//go:build !js && !wasm
// +build !js,!wasm

package mux

import "net/http"

// A function that handles a request.
func (r *Route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Handler.ServeHTTP(w, req)
}

func (r *Route) Get(path string, handler Handler, name ...string) *Route {
	return r.Handle(GET, path, handler, name...)
}

func (r *Route) Post(path string, handler Handler, name ...string) *Route {
	return r.Handle(POST, path, handler, name...)
}

func (r *Route) Put(path string, handler Handler, name ...string) *Route {
	return r.Handle(PUT, path, handler, name...)
}

func (r *Route) Delete(path string, handler Handler, name ...string) *Route {
	return r.Handle(DELETE, path, handler, name...)
}

func (r *Route) Head(path string, handler Handler, name ...string) *Route {
	return r.Handle(HEAD, path, handler, name...)
}

func (r *Route) Patch(path string, handler Handler, name ...string) *Route {
	return r.Handle(PATCH, path, handler, name...)
}

func (r *Route) Options(path string, handler Handler, name ...string) *Route {
	return r.Handle(OPTIONS, path, handler, name...)
}

func (r *Route) Any(path string, handler Handler, name ...string) *Route {
	return r.Handle(ANY, path, handler, name...)
}

func (r *Route) HandleFunc(method string, path string, handler func(w http.ResponseWriter, r *http.Request), name ...string) *Route {
	return r.Handle(method, path, NewHandler(handler), name...)
}

func (r *Route) AddRoute(rt *Route) {
	rt.Parent = r
	rt.ParentMux = r.ParentMux
	if rt.identifier == 0 {
		rt.identifier = randInt64()
	}

	rt.Path = r.Path.CopyAppend(
		rt.Path,
	)

	r.Children = append(r.Children, rt)
}

// Handle adds a handler to the route.
//
// It returns the route that was added so that it can be used to add children.
func (r *Route) Handle(method string, path string, handler Handler, name ...string) *Route {
	var route = NewRoute(method, path, handler, name...)
	route.Path = r.Path.CopyAppend(
		route.Path,
	)
	route.Parent = r
	route.ParentMux = r.ParentMux
	r.Children = append(r.Children, route)
	return route
}
