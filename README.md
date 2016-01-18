# issuesapp

Installation
------------

```bash
go get -u github.com/shurcooL/issuesapp
```

Development
-----------

This project relies on `go generate` directives to process and statically embed assets. For development only, you'll need extra dependencies:

```bash
go get -u -d -tags=generate github.com/shurcooL/issuesapp/...
go get -u -d -tags=js github.com/shurcooL/issuesapp/...
```

Afterwards, you can build and run in development mode, where all assets are always read and processed from disk:

```bash
go build -tags=dev something/that/uses/tracker
```

When you're done with development, you should run `go generate` and commit that:

```bash
go generate github.com/shurcooL/issuesapp/...
```

License
-------

- [MIT License](http://opensource.org/licenses/mit-license.php)
