package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/go-github/github"
	"github.com/gopherjs/gopherpen/common"
	"github.com/gopherjs/gopherpen/issues"
	"github.com/gorilla/mux"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go-goon"
	"github.com/shurcooL/go/gzip_file_server"
	"github.com/shurcooL/go/vfs/httpfs/html/vfstemplate"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	ghissues "github.com/gopherjs/gopherpen/issues/github"
	//fsissues "github.com/gopherjs/gopherpen/issues/fs"
)

var httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")

var t *template.Template

func loadTemplates() error {
	var err error
	t = template.New("").Funcs(template.FuncMap{
		"dump": func(v interface{}) string { return goon.Sdump(v) },
		"jsonfmt": func(v interface{}) (string, error) {
			b, err := json.MarshalIndent(v, "", "\t")
			return string(b), err
		},
		"reltime": humanize.Time,
		"gfm":     func(s string) template.HTML { return template.HTML(github_flavored_markdown.Markdown([]byte(s))) },
		"event":   func(e issues.Event) event { return event{e} },
	})
	t, err = vfstemplate.ParseGlob(assets, t, "/assets/*.tmpl")
	return err
}

type state struct {
	BaseState
}

type BaseState struct {
	ctx  context.Context
	req  *http.Request
	vars map[string]string

	common.State
}

func baseState(req *http.Request) BaseState {
	ctx := context.TODO()
	return BaseState{
		ctx:  ctx,
		req:  req,
		vars: mux.Vars(req),

		State: common.State{
			//BaseURI:   pctx.BaseURI(ctx),
			ReqPath: req.URL.Path,
			//CSRFToken: pctx.CSRFToken(ctx),
		},
	}
}

func (s state) Tabs() (template.HTML, error) {
	return tabs(&s, s.ReqPath, s.req.URL.RawQuery)
}

func (s state) Issues() ([]issues.Issue, error) {
	var opt issues.IssueListOptions
	if selectedTab := s.req.URL.Query().Get(queryKeyState); selectedTab == string(issues.ClosedState) {
		opt.State = issues.ClosedState
	}
	return is.List(s.ctx, issues.RepoSpec{Owner: s.vars["owner"], Repo: s.vars["repo"]}, opt)
}

func (s state) OpenCount() (uint64, error) {
	return is.Count(s.ctx, issues.RepoSpec{Owner: s.vars["owner"], Repo: s.vars["repo"]}, issues.IssueListOptions{State: issues.OpenState})
}

func (s state) ClosedCount() (uint64, error) {
	return is.Count(s.ctx, issues.RepoSpec{Owner: s.vars["owner"], Repo: s.vars["repo"]}, issues.IssueListOptions{State: issues.ClosedState})
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func (s state) Issue() (interface{}, error) {
	return is.Get(s.ctx, issues.RepoSpec{Owner: s.vars["owner"], Repo: s.vars["repo"]}, uint64(mustAtoi(s.vars["id"])))
}

func (s state) Comments() (interface{}, error) {
	return is.ListComments(s.ctx, issues.RepoSpec{Owner: s.vars["owner"], Repo: s.vars["repo"]}, uint64(mustAtoi(s.vars["id"])), nil)
}

func (s state) Events() (interface{}, error) {
	return is.ListEvents(s.ctx, issues.RepoSpec{Owner: s.vars["owner"], Repo: s.vars["repo"]}, uint64(mustAtoi(s.vars["id"])), nil)
}

func (s state) Items() (interface{}, error) {
	cs, err := is.ListComments(s.ctx, issues.RepoSpec{Owner: s.vars["owner"], Repo: s.vars["repo"]}, uint64(mustAtoi(s.vars["id"])), nil)
	if err != nil {
		return nil, err
	}
	es, err := is.ListEvents(s.ctx, issues.RepoSpec{Owner: s.vars["owner"], Repo: s.vars["repo"]}, uint64(mustAtoi(s.vars["id"])), nil)
	if err != nil {
		return nil, err
	}
	var items []issueItem
	for _, comment := range cs {
		items = append(items, issueItem{comment})
	}
	for _, event := range es {
		items = append(items, issueItem{event})
	}
	sort.Sort(byCreatedAt(items))
	return items, nil
}

//var is issues.Service = fsissues.NewService()
var is issues.Service = ghissues.NewService(gh)

var gh = github.NewClient(oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ""})))

func (s state) Comment() issues.Comment {
	return issues.Comment{
		User:      is.CurrentUser(),
		CreatedAt: time.Unix(1443244474, 0).UTC(),
		Body:      "I've resolved this in 4387efb. Please re-open or leave a comment if there's still room for improvement here.",
	}
}

func (s state) Event() issues.Event {
	return issues.Event{
		Actor:     is.CurrentUser(),
		CreatedAt: time.Now(),
		Type:      issues.Closed,
	}
}

// Apparently needed for "new-comment" component, etc.
func (state) CurrentUser() issues.User {
	return is.CurrentUser()
}

