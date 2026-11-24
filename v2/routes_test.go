package mux_test

import (
	"errors"
	"slices"
	"testing"

	mux "github.com/Nigel2392/mux/v2"
)

func TestChangeRoutePathReindexesDescendants(t *testing.T) {
	router := mux.New()
	root := router.Get("/stores/<<store>>", nil, "store")
	product := root.Get("/products/<<int:id>>", handler, "product")
	unrelated := router.Get("/stores/<<shop>>/status", handler, "status")
	id := product.ID()

	if err := root.SetPath("/shops/<<region>>/<<store>>"); err != nil {
		t.Fatal(err)
	}
	if got, _ := router.Match(mux.GET, "/stores/london/products/42"); got != nil {
		t.Fatalf("old path still matches %v", got)
	}
	got, vars := router.Match(mux.GET, "/shops/europe/london/products/42")
	if got != product || vars.Get("region") != "europe" || vars.Get("store") != "london" || vars.GetInt("id") != 42 {
		t.Fatalf("updated path matched %v with %v", got, vars)
	}
	if got, vars := router.Match(mux.GET, "/stores/london/status"); got != unrelated || vars.Get("shop") != "london" {
		t.Fatalf("shared branch was damaged: %v, %v", got, vars)
	}
	if product.ID() != id || product.Parent != root || product.ParentMux != router || product.Path.Parent != root.Path {
		t.Fatal("path change modified route identity or detached the child")
	}
	if path, err := router.Reverse("store:product", "europe", "london", 42); err != nil || path != "/shops/europe/london/products/42/" {
		t.Fatalf("Reverse() = %q, %v", path, err)
	}

	if err := product.SetPathInfo(mux.NewPathInfo(product, "/detail/<<int:productID>>")); err != nil {
		t.Fatal(err)
	}
	if got, vars := router.Match(mux.GET, "/shops/europe/london/detail/42"); got != product || vars.GetInt("productID") != 42 {
		t.Fatalf("SetPathInfo replacement matched %v with %v", got, vars)
	}
}

func TestChangeRoutePathRejectsInvalidEdits(t *testing.T) {
	router := mux.New()
	root := router.Get("/original", nil)
	child := root.Get("/child", handler)
	for _, tt := range []struct {
		path string
		err  error
	}{
		{"/new//prefix", mux.ErrInvalidMatcherArg},
		{"/new/*/prefix", mux.ErrTerminalNotLast},
		{"/new/*", mux.ErrHasChildren},
		{"/<<unknown:name>>", mux.ErrMatcherNotFound},
	} {
		t.Run(tt.path, func(t *testing.T) {
			if err := root.SetPath(tt.path); !errors.Is(err, tt.err) {
				t.Fatalf("SetPath(%q) = %v, want %v", tt.path, err, tt.err)
			}
			if got, _ := router.Match(mux.GET, "/original/child"); got != child {
				t.Fatalf("failed update changed original route: got %v", got)
			}
			if child.String() != "/original/child" || root.String() != "/original" {
				t.Fatalf("failed update changed path metadata: %s, %s", root, child)
			}
		})
	}
}

func TestChangeCaptureNameKeepsRoutePrecedence(t *testing.T) {
	router := mux.New()
	first := router.Get("/<<regex:id:[a-z]+>>", handler)
	router.Get("/<<regex:same:[a-z]+>>", handler)
	router.Get("/<<regex:overlap:[a-z]{3}>>", handler)
	if err := first.SetPath("/<<regex:slug:[a-z]+>>"); err != nil {
		t.Fatal(err)
	}
	if got, vars := router.Match(mux.GET, "/abc"); got != first || vars.Get("slug") != "abc" {
		t.Fatalf("capture rename changed precedence: got %v with %v", got, vars)
	}
}

