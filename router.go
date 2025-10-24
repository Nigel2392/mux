package mux

import (
	"strings"
)

const (
	// A special method, which will not be returned by any request, but can be set on the route to allow any method to pass.
	ANY     = "*"
	GET     = "GET"     // Equivalent to http.MethodGet
	POST    = "POST"    // Equivalent to http.MethodPost
	PUT     = "PUT"     // Equivalent to http.MethodPut
	DELETE  = "DELETE"  // Equivalent to http.MethodDelete
	HEAD    = "HEAD"    // Equivalent to http.MethodHead
	PATCH   = "PATCH"   // Equivalent to http.MethodPatch
	OPTIONS = "OPTIONS" // Equivalent to http.MethodOptions

	NAME_SEPARATOR = ":"
)

// Middleware which will run before/after the HandleFunc.
type Middleware func(next Handler) Handler

func New() *Mux {
	return &Mux{}
}

func (r *Mux) Use(middleware ...Middleware) {
	r.middleware = append(r.middleware, middleware...)
}

func (r *Mux) RemoveByPath(path string) {
	for _, route := range r.routes {
		route.RemoveByPath(path)
	}
}

func (r *Mux) RemoveRoute(route *Route) {
	r.routes = removeRoute(r.routes, route)
}

func (r *Mux) ResetRoutes() {
	r.routes = make([]*Route, 0)
}

func (r *Mux) Find(name string) *Route {
	var nameparts = strings.Split(name, NAME_SEPARATOR)
	for _, route := range r.routes {
		var route, ok = route.Find(nameparts)
		if ok {
			return route
		}
	}
	return nil
}

func (r *Mux) Reverse(name string, variables ...interface{}) (string, error) {
	var route = r.Find(name)
	if route == nil {
		return "", ErrRouteNotFound
	}
	return route.Path.Reverse(variables...)
}
