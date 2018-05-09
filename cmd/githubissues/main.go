// githubissues is a simple test program for issuesapp that uses GitHub API-backed services.
//
// Warning: It performs queries (and mutations, if given an access token via
// GITHUBISSUES_GITHUB_TOKEN environment variable) against real GitHub API.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	githubv3 "github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/issuesapp/httphandler"
	"github.com/shurcooL/issuesapp/httproute"
	"github.com/shurcooL/notificationsapp"
	"github.com/shurcooL/reactions/emojis"
	"golang.org/x/oauth2"

	ghissues "github.com/shurcooL/issues/githubapi"
	ghnotifications "github.com/shurcooL/notifications/githubapi"
	ghusers "github.com/shurcooL/users/githubapi"
)

var httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")

func main() {
	flag.Parse()

	cacheTransport := httpcache.NewMemoryCacheTransport()
	// Optionally, perform GitHub API authentication with provided token.
	if token := os.Getenv("GITHUBISSUES_GITHUB_TOKEN"); token != "" {
		authTransport := &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
		}
		cacheTransport.Transport = authTransport
	}
	httpClient := &http.Client{Transport: cacheTransport}
	ghV3 := githubv3.NewClient(httpClient)
	ghV4 := githubv4.NewClient(httpClient)

	usersService, err := ghusers.NewService(ghV3)
	if err != nil {
		log.Fatalln("ghusers.NewService:", err)
	}
	notificationsService := ghnotifications.NewService(ghV3, ghV4, nil)
	service := ghissues.NewService(ghV3, ghV4, notificationsService, nil)

	r := mux.NewRouter()

	// Register HTTP API endpoints.
	apiMux := http.NewServeMux()
	apiHandler := httphandler.Issues{Issues: service}
	apiMux.Handle(httproute.List, httputil.ErrorHandler(usersService, apiHandler.List))
	apiMux.Handle(httproute.Count, httputil.ErrorHandler(usersService, apiHandler.Count))
	apiMux.Handle(httproute.ListComments, httputil.ErrorHandler(usersService, apiHandler.ListComments))
	apiMux.Handle(httproute.ListEvents, httputil.ErrorHandler(usersService, apiHandler.ListEvents))
	apiMux.Handle(httproute.EditComment, httputil.ErrorHandler(usersService, apiHandler.EditComment))
	r.PathPrefix("/api/").Handler(apiMux)

	issuesOpt := issuesapp.Options{
		Notifications: notificationsService,

		HeadPre: `<style type="text/css">
	body {
		margin: 20px;
		font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
		font-size: 14px;
		line-height: initial;
		color: #373a3c;
	}
	.btn {
		font-size: 11px;
		line-height: 11px;
		border-radius: 4px;
		border: solid #d2d2d2 1px;
		background-color: #fff;
		box-shadow: 0 1px 1px rgba(0, 0, 0, .05);
	}
</style>`,
	}
	issuesApp := issuesapp.New(service, usersService, issuesOpt)

	notificationsOpt := notificationsapp.Options{
		HeadPre: `<link href="//cdnjs.cloudflare.com/ajax/libs/octicons/3.1.0/octicons.css" media="all" rel="stylesheet" type="text/css" />
<style type="text/css">
	body {
		margin: 20px;
	}
	body, table {
		font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
		font-size: 14px;
		line-height: initial;
		color: #373a3c;
	}
</style>`,
	}
	notificationsApp := notificationsapp.New(notificationsService, usersService, notificationsOpt)

	githubHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		prefixLen := 20 + len(vars["owner"]) + len(vars["repo"]) // len("/github.com/{owner}/{repo}/issues") with substitutions.
		if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
			baseURL := prefix
			if req.URL.RawQuery != "" {
				baseURL += "?" + req.URL.RawQuery
			}
			http.Redirect(w, req, baseURL, http.StatusFound)
			return
		}
		req.URL.Path = req.URL.Path[prefixLen:]
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		req = req.WithContext(context.WithValue(req.Context(),
			issuesapp.RepoSpecContextKey, issues.RepoSpec{URI: "github.com/" + vars["owner"] + "/" + vars["repo"]}))
		req = req.WithContext(context.WithValue(req.Context(),
			issuesapp.BaseURIContextKey, fmt.Sprintf("/github.com/%s/%s/issues", vars["owner"], vars["repo"])))
		issuesApp.ServeHTTP(w, req)
	})
	r.Path("/github.com/{owner}/{repo}/issues").Handler(githubHandler)
	r.PathPrefix("/github.com/{owner}/{repo}/issues/").Handler(githubHandler)

	notificationsHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		prefixLen := len("/notifications")
		if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
			baseURL := prefix
			if req.URL.RawQuery != "" {
				baseURL += "?" + req.URL.RawQuery
			}
			http.Redirect(w, req, baseURL, http.StatusFound)
			return
		}
		req.URL.Path = req.URL.Path[prefixLen:]
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		req = req.WithContext(context.WithValue(req.Context(), notificationsapp.BaseURIContextKey, "/notifications"))
		notificationsApp.ServeHTTP(w, req)
	})
	r.Path("/notifications").Handler(notificationsHandler)
	r.PathPrefix("/notifications/").Handler(notificationsHandler)

	r.HandleFunc("/login/github", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Sorry, this is just a demo instance and it doesn't support signing in.")
	})

	emojisHandler := httpgzip.FileServer(emojis.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	r.PathPrefix("/emojis/").Handler(http.StripPrefix("/emojis", emojisHandler))

	printServingAt(*httpFlag)
	err = http.ListenAndServe(*httpFlag, r)
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
