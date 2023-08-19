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

// Handle adds a handler to the route.
//
// It returns the route that was added so that it can be used to add children.
func (r *Route) Handle(method string, path string, handler Handler, name ...string) *Route {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	var route = &Route{
		Path: r.Path.CopyAppend(
			NewPathInfo(path),
		),
		Handler:    handler,
		Name:       n,
		Method:     method,
		Parent:     r,
		ParentMux:  r.ParentMux,
		identifier: randInt64(),
	}
	r.Children = append(r.Children, route)
	return route
}
