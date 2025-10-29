//go:build !js && !wasm
// +build !js,!wasm

package mux

import "net/http"

type Handler = http.Handler

type BindableHandler interface {
	Handler
	Bind(r *http.Request, rt *Route, vars Variables) Handler
}

type HandleFunc = func(w http.ResponseWriter, req *http.Request)

type FuncHandler struct {
	Func HandleFunc
}

func (h *FuncHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.Func(w, req)
}

func NewHandler(f HandleFunc) Handler {
	if f == nil {
		return nil
	}
	return &FuncHandler{
		Func: f,
	}
}
