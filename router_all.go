//go:build !js && !wasm
// +build !js,!wasm

package mux

import "net/http"

// The muxer.
type Mux struct {
	routes          []*Route
	middleware      []Middleware
	NotFoundHandler http.HandlerFunc
	ErrorHandler    func(w http.ResponseWriter, r *http.Request, err error)
}

func (r *Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var route, variables, err = r.Match(req.Method, req.URL.Path)
	if err != nil {
		r.HandleError(w, req, err)
		return
	}

	if route == nil || route.Handler == nil {
		r.NotFound(w, req)
		return
	}

	req = SetVariables(req, variables)

	var handler Handler = route
	for i := len(route.Middleware) - 1; i >= 0; i-- {
		handler = route.Middleware[i](handler)
	}

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

func (r *Mux) HandleError(w http.ResponseWriter, req *http.Request, err error) {
	if r.ErrorHandler != nil {
		r.ErrorHandler(w, req, err)
		return
	}
	http.Error(w,
		http.StatusText(http.StatusInternalServerError),
		http.StatusInternalServerError,
	)
}

func (r *Mux) Match(method string, path string) (*Route, Variables, error) {
	var parts = SplitPath(path)
	for _, route := range r.routes {
		var rt, matched, variables, err = route.Match(method, parts)
		if matched {
			return rt, variables, err
		}
	}
	return nil, nil, nil
}

func (r *Mux) Handle(method string, path string, handler Handler, name ...string) *Route {
	var route = NewRoute(method, path, handler, name...)
	r.routes = append(r.routes, route)
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
