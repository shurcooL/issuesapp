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
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go-goon"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issuesapp/assets"
	"github.com/shurcooL/issuesapp/common"
	"github.com/shurcooL/issuesapp/component"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/reactions"
	reactionscomponent "github.com/shurcooL/reactions/component"
	"github.com/shurcooL/users"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "github.com/shurcooL/issuesapp context value " + k.name }

// RepoSpecContextKey is a context key for the request's issues.RepoSpec.
// That value specifies which repo the issues are to be displayed for.
// The associated value will be of type issues.RepoSpec.
var RepoSpecContextKey = &contextKey{"RepoSpec"}

// BaseURIContextKey is a context key for the request's base URI.
// That value specifies the base URI prefix to use for all absolute URLs.
// The associated value will be of type string.
var BaseURIContextKey = &contextKey{"BaseURI"}

type Options struct {
	Notifications    notifications.Service // If not nil, issues containing unread notifications are highlighted.
	DisableReactions bool                  // Disable all support for displaying and toggling reactions.

	HeadPre template.HTML
	BodyPre string // An html/template definition of "body-pre" template.

	// BodyTop provides components to include on top of <body> of page rendered for req. It can be nil.
	BodyTop func(req *http.Request) ([]htmlg.Component, error)
}

type handler struct {
	http.Handler

	is issues.Service
	us users.Service

	Options
}

// TODO: Find a better way for issuesapp to be able to ensure registration of a top-level route:
//
// 	emojisHandler := httpgzip.FileServer(emojis.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
// 	http.Handle("/emojis/", http.StripPrefix("/emojis", emojisHandler))
//
// So that it can depend on it.

// New returns an issues app http.Handler using given services and options.
// If usersService is nil, then there is no way to have an authenticated user.
// Emojis image data is expected to be available at /emojis/emojis.png, unless
// opt.DisableReactions is true.
//
// In order to serve HTTP requests, the returned http.Handler expects each incoming
// request to have 2 parameters provided to it via RepoSpecContextKey and BaseURIContextKey
// context keys. For example:
//
// 	issuesApp := issuesapp.New(...)
//
// 	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
// 		req = req.WithContext(context.WithValue(req.Context(), issuesapp.RepoSpecContextKey, issues.RepoSpec{...}))
// 		req = req.WithContext(context.WithValue(req.Context(), issuesapp.BaseURIContextKey, string(...)))
// 		issuesApp.ServeHTTP(w, req)
// 	})
//
// An HTTP API must be available (currently, only EditComment endpoint is used for reacting):
//
// 	// Register HTTP API endpoints.
// 	apiHandler := httphandler.Issues{Issues: service}
// 	http.Handle(httproute.List, errorHandler(apiHandler.List))
// 	http.Handle(httproute.Count, errorHandler(apiHandler.Count))
// 	http.Handle(httproute.ListComments, errorHandler(apiHandler.ListComments))
// 	http.Handle(httproute.EditComment, errorHandler(apiHandler.EditComment))
func New(service issues.Service, usersService users.Service, opt Options) http.Handler {
	handler := &handler{
		is:      service,
		us:      usersService,
		Options: opt,
	}

	err := handler.loadTemplates(common.State{})
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
	r.HandleFunc("/new", handler.createIssueHandler).Methods("GET")
	r.HandleFunc("/new", handler.postCreateIssueHandler).Methods("POST")
	h.Handle("/", r)
	assetsFileServer := httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	h.Handle("/assets/", assetsFileServer)
	h.Handle("/assets/octicons/", http.StripPrefix("/assets", assetsFileServer))
	h.Handle("/assets/gfm/", http.StripPrefix("/assets", assetsFileServer))

	handler.Handler = h
	return handler
}

var t *template.Template

