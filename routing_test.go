package mux_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	mux "github.com/Nigel2392/mux/v2"
)

func TestHTTPRegistrationHelpers(t *testing.T) {
	var router mux.Mux
	parent := router.Get("/parent", nil)
	for _, tt := range []struct {
		method string
		top    func(string, http.Handler, ...string) *mux.Route
		child  func(string, http.Handler, ...string) *mux.Route
	}{
		{mux.GET, router.Get, parent.Get},
		{mux.POST, router.Post, parent.Post},
		{mux.PUT, router.Put, parent.Put},
		{mux.DELETE, router.Delete, parent.Delete},
		{mux.HEAD, router.Head, parent.Head},
		{mux.PATCH, router.Patch, parent.Patch},
		{mux.OPTIONS, router.Options, parent.Options},
		{mux.ANY, router.Any, parent.Any},
	} {
		t.Run(tt.method, func(t *testing.T) {
			path := "/method-" + strings.ToLower(tt.method)
			top := tt.top(path, handler, "top", "ignored")
			child := tt.child(path, handler, "child")
			for _, item := range []struct {
				path, name string
				route      *mux.Route
			}{
				{path, "top", top},
				{"/parent" + path, "child", child},
			} {
				if item.route.Method != tt.method || item.route.Name != item.name || item.route.String() != item.path {
					t.Fatalf("unexpected route metadata: %#v", item.route)
				}
				if got, vars := router.Match(tt.method, item.path); got != item.route || vars != nil {
					t.Fatalf("Match(%q, %q) = %p, %v; want %p, nil", tt.method, item.path, got, vars, item.route)
				}
			}
		})
	}
}

func TestHandleFuncDispatch(t *testing.T) {
	router := mux.New()
	parent := router.Get("/parent", nil)
	for _, tt := range []struct {
		pattern, request string
		register         func(string, string, func(http.ResponseWriter, *http.Request), ...string) *mux.Route
	}{
		{"/top", "/top", router.HandleFunc},
		{"/child", "/parent/child", parent.HandleFunc},
	} {
		t.Run(tt.request, func(t *testing.T) {
			want := tt.register("CUSTOM", tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
				fmt.Fprint(w, r.URL.Path)
			})
			req := httptest.NewRequest("CUSTOM", tt.request, nil)
			got, _ := router.Match(req.Method, req.URL.Path)
			if got != want {
				t.Fatalf("Match() got %p, want %p", got, want)
			}
			out := httptest.NewRecorder()
			got.Handler.ServeHTTP(out, req)
			if out.Code != http.StatusAccepted || out.Body.String() != tt.request {
				t.Fatalf("handler wrote status %d and body %q", out.Code, out.Body)
			}
		})
	}
}

func TestMethodSelectionWithinSharedPath(t *testing.T) {
	router := mux.New()
	router.Any("/item/<<ignored>>", nil)
	post := router.Post("/item/<<posted>>", handler)
	first := router.Get("/item/<<first>>", handler)
	router.Get("/item/<<duplicate>>", handler)
	wild := router.Any("/item/<<wild>>", handler)
	router.Put("/item/<<late>>", handler)
	for _, tt := range []struct {
		method string
		route  *mux.Route
		key    string
	}{
		{mux.POST, post, "posted"},
		{mux.GET, first, "first"},
		{mux.ANY, post, "posted"},
		{mux.HEAD, wild, "wild"},
		{mux.PUT, wild, "wild"},
	} {
		t.Run(tt.method, func(t *testing.T) {
			got, vars := router.Match(tt.method, "/item/value")
			if got != tt.route || vars.Get(tt.key) != "value" || vars.Len() != 1 {
				t.Fatalf("got route %p with variables %v, want %p with %q", got, vars, tt.route, tt.key)
			}
		})
	}
}

