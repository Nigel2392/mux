//go:build js && wasm
// +build js,wasm

package mux

import (
	"fmt"
	"net/url"
	"strings"
	"syscall/js"
)

var global = js.Global()

func init() {
	global.Set("newMux", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var m = New()
		var muxObject = global.Get("Object").New()
		muxObject.Set("handlePath", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) < 1 {
				panic("handlePath requires at least one argument")
			}
			var path = args[0].String()
			m.HandlePath(path)
			return nil
		}))
		muxObject.Set("run", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			m.ListenForChanges()
			return nil
		}))
		muxObject.Set("initialPage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) < 1 {
				panic("initialPage requires at least one argument")
			}
			var path = args[0].String()
			m.FirstPage(path)
			return nil
		}))
		muxObject.Set("alwaysInvokeHandler", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) < 1 {
				panic("alwaysInvokeHandler requires at least one argument")
			}
			var b = args[0].Bool()
			m.InvokeHandler(b)
			return nil
		}))
		muxObject.Set("handle", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) < 2 {
				panic("handle requires at least two arguments")
			}
			var path = args[0].String()
			var handler = args[1]
			m.Handle(path, NewHandler(func(v Variables) {
				var varsObject = global.Get("Object").New()
				for k, value := range v {
					varsObject.Set(k, sliceToJSArray(value))
				}
				handler.Invoke(varsObject)
			}))
			return nil
		}))
		return muxObject
	}))
}

func sliceToJSArray(slice []string) js.Value {
	var array = global.Get("Array").New(len(slice))
	for i, v := range slice {
		array.SetIndex(i, v)
	}
	return array
}

const RT_PREFIX_EXTERNAL = "external://"

// The muxer.
type Mux struct {
	routes          []*Route
	middleware      []Middleware
	NotFoundHandler Handler

	running              bool
	routerChangePageFunc js.Func
	jsChangePageFunc     js.Func
	firstPageURL         string
	currentPath          string
	alwaysInvokeRoute    bool
}

func (r *Mux) FirstPage(path string) {
	r.firstPageURL = path
}

func (r *Mux) InvokeHandler(b bool) {
	r.alwaysInvokeRoute = b
}

func (r *Mux) ListenForChanges() {
	if r.running {
		// remove event listeners
		if r.routerChangePageFunc.Type() != js.TypeUndefined {
			global.Get("document").Call("removeEventListener", "click", r.routerChangePageFunc)
		}
		if r.jsChangePageFunc.Type() != js.TypeUndefined {
			global.Get("window").Call("removeEventListener", "popstate", r.jsChangePageFunc)
		}
	}

	// add event listeners
	r.routerChangePageFunc = js.FuncOf(r.changePage)
	r.jsChangePageFunc = js.FuncOf(r.jsChangePage)

	global.Get("document").Call("addEventListener", "click", r.routerChangePageFunc)
	global.Get("window").Call("addEventListener", "popstate", r.jsChangePageFunc)

	r.running = true
	if r.firstPageURL != "" {
		r.HandlePath(r.firstPageURL)
	} else {
		var path = global.Get("location").Get("href").String()
		r.HandlePath(path)
	}

}

func (r *Mux) compile() {
	if r.running {
		// remove event listeners
		if r.routerChangePageFunc.Type() != js.TypeUndefined {
			global.Get("document").Call("removeEventListener", "click", r.routerChangePageFunc)
		}
		if r.jsChangePageFunc.Type() != js.TypeUndefined {
			global.Get("window").Call("removeEventListener", "popstate", r.jsChangePageFunc)
		}
	}

	// add event listeners
	r.routerChangePageFunc = js.FuncOf(r.changePage)
	r.jsChangePageFunc = js.FuncOf(func(this js.Value, args []js.Value) any {
		return _jsChangePage(this, args, r)
	})

	global.Get("document").Call("addEventListener", "click", r.routerChangePageFunc)
	global.Get("window").Call("addEventListener", "popstate", r.jsChangePageFunc)
}

