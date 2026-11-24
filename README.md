# A multiplexer for Go (V2)

`mux` routes standard Go HTTP handlers by method and path. v2 adds tree-based
matching, typed path captures, and URL updates for registered routes.

## Installation

This checkout requires Go 1.27.3 or newer. If your application has no
`go.mod` yet, run `go mod init example.com/myapp` in its directory.

After a release with the corrected module layout is published, install v2
with the following command.

```sh
go get github.com/Nigel2392/mux/v2@latest
```

If you previously fetched `v2.0.0` with the router in a subdirectory, use a
release with the corrected layout or follow the local checkout instructions
below.

For v1, use the import path `github.com/Nigel2392/mux` and a v1 release tag.

```sh
go get github.com/Nigel2392/mux@v1.6.1
```

To use v2 from a local checkout with the corrected layout, point your
application at that checkout. In this example, `../mux` is relative to your
application's `go.mod`.

```sh
go mod edit -replace github.com/Nigel2392/mux/v2=../mux
go get github.com/Nigel2392/mux/v2
```

## Quick start

Save this as `main.go`.

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Nigel2392/mux/v2"
)

func main() {
	router := mux.New()
	router.Get("/hello/<<name>>", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!\n", mux.Vars(r).Get("name"))
	}))

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", router))
}
```

Run `go run .`, then visit `http://127.0.0.1:8080/hello/Ada`.

Use `/users/<<uuid:id>>` to require a UUID. The capture preserves the request
text, and `Reverse` accepts a UUID string or a `uuid.UUID` from Go's standard
library.

## Nested routes, middleware, and named URLs

Add this function to the quick-start file. It builds an entire subtree before
registering it.

```go
func mountShop(router *mux.Mux) (string, error) {
	root := mux.NewRoute(http.MethodGet, "/site", nil)
	app := root.Get("/shop", nil, "myapp")
	catalog := app.Get("/catalog", nil)
	product := catalog.Get("/products/<<int:id>>", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Product %d\n", mux.Vars(r).GetInt("id"))
	}), "product")
	product.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Shop-Route", "product")
			next.ServeHTTP(w, r)
		})
	})

	router.AddRoute(root)
	if err := app.SetPath("/store"); err != nil {
		return "", err
	}
	return router.Reverse("myapp:product", 42)
}
```

Call it from `main` before starting the server.

```go
shopURL, err := mountShop(router)
if err != nil {
	log.Fatal(err)
}
log.Printf("Example product URL: %s", shopURL)
```

`SetPath` changes the route's relative URL and updates its descendants. Apply
route edits outside concurrent lookups. Both `Find` and `Reverse` skip unnamed
routes in the name chain, so `myapp:product` reverses to
`/site/store/catalog/products/42/` with all of its URL segments.

`Route.Use` applies middleware to the route and its descendants. Use
`Mux.Use` for middleware shared by all matched routes, `Route.Preprocess` to
run route middleware before mux middleware, or `Route.RunsMiddleware(false)`
to skip middleware for a route. See the [namespace examples](namespace_test.go)
and the [middleware package](middleware), available at
`github.com/Nigel2392/mux/v2/middleware`. It includes an
[SCS session integration](middleware/sessions).

## Benchmarks

Run the side-by-side v1 and v2 benchmarks from the repository root.

```sh
go test -run '^$' -bench '^BenchmarkCompare' -benchmem .
```

Each pair uses the same route table and registration order. Lookup benchmarks
also replay the same requests and verify the matched routes and captures
before timing. Cases cover literal hits and misses, path depth, captures,
globs, method selection, and mixed traffic. Registration benchmarks measure
building an entire router per operation.
