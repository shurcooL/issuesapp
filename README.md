issuesapp
=========

[![Go Reference](https://pkg.go.dev/badge/github.com/shurcooL/issuesapp.svg)](https://pkg.go.dev/github.com/shurcooL/issuesapp)

Package issuesapp is a web frontend for an issues service.

Note, the canonical issue tracker for this package is currently hosted at
https://dmitri.shuralyov.com/issues/github.com/shurcooL/issuesapp.
It is implemented using this very package.

Installation
------------

```sh
go get github.com/shurcooL/issuesapp
```

Development
-----------

This package relies on `go generate` directives to process and statically embed assets. For development only, you may need extra dependencies. You can build and run the package in development mode, where all assets are always read and processed from disk:

```bash
go build -tags=issuesappdev something/that/uses/issuesapp
```

When you're done with development, you should run `go generate` and commit that:

```bash
go generate github.com/shurcooL/issuesapp/...
```

Directories
-----------

| Path                                                                                  | Synopsis                                                                                  |
|---------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------|
| [assets](https://pkg.go.dev/github.com/shurcooL/issuesapp/assets)                     | Package assets contains assets for issuesapp.                                             |
| [cmd/githubissues](https://pkg.go.dev/github.com/shurcooL/issuesapp/cmd/githubissues) | githubissues is a simple test program for issuesapp that uses GitHub API-backed services. |
| [common](https://pkg.go.dev/github.com/shurcooL/issuesapp/common)                     | Package common contains common code for backend and frontend.                             |
| [component](https://pkg.go.dev/github.com/shurcooL/issuesapp/component)               | Package component contains individual components that can render themselves as HTML.      |
| [frontend](https://pkg.go.dev/github.com/shurcooL/issuesapp/frontend)                 | frontend script for issuesapp.                                                            |
| [httpclient](https://pkg.go.dev/github.com/shurcooL/issuesapp/httpclient)             | Package httpclient contains issues.Service implementation over HTTP.                      |
| [httphandler](https://pkg.go.dev/github.com/shurcooL/issuesapp/httphandler)           | Package httphandler contains an API handler for issues.Service.                           |
| [httproute](https://pkg.go.dev/github.com/shurcooL/issuesapp/httproute)               | Package httproute contains route paths for httpclient, httphandler.                       |

License
-------

-	[MIT License](LICENSE)
