//go:build !js && !wasm
// +build !js,!wasm

package mux

import "net/http"

var _ Multiplexer = (*nameSpace[*Mux])(nil)

type NamespaceOptions struct {
	OnRouteAdded func(route *Route)
	OnRouteServe func(r *http.Request) *http.Request
}

type nameSpace[T Multiplexer] struct {
	Multiplexer  T
	onRouteAdded func(route *Route)
	onRouteServe func(r *http.Request) *http.Request
}

func newNamespace[T Multiplexer](mux T, options NamespaceOptions) *nameSpace[T] {
	ns := &nameSpace[T]{
		Multiplexer:  mux,
		onRouteAdded: options.OnRouteAdded,
		onRouteServe: options.OnRouteServe,
	}
	if ns.onRouteAdded == nil && ns.onRouteServe == nil {
		panic("mux: NamespaceOptions must have at least one of OnRouteAdded or OnRouteServe set")
	}
	return ns
}

func (ns *nameSpace[T]) Use(middleware ...Middleware) {
	ns.Multiplexer.Use(middleware...)
}

func (ns *nameSpace[T]) routeSetup(r *Route) {
	if ns.onRouteAdded != nil {
		ns.onRouteAdded(r)
	}

	if ns.onRouteServe != nil {
		r.Middleware = append(r.Middleware, func(next Handler) Handler {
			return NewHandler(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, ns.onRouteServe(r))
			})
		})
	}
}

func (ns *nameSpace[T]) Handle(method string, path string, handler Handler, name ...string) *Route {
	rt := ns.Multiplexer.Handle(method, path, handler, name...)
	ns.routeSetup(rt)
	return rt
}

func (ns *nameSpace[T]) HandleFunc(method string, path string, handler func(w http.ResponseWriter, r *http.Request), name ...string) *Route {
	return ns.Handle(method, path, http.HandlerFunc(handler), name...)
}

func (ns *nameSpace[T]) AddRoute(route *Route) {
	ns.routeSetup(route)
	ns.Multiplexer.AddRoute(route)
}

func (ns *nameSpace[T]) Any(path string, handler Handler, name ...string) *Route {
	return ns.Handle(http.MethodGet, path, handler, name...)
}

func (ns *nameSpace[T]) Get(path string, handler Handler, name ...string) *Route {
	return ns.Handle(http.MethodGet, path, handler, name...)
}

func (ns *nameSpace[T]) Post(path string, handler Handler, name ...string) *Route {
	return ns.Handle(http.MethodPost, path, handler, name...)
}

func (ns *nameSpace[T]) Put(path string, handler Handler, name ...string) *Route {
	return ns.Handle(http.MethodPut, path, handler, name...)
}

func (ns *nameSpace[T]) Patch(path string, handler Handler, name ...string) *Route {
	return ns.Handle(http.MethodPatch, path, handler, name...)
}

func (ns *nameSpace[T]) Delete(path string, handler Handler, name ...string) *Route {
	return ns.Handle(http.MethodDelete, path, handler, name...)
}

func (ns *nameSpace[T]) Unwrap() Multiplexer {
	return ns.Multiplexer
}
