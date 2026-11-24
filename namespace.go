package mux

import (
	"net/http"

	"github.com/Nigel2392/mux/v2/middleware"
)

// Multiplexer registers routes and middleware.
type Multiplexer interface {
	Use(...middleware.Middleware)
	Handle(method, path string, handler http.Handler, name ...string) *Route
	HandleFunc(method, path string, handler func(http.ResponseWriter, *http.Request), name ...string) *Route
	AddRoute(route *Route)

	Any(path string, handler http.Handler, name ...string) *Route
	Get(path string, handler http.Handler, name ...string) *Route
	Post(path string, handler http.Handler, name ...string) *Route
	Put(path string, handler http.Handler, name ...string) *Route
	Patch(path string, handler http.Handler, name ...string) *Route
	Delete(path string, handler http.Handler, name ...string) *Route
}

// NamespaceOptions configures the registration and request hooks for a namespace.
type NamespaceOptions struct {
	OnRouteAdded func(route *Route)
	OnRouteServe func(r *http.Request) *http.Request
}

type namespace struct {
	Multiplexer
	options NamespaceOptions
}

var (
	_ Multiplexer = (*Mux)(nil)
	_ Multiplexer = (*Route)(nil)
	_ Multiplexer = (*namespace)(nil)
)

// Namespace adds hooks to routes registered through the returned Multiplexer.
func (m *Mux) Namespace(opts NamespaceOptions) Multiplexer {
	if opts.OnRouteAdded == nil && opts.OnRouteServe == nil {
		panic("mux: NamespaceOptions must have at least one of OnRouteAdded or OnRouteServe set")
	}
	return &namespace{Multiplexer: m, options: opts}
}

func (ns *namespace) Unwrap() Multiplexer {
	return ns.Multiplexer
}

func (ns *namespace) Handle(method, path string, handler http.Handler, name ...string) *Route {
	route := NewRoute(method, path, handler, name...)
	ns.AddRoute(route)
	return route
}

func (ns *namespace) AddRoute(route *Route) {
	ns.Multiplexer.AddRoute(route)
	if hook := ns.options.OnRouteAdded; hook != nil {
		hook(route)
	}
	if hook := ns.options.OnRouteServe; hook != nil {
		// A namespace hook covers the entire mounted subtree, including
		// children built before the subtree entered the namespace.
		route.namespaceMiddleware = append(route.namespaceMiddleware, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, hook(r))
			})
		})
	}
}

func (ns *namespace) HandleFunc(method, path string, handler func(http.ResponseWriter, *http.Request), name ...string) *Route {
	return ns.Handle(method, path, http.HandlerFunc(handler), name...)
}

func (ns *namespace) Any(path string, handler http.Handler, name ...string) *Route {
	return ns.Handle(ANY, path, handler, name...)
}

func (ns *namespace) Get(path string, handler http.Handler, name ...string) *Route {
	return ns.Handle(GET, path, handler, name...)
}

func (ns *namespace) Post(path string, handler http.Handler, name ...string) *Route {
	return ns.Handle(POST, path, handler, name...)
}

func (ns *namespace) Put(path string, handler http.Handler, name ...string) *Route {
	return ns.Handle(PUT, path, handler, name...)
}

func (ns *namespace) Patch(path string, handler http.Handler, name ...string) *Route {
	return ns.Handle(PATCH, path, handler, name...)
}

func (ns *namespace) Delete(path string, handler http.Handler, name ...string) *Route {
	return ns.Handle(DELETE, path, handler, name...)
}

func (ns *namespace) Head(path string, handler http.Handler, name ...string) *Route {
	return ns.Handle(HEAD, path, handler, name...)
}

func (ns *namespace) Options(path string, handler http.Handler, name ...string) *Route {
	return ns.Handle(OPTIONS, path, handler, name...)
}
