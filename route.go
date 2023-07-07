package mux

import (
	"crypto/rand"
	"math"
	"math/big"
	"net/http"
	"strings"
)

type Route struct {
	Name       string
	Method     string
	Path       *PathInfo
	Children   []*Route
	HandleFunc HandleFunc
	Parent     *Route
	ParentMux  *Mux

	identifier int64
}

// String returns a string representation of the route.
func (r *Route) String() string {
	return r.Path.String()
}

func (r *Route) Find(names []string) (*Route, bool) {
	return r.find(names, 0)
}

func (r *Route) find(names []string, index int) (*Route, bool) {
	if len(names) <= index {
		return nil, false
	}
	if r.Name == names[index] && len(names)-1 == index {
		return r, true
	}
	for _, child := range r.Children {
		var route, ok = child.find(names, index+1)
		if ok {
			return route, ok
		}
	}
	return nil, false
}

func (r *Route) RemoveByPath(path string) bool {
	path = strings.Trim(path, "/")
	var routePath = strings.Trim(r.Path.String(), "/")
	if path == routePath && r.Parent != nil {
		r.Parent.RemoveChild(r)
		return true
	} else if path == routePath && r.Parent == nil {
		r.ParentMux.RemoveRoute(r)
		return true
	}
	for _, child := range r.Children {
		if child.RemoveByPath(path) {
			return true
		}
	}
	return false
}

func (r *Route) RemoveChild(child *Route) {
	removeRoute(r.Children, child)
}

// A function that handles a request.
func (r *Route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.HandleFunc(w, req)
}

// Helper function to check if the route matches the method and path.
func routeMatched(matched bool, method string, route *Route) bool {
	return matched && (route.Method == ANY || route.Method == method || method == ANY) && route.HandleFunc != nil
}

func (r *Route) Match(method string, path []string) (*Route, bool, Variables) {
	var matched, variables = r.Path.Match(path)
	if routeMatched(matched, method, r) {
		return r, matched, variables
	}
	for _, child := range r.Children {
		var rt, matched, variables = child.Match(method, path)
		if routeMatched(matched, method, rt) {
			return rt, matched, variables
		}
	}
	return nil, false, nil
}

// Handle adds a handler to the route.
//
// It returns the route that was added so that it can be used to add children.
func (r *Route) Handle(method string, path string, handler HandleFunc, name ...string) *Route {
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
		Method:     method,
		Parent:     r,
		ParentMux:  r.ParentMux,
		identifier: randInt64(),
	}
	r.Children = append(r.Children, route)
	return route
}

func (r *Route) Get(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(GET, path, handler, name...)
}

func (r *Route) Post(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(POST, path, handler, name...)
}

func (r *Route) Put(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(PUT, path, handler, name...)
}

func (r *Route) Delete(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(DELETE, path, handler, name...)
}

func (r *Route) Head(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(HEAD, path, handler, name...)
}

func (r *Route) Patch(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(PATCH, path, handler, name...)
}

func (r *Route) Options(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(OPTIONS, path, handler, name...)
}

func (r *Route) Any(path string, handler HandleFunc, name ...string) *Route {
	return r.Handle(ANY, path, handler, name...)
}

func randInt64() int64 {
	var n, _ = rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	return n.Int64()
}

func removeRoute(s []*Route, match *Route) []*Route {
	for i, r := range s {
		if r.identifier == match.identifier {
			if len(s) == 1 {
				return make([]*Route, 0)
			} else if i == len(s)-1 {
				return s[:i]
			} else {
				return append(s[:i], s[i+1:]...)
			}
		}
	}
	return s
}
