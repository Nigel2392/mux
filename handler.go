//go:build !js && !wasm
// +build !js,!wasm

package mux

import "net/http"

type Handler http.Handler

type HandleFunc func(w http.ResponseWriter, req *http.Request)

type handler struct {
	handleFunc HandleFunc
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.handleFunc(w, req)
}

func NewHandler(f HandleFunc) Handler {
	if f == nil {
		return nil
	}
	return &handler{
		handleFunc: f,
	}
}
