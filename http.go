package mux

import (
	"net/http"
	"slices"

	"github.com/Nigel2392/mux/v2/middleware"
)

var _ http.Handler = (*Mux)(nil)

// BindableHandler selects a handler for a request after routing.
type BindableHandler interface {
	http.Handler
	Bind(r *http.Request, route *Route, vars Variables) http.Handler
}

// Use adds middleware for matched requests.
func (m *Mux) Use(middlewares ...middleware.Middleware) {
	m.middleware = append(m.middleware, middlewares...)
}

// Use adds middleware to the route. Children inherit it when they are added.
func (r *Route) Use(middlewares ...middleware.Middleware) {
	r.Middleware = append(r.Middleware, middlewares...)
}

func (r *Route) setParent(parent *Route) {
	r.Parent = parent
	r.parentMiddleware = nil
	r.middlewareBeforeParent = 0
	if parent != nil {
		r.parentMiddleware = slices.Clone(parent.Middleware)
		r.middlewareBeforeParent = len(r.Middleware)
	}
}

// Preprocess adds middleware that runs before the mux and route middleware.
func (r *Route) Preprocess(middlewares ...middleware.Middleware) {
	r.PreMiddleware = append(r.PreMiddleware, middlewares...)
}

// RunsMiddleware controls whether requests matched to the route run middleware.
func (r *Route) RunsMiddleware(enabled bool) {
	r.DisabledMiddleware = !enabled
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, vars := m.Match(req.Method, req.URL.Path)
	if route == nil {
		m.NotFound(w, req)
		return
	}
	m.serveRoute(w, req, route, vars)
}

// Resolve returns the handler and captures for a method and path. On a miss
// the returned handler serves the configured NotFound response.
func (m *Mux) Resolve(method, path string) (http.Handler, Variables, bool) {
	route, vars := m.Match(method, path)
	if route == nil {
		return http.HandlerFunc(m.NotFound), nil, false
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m.serveRoute(w, req, route, vars)
	}), vars, true
}

func (m *Mux) NotFound(w http.ResponseWriter, req *http.Request) {
	if m.NotFoundHandler != nil {
		m.NotFoundHandler(w, req)
		return
	}
	http.NotFound(w, req)
}

func (m *Mux) serveRoute(w http.ResponseWriter, req *http.Request, route *Route, vars Variables) {
	req = req.WithContext(contextWithVars(ContextWithRoute(req.Context(), route), vars))

	handler := route.Handler
	if binder, ok := handler.(BindableHandler); ok {
		handler = binder.Bind(req, route, vars)
	}
	if !route.DisabledMiddleware {
		handler = wrapRouteMiddleware(handler, route, route.Middleware)
		handler = wrapMiddleware(handler, m.middleware)
		handler = wrapMiddleware(handler, route.PreMiddleware)
	}
	handler.ServeHTTP(w, req)
}

func wrapRouteMiddleware(handler http.Handler, route *Route, local []middleware.Middleware) http.Handler {
	handler = wrapMiddleware(handler, route.namespaceMiddleware)
	if route.Parent != nil {
		// Middleware added to the child after attachment runs after the chain
		// it inherited at registration, as it did in v1.
		split := min(route.middlewareBeforeParent, len(local))
		handler = wrapMiddleware(handler, local[split:])
		handler = wrapRouteMiddleware(handler, route.Parent, route.parentMiddleware)
		local = local[:split]
	}
	return wrapMiddleware(handler, local)
}

func wrapMiddleware(handler http.Handler, middlewares []middleware.Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
