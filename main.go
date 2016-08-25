package issuesapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go-goon"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issuesapp/common"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

type Options struct {
	RepoSpec func(req *http.Request) issues.RepoSpec
	BaseURI  func(req *http.Request) string
	HeadPre  template.HTML
	BodyPre  string // An html/template definition of "body-pre" template.

	// TODO.
	BaseState func(req *http.Request) BaseState
}

type handler struct {
	http.Handler

	is issues.Service
	us users.Service

	Options
}

// New returns an issues app http.Handler using given services and options.
// If usersService is nil, then there is no way to have an authenticated user.
func New(service issues.Service, usersService users.Service, opt Options) http.Handler {
	handler := &handler{
		is:      service,
		us:      usersService,
		Options: opt,
	}

	err := handler.loadTemplates(users.User{})
	if err != nil {
		log.Fatalln("loadTemplates:", err)
	}

	h := http.NewServeMux()
	h.HandleFunc("/mock/", handler.mockHandler)
	r := mux.NewRouter()
	// TODO: Make redirection work.
	//r.StrictSlash(true) // THINK: Can't use this due to redirect not taking baseURI into account.
	r.HandleFunc("/", handler.issuesHandler).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}", handler.issueHandler).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/edit", handler.postEditIssueHandler).Methods("POST")
	r.HandleFunc("/{id:[0-9]+}/comment", handler.postCommentHandler).Methods("POST")
	r.HandleFunc("/{id:[0-9]+}/comment/{commentID:[0-9]+}", handler.postEditCommentHandler).Methods("POST")
	r.HandleFunc("/{id:[0-9]+}/comment/{commentID:[0-9]+}/react", handler.postToggleReactionHandler).Methods("POST")
	r.HandleFunc("/new", handler.createIssueHandler).Methods("GET")
	r.HandleFunc("/new", handler.postCreateIssueHandler).Methods("POST")
	h.Handle("/", r)
	fileServer := httpgzip.FileServer(Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	h.Handle("/assets/", fileServer)
	h.Handle("/assets/octicons/", http.StripPrefix("/assets", fileServer))
	h.Handle("/assets/gfm/", http.StripPrefix("/assets", fileServer))

	handler.Handler = h
	return handler
}

var t *template.Template

func (h *handler) loadTemplates(currentUser users.User) error {
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
		"reactionPosition": func(emojiID reactions.EmojiID) string { return reactions.Position(":" + string(emojiID) + ":") },
		// THINK.
		"containsCurrentUser": func(users []users.User) bool {
			if currentUser.ID == 0 {
				return false
			}
			for _, u := range users {
				if u.ID == currentUser.ID {
					return true
				}
			}
			return false
		},
		"reactionTooltip": func(reaction reactions.Reaction) string {
			var users string
			for i, u := range reaction.Users {
				if i != 0 {
					if i < len(reaction.Users)-1 {
						users += ", "
					} else {
						users += " and "
					}
				}
				if currentUser.ID != 0 && u.ID == currentUser.ID {
					if i == 0 {
						users += "You"
					} else {
						users += "you"
					}
				} else {
					users += u.Login
				}
			}
			// TODO: Handle when there are too many users and their details are left out by backend.
			//       Count them and add "and N others" here.
			return fmt.Sprintf("%v reacted with :%v:.", users, reaction.Reaction)
		},
	})
	t, err = vfstemplate.ParseGlob(Assets, t, "/assets/*.tmpl")
	if err != nil {
		return err
	}
	t, err = t.New("body-pre").Parse(h.BodyPre) // HACK: This is a temporary experiment.
	return err
}

type BaseState struct {
	ctx  context.Context
	req  *http.Request
	vars map[string]string

	repoSpec issues.RepoSpec
	HeadPre  template.HTML

	is issues.Service

	common.State
}

func (h *handler) baseState(req *http.Request) (BaseState, error) {
	b := h.BaseState(req)
	b.ctx = req.Context()
	b.req = req
	b.vars = mux.Vars(req)
	b.repoSpec = h.RepoSpec(req)
	b.HeadPre = h.HeadPre

	b.is = h.is

	if h.us == nil {
		// No user service provided, so there can never be an authenticated user.
		b.CurrentUser = users.User{}
	} else if user, err := h.us.GetAuthenticated(b.ctx); err == nil {
		b.CurrentUser = user
	} else {
		return BaseState{}, err
	}

	return b, nil
}

