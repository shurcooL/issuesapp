issuesapp
=========

[![Build Status](https://travis-ci.org/shurcooL/issuesapp.svg?branch=master)](https://travis-ci.org/shurcooL/issuesapp) [![GoDoc](https://godoc.org/github.com/shurcooL/issuesapp?status.svg)](https://godoc.org/github.com/shurcooL/issuesapp)

Package issuesapp is a web frontend for an issues service.

Note, the canonical issue tracker for this package is currently hosted at
https://dmitri.shuralyov.com/issues/github.com/shurcooL/issuesapp.
It is implemented using this very package.

Installation
------------

```bash
go get -u github.com/shurcooL/issuesapp
```

Development
-----------

This package relies on `go generate` directives to process and statically embed assets. For development only, you may need extra dependencies. You can build and run the package in development mode, where all assets are always read and processed from disk:

```bash
go build -tags=dev something/that/uses/issuesapp
```

When you're done with development, you should run `go generate` and commit that:

```bash
go generate github.com/shurcooL/issuesapp/...
```

License
-------

-	[MIT License](LICENSE)