// Change the page to the given path.
func (r *Mux) changePage(this js.Value, args []js.Value) any {
	// Get the object if it is valid.
	var event = args[0]
	if event.Type() != js.TypeObject {
		return nil
	}
	var target = event.Get("target")
	// If the target is not a link, return.
	if target.Type() == js.TypeUndefined {
		return nil
	}

	var max int = 5
	var i int
	for (!target.Equal(js.Undefined()) && !target.Equal(js.Null())) && target.Get("nodeName").String() != "A" && target.Get("nodeName").String() != "a" && i < max {
		target = target.Get("parentNode")
		i++
	}

	if target.Equal(js.Undefined()) || target.Equal(js.Null()) {
		return nil
	}

	// Check if the event came from a link.
	if tagname := target.Get("nodeName").String(); tagname != "A" && tagname != "a" {
		return nil
	}

	event.Call("preventDefault")

	// Only stop the default action if the link is an internal link
	// Which means it starts with the RT_PREFIX and we need to handle it
	var path = target.Get("href").String()
	var url, err = url.Parse(path)
	if err != nil {
		return nil
	}

	if strings.HasPrefix(path, RT_PREFIX_EXTERNAL) {
		path = strings.TrimPrefix(path, RT_PREFIX_EXTERNAL)
		global.Get("window").Call("open", path, "_blank")
		return nil
	}

	// If the path is empty, set it to the root.
	if url.Path == "" {
		url.Path = "/"
	}

	// If the path is the same as the current path, return.
	if url.Path == r.currentPath && !r.alwaysInvokeRoute {
		return nil
	}

	r.HandlePath(url.Path)
	return nil
}

func _jsChangePage(this js.Value, args []js.Value, r *Mux) interface{} {
	var event = args[0]
	event.Call("preventDefault")
	var path = global.Get("location").Get("href").String()
	r.HandlePath(path)
	return nil
}

func (r *Mux) jsChangePage(this js.Value, args []js.Value) interface{} {
	return _jsChangePage(this, args, r)
}

func (r *Mux) HandlePath(path string) {
	var url, err = url.Parse(path)
	if err != nil {
		goto execPath
	}
	path = url.Path

execPath:
	var route, variables = r.Match(path)
	if route == nil {
		r.NotFound(map[string][]string{"path": {path}})
		return
	}

	for k, v := range url.Query() {
		if len(v) == 0 {
			continue
		}
		if values, ok := variables[k]; ok {
			variables["queryparam_"+k] = append(values, v...)
		} else {
			variables["queryparam_"+k] = v
		}
	}

	r.currentPath = path
	if variables == nil {
		variables = make(map[string][]string)
	}

	if paths, ok := variables["path"]; ok {
		variables["path"] = append(paths, path)
	} else {
		variables["path"] = []string{path}
	}

	pushState(js.Null(), path)

	var handler Handler = route
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	go handler.ServeHTTP(variables)
}

func (r *Mux) NotFound(v Variables) {
	if r.NotFoundHandler != nil {
		r.NotFoundHandler.ServeHTTP(v)
		return
	}

	fmt.Printf("404: %s\n", v.Get("path"))
}

func (r *Mux) Match(path string) (*Route, Variables) {
	var parts = SplitPath(path)
	for _, route := range r.routes {
		var rt, matched, variables = route.Match(ANY, parts)
		if matched {
			return rt, variables
		}
	}
	return nil, nil
}

// Handle adds a handler to the route.
//
// It returns the route that was added so that it can be used to add children.
func (r *Mux) Handle(path string, handler Handler, name ...string) *Route {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	var route = &Route{
		Path:       NewPathInfo(path),
		Handler:    handler,
		Name:       n,
		Method:     ANY,
		ParentMux:  r,
		identifier: randInt64(),
	}
	r.routes = append(r.routes, route)

	if r.running {
		r.compile()
	}

	return route
}

func pushState(state js.Value, path string) {
	global.Get("history").Call("pushState", state, nil, path)
}
