package mux_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	mux "github.com/Nigel2392/mux/v2"
	"github.com/Nigel2392/mux/v2/middleware"
)

func recordMiddleware(events *[]string, label string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			*events = append(*events, label)
			next.ServeHTTP(w, req)
			*events = append(*events, "/"+label)
		})
	}
}

func TestHTTPMiddlewarePhases(t *testing.T) {
	for _, mode := range []string{"ServeHTTP", "Resolve"} {
		t.Run(mode, func(t *testing.T) {
			router := mux.New()
			var events []string
			router.Use(recordMiddleware(&events, "mux-first"), recordMiddleware(&events, "mux-second"))
			parent := mux.NewRoute(mux.GET, "/teams/<<team>>", nil)
			parent.Use(recordMiddleware(&events, "parent"))
			parent.Preprocess(recordMiddleware(&events, "parent-preprocess"))
			var child *mux.Route
			child = parent.Get("/members/<<int:id>>", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				events = append(events, "handler")
				if got := mux.RouteFromContext(req.Context()); got != child {
					t.Errorf("RouteFromContext = %p, want %p", got, child)
				}
				if vars := mux.Vars(req); vars.Get("team") != "ops" || vars.Get("id") != "42" {
					t.Errorf("Vars = %v, want team=ops and id=42", vars)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			child.Use(recordMiddleware(&events, "child-first"))
			child.Preprocess(recordMiddleware(&events, "pre-first"), recordMiddleware(&events, "pre-second"))
			router.AddRoute(parent)
			child.Use(recordMiddleware(&events, "child-second"))

			const path = "/teams/ops/members/42"
			req := httptest.NewRequest(mux.GET, path, nil)
			var h http.Handler = router
			if mode == "Resolve" {
				resolved, vars, ok := router.Resolve(mux.GET, path)
				if !ok || vars.Get("id") != "42" {
					t.Fatalf("Resolve = %v, %v; want id=42 and a match", vars, ok)
				}
				h = resolved
			}

			want := []string{
				"pre-first", "pre-second", "mux-first", "mux-second", "parent", "child-first", "child-second",
				"handler", "/child-second", "/child-first", "/parent", "/mux-second", "/mux-first", "/pre-second", "/pre-first",
			}
			for _, tt := range []struct {
				enabled bool
				want    []string
			}{{true, want}, {false, []string{"handler"}}, {true, want}} {
				child.RunsMiddleware(tt.enabled)
				events = nil
				out := httptest.NewRecorder()
				h.ServeHTTP(out, req)
				if out.Code != http.StatusNoContent || !slices.Equal(events, tt.want) {
					t.Fatalf("middleware enabled=%v: got status %d and calls %v, want status 204 and calls %v", tt.enabled, out.Code, events, tt.want)
				}
			}
		})
	}
}

func TestHTTPRouteMiddlewareSnapshotsKeepLoginPublic(t *testing.T) {
	for _, mode := range []string{"detached", "attached"} {
		t.Run(mode, func(t *testing.T) {
			router := mux.New()
			admin := mux.NewRoute(mux.ANY, "/admin", nil)
			if mode == "attached" {
				router.AddRoute(admin)
			}
			admin.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-App", "admin")
					next.ServeHTTP(w, r)
				})
			})
			public := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			login := admin.Get("/login/", public)
			admin.Post("/login/", public)
			admin.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") == "" {
						http.Redirect(w, r, "/admin/login/", http.StatusFound)
						return
					}
					next.ServeHTTP(w, r)
				})
			})
			login.Get("/help", public)
			app := admin.Get("/apps/admin_test", public)
			app.Get("/settings", public)
			if mode == "detached" {
				router.AddRoute(admin)
			}

			for _, tt := range []struct {
				method, path  string
				authorization string
				status        int
				location      string
			}{
				{mux.GET, "/admin/apps/admin_test", "", http.StatusFound, "/admin/login/"},
				{mux.GET, "/admin/apps/admin_test/settings", "", http.StatusFound, "/admin/login/"},
				{mux.GET, "/admin/login/", "", http.StatusOK, ""},
				{mux.POST, "/admin/login/", "", http.StatusOK, ""},
				{mux.GET, "/admin/login/help", "", http.StatusOK, ""},
				{mux.GET, "/admin/apps/admin_test", "test-user", http.StatusOK, ""},
			} {
				t.Run(tt.method+" "+tt.path+" "+tt.authorization, func(t *testing.T) {
					req := httptest.NewRequest(tt.method, tt.path, nil)
					req.Header.Set("Authorization", tt.authorization)
					out := httptest.NewRecorder()
					router.ServeHTTP(out, req)
					if loc := out.Header().Get("Location"); out.Code != tt.status || loc != tt.location {
						t.Fatalf("response status and location = %d, %q; want %d, %q", out.Code, loc, tt.status, tt.location)
					}
					if app := out.Header().Get("X-App"); app != "admin" {
						t.Errorf("app middleware set header to %q, want admin", app)
					}
				})
			}
		})
	}
}

