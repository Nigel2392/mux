package mux_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"

	mux "github.com/Nigel2392/mux/v2"
)

var handler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

func TestMatchPriorityAndFallback(t *testing.T) {
	router := mux.New()
	glob := router.Any("/items/*", handler)
	variable := router.Get("/items/<<slug>>", handler)
	integer := router.Get("/items/<<int:id>>", handler)
	regex := router.Get("/items/<<regex:code:[0-9]{2}>>", handler)
	literal := router.Get("/items/42", handler)
	router.Get("/items/50/edit", handler)
	root := router.Get("/", handler)

	for _, tt := range []struct {
		path, method string
		want         *mux.Route
		key, value   string
	}{
		{"/", mux.GET, root, "", ""},
		{"/items/42/", mux.GET, literal, "", ""},
		{"/items/42", mux.POST, glob, "*", "42"},
		{"/items/50", mux.GET, regex, "code", "50"},
		{"/items/507", mux.GET, integer, "id", "507"},
		{"/items/abc", mux.GET, variable, "slug", "abc"},
		{"/items/abc/def", mux.GET, glob, "*", "abc"},
		{"/items", mux.GET, glob, "*", ""},
		{"/missing", mux.GET, nil, "", ""},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			route, vars := router.Match(tt.method, tt.path)
			if route != tt.want {
				t.Fatalf("got route %v, want %v", route, tt.want)
			}
			if tt.key == "" {
				if vars != nil {
					t.Fatalf("expected nil variables, got %v", vars)
				}
				return
			}
			if _, ok := vars.GetAllOK(tt.key); !ok || vars.Get(tt.key) != tt.value || vars.Len() != 1 {
				got := vars.Get(tt.key)
				t.Fatalf("got %q, %v for %q with variables %v", got, ok, tt.key, vars)
			}
		})
	}
}

func TestSharedEdgesKeepRouteCaptureNames(t *testing.T) {
	router := mux.New()
	parent := mux.NewRoute(mux.GET, "/<<org>>", nil)
	child := parent.Get("/<<org>>/details", handler)
	alternative := router.Post("/<<tenant>>/<<id>>/details", handler)
	router.AddRoute(parent)

	route, vars := router.Match(mux.GET, "/one/two/details")
	if route != child || !slices.Equal(vars.GetAll("org"), []string{"one", "two"}) {
		t.Fatalf("got route %v with variables %v", route, vars)
	}
	route, vars = router.Match(mux.POST, "/one/two/details")
	if route != alternative || vars.Get("tenant") != "one" || vars.Get("id") != "two" {
		t.Fatalf("got route %v with variables %v", route, vars)
	}
	route, ok, vars := child.Match(mux.GET, "/two/details")
	if !ok || route != child || !slices.Equal(vars.GetAll("org"), []string{"two"}) {
		t.Fatalf("got relative route %v, %v with variables %v", route, ok, vars)
	}
	if route, ok, _ := parent.Match(mux.POST, "/one/two/details"); ok || route != nil {
		t.Fatalf("scoped match returned unrelated route %v", route)
	}
}

func TestRejectedSubtreeDoesNotIndexRoutes(t *testing.T) {
	router := mux.New()
	root := mux.NewRoute(mux.GET, "/valid", handler)
	root.Get("/*/invalid", handler)
	func() {
		defer func() {
			if got := recover(); got != mux.ErrTerminalNotLast {
				t.Fatalf("expected ErrTerminalNotLast, got %v", got)
			}
		}()
		router.AddRoute(root)
	}()
	if route, _ := router.Match(mux.GET, "/valid"); route != nil {
		t.Fatalf("rejected subtree left route %v indexed", route)
	}
	if _, err := router.Matchers().ParsePath("/double//slash"); !errors.Is(err, mux.ErrInvalidMatcherArg) {
		t.Fatalf("expected ErrInvalidMatcherArg, got %v", err)
	}
}