func TestEqualPriorityMatchersKeepRegistrationOrder(t *testing.T) {
	router := mux.New()
	first := router.Get("/<<regex:first:[a-z]+>>/a", handler)
	second := router.Get("/<<regex:second:[a-z]{2}>>/b", handler)
	third := router.Get("/<<regex:third:[a-z]{2}>>/a", handler)
	for _, tt := range []struct {
		path, key string
		route     *mux.Route
	}{
		{"/ab/a", "first", first},
		{"/ab/b", "second", second},
	} {
		got, vars := router.Match(mux.GET, tt.path)
		if got != tt.route || vars.Get(tt.key) != "ab" || vars.Len() != 1 {
			t.Errorf("Match(%q) = %p, %v; want %p with %q", tt.path, got, vars, tt.route, tt.key)
		}
	}
	if got, ok, vars := third.Match(mux.GET, "/ab/a"); !ok || got != third || vars.Get("third") != "ab" {
		t.Fatalf("scoped Match() = %p, %v, %v", got, ok, vars)
	}
}

func TestCaptureRollbackBeyondInlineBuffer(t *testing.T) {
	router := mux.New()
	const count = 18
	prefix := strings.Repeat("/<<part>>", count)
	router.Get(prefix+"/<<int:rejected>>/missing", handler)
	route := router.Get(prefix+"/<<tail>>/end", handler)
	want := make([]string, count)
	for i := range want {
		want[i] = strconv.Itoa(i)
	}
	path := "/" + strings.Join(want, "/") + "/123/end"
	got, vars := router.Match(mux.GET, path)
	if got != route || !slices.Equal(vars.GetAll("part"), want) || vars.Get("tail") != "123" || vars.Len() != 2 {
		t.Fatalf("Match(%q) = %p with %v, want %p with %v and tail=123", path, got, vars, route, want)
	}
	if _, found := vars.GetOK("rejected"); found {
		t.Fatal("failed branch left a capture behind")
	}
}

func TestRouteIdentityAtSamePattern(t *testing.T) {
	router := mux.New()
	left := router.Get("/<<left>>", nil)
	right := router.Get("/<<right>>", nil)
	leftChild := left.Get("/child", handler)
	rightChild := right.Get("/child", handler)
	emptyChild := right.Post("/", handler)

	if got, vars := router.Match(mux.GET, "/value/child"); got != leftChild || vars.Get("left") != "value" {
		t.Fatalf("mux match = %p with %v", got, vars)
	}
	if got, ok, vars := right.Match(mux.GET, "/value/child"); !ok || got != rightChild || vars.Get("right") != "value" {
		t.Fatalf("right scoped match = %p, %v, %v", got, ok, vars)
	}
	if got, ok, vars := emptyChild.Match(mux.POST, "/"); !ok || got != emptyChild || vars != nil {
		t.Fatalf("empty child scoped match = %p, %v, %v", got, ok, vars)
	}
	if got := emptyChild.String(); got != right.String() {
		t.Errorf("empty child pattern %q differs from parent pattern %q", got, right.String())
	}
}

func TestExactEmptyPathBeforeGlob(t *testing.T) {
	router := mux.New()
	glob := router.Any("/<<glob:root>>", handler)
	exact := router.Get("/", handler)
	if got, vars := router.Match(mux.GET, "///"); got != exact || vars != nil {
		t.Fatalf("exact root match = %p, %v", got, vars)
	}
	if got, vars := router.Match(mux.POST, ""); got != glob || vars.Len() != 1 {
		t.Fatalf("root glob match = %p, %v", got, vars)
	} else if value, found := vars.GetOK("root"); value != "" || !found {
		t.Fatalf("empty capture = %q, %v", value, found)
	}
	if got, vars := router.Match(mux.GET, "/one//two/"); got != glob || vars.Get("root") != "one//two" {
		t.Fatalf("root glob match with path remainder = %p, %v", got, vars)
	}
}

func TestScopedRouteRejectsIncompleteOrInvalidPrefix(t *testing.T) {
	router := mux.New()
	parent := router.Get("/team/<<int:id>>", nil)
	child := parent.Get("/settings", handler)
	for _, path := range []string{"/", "/team", "/project/42/settings", "/team/no/settings", "/team/42", "/team/42/settings/more"} {
		t.Run(path, func(t *testing.T) {
			if got, ok, vars := parent.Match(mux.GET, path); got != nil || ok || vars != nil {
				t.Fatalf("Match(%q) = %p, %v, %v; want a miss", path, got, ok, vars)
			}
		})
	}
	if got, ok, vars := parent.Match(mux.GET, "/team/42/settings"); got != child || !ok || vars.Get("id") != "42" {
		t.Fatalf("Match(valid path) = %p, %v, %v; want %p with id=42", got, ok, vars, child)
	}
}

