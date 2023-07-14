//go:build js && wasm
// +build js,wasm

package mux

// A function that handles a request.
func (r *Route) ServeHTTP(v Variables) {
	r.HandleFunc(v)
}

// Handle adds a handler to the route.
//
// It returns the route that was added so that it can be used to add children.
func (r *Route) Handle(path string, handler HandleFunc, name ...string) *Route {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	var route = &Route{
		Path: r.Path.CopyAppend(
			NewPathInfo(path),
		),
		HandleFunc: handler,
		Name:       n,
		Method:     ANY,
		Parent:     r,
		ParentMux:  r.ParentMux,
		identifier: randInt64(),
	}
	r.Children = append(r.Children, route)
	if r.ParentMux.running {
		r.ParentMux.compile()
	}
	return route
}
