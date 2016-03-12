package issuesapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go-goon"
	"github.com/shurcooL/go/gzip_file_server"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/shurcooL/issuesapp/common"
	"github.com/shurcooL/reactions"
	"golang.org/x/net/context"
	"src.sourcegraph.com/apps/tracker/issues"
)

type Options struct {
	Context  func(req *http.Request) context.Context
	RepoSpec func(req *http.Request) issues.RepoSpec
	BaseURI  func(req *http.Request) string
	HeadPre  template.HTML

	// TODO.
	BaseState func(req *http.Request) BaseState
}

type handler struct {
	http.Handler

	Options
}

// New returns an issues app http.Handler using given service and options.
func New(service issues.Service, opt Options) http.Handler {
	err := loadTemplates(nil)
	if err != nil {
		log.Fatalln("loadTemplates:", err)
	}

	// TODO: Move into handler?
	is = service

	h := http.NewServeMux()
	h.HandleFunc("/mock/", mockHandler)
	r := mux.NewRouter()
	// TODO: Make redirection work.
	//r.StrictSlash(true) // THINK: Can't use this due to redirect not taking baseURI into account.
	r.HandleFunc("/", issuesHandler).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}", issueHandler).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/edit", postEditIssueHandler).Methods("POST")
	r.HandleFunc("/{id:[0-9]+}/comment", postCommentHandler).Methods("POST")
	r.HandleFunc("/{id:[0-9]+}/comment/{commentID:[0-9]+}", postEditCommentHandler).Methods("POST")
	r.HandleFunc("/new", createIssueHandler).Methods("GET")
	r.HandleFunc("/new", postCreateIssueHandler).Methods("POST")
	h.Handle("/", r)
	fileServer := gzip_file_server.New(Assets)
	h.Handle("/assets/", fileServer)
	h.Handle("/assets/octicons/", http.StripPrefix("/assets/", fileServer))
	h.HandleFunc("/debug", debugHandler)

	globalHandler = &handler{
		Options: opt,
		Handler: h,
	}
	return globalHandler
}

// TODO: Refactor to avoid global.
var globalHandler *handler

var t *template.Template

func loadTemplates(currentUser *issues.User) error {
	var err error
	t = template.New("").Funcs(template.FuncMap{
		"dump": func(v interface{}) string { return goon.Sdump(v) },
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
		"jsonfmt": func(v interface{}) (string, error) {
			b, err := json.MarshalIndent(v, "", "\t")
			return string(b), err
		},
		"reltime":          humanize.Time,
		"gfm":              func(s string) template.HTML { return template.HTML(github_flavored_markdown.Markdown([]byte(s))) },
		"reactionPosition": func(emojiID issues.EmojiID) string { return reactions.Position(":" + string(emojiID) + ":") },
		// THINK.
		"containsCurrentUser": func(users []issues.User) bool {
			if currentUser == nil {
				return false
			}
			if len(users) >= 2 { // HACK.
				return true
			}
			for _, u := range users {
				if u.ID == currentUser.ID {
					return true
				}
			}
			return false
		},
	})
	t, err = vfstemplate.ParseGlob(Assets, t, "/assets/*.tmpl")
	return err
}

type state struct {
	BaseState
}

type BaseState struct {
	ctx  context.Context
	req  *http.Request
	vars map[string]string

	repoSpec issues.RepoSpec
	HeadPre  template.HTML

	common.State
}

func baseState(req *http.Request) (BaseState, error) {
	b := globalHandler.BaseState(req)
	b.ctx = globalHandler.Context(req)
	b.req = req
	b.vars = mux.Vars(req)
	b.repoSpec = globalHandler.RepoSpec(req)
	b.HeadPre = globalHandler.HeadPre

	if u, err := is.CurrentUser(b.ctx); err != nil {
		return BaseState{}, err
	} else {
		b.CurrentUser = u
	}

	return b, nil
}

func (s state) Tab() (issues.State, error) {
	return tab(s.req.URL.Query())
}

func (s state) Tabs() (template.HTML, error) {
	return tabs(&s, s.BaseURI+s.ReqPath, s.req.URL.RawQuery)
}

func (s state) Issues() ([]issues.Issue, error) {
	var opt issues.IssueListOptions
	switch selectedTab := s.req.URL.Query().Get(queryKeyState); selectedTab {
	case "": // Default. TODO: Make this cleaner.
		opt.State = issues.StateFilter(issues.OpenState)
	case string(issues.ClosedState):
		opt.State = issues.StateFilter(issues.ClosedState)
	}
	return is.List(s.ctx, s.repoSpec, opt)
}

func (s state) OpenCount() (uint64, error) {
	return is.Count(s.ctx, s.repoSpec, issues.IssueListOptions{State: issues.StateFilter(issues.OpenState)})
}

