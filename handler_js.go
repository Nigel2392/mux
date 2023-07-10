//go:build js && wasm
// +build js,wasm

package mux

type Handler interface {
	ServeHTTP(vars Variables)
}

type HandleFunc func(vars Variables)

type handler struct {
	handleFunc HandleFunc
}

func (h *handler) ServeHTTP(vars Variables) {
	h.handleFunc(vars)
}

func NewHandler(f HandleFunc) Handler {
	return &handler{
		handleFunc: f,
	}
}
