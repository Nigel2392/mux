//go:build !js && !wasm
// +build !js,!wasm

package mux

import "net/http"

// A function that handles a request.
func (r *Route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.HandleFunc(w, req)
}