func (h *handler) loadTemplates(state common.State) error {
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
		"equalUsers": func(a, b users.User) bool {
			return a.UserSpec == b.UserSpec
		},
		"reactableID": func(commentID uint64) string {
			return fmt.Sprintf("%d/%d", state.IssueID, commentID)
		},
		"reactionsBar": func(reactions []reactions.Reaction, reactableID string) htmlg.Component {
			return reactionscomponent.ReactionsBar{
				Reactions:   reactions,
				CurrentUser: state.CurrentUser,
				ID:          reactableID,
			}
		},
		"newReaction": func(reactableID string) htmlg.Component {
			return reactionscomponent.NewReaction{
				ReactableID: reactableID,
			}
		},
		"state": func() common.State { return state },

		"render": func(c htmlg.Component) template.HTML {
			return template.HTML(htmlg.Render(c.Render()...))
		},
		"issueBadge": func(state issues.State) htmlg.Component {
			return component.IssueBadge{State: state}
		},
		"issueIcon": func(state issues.State) htmlg.Component {
			return component.IssueIcon{State: state}
		},
		"time": func(t time.Time) htmlg.Component { return component.Time{Time: t} },
		"user": func(u users.User) htmlg.Component { return component.User{User: u} },
	})
	var err error
	t, err = vfstemplate.ParseGlob(assets.Assets, t, "/assets/*.tmpl")
	if err != nil {
		return err
	}
	t, err = t.New("body-pre").Parse(h.BodyPre)
	return err
}

func (h *handler) state(req *http.Request) (state, error) {
	vars := mux.Vars(req)
	repoSpec, ok := req.Context().Value(RepoSpecContextKey).(issues.RepoSpec)
	if !ok {
		return state{}, fmt.Errorf("request to %v doesn't have issuesapp.RepoSpecContextKey context key set", req.URL.Path)
	}
	baseURI, ok := req.Context().Value(BaseURIContextKey).(string)
	if !ok {
		return state{}, fmt.Errorf("request to %v doesn't have issuesapp.BaseURIContextKey context key set", req.URL.Path)
	}

	// TODO: Caller still does a lot of work outside to calculate req.URL.Path by
	//       subtracting BaseURI from full original req.URL.Path. We should be able
	//       to compute it here internally by using req.RequestURI and BaseURI.
	reqPath := req.URL.Path
	if reqPath == "/" {
		reqPath = "" // This is needed so that absolute URL for root view, i.e., /issues, is "/issues" and not "/issues/" because of "/issues" + "/".
	}
	issueID, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		issueID = 0
	}
	b := state{
		State: common.State{
			BaseURI:  baseURI,
			ReqPath:  reqPath,
			RepoSpec: repoSpec,
			IssueID:  issueID,
		},
	}
	b.req = req
	b.HeadPre = h.HeadPre
	if h.BodyTop != nil {
		c, err := h.BodyTop(req)
		if err != nil {
			return state{}, err
		}
		var buf bytes.Buffer
		err = htmlg.RenderComponents(&buf, c...)
		if err != nil {
			return state{}, err
		}
		b.BodyTop = template.HTML(buf.String())
	}

	b.is = h.is

	b.notifications = h.Options.Notifications

	b.DisableReactions = h.Options.DisableReactions

	if h.us == nil {
		// No user service provided, so there can never be an authenticated user.
		b.CurrentUser = users.User{}
	} else if user, err := h.us.GetAuthenticated(req.Context()); err == nil {
		b.CurrentUser = user
	} else {
		return state{}, err
	}

	return b, nil
}

type state struct {
	req *http.Request

	HeadPre template.HTML
	BodyTop template.HTML

	is issues.Service

	notifications notifications.Service

	common.State
}

func (s state) Tab() (issues.State, error) {
	return tab(s.req.URL.Query())
}

func (s state) Tabs() (template.HTML, error) {
	return tabs(&s, s.BaseURI+s.ReqPath, s.req.URL.RawQuery)
}

func (s state) Issues() ([]issue, error) {
	var opt issues.IssueListOptions
	switch selectedTab := s.req.URL.Query().Get(queryKeyState); selectedTab {
	case "": // Default. TODO: Make this cleaner.
		opt.State = issues.StateFilter(issues.OpenState)
	case string(issues.ClosedState):
		opt.State = issues.StateFilter(issues.ClosedState)
	}
	is, err := s.is.List(s.req.Context(), s.RepoSpec, opt)
	if err != nil {
		return nil, err
	}
	var dis []issue
	for _, i := range is {
		dis = append(dis, issue{Issue: i})
	}
	dis = s.augmentUnread(dis)
	return dis, nil
}

