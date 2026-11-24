package mux_test

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	mux "github.com/Nigel2392/mux/v2"
)

func TestPathInfoMetadataAndPartialMatch(t *testing.T) {
	parent := mux.NewPathInfo(nil, "/org/<<name>>")
	child := mux.NewPathInfo(nil, "/team/<<int:id>>/*").WithParent(parent)
	if got := child.String(); got != "/org/<<name>>/team/<<int:id>>/*" {
		t.Fatalf("String() = %q", got)
	}
	if !child.IsGlob || len(child.Path) != 3 || !child.Path[1].IsVariable || !child.Path[2].IsGlob {
		t.Fatalf("unexpected path metadata: %#v", child)
	}
	path := mux.SplitPath("/org/acme/team/123/assets/logo.svg/")
	done, index, vars := parent.Match(path, 0, nil)
	if done || index != 2 || vars.Get("name") != "acme" {
		t.Fatalf("parent Match = %v, %d, %v", done, index, vars)
	}
	done, index, vars = child.Match(path, index, vars)
	if !done || index != len(path) || vars.GetInt("id") != 123 ||
		!slices.Equal(vars.GetAll(mux.GLOB), []string{"assets", "logo.svg"}) {
		t.Fatalf("child Match = %v, %d, %v", done, index, vars)
	}
	for _, from := range []int{-1, 0, len(path) + 1} {
		if ok, next, vars := child.Match(path, from, nil); ok || next != -1 || vars != nil {
			t.Errorf("Match from index %d = %v, %d, %v", from, ok, next, vars)
		}
	}
}

func TestPathInfoReverseUsesMatchers(t *testing.T) {
	for _, tt := range []struct {
		pattern string
		args    []any
		want    string
		err     error
	}{
		{"/", nil, "/", nil},
		{"/status", nil, "/status/", nil},
		{"/users/<<id>>", []any{1}, "/users/1/", nil},
		{"/users/<<id>>", nil, "", mux.ErrNotEnoughVariables},
		{"/users/<<id>>", []any{1, 2}, "", mux.ErrTooManyVariables},
		{"/tags/<<regex:name:[a-z]+>>", []any{"go"}, "/tags/go/", nil},
		{"/tags/<<regex:name:[a-z]+>>", []any{1}, "", mux.ErrInvalidMatcherArg},
		{"/assets/*", []any{"icons", "plus.svg"}, "/assets/icons/plus.svg", nil},
		{"/assets/*", nil, "/assets/", nil},
		{"/assets/<<glob:file>>", []any{"icons/plus.svg"}, "/assets/icons/plus.svg", nil},
	} {
		t.Run(tt.pattern+fmt.Sprint(tt.args), func(t *testing.T) {
			got, err := mux.NewPathInfo(nil, tt.pattern).Reverse(tt.args...)
			if got != tt.want || !errors.Is(err, tt.err) {
				t.Fatalf("Reverse(%v) = %q, %v; want %q, %v", tt.args, got, err, tt.want, tt.err)
			}
		})
	}
}

type assetResolver struct{}

func (assetResolver) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (assetResolver) Match(vars mux.Variables, path []string) (bool, mux.Variables) {
	if vars.Get("org") != "acme" || len(path) == 0 || path[0] != "accepted" {
		vars["discarded"] = []string{"unmatched"}
		return false, nil
	}
	return true, mux.Variables{"asset": path[1:]}
}

func (assetResolver) Reverse(base string, variables ...any) (string, error) {
	return base + "accepted/" + fmt.Sprint(variables[0]), nil
}