func TestDetachAndReattachSubtree(t *testing.T) {
	first := mux.New()
	root := first.Get("/old", nil, "old")
	subtree := mux.NewRoute(mux.GET, "/group", nil, "group")
	child := &mux.Route{
		Name: "child", Method: mux.GET, Handler: handler, Path: mux.NewPathInfo(nil, "/item"),
	}
	subtree.Children = append(subtree.Children, child)
	root.AddRoute(subtree)
	id := child.ID()
	if id == 0 || child.Parent != subtree || child.ParentMux != first {
		t.Fatal("pre-initialized child did not acquire identity and ownership")
	}

	root.RemoveChild(subtree)
	if got, _ := first.Match(mux.GET, "/old/group/item"); got != nil {
		t.Fatalf("removed subtree still matches %v", got)
	}
	if subtree.Parent != nil || subtree.ParentMux != nil || child.ParentMux != nil || child.Parent != subtree {
		t.Fatal("detaching did not preserve the subtree and clear its mux")
	}
	if err := subtree.SetPath("/moved/<<team>>"); err != nil {
		t.Fatal(err)
	}
	if got, ok, vars := subtree.Match(mux.GET, "/moved/blue/item"); got != child || !ok || vars.Get("team") != "blue" {
		t.Fatalf("detached subtree matched %v, %v, %v", got, ok, vars)
	}
	second := mux.New()
	second.AddRoute(subtree)
	second.RemoveByPath("/moved/<<team>>/item/")
	if got := second.Find("group:child"); got != nil || child.Parent != nil {
		t.Fatalf("RemoveByPath left named child %v attached", got)
	}
	subtree.AddRoute(child)
	if got, vars := second.Match(mux.GET, "/moved/blue/item"); got != child || got.ID() != id || vars.Get("team") != "blue" {
		t.Fatalf("reattached child matched %v with %v", got, vars)
	}
	if !subtree.RemoveByPath("/moved/<<team>>/item") {
		t.Fatal("Route.RemoveByPath did not remove the descendant")
	}
}

func TestResetRoutesClearsTreeForReuse(t *testing.T) {
	router := mux.New()
	root := router.Get("/", nil, "root")
	child := root.Get("/<<id>>", handler, "child")
	unrelated := router.Post("/<<name>>", handler, "other")
	router.RemoveRoute(child) // Only top-level routes are removed here.
	if got, _ := router.Match(mux.GET, "/one"); got != child {
		t.Fatalf("RemoveRoute affected a non-root route: %v", got)
	}
	router.RemoveRoute(unrelated)
	if got, _ := router.Match(mux.POST, "/one"); got != nil {
		t.Fatalf("RemoveRoute left the removed route indexed: %v", got)
	}
	router.ResetRoutes()
	if got := router.Find("root"); got != nil || root.ParentMux != nil || child.ParentMux != nil {
		t.Fatal("ResetRoutes did not clear names and ownership")
	}
	router.AddRoute(root)
	if got, _ := router.Match(mux.GET, "/one"); got != child {
		t.Fatalf("router could not reuse its detached subtree: %v", got)
	}
}

func TestFindAndReverseSkipUnnamedRoutes(t *testing.T) {
	router := mux.New()
	root := router.Get("/accounts", nil)
	space := root.Get("/<<account>>", nil)
	app := space.Get("/shop", nil, "myapp")
	group := app.Get("/catalog", nil)
	product := group.Get("/<<int:id>>", handler, "product")
	root.Get("/competing", nil, "other").Get("/item", handler, "product")

	if !slices.Equal(product.PathName(), []string{"myapp", "product"}) {
		t.Fatalf("PathName() = %q", product.PathName())
	}
	if got := router.Find("myapp:product"); got != product {
		t.Fatalf("Find(myapp:product) = %v, want %v", got, product)
	}
	if got := router.Find("myapp"); got != app {
		t.Fatalf("Find(myapp) = %v, want %v", got, app)
	}
	if got, ok := space.Find([]string{"myapp", "product"}); got != product || !ok {
		t.Fatalf("scoped Find() = %v, %v", got, ok)
	}
	if got := router.Find("product"); got != nil {
		t.Fatalf("Find(product) skipped a named ancestor: %v", got)
	}
	if path, err := router.Reverse("myapp:product", "alice", 42); err != nil || path != "/accounts/alice/shop/catalog/42/" {
		t.Fatalf("Reverse(myapp:product) = %q, %v", path, err)
	}
	if _, err := router.Reverse("myapp:missing"); !errors.Is(err, mux.ErrRouteNotFound) {
		t.Fatalf("Reverse(missing route) = %v", err)
	}
}
