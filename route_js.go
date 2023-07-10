//go:build js && wasm
// +build js,wasm

package mux

// A function that handles a request.
func (r *Route) ServeHTTP(v Variables) {
	r.HandleFunc(v)
}