func TestMountedResolverSharesCapturesAndAllowsFallback(t *testing.T) {
	router := mux.New()
	org := router.Get("/org/<<org>>", nil)
	resolved := org.Get("/assets/*", assetResolver{})
	fallback := org.Get("/assets/*", handler)

	got, vars := router.Match(mux.GET, "/org/acme/assets/accepted/icon.svg")
	if got != resolved || vars.Get("org") != "acme" || !slices.Equal(vars["asset"], []string{"icon.svg"}) || vars.Len() != 2 {
		t.Fatalf("resolved match = %p, %v", got, vars)
	}
	got, vars = router.Match(mux.GET, "/org/acme/assets/other/icon.svg")
	if got != fallback || !slices.Equal(vars[mux.GLOB], []string{"other", "icon.svg"}) || vars.Len() != 2 {
		t.Fatalf("fallback match = %p, %v", got, vars)
	}
	if path, err := resolved.Path.Reverse("acme", "icon.svg"); err != nil || path != "/org/acme/assets/accepted/icon.svg" {
		t.Fatalf("Reverse() = %q, %v", path, err)
	}
}

func TestPathInfoMatchesResolverAcrossParts(t *testing.T) {
	path := mux.NewPathInfo(nil, "/<<org>>/*")
	path.Resolver = assetResolver{}
	got, position, vars := path.Match([]string{"acme", "accepted", "icons", "a.svg"}, 0, nil)
	if !got || position != 4 || vars.Get("org") != "acme" || !slices.Equal(vars.GetAll("asset"), []string{"icons", "a.svg"}) {
		t.Fatalf("resolver Match = %v, %d, %v", got, position, vars)
	}
	if ok, index, vars := path.Match([]string{"acme", "discard"}, 0, nil); ok || index != -1 || vars != nil {
		t.Fatalf("rejected resolver Match = %v, %d, %v", ok, index, vars)
	}
}

type keyedGlob struct{ mux.Matcher }

func (m keyedGlob) Bind(name string) (mux.BoundMatcher, error) {
	matcher, err := m.Matcher.Bind(name)
	if err != nil {
		return nil, err
	}
	return boundKeyedGlob{matcher}, nil
}

type boundKeyedGlob struct{ mux.BoundMatcher }

func (boundKeyedGlob) Key() string { return "glob:specialized" }

func TestResolverInferredForPreInitializedPath(t *testing.T) {
	router := mux.New()
	registry := router.Matchers()
	original, _ := registry.Get("glob")
	registry.Register(keyedGlob{original})

	route := &mux.Route{
		Method:  mux.GET,
		Handler: assetResolver{},
		Path:    mux.NewPathInfo(nil, "/org/<<org>>/*"),
	}
	router.AddRoute(route)
	got, vars := router.Match(mux.GET, "/org/acme/accepted/icon.svg")
	if got != route || vars.Get("asset") != "icon.svg" {
		t.Fatalf("Match() = %p, %v", got, vars)
	}
	if path, err := route.Path.Reverse("acme", "icon.svg"); err != nil || path != "/org/acme/accepted/icon.svg" {
		t.Fatalf("Reverse() = %q, %v", path, err)
	}
}

func TestGlobalRouteSyntax(t *testing.T) {
	delimiters, separator, glob := mux.VARIABLE_DELIMS, mux.URL_DELIM, mux.GLOB
	t.Cleanup(func() {
		mux.VARIABLE_DELIMS, mux.URL_DELIM, mux.GLOB = delimiters, separator, glob
	})
	mux.VARIABLE_DELIMS, mux.URL_DELIM, mux.GLOB = []string{"{", "}"}, "::", "**"

	router := mux.New()
	parent := router.Get("::org::{owner}::", nil)
	want := parent.Get("assets::**", handler)
	path := "::org::acme::assets::icons::plus.svg::"
	got, vars := router.Match(mux.GET, path)
	if got != want || vars.Get("owner") != "acme" || !slices.Equal(vars.GetAll("**"), []string{"icons", "plus.svg"}) {
		t.Fatalf("Match(%q) = %p, %v, want %p with both captures", path, got, vars, want)
	}
	if got := want.Path.String(); got != "::org::{owner}::assets::**" {
		t.Fatalf("String() = %q", got)
	}
	if got, err := want.Path.Reverse("acme", "icons", "plus.svg"); got != strings.TrimSuffix(path, "::") || err != nil {
		t.Fatalf("Reverse() = %q, %v", got, err)
	}
	if got := mux.SplitPath("::one::::two::"); !slices.Equal(got, []string{"one", "", "two"}) {
		t.Fatalf("SplitPath() = %q", got)
	}
}