type state struct {
	BaseState
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
	return s.is.List(s.ctx, s.repoSpec, opt)
}

func (s state) OpenCount() (uint64, error) {
	return s.is.Count(s.ctx, s.repoSpec, issues.IssueListOptions{State: issues.StateFilter(issues.OpenState)})
}

func (s state) ClosedCount() (uint64, error) {
	return s.is.Count(s.ctx, s.repoSpec, issues.IssueListOptions{State: issues.StateFilter(issues.ClosedState)})
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func (s state) Issue() (issues.Issue, error) {
	return s.is.Get(s.ctx, s.repoSpec, uint64(mustAtoi(s.vars["id"])))
}

func (s state) Items() ([]issueItem, error) {
	cs, err := s.is.ListComments(s.ctx, s.repoSpec, uint64(mustAtoi(s.vars["id"])), nil)
	if err != nil {
		return nil, err
	}
	es, err := s.is.ListEvents(s.ctx, s.repoSpec, uint64(mustAtoi(s.vars["id"])), nil)
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

func (h *handler) issuesHandler(w http.ResponseWriter, req *http.Request) {
	if err := h.loadTemplates(users.User{}); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseState, err := h.baseState(req)
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

func (h *handler) issueHandler(w http.ResponseWriter, req *http.Request) {
	baseState, err := h.baseState(req)
	if err != nil {
		log.Println("baseState:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// THINK.
	if err := h.loadTemplates(baseState.CurrentUser); err != nil {
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

func (h *handler) createIssueHandler(w http.ResponseWriter, req *http.Request) {
	if err := h.loadTemplates(users.User{}); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseState, err := h.baseState(req)
	if err != nil {
		log.Println("baseState:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if baseState.CurrentUser.ID == 0 {
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

func (h *handler) postCreateIssueHandler(w http.ResponseWriter, req *http.Request) {
	baseURI := h.BaseURI(req)
	repoSpec := h.RepoSpec(req)

	var issue issues.Issue
	err := json.NewDecoder(req.Body).Decode(&issue)
	if err != nil {
		log.Println("json.Decode:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	issue, err = h.is.Create(req.Context(), repoSpec, issue)
	if err != nil {
		log.Println("is.Create:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s/%d", baseURI, issue.ID)
}

func (h *handler) postEditIssueHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(req)
	repoSpec := h.RepoSpec(req)

	var ir issues.IssueRequest
	err := json.Unmarshal([]byte(req.PostForm.Get("value")), &ir)
	if err != nil {
		log.Println("postEditIssueHandler: json.Unmarshal value:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	issue, events, err := h.is.Edit(req.Context(), repoSpec, uint64(mustAtoi(vars["id"])), ir)
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

func (h *handler) postCommentHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(req)
	repoSpec := h.RepoSpec(req)

	comment := issues.Comment{
		Body: req.PostForm.Get("value"),
	}

	issueID := uint64(mustAtoi(vars["id"]))
	comment, err := h.is.CreateComment(req.Context(), repoSpec, issueID, comment)
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

func (h *handler) postEditCommentHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(req)
	repoSpec := h.RepoSpec(req)

	body := req.PostForm.Get("value")
	cr := issues.CommentRequest{
		ID:   uint64(mustAtoi(vars["commentID"])),
		Body: &body,
	}

	_, err := h.is.EditComment(req.Context(), repoSpec, uint64(mustAtoi(vars["id"])), cr)
	if err != nil {
		log.Println("is.EditComment:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handler) postToggleReactionHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(req)
	repoSpec := h.RepoSpec(req)

	reaction := reactions.EmojiID(req.PostForm.Get("reaction"))
	cr := issues.CommentRequest{
		ID:       uint64(mustAtoi(vars["commentID"])),
		Reaction: &reaction,
	}

	comment, err := h.is.EditComment(req.Context(), repoSpec, uint64(mustAtoi(vars["id"])), cr)
	if os.IsPermission(err) { // TODO: Move this to a higher level (and upate all other similar code too).
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("is.EditComment:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Deduplicate.
	// {{template "reactions" .Reactions}}{{template "new-reaction" .ID}}
	err = t.ExecuteTemplate(w, "reactions", comment.Reactions)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.ExecuteTemplate(w, "new-reaction", comment.ID)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
