package mux

import (
	"crypto/rand"
	"math"
	"math/big"
	"strings"
)

type Route struct {
	Name               string
	Method             string
	Middleware         []Middleware
	PreMiddleware      []Middleware
	Path               *PathInfo
	Children           []*Route
	Handler            Handler
	Parent             *Route
	ParentMux          *Mux
	DisabledMiddleware bool // Is middleware disabled for this route?

	identifier int64
}

func newRoute(method string, handler Handler, name ...string) *Route {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	var route = &Route{
		Handler:    handler,
		Name:       n,
		Method:     method,
		identifier: randInt64(),
	}
	return route
}

func NewRoute(method, path string, handler Handler, name ...string) *Route {
	var rt = newRoute(method, handler, name...)
	rt.Path = NewPathInfo(rt, path)
	return rt
}

func (r *Route) ID() int64 {
	return r.identifier
}

// Use adds middleware to the route.
func (r *Route) Use(middleware ...Middleware) {
	r.Middleware = append(r.Middleware, middleware...)
}

// Preprocess adds middleware to the route that will be executed before any other middleware or handler.
func (r *Route) Preprocess(middleware ...Middleware) {
	r.PreMiddleware = append(r.PreMiddleware, middleware...)
}

// DisableMiddleware disables the middleware for the route.
func (r *Route) RunsMiddleware(b bool) {
	r.DisabledMiddleware = !b
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
	// If the name matches and we are at the end of the names slice, return the route.
	if r.Name == names[index] && len(names)-1 == index {
		return r, true
	}
	// If the name matches, but we are not at the end of the names slice, continue searching.
	if r.Name != names[index] {
		return nil, false
	}
	// Check next parts for each child.
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

// Helper function to check if the route matches the method and path.
func routeMatched(matched bool, method string, route *Route) bool {
	return matched && (route.Method == ANY || route.Method == method || method == ANY) && route.Handler != nil
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