func (s state) ClosedCount() (uint64, error) {
	return is.Count(s.ctx, s.repoSpec, issues.IssueListOptions{State: issues.StateFilter(issues.ClosedState)})
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func (s state) Issue() (issues.Issue, error) {
	return is.Get(s.ctx, s.repoSpec, uint64(mustAtoi(s.vars["id"])))
}

func (s state) Items() ([]issueItem, error) {
	cs, err := is.ListComments(s.ctx, s.repoSpec, uint64(mustAtoi(s.vars["id"])), nil)
	if err != nil {
		return nil, err
	}
	es, err := is.ListEvents(s.ctx, s.repoSpec, uint64(mustAtoi(s.vars["id"])), nil)
	if err != nil {
		return nil, err
	}
	var items []issueItem
	for _, comment := range cs {
		items = append(items, issueItem{comment})
	}
	for _, e := range es {
		items = append(items, issueItem{event{e}})
	}
	sort.Sort(byCreatedAtID(items))
	return items, nil
}

var is issues.Service

func issuesHandler(w http.ResponseWriter, req *http.Request) {
	if err := loadTemplates(nil); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseState, err := baseState(req)
	if err != nil {
		log.Println("baseState:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	state := state{
		BaseState: baseState,
	}
	err = t.ExecuteTemplate(w, "issues.html.tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func issueHandler(w http.ResponseWriter, req *http.Request) {
	baseState, err := baseState(req)
	if err != nil {
		log.Println("baseState:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// THINK.
	if err := loadTemplates(baseState.CurrentUser); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	state := state{
		BaseState: baseState,
	}
	err = t.ExecuteTemplate(w, "issue.html.tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func debugHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("debugHandler:", req.URL.Path)

	/*ctx := putil.Context(req)
	if repoRevSpec, ok := pctx.RepoRevSpec(ctx); ok {
		goon.DumpExpr(issues.RepoSpec{URI: repoRevSpec.URI})
	}
	goon.DumpExpr(pctx.RepoRevSpec(ctx))*/

	//io.WriteString(w, req.PostForm.Get("value"))
}

func createIssueHandler(w http.ResponseWriter, req *http.Request) {
	if err := loadTemplates(nil); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseState, err := baseState(req)
	if err != nil {
		log.Println("baseState:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if baseState.CurrentUser == nil {
		http.Error(w, "this page requires an authenticated user", http.StatusUnauthorized)
		return
	}
	state := state{
		BaseState: baseState,
	}
	err = t.ExecuteTemplate(w, "new-issue.html.tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func postCreateIssueHandler(w http.ResponseWriter, req *http.Request) {
	ctx := globalHandler.Context(req)
	baseURI := globalHandler.BaseURI(req)
	repoSpec := globalHandler.RepoSpec(req)

	var issue issues.Issue
	err := json.NewDecoder(req.Body).Decode(&issue)
	if err != nil {
		log.Println("json.Decode:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	issue, err = is.Create(ctx, repoSpec, issue)
	if err != nil {
		log.Println("is.Create:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s/%d", baseURI, issue.ID)
}

func postEditIssueHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := globalHandler.Context(req)
	vars := mux.Vars(req)
	repoSpec := globalHandler.RepoSpec(req)

	var ir issues.IssueRequest
	err := json.Unmarshal([]byte(req.PostForm.Get("value")), &ir)
	if err != nil {
		log.Println("postEditIssueHandler: json.Unmarshal value:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	issue, events, err := is.Edit(ctx, repoSpec, uint64(mustAtoi(vars["id"])), ir)
	if err != nil {
		log.Println("is.Edit:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = func(w io.Writer, issue issues.Issue) error {
		var resp = make(url.Values)

		var buf bytes.Buffer
		err := t.ExecuteTemplate(&buf, "issue-state-badge", issue)
		if err != nil {
			return err
		}
		resp.Set("issue-state-badge", buf.String())

		buf.Reset()
		err = t.ExecuteTemplate(&buf, "toggle-button", issue.State)
		if err != nil {
			return err
		}
		resp.Set("issue-toggle-button", buf.String())

		for _, e := range events {
			buf.Reset()
			err = t.ExecuteTemplate(&buf, "event", event{e})
			if err != nil {
				return err
			}
			resp.Add("new-event", buf.String())
		}

		_, err = io.WriteString(w, resp.Encode())
		return err
	}(w, issue)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func postCommentHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := globalHandler.Context(req)
	vars := mux.Vars(req)
	repoSpec := globalHandler.RepoSpec(req)

	comment := issues.Comment{
		Body: req.PostForm.Get("value"),
	}

	issueID := uint64(mustAtoi(vars["id"]))
	comment, err := is.CreateComment(ctx, repoSpec, issueID, comment)
	if err != nil {
		log.Println("is.CreateComment:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.ExecuteTemplate(w, "comment", comment)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func postEditCommentHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := globalHandler.Context(req)
	vars := mux.Vars(req)
	repoSpec := globalHandler.RepoSpec(req)

	body := req.PostForm.Get("value")
	cr := issues.CommentRequest{
		ID:   uint64(mustAtoi(vars["commentID"])),
		Body: &body,
	}

	_, err := is.EditComment(ctx, repoSpec, uint64(mustAtoi(vars["id"])), cr)
	if err != nil {
		log.Println("is.EditComment:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
