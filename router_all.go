//go:build !js && !wasm
// +build !js,!wasm

package mux

import "net/http"

// The muxer.
type Mux struct {
	routes          []*Route
	middleware      []Middleware
	NotFoundHandler http.HandlerFunc
}

func (r *Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var route, variables = r.Match(req.Method, req.URL.Path)
	if route == nil || route.Handler == nil {
		r.NotFound(w, req)
		return
	}

	req = SetVariables(req, variables)

	var handler Handler = route

	// Do not run middleware if disabled.
	//lint:ignore S1002 Extra verbose to make it more clear.
	if route.DisabledMiddleware == false {
		for i := len(route.Middleware) - 1; i >= 0; i-- {
			handler = route.Middleware[i](handler)
		}

		for i := len(r.middleware) - 1; i >= 0; i-- {
			handler = r.middleware[i](handler)
		}

		for i := len(route.PreMiddleware) - 1; i >= 0; i-- {
			handler = route.PreMiddleware[i](handler)
		}
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

func (r *Mux) Handle(method string, path string, handler Handler, name ...string) *Route {
	var route = NewRoute(method, path, handler, name...)
	r.routes = append(r.routes, route)

	setChildData(route, nil)

	for _, child := range route.Children {
		setChildData(child, route)
	}

	return route
}

func (r *Mux) AddRoute(rt *Route) {
	rt.ParentMux = r

	if rt.identifier == 0 {
		rt.identifier = randInt64()
	}

	for _, child := range rt.Children {
		setChildData(child, rt)
	}

	r.routes = append(r.routes, rt)
}

func (r *Mux) HandleFunc(method string, path string, handler http.HandlerFunc, name ...string) *Route {
	return r.Handle(method, path, handler, name...)
}

func (r *Mux) Get(path string, handler Handler, name ...string) *Route {
	return r.Handle(GET, path, handler, name...)
}

func (r *Mux) Post(path string, handler Handler, name ...string) *Route {
	return r.Handle(POST, path, handler, name...)
}

func (r *Mux) Put(path string, handler Handler, name ...string) *Route {
	return r.Handle(PUT, path, handler, name...)
}

func (r *Mux) Delete(path string, handler Handler, name ...string) *Route {
	return r.Handle(DELETE, path, handler, name...)
}

func (r *Mux) Head(path string, handler Handler, name ...string) *Route {
	return r.Handle(HEAD, path, handler, name...)
}

func (r *Mux) Patch(path string, handler Handler, name ...string) *Route {
	return r.Handle(PATCH, path, handler, name...)
}

func (r *Mux) Options(path string, handler Handler, name ...string) *Route {
	return r.Handle(OPTIONS, path, handler, name...)
}

func (r *Mux) Any(path string, handler Handler, name ...string) *Route {
	return r.Handle(ANY, path, handler, name...)
}
