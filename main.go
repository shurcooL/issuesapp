package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go-goon"
	"github.com/shurcooL/go/gzip_file_server"
	"github.com/shurcooL/go/vfs/httpfs/html/vfstemplate"
)

var httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")

var t *template.Template

func loadTemplates() error {
	var err error
	t = template.New("").Funcs(template.FuncMap{
		"dump": func(v interface{}) string { return goon.Sdump(v) },
		"time": humanize.Time,
		"gfm":  func(s string) template.HTML { return template.HTML(github_flavored_markdown.Markdown([]byte(s))) },
	})
	t, err = vfstemplate.ParseGlob(assets, t, "/assets/*.tmpl")
	return err
}

var gh = github.NewClient(nil)

var state stateType

type stateType struct {
	mu   sync.Mutex
	vars map[string]string
}

func (s stateType) Issues() ([]github.Issue, error) {
	issuesPRs, _, err := gh.Issues.ListByRepo(s.vars["owner"], s.vars["repo"], &github.IssueListByRepoOptions{State: "open"})
	if err != nil {
		return nil, err
	}

	// Filter out PRs.
	var issues []github.Issue
	for _, issue := range issuesPRs {
		if issue.PullRequestLinks != nil {
			continue
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func (s stateType) Issue() (interface{}, error) {
	issue, _, err := gh.Issues.Get(s.vars["owner"], s.vars["repo"], mustAtoi(s.vars["id"]))
	if err != nil {
		return nil, err
	}

	return issue, nil
}

func (s stateType) Comments() (interface{}, error) {
	comments, _, err := gh.Issues.ListComments(s.vars["owner"], s.vars["repo"], mustAtoi(s.vars["id"]), nil)
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (s stateType) Comment() github.IssueComment {
	newInt := func(v int) *int {
		return &v
	}
	newString := func(v string) *string {
		return &v
	}
	newTime := func(v time.Time) *time.Time {
		return &v
	}

	return (github.IssueComment)(github.IssueComment{
		ID:        (*int)(newInt(143399786)),
		Body:      (*string)(newString("I've resolved this in 4387efb. Please re-open or leave a comment if there's still room for improvement here.")),
		User:      s.User(),
		CreatedAt: (*time.Time)(newTime(time.Unix(1443244474, 0).UTC())),
		UpdatedAt: (*time.Time)(newTime(time.Unix(1443244474, 0).UTC())),
		URL:       (*string)(newString("https://api.github.com/repos/shurcooL/vfsgen/issues/comments/143399786")),
		HTMLURL:   (*string)(newString("https://github.com/shurcooL/vfsgen/issues/8#issuecomment-143399786")),
		IssueURL:  (*string)(newString("https://api.github.com/repos/shurcooL/vfsgen/issues/8")),
	})
}

func (stateType) User() *github.User {
	newInt := func(v int) *int {
		return &v
	}
	newBool := func(v bool) *bool {
		return &v
	}
	newString := func(v string) *string {
		return &v
	}

	return (*github.User)(&github.User{
		Login:             (*string)(newString("shurcooL")),
		ID:                (*int)(newInt(1924134)),
		AvatarURL:         (*string)(newString("https://avatars.githubusercontent.com/u/1924134?v=3")),
		HTMLURL:           (*string)(newString("https://github.com/shurcooL")),
		GravatarID:        (*string)(newString("")),
		Name:              (*string)(nil),
		Company:           (*string)(nil),
		Blog:              (*string)(nil),
		Location:          (*string)(nil),
		Email:             (*string)(nil),
		Hireable:          (*bool)(nil),
		Bio:               (*string)(nil),
		PublicRepos:       (*int)(nil),
		PublicGists:       (*int)(nil),
		Followers:         (*int)(nil),
		Following:         (*int)(nil),
		CreatedAt:         (*github.Timestamp)(nil),
		UpdatedAt:         (*github.Timestamp)(nil),
		Type:              (*string)(newString("User")),
		SiteAdmin:         (*bool)(newBool(false)),
		TotalPrivateRepos: (*int)(nil),
		OwnedPrivateRepos: (*int)(nil),
		PrivateGists:      (*int)(nil),
		DiskUsage:         (*int)(nil),
		Collaborators:     (*int)(nil),
		Plan:              (*github.Plan)(nil),
		URL:               (*string)(newString("https://api.github.com/users/shurcooL")),
		EventsURL:         (*string)(newString("https://api.github.com/users/shurcooL/events{/privacy}")),
		FollowingURL:      (*string)(newString("https://api.github.com/users/shurcooL/following{/other_user}")),
		FollowersURL:      (*string)(newString("https://api.github.com/users/shurcooL/followers")),
		GistsURL:          (*string)(newString("https://api.github.com/users/shurcooL/gists{/gist_id}")),
		OrganizationsURL:  (*string)(newString("https://api.github.com/users/shurcooL/orgs")),
		ReceivedEventsURL: (*string)(newString("https://api.github.com/users/shurcooL/received_events")),
		ReposURL:          (*string)(newString("https://api.github.com/users/shurcooL/repos")),
		StarredURL:        (*string)(newString("https://api.github.com/users/shurcooL/starred{/owner}{/repo}")),
		SubscriptionsURL:  (*string)(newString("https://api.github.com/users/shurcooL/subscriptions")),
		TextMatches:       ([]github.TextMatch)(nil),
		Permissions:       (*map[string]bool)(nil),
	})
}

func mainHandler(w http.ResponseWriter, req *http.Request) {
	if err := loadTemplates(); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := path.Base(req.URL.Path)

	state.mu.Lock()
	err := t.ExecuteTemplate(w, tmpl+".tmpl", &state)
	state.mu.Unlock()
	if err != nil {
		log.Println("t.Execute:", err)
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

	vars := mux.Vars(req)

	state.mu.Lock()
	state.vars = vars
	err := t.ExecuteTemplate(w, "issues.html.tmpl", &state)
	state.mu.Unlock()
	if err != nil {
		log.Println("t.Execute:", err)
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

	vars := mux.Vars(req)

	state.mu.Lock()
	state.vars = vars
	err := t.ExecuteTemplate(w, "issue.html.tmpl", &state)
	state.mu.Unlock()
	if err != nil {
		log.Println("t.Execute:", err)
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

	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/pages/", mainHandler)
	r := mux.NewRouter()
	r.HandleFunc("/github.com/{owner}/{repo}/issues", issuesHandler)
	r.HandleFunc("/github.com/{owner}/{repo}/issues/{id:[0-9]+}", issueHandler)
	http.Handle("/", r)
	http.Handle("/assets/", gzip_file_server.New(assets))

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
