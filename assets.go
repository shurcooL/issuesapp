// +build dev

package issuesapp

import (
	"go/build"
	"log"
	"net/http"

	"github.com/shurcooL/github_flavored_markdown/gfmstyle"
	"github.com/shurcooL/go/gopherjs_http"
	"github.com/shurcooL/httpfs/union"
	"github.com/shurcooL/octicons"
)

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}

var Assets = union.New(map[string]http.FileSystem{
	"/assets":   gopherjs_http.NewFS(http.Dir(importPathToDir("github.com/shurcooL/issuesapp/assets"))),
	"/octicons": octicons.Assets,
	"/gfm":      gfmstyle.Assets,
})