func mainHandler(w http.ResponseWriter, req *http.Request) {
	if err := loadTemplates(); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := path.Base(req.URL.Path)
	state := state{
		BaseState: baseState(req),
	}
	err := t.ExecuteTemplate(w, tmpl+".tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func issuesHandler(w http.ResponseWriter, req *http.Request) {
	if err := loadTemplates(); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	state := state{
		BaseState: baseState(req),
	}
	err := t.ExecuteTemplate(w, "issues.html.tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func issueHandler(w http.ResponseWriter, req *http.Request) {
	if err := loadTemplates(); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	state := state{
		BaseState: baseState(req),
	}
	err := t.ExecuteTemplate(w, "issue.html.tmpl", &state)
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
		_ = repoRevSpec
	}
	goon.DumpExpr(pctx.RepoRevSpec(ctx))*/

	//io.WriteString(w, req.PostForm.Get("value"))
}
func debugIssueHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	ie, _, err := gh.Issues.ListIssueEvents(vars["owner"], vars["repo"], mustAtoi(vars["id"]), nil)
	if err != nil {
		log.Println("ListIssueEvents:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, goon.SdumpExpr(ie))
}

func createIssueHandler(w http.ResponseWriter, req *http.Request) {
	if err := loadTemplates(); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	state := state{
		BaseState: baseState(req),
	}
	err := t.ExecuteTemplate(w, "new-issue.html.tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func postCreateIssueHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.TODO()
	vars := mux.Vars(req)

	issue := issues.Issue{
		Title: req.PostForm.Get("title"),
		Comment: issues.Comment{
			Body: req.PostForm.Get("body"),
		},
	}

	issue, err := is.Create(ctx, issues.RepoSpec{Owner: vars["owner"], Repo: vars["repo"]}, issue)
	if err != nil {
		log.Println("is.Create:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Make this work...
	//http.Redirect(w, req, fmt.Sprintf("%s/issues/%d", baseURI, issue.ID), http.StatusSeeOther)
}

func postEditIssueHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.TODO()
	vars := mux.Vars(req)

	var ir issues.IssueRequest
	err := json.Unmarshal([]byte(req.PostForm.Get("value")), &ir)
	if err != nil {
		log.Println("postEditIssueHandler json.Unmarshal:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	issue, err := is.Edit(ctx, issues.RepoSpec{Owner: vars["owner"], Repo: vars["repo"]}, uint64(mustAtoi(vars["id"])), ir)
	if err != nil {
		log.Println("is.Edit:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Move to right place?
	issueEvent := issues.Event{
		Actor:     is.CurrentUser(),
		CreatedAt: time.Now(),
	}
	switch {
	case ir.State != nil && *ir.State == issues.OpenState:
		issueEvent.Type = issues.Reopened
	case ir.State != nil && *ir.State == issues.ClosedState:
		issueEvent.Type = issues.Closed
	case ir.Title != nil:
		issueEvent.Type = issues.Renamed
		issueEvent.Rename = &issues.Rename{
			From: "TODO",
			To:   *ir.Title,
		}
	}

	err = func(w io.Writer, issue issues.Issue) error {
		var resp = make(url.Values)

		var buf bytes.Buffer
		err := t.ExecuteTemplate(&buf, "issue-badge", issue.State)
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
		buf.Reset()
		err = t.ExecuteTemplate(&buf, "event", issueEvent)
		if err != nil {
			return err
		}
		resp.Set("new-event", buf.String())

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.TODO()
	vars := mux.Vars(req)

	comment := issues.Comment{
		Body: req.PostForm.Get("value"),
	}

	comment, err := is.CreateComment(ctx, issues.RepoSpec{Owner: vars["owner"], Repo: vars["repo"]}, uint64(mustAtoi(vars["id"])), comment)
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

func main() {
	flag.Parse()

	err := loadTemplates()
	if err != nil {
		log.Fatalln("loadTemplates:", err)
	}

	http.HandleFunc("/pages/", mainHandler)
	r := mux.NewRouter()
	r.HandleFunc("/github.com/{owner}/{repo}/issues", issuesHandler).Methods("GET")
	r.HandleFunc("/github.com/{owner}/{repo}/issues/{id:[0-9]+}", issueHandler).Methods("GET")
	r.HandleFunc("/github.com/{owner}/{repo}/issues/{id:[0-9]+}/comment", postCommentHandler).Methods("POST")
	r.HandleFunc("/github.com/{owner}/{repo}/issues/{id:[0-9]+}/edit", postEditIssueHandler).Methods("POST")
	r.HandleFunc("/github.com/{owner}/{repo}/issues/new", createIssueHandler).Methods("GET")
	r.HandleFunc("/github.com/{owner}/{repo}/issues/new", postCreateIssueHandler).Methods("POST")
	http.Handle("/", r)
	http.Handle("/assets/", gzip_file_server.New(assets))
	http.HandleFunc("/debug", debugHandler)
	r.HandleFunc("/github.com/{owner}/{repo}/issues/{id:[0-9]+}/debug", debugIssueHandler).Methods("GET")

	printServingAt(*httpFlag)
	err = http.ListenAndServe(*httpFlag, nil)
	if err != nil {
		log.Fatalln("ListenAndServe:", err)
	}
}

func printServingAt(addr string) {
	hostPort := addr
	if strings.HasPrefix(hostPort, ":") {
		hostPort = "localhost" + hostPort
	}
	fmt.Printf("serving at http://%s/\n", hostPort)
}
