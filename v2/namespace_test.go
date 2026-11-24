package mux_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mux "github.com/Nigel2392/mux/v2"
)

func TestNamespaceRegistrationAndServeHooks(t *testing.T) {
	type hookKey struct{}
	router := mux.New()
	var added []*mux.Route
	ns := router.Namespace(mux.NamespaceOptions{
		OnRouteAdded: func(route *mux.Route) {
			if got, _ := router.Match(route.Method, route.String()); got != route {
				t.Fatalf("hook saw unregistered route %q", route.String())
			}
			added = append(added, route)
			route.Name = "api:" + route.Name
		},
		OnRouteServe: func(r *http.Request) *http.Request {
			return r.WithContext(context.WithValue(r.Context(), hookKey{}, "api"))
		},
	})

	ns.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Mux", "global")
			next.ServeHTTP(w, r)
		})
	})
	write := func(w http.ResponseWriter, r *http.Request) {
		route := mux.RouteFromContext(r.Context())
		fmt.Fprintf(w, "%s %v", route.Name, r.Context().Value(hookKey{}))
	}
	first := ns.HandleFunc(mux.GET, "/first", write, "first")
	second := mux.NewRoute(mux.GET, "/second", http.HandlerFunc(write), "second")
	ns.AddRoute(second)
	router.HandleFunc(mux.GET, "/outside", write, "outside")
	if len(added) != 2 || added[0] != first || added[1] != second {
		t.Fatalf("registration hook received %v", added)
	}
	if got := ns.(interface{ Unwrap() mux.Multiplexer }).Unwrap(); got != router {
		t.Fatalf("Unwrap() = %v, want original router", got)
	}

	for _, tt := range []struct{ path, body string }{
		{"/first", "api:first api"},
		{"/second", "api:second api"},
		{"/outside", "outside <nil>"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, httptest.NewRequest(mux.GET, tt.path, nil))
			if resp.Code != http.StatusOK || resp.Body.String() != tt.body || resp.Header().Get("X-Mux") != "global" {
				t.Fatalf("got status %d, body %q, and headers %v", resp.Code, resp.Body, resp.Header())
			}
		})
	}
}

func TestNamespaceServeHookRunsOnceForNestedRoutes(t *testing.T) {
	router := mux.New()
	count := 0
	ns := router.Namespace(mux.NamespaceOptions{
		OnRouteServe: func(r *http.Request) *http.Request {
			count++
			return r
		},
	})
	root := mux.NewRoute(mux.GET, "/team", nil)
	child := root.Get("/child", nil)
	child.Get("/early", handler)
	ns.AddRoute(root)
	child.Get("/late", handler)

	for _, path := range []string{"/team/child/early", "/team/child/late"} {
		count = 0
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest(mux.GET, path, nil))
		if resp.Code != http.StatusOK || count != 1 {
			t.Errorf("%s: got status %d and %d hook calls", path, resp.Code, count)
		}
	}
}

func TestNamespaceRegistrationHookCanMoveSubtree(t *testing.T) {
	router := mux.New()
	ns := router.Namespace(mux.NamespaceOptions{
		OnRouteAdded: func(route *mux.Route) {
			if err := route.SetPath("/new"); err != nil {
				t.Fatalf("SetPath() failed: %v", err)
			}
		},
	})
	root := mux.NewRoute(mux.GET, "/old", nil)
	child := root.Get("/child", handler)
	ns.AddRoute(root)

	if got, _ := router.Match(mux.GET, "/new/child"); got != child {
		t.Fatalf("new path matched %v, want %v", got, child)
	}
	if got, _ := router.Match(mux.GET, "/old/child"); got != nil {
		t.Fatalf("old path still matches %v", got)
	}
}

func TestNamespaceHTTPMethodHelpers(t *testing.T) {
	router := mux.New()
	count := 0
	ns := router.Namespace(mux.NamespaceOptions{
		OnRouteAdded: func(*mux.Route) { count++ },
	})
	extras := ns.(interface {
		Head(string, http.Handler, ...string) *mux.Route
		Options(string, http.Handler, ...string) *mux.Route
	})
	for _, tt := range []struct {
		method, path string
		register     func(string, http.Handler, ...string) *mux.Route
	}{
		{mux.GET, "/get", ns.Get},
		{mux.POST, "/post", ns.Post},
		{mux.PUT, "/put", ns.Put},
		{mux.PATCH, "/patch", ns.Patch},
		{mux.DELETE, "/delete", ns.Delete},
		{mux.HEAD, "/head", extras.Head},
		{mux.OPTIONS, "/options", extras.Options},
		{"CUSTOM", "/any", ns.Any},
	} {
		t.Run(tt.method, func(t *testing.T) {
			want := tt.register(tt.path, handler)
			if route, _ := router.Match(tt.method, tt.path); route != want {
				t.Fatalf("Match(%q, %q) = %v, want %v", tt.method, tt.path, route, want)
			}
		})
	}
	if count != 8 {
		t.Fatalf("registration hooks called %d times, want 8", count)
	}
}

func TestNamespaceRejectedAttachmentDoesNotRunHooks(t *testing.T) {
	router := mux.New()
	registered := router.Get("/already-registered", handler)
	count := 0
	ns := router.Namespace(mux.NamespaceOptions{
		OnRouteAdded: func(*mux.Route) { count++ },
	})
	defer func() {
		recovered := recover()
		if err, ok := recovered.(error); !ok || !errors.Is(err, mux.ErrRouteAttached) {
			t.Errorf("AddRoute panic = %v, want %v", recovered, mux.ErrRouteAttached)
		}
		if count != 0 {
			t.Errorf("rejected registration ran %d hooks", count)
		}
	}()
	ns.AddRoute(registered)
}

func TestNamespaceRequiresHook(t *testing.T) {
	defer func() {
		if got := recover(); got == nil {
			t.Error("Namespace without hooks succeeded")
		}
	}()
	mux.New().Namespace(mux.NamespaceOptions{})
}
