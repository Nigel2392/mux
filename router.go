package mux

import (
	"net/http"
	"strings"
)

const (
	// A special method, which will not be returned by any request, but can be set on the route to allow any method to pass.
	ANY     = "*"
	GET     = http.MethodGet     // Equivalent to http.MethodGet
	POST    = http.MethodPost    // Equivalent to http.MethodPost
	PUT     = http.MethodPut     // Equivalent to http.MethodPut
	DELETE  = http.MethodDelete  // Equivalent to http.MethodDelete
	HEAD    = http.MethodHead    // Equivalent to http.MethodHead
	PATCH   = http.MethodPatch   // Equivalent to http.MethodPatch
	OPTIONS = http.MethodOptions // Equivalent to http.MethodOptions
)

// Middleware which will run before/after the HandleFunc.
type Middleware func(next Handler) Handler

// The muxer.
type Mux struct {
	routes          []*Route
	middleware      []Middleware
	NotFoundHandler http.HandlerFunc
}

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

func (r *Mux) Find(name string) *Route {
	var nameparts = strings.Split(name, ":")
	for _, route := range r.routes {
		var route, ok = route.Find(nameparts)
		if ok {
			return route
		}
	}
	return nil
}

func (r *Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var route, variables = r.Match(req.Method, req.URL.Path)
	if route == nil {
		r.NotFound(w, req)
		return
	}

	req = SetVariables(req, variables)

	var handler Handler = route
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}
	handler.ServeHTTP(w, req)
}

func (r *Mux) NotFound(w http.ResponseWriter, req *http.Request) {
	if r.NotFoundHandler != nil {
		r.NotFoundHandler(w, req)
		return
	}
	http.NotFound(w, req)
}

func (r *Mux) Match(method string, path string) (*Route, Variables) {
	var parts = SplitPath(path)
	for _, route := range r.routes {
		var rt, matched, variables = route.Match(method, parts)
		if matched {
			return rt, variables
		}
	}
	return nil, nil
}

func (r *Mux) Handle(method string, path string, handler HandleFunc, name ...string) *Route {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	var route = &Route{
		Path:       NewPathInfo(path),
		HandleFunc: handler,
		Name:       n,
		Method:     method,
		ParentMux:  r,
		identifier: randInt64(),
	}
	r.routes = append(r.routes, route)
	return route
}

func (r *Mux) Get(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(GET, path, handler, name...)
}

func (r *Mux) Post(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(POST, path, handler, name...)
}

func (r *Mux) Put(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(PUT, path, handler, name...)
}

func (r *Mux) Delete(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(DELETE, path, handler, name...)
}

func (r *Mux) Head(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(HEAD, path, handler, name...)
}

func (r *Mux) Patch(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(PATCH, path, handler, name...)
}

func (r *Mux) Options(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(OPTIONS, path, handler, name...)
}

func (r *Mux) Any(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(ANY, path, handler, name...)
}