type bindingHandler struct {
	bind func(*http.Request, *mux.Route, mux.Variables) http.Handler
}

func (h bindingHandler) Bind(r *http.Request, route *mux.Route, vars mux.Variables) http.Handler {
	return h.bind(r, route, vars)
}

func (bindingHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	panic("ServeHTTP called on an unbound handler")
}

func TestHTTPBindReceivesMatchedRequest(t *testing.T) {
	router := mux.New()
	var route *mux.Route
	var events []string
	route = router.Get("/files/*", bindingHandler{bind: func(r *http.Request, rt *mux.Route, vars mux.Variables) http.Handler {
		events = append(events, "bind")
		rest := []string{"image", "blue.png"}
		if rt != route || mux.RouteFromContext(r.Context()) != route ||
			!slices.Equal(mux.Vars(r).GetAll("*"), rest) || !slices.Equal(vars.GetAll("*"), rest) {
			t.Errorf("Bind received route %p, context route %p and captures %v", rt, mux.RouteFromContext(r.Context()), vars)
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			events = append(events, "bound")
			fmt.Fprint(w, strings.Join(vars.GetAll("*"), "/"))
		})
	}})
	router.Use(recordMiddleware(&events, "mux"))
	out := httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest(mux.GET, "/files/image/blue.png", nil))
	if out.Body.String() != "image/blue.png" || !slices.Equal(events, []string{"bind", "mux", "bound", "/mux"}) {
		t.Fatalf("bound request returned body %q with calls %v", out.Body.String(), events)
	}
}

func TestHTTPNotFound(t *testing.T) {
	router := mux.New()
	router.Get("/present", handler)
	router.Use(func(next http.Handler) http.Handler {
		t.Error("middleware ran for an unmatched request")
		return next
	})
	req := httptest.NewRequest(mux.POST, "/present", nil)
	h, vars, ok := router.Resolve(req.Method, req.URL.Path)
	if h == nil || vars != nil || ok {
		t.Fatalf("Resolve on miss = %v, %v, %v; want fallback handler, nil, false", h, vars, ok)
	}
	for _, endpoint := range []http.Handler{router, h} {
		out := httptest.NewRecorder()
		endpoint.ServeHTTP(out, req)
		if out.Code != http.StatusNotFound {
			t.Errorf("default missing status = %d, want 404", out.Code)
		}
	}

	router.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		if r != req {
			t.Error("NotFoundHandler received a different request")
		}
		w.WriteHeader(http.StatusGone)
	}
	for _, endpoint := range []http.Handler{router, h} {
		out := httptest.NewRecorder()
		endpoint.ServeHTTP(out, req)
		if out.Code != http.StatusGone {
			t.Errorf("custom missing status = %d, want 410", out.Code)
		}
	}
}

func TestHTTPContextHelpers(t *testing.T) {
	router := mux.New()
	route := router.Get("/files/<<id>>", handler)
	_, vars := router.Match(mux.GET, "/files/foo")
	req := httptest.NewRequest(mux.GET, "/files/foo", nil)
	withRoute := req.WithContext(mux.ContextWithRoute(req.Context(), route))
	withValues := mux.SetContextVars(withRoute, vars)
	if mux.Vars(req) != nil || mux.RouteFromContext(req.Context()) != nil {
		t.Fatal("context helpers mutated the original request")
	}
	if got := mux.RouteFromContext(withValues.Context()); got != route {
		t.Errorf("RouteFromContext = %p, want %p", got, route)
	}
	if got := mux.Vars(withValues).Get("id"); got != "foo" {
		t.Errorf("Vars.Get(id) = %q, want foo", got)
	}
	foreign := context.WithValue(req.Context(), mux.ContextKey{K: "mux.variables"}, 42)
	if got := mux.Vars(req.WithContext(foreign)); got != nil {
		t.Errorf("Vars with a foreign context value = %v, want nil", got)
	}
}
