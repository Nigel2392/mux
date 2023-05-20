package mux

import "net/http"

type HandleFunc func(w http.ResponseWriter, req *http.Request)

type handler struct {
	handleFunc HandleFunc
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.handleFunc(w, req)
}

func Handler(f HandleFunc) http.Handler {
	return &handler{
		handleFunc: f,
	}
}
