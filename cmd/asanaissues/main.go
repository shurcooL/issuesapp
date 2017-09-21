// asanaissues is a simple test program for issuesapp that uses Asana API-backed services.
//
// Warning: It performs queries (and mutations, if given an access token via
// ASANAISSUES_ASANA_TOKEN environment variable) against real Asana API.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issuesapp"
	"github.com/tambet/go-asana/asana"
	"golang.org/x/oauth2"

	anissues "github.com/shurcooL/issues/asanaapi"
	anusers "github.com/shurcooL/users/asanaapi"
)

var httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")

func main() {
	flag.Parse()

	cacheTransport := httpcache.NewMemoryCacheTransport()
	// Optionally, perform Asana API authentication with provided token.
	if token := os.Getenv("ASANAISSUES_ASANA_TOKEN"); token != "" {
		authTransport := &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
		}
		cacheTransport.Transport = authTransport
	}
	an := asana.NewClient(&http.Client{Transport: cacheTransport})

	usersService := anusers.NewService(an)
	service := anissues.NewService(an, usersService)

	issuesOpt := issuesapp.Options{
		HeadPre: `<link href="//cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.0.0-alpha/css/bootstrap.css" media="all" rel="stylesheet" type="text/css" />
<style type="text/css">
	body {
		margin: 20px;
		font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
		font-size: 14px;
		line-height: initial;
	}
	.btn {
		font-size: 14px;
	}
</style>`,
	}
	issuesApp := issuesapp.New(service, usersService, issuesOpt)

	r := mux.NewRouter()

	asanaHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		prefixLen := len("/app.asana.com/0/") + len(vars["projectID"])
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
			issuesapp.RepoSpecContextKey, issues.RepoSpec{URI: vars["projectID"]}))
		req = req.WithContext(context.WithValue(req.Context(),
			issuesapp.BaseURIContextKey, fmt.Sprintf("/app.asana.com/0/%s", vars["projectID"])))
		issuesApp.ServeHTTP(w, req)
	})
	r.Path("/app.asana.com/0/{projectID}").Handler(asanaHandler)
	r.PathPrefix("/app.asana.com/0/{projectID}/").Handler(asanaHandler)

	printServingAt(*httpFlag)
	err := http.ListenAndServe(*httpFlag, r)
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
