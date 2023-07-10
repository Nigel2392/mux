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
