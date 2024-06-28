package mux

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"strings"
)

type Route struct {
	Name       string
	Method     string
	Middleware []Middleware
	Path       *PathInfo
	Children   []*Route
	Handler    Handler
	Parent     *Route
	ParentMux  *Mux

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
		Middleware: make([]Middleware, 0),
		identifier: randInt64(),
	}
	return route
}

func NewRoute(method, path string, handler Handler, name ...string) *Route {
	var rt = newRoute(method, handler, name...)
	rt.Path = NewPathInfo(path)
	return rt
}

func (r *Route) ID() int64 {
	return r.identifier
}

// Use adds middleware to the route.
func (r *Route) Use(middleware ...Middleware) {
	r.Middleware = append(r.Middleware, middleware...)
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

var ErrMethodNotAllowed = errors.New("method not allowed")

// Helper function to check if the route matches the method and path.
func routeMatched(matched bool, method string, route *Route) (bool, error) {
	if !matched || route.Handler == nil {
		return false, nil
	}

	if route.Method != ANY && route.Method != method {
		return false, ErrMethodNotAllowed
	}

	return matched && (route.Method == ANY || route.Method == method) && route.Handler != nil, nil
}

func (r *Route) Match(method string, path []string) (*Route, bool, Variables, error) {
	var matched, variables = r.Path.Match(path)
	if ok, err := routeMatched(matched, method, r); ok || err != nil {
		return r, matched, variables, err
	}

	for _, child := range r.Children {
		var rt, matched, variables, err = child.Match(method, path)
		if err != nil {
			return nil, false, nil, err
		}
		if ok, err := routeMatched(matched, method, rt); ok || err != nil {
			return rt, matched, variables, err
		}
	}
	return nil, false, nil, nil
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