func TestNewRouteMatchesBeforeAndAfterAttachment(t *testing.T) {
	route := mux.NewRoute(mux.GET, "///team///", handler)
	if got, ok, vars := route.Match(mux.GET, "/team"); got != route || !ok || vars != nil || route.ParentMux != nil {
		t.Fatalf("detached Match() = %p, %v, %v; want %p without attachment", got, ok, vars, route)
	}
	var router mux.Mux
	if got, vars := router.Match(mux.GET, "/team"); got != nil || vars != nil {
		t.Fatalf("empty mux Match() = %p, %v; want a miss", got, vars)
	}
	router.AddRoute(route)
	if got, ok, vars := route.Match(mux.GET, "/team"); got != route || !ok || vars != nil {
		t.Fatalf("attached Match() = %p, %v, %v; want %p", got, ok, vars, route)
	}
}

type pairMatcher struct{}

func (pairMatcher) Key() string { return "pair" }

func (pairMatcher) Bind(name string) (mux.BoundMatcher, error) {
	bound, err := (mux.VarMatcher{}).Bind(name)
	if err != nil {
		return nil, err
	}
	return boundPair{BoundMatcher: bound}, nil
}

type boundPair struct{ mux.BoundMatcher }

func (boundPair) Key() string   { return "pair" }
func (boundPair) Priority() int { return 250 }

func (boundPair) Match(path *mux.PathBuf) (string, bool) {
	first, second := path.Consume(), path.Consume()
	if first != "prefix" || second == "" {
		return "", false
	}
	return second, true
}

func TestCustomMatcherConsumesMultipleSegments(t *testing.T) {
	router := mux.New()
	router.Matchers().Register(pairMatcher{})
	pair := router.Get("/<<pair:value>>/end", handler)
	fallback := router.Get("/<<first>>/<<second>>/end", handler)
	suffix := router.Get("/<<first>>/end", handler)
	for _, tt := range []struct {
		path string
		want *mux.Route
		vars map[string]string
	}{
		{"/prefix/x/end", pair, map[string]string{"value": "x"}},
		{"/other/x/end", fallback, map[string]string{"first": "other", "second": "x"}},
		{"/other/end", suffix, map[string]string{"first": "other"}},
		{"/prefix//end", nil, nil},
	} {
		t.Run(tt.path, func(t *testing.T) {
			got, vars := router.Match(mux.GET, tt.path)
			if got != tt.want {
				t.Fatalf("got route %p, want %p", got, tt.want)
			}
			if tt.vars == nil {
				if vars != nil {
					t.Fatalf("unexpected captures on miss: %v", vars)
				}
				return
			}
			if vars == nil || vars.Len() != len(tt.vars) {
				t.Fatalf("got variables %v, want %v", vars, tt.vars)
			}
			for key, want := range tt.vars {
				if value := vars.Get(key); value != want {
					t.Errorf("vars.Get(%q) = %q, want %q", key, value, want)
				}
			}
		})
	}
}

