// +build generate

package main

import (
	"log"

	"github.com/shurcooL/vfsgen"

	"github.com/shurcooL/issuesapp"
)

func main() {
	err := vfsgen.Generate(issuesapp.Assets, vfsgen.Options{
		PackageName:  "issuesapp",
		BuildTags:    "!dev",
		VariableName: "Assets",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
