//go:build !uncompliant
// +build !uncompliant

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
	return &handler{
		handleFunc: f,
	}
}