func TestAddRouteRejectsInvalidAttachments(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		run  func()
	}{
		{"self-cycle", mux.ErrCyclicParent, func() {
			r := mux.NewRoute(mux.GET, "/", handler)
			r.AddRoute(r)
		}},
		{"ancestor-cycle", mux.ErrCyclicParent, func() {
			r := mux.NewRoute(mux.GET, "/", handler)
			r.Get("/child", handler).Get("/grandchild", handler).AddRoute(r)
		}},
		{"cycle-in-prebuilt-children", mux.ErrCyclicParent, func() {
			r := mux.NewRoute(mux.GET, "/", handler)
			child := mux.NewRoute(mux.GET, "/child", handler)
			r.Children = []*mux.Route{child}
			child.Children = []*mux.Route{r}
			mux.New().AddRoute(r)
		}},
		{"duplicate-in-prebuilt-children", mux.ErrRouteAttached, func() {
			r := mux.NewRoute(mux.GET, "/", handler)
			child := mux.NewRoute(mux.GET, "/child", handler)
			r.Children = []*mux.Route{child, child}
			mux.New().AddRoute(r)
		}},
		{"double-registration", mux.ErrRouteAttached, func() {
			router := mux.New()
			r := router.Get("/already", handler)
			router.AddRoute(r)
		}},
		{"second-mux", mux.ErrRouteAttached, func() {
			r := mux.New().Get("/already", handler)
			mux.New().AddRoute(r)
		}},
		{"second-parent", mux.ErrRouteAttached, func() {
			first := mux.NewRoute(mux.GET, "/first", handler)
			second := mux.NewRoute(mux.GET, "/second", handler)
			second.AddRoute(first.Get("/child", handler))
		}},
		{"attached-terminal", mux.ErrParentIsTerminal, func() {
			mux.New().Get("/*", handler).Get("/child", handler)
		}},
		{"detached-terminal-with-child", mux.ErrHasChildren, func() {
			r := mux.NewRoute(mux.GET, "/*", handler)
			r.Get("/child", handler)
			mux.New().AddRoute(r)
		}},
		{"unknown-matcher", mux.ErrMatcherNotFound, func() {
			mux.New().Get("/<<unknown:id>>", handler)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if err, ok := got.(error); !ok || !errors.Is(err, tt.err) {
					t.Errorf("panic = %v, want %v", got, tt.err)
				}
			}()
			tt.run()
		})
	}
}

func TestConcurrentMatchesAreIndependent(t *testing.T) {
	router := mux.New()
	want := router.Get("/<<int:id>>/<<id>>/end", handler)
	router.Get("/<<regex:dead:[0-9]+>>/<<id>>/missing", handler)
	var workers sync.WaitGroup
	for i := range 16 {
		workers.Go(func() {
			id := strconv.Itoa(i)
			path := "/" + id + "/value/end"
			for range 50 {
				got, vars := router.Match(mux.GET, path)
				if got != want || !slices.Equal(vars.GetAll("id"), []string{id, "value"}) {
					t.Errorf("Match(%q) = %p, %v; want %p with captures %q and value", path, got, vars, want, id)
					return
				}
				vars.GetAll("id")[0] = "changed"
			}
		})
	}
	workers.Wait()
}

func FuzzMatchFallback(f *testing.F) {
	router := mux.New()
	literal := router.Get("/items/new", handler)
	integer := router.Get("/items/<<int:id>>", handler)
	variable := router.Post("/items/<<slug>>", handler)
	glob := router.Any("/items/<<glob:rest>>", handler)

	for _, seed := range []struct{ method, path string }{
		{mux.GET, "/items/new"},
		{mux.GET, "/items/0004"},
		{mux.POST, "//items/new///"},
		{mux.POST, "/items/東京"},
		{mux.DELETE, "/items/new"},
		{mux.GET, "/items"},
		{mux.POST, "/items//x/"},
		{mux.ANY, "/items/42"},
		{mux.GET, "/missing/42"},
	} {
		f.Add(seed.method, seed.path)
	}
	f.Fuzz(func(t *testing.T, method, path string) {
		var want *mux.Route
		var key, value string
		base, rest, hasRest := strings.Cut(strings.Trim(path, "/"), "/")
		if base == "items" {
			singleSegment := hasRest && !strings.Contains(rest, "/")
			isNumber := singleSegment && strings.Trim(rest, "0123456789") == ""
			switch {
			case (method == mux.GET || method == mux.ANY) && rest == "new":
				want = literal
			case (method == mux.GET || method == mux.ANY) && isNumber:
				want, key, value = integer, "id", rest
			case (method == mux.POST || method == mux.ANY) && singleSegment:
				want, key, value = variable, "slug", rest
			default:
				want, key, value = glob, "rest", rest
			}
		}

		got, vars := router.Match(method, path)
		if got != want {
			t.Fatalf("Match(%q, %q) = %p, want %p", method, path, got, want)
		}
		if key == "" {
			if vars != nil {
				t.Fatalf("unexpected captures: %v", vars)
			}
		} else if actual, found := vars.GetOK(key); !found || actual != value || vars.Len() != 1 {
			t.Fatalf("captures = %v, want %q=%q", vars, key, value)
		}
	})
}