func (s state) augmentUnread(dis []issue) []issue {
	if s.notifications == nil {
		return dis
	}

	tt, ok := s.is.(interface {
		ThreadType() string
	})
	if !ok {
		log.Println("augmentUnread: issues service doesn't implement ThreadType")
		return dis
	}
	threadType := tt.ThreadType()

	if s.CurrentUser.ID == 0 {
		// Unauthenticated user cannot have any unread issues.
		return dis
	}

	// TODO: Consider starting to do this in background in parallel with s.is.List.
	ns, err := s.notifications.List(s.req.Context(), notifications.ListOptions{
		Repo: &notifications.RepoSpec{URI: s.RepoSpec.URI},
	})
	if err != nil {
		log.Println("augmentUnread: failed to s.notifications.List:", err)
		return dis
	}

	unreadThreads := make(map[uint64]struct{}) // Set of unread thread IDs.
	for _, n := range ns {
		if n.AppID != threadType { // Assumes RepoSpec matches because we filtered via notifications.ListOptions.
			continue
		}
		unreadThreads[n.ThreadID] = struct{}{}
	}

	for i, di := range dis {
		_, unread := unreadThreads[di.ID]
		dis[i].Unread = unread
	}
	return dis
}

func (s state) OpenCount() (uint64, error) {
	return s.is.Count(s.req.Context(), s.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(issues.OpenState)})
}

func (s state) ClosedCount() (uint64, error) {
	return s.is.Count(s.req.Context(), s.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(issues.ClosedState)})
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func (s state) Issue() (issues.Issue, error) {
	return s.is.Get(s.req.Context(), s.RepoSpec, s.IssueID)
}

func (s state) Items() ([]issueItem, error) {
	cs, err := s.is.ListComments(s.req.Context(), s.RepoSpec, s.IssueID, nil)
	if err != nil {
		return nil, err
	}
	es, err := s.is.ListEvents(s.req.Context(), s.RepoSpec, s.IssueID, nil)
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

// ForceIssuesApp reports whether "issuesapp" query is true.
// This is a temporary solution for external users to use when overriding templates.
// It's going to go away eventually, so its use is discouraged.
func (s state) ForceIssuesApp() bool {
	forceIssuesApp, _ := strconv.ParseBool(s.req.URL.Query().Get("issuesapp"))
	return forceIssuesApp
}

func (h *handler) issuesHandler(w http.ResponseWriter, req *http.Request) {
	if err := h.loadTemplates(common.State{}); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	state, err := h.state(req)
	if err != nil {
		log.Println("state:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = t.ExecuteTemplate(w, "issues.html.tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handler) issueHandler(w http.ResponseWriter, req *http.Request) {
	state, err := h.state(req)
	if err != nil {
		log.Println("state:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// THINK.
	if err := h.loadTemplates(state.State); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	err = t.ExecuteTemplate(&buf, "issue.html.tmpl", &state)
	if err != nil && (strings.Contains(err.Error(), "no such file or directory") || strings.Contains(err.Error(), "does not exist")) { // TODO: Better error handling.
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.Copy(w, &buf)
}

func (h *handler) createIssueHandler(w http.ResponseWriter, req *http.Request) {
	if err := h.loadTemplates(common.State{}); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	state, err := h.state(req)
	if err != nil {
		log.Println("state:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if state.CurrentUser.ID == 0 {
		http.Error(w, "this page requires an authenticated user", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = t.ExecuteTemplate(w, "new-issue.html.tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handler) postCreateIssueHandler(w http.ResponseWriter, req *http.Request) {
	baseURI := req.Context().Value(BaseURIContextKey).(string)
	repoSpec := req.Context().Value(RepoSpecContextKey).(issues.RepoSpec)

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
	repoSpec := req.Context().Value(RepoSpecContextKey).(issues.RepoSpec)

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
	repoSpec := req.Context().Value(RepoSpecContextKey).(issues.RepoSpec)

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
	repoSpec := req.Context().Value(RepoSpecContextKey).(issues.RepoSpec)

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
