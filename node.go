package mux

import (
	"errors"
	"net/http"
	"sync/atomic"

	"github.com/Nigel2392/mux/v2/middleware"
)

const (
	ANY     = "*"
	GET     = http.MethodGet
	POST    = http.MethodPost
	PUT     = http.MethodPut
	DELETE  = http.MethodDelete
	HEAD    = http.MethodHead
	PATCH   = http.MethodPatch
	OPTIONS = http.MethodOptions

	NAME_SEPARATOR = ":"
)

// Mux matches HTTP methods and paths against a tree of registered routes.
// Its zero value is ready to use. Configure and edit routes outside concurrent
// lookups.
type Mux struct {
	NotFoundHandler http.HandlerFunc

	root       branch
	routes     []*Route
	matchers   *MatcherRegistry
	middleware []middleware.Middleware
}

func New() *Mux {
	return &Mux{}
}

// Matchers returns the registry used to parse newly added routes.
func (m *Mux) Matchers() *MatcherRegistry {
	if m.matchers == nil {
		m.matchers = NewMatcherRegistry()
	}
	return m.matchers
}

// Route stores a handler and can serve as the parent of relative routes.
// Use SetPath, SetPathInfo, AddRoute and RemoveChild to edit an attached
// route's path and children so the mux can update its index.
type Route struct {
	Name               string
	Method             string
	Middleware         []middleware.Middleware
	PreMiddleware      []middleware.Middleware
	Path               *PathInfo
	Children           []*Route
	Handler            http.Handler
	Parent             *Route
	ParentMux          *Mux
	DisabledMiddleware bool

	identifier int64
	branch     *branch
	segments   []BoundMatcher
	names      []string
}

var routeIDs atomic.Int64

func NewRoute(method, path string, handler http.Handler, name ...string) *Route {
	route := &Route{Method: method, Handler: handler, identifier: routeIDs.Add(1)}
	route.Path = newRoutePath(route, path)
	if len(name) != 0 {
		route.Name = name[0]
	}
	return route
}

func (r *Route) ID() int64 { return r.identifier }

func (m *Mux) Handle(method, path string, handler http.Handler, name ...string) *Route {
	route := NewRoute(method, path, handler, name...)
	m.AddRoute(route)
	return route
}

func (r *Route) Handle(method, path string, handler http.Handler, name ...string) *Route {
	route := NewRoute(method, path, handler, name...)
	r.AddRoute(route)
	return route
}

// AddRoute attaches a route and its children. It panics for an invalid pattern
// or a route that already belongs to a parent or mux.
func (m *Mux) AddRoute(route *Route) {
	m.addRoute(nil, route)
}

// AddRoute adds a child whose path is relative to this route.
func (r *Route) AddRoute(route *Route) {
	for parent := r; parent != nil; parent = parent.Parent {
		if parent == route {
			panic(ErrCyclicParent)
		}
	}
	if r.ParentMux != nil {
		r.ParentMux.addRoute(r, route)
		return
	}
	checkUnattached(route)
	route.Parent = r
	if route.Path != nil {
		route.Path.Parent = r.Path
	}
	r.Children = append(r.Children, route)
}

func checkUnattached(route *Route) {
	if route.ParentMux != nil || route.Parent != nil {
		panic(ErrRouteAttached)
	}
}

func (m *Mux) addRoute(parent, route *Route) {
	checkUnattached(route)
	if parent != nil && isTerminal(parent.segments) {
		panic(ErrParentIsTerminal)
	}

	registry := m.Matchers()
	parsed, err := registry.parseRoute(route, parent, nil, nil)
	if err != nil {
		panic(err)
	}
	if parent != nil {
		parent.Children = append(parent.Children, route)
	} else {
		m.routes = append(m.routes, route)
	}
	for _, item := range parsed {
		item.route.Parent = item.parent
		item.route.Path = item.path
		if item.route.identifier == 0 {
			item.route.identifier = routeIDs.Add(1)
		}
		m.indexRoute(item.route, item.segments)
	}
}

type parsedRoute struct {
	route    *Route
	parent   *Route
	path     *PathInfo
	segments []BoundMatcher
}

func (registry *MatcherRegistry) parseRoute(route, parent *Route, parsed []parsedRoute, seen map[*Route]bool) ([]parsedRoute, error) {
	if route == parent {
		return nil, ErrCyclicParent
	}
	if active, exists := seen[route]; exists {
		if active {
			return nil, ErrCyclicParent
		}
		return nil, ErrRouteAttached
	}
	if route.ParentMux != nil || (route.Parent != nil && route.Parent != parent) {
		return nil, ErrRouteAttached
	}
	path := route.Path
	if path == nil {
		path = newRoutePath(route, "")
	}
	segments, err := path.compile(registry)
	if err != nil {
		return nil, err
	}
	if len(route.Children) != 0 && isTerminal(segments) {
		return nil, ErrHasChildren
	}
	parsed = append(parsed, parsedRoute{route: route, parent: parent, path: path, segments: segments})
	if len(route.Children) != 0 && seen == nil {
		seen = make(map[*Route]bool)
	}
	if seen != nil {
		seen[route] = true
	}
	for _, child := range route.Children {
		parsed, err = registry.parseRoute(child, route, parsed, seen)
		if err != nil {
			return nil, err
		}
	}
	if seen != nil {
		seen[route] = false
	}
	return parsed, nil
}

func isTerminal(segments []BoundMatcher) bool {
	return len(segments) != 0 && segments[len(segments)-1].Flags().Has(M_TERMINAL)
}

func (m *Mux) indexRoute(route *Route, segments []BoundMatcher) {
	node := &m.root
	var inherited []string
	var parentPath *PathInfo
	if route.Parent != nil {
		node = route.Parent.branch
		inherited = route.Parent.names
		parentPath = route.Parent.Path
	}
	// A child's captures must never overwrite its parent's backing array.
	names := inherited[:len(inherited):len(inherited)]
	for _, segment := range segments {
		node = node.add(segment)
		if segment.Flags().Has(M_CAPTURE) {
			names = append(names, segment.Name())
		}
	}
	if old := route.branch; old != node {
		node.routes = append(node.routes, route)
		if old != nil {
			old.removeRoute(route)
		}
	}
	route.Path.refresh(parentPath, segments, m.matchers)
	route.Path.inferResolver(route.Handler)
	route.ParentMux, route.branch, route.segments, route.names = m, node, segments, names
}

// Match finds the best route for a method and path. A miss returns two nils.
func (m *Mux) Match(method, path string) (*Route, Variables) {
	var captures [8]string
	route, values, vars := m.root.match(method, m.matchers.pathBuf(path), nil, nil, captures[:0])
	if len(values) != 0 {
		vars = makeVariables(route, values, 0)
	}
	return route, vars
}

// Match starts with this route's own pattern and searches its children. The
// path is relative to this route's parent.
func (r *Route) Match(method, path string) (*Route, bool, Variables) {
	if r.ParentMux == nil {
		return r.matchDetached(method, path)
	}
	var captures [8]string
	values := captures[:0]
	buf := r.ParentMux.matchers.pathBuf(path)
	for _, segment := range r.segments {
		flags := segment.Flags()
		if buf.Done() && !flags.Has(M_TERMINAL) {
			return nil, false, nil
		}
		value, ok := segment.Match(&buf)
		if !ok {
			return nil, false, nil
		}
		if flags.Has(M_CAPTURE) {
			values = append(values, value)
		}
	}
	route, values, vars := r.branch.match(method, buf, &buf, r, values)
	if len(values) != 0 {
		vars = makeVariables(route, values, r.captureOffset())
	}
	return route, route != nil, vars
}

func (r *Route) captureOffset() int {
	if r != nil && r.Parent != nil {
		return len(r.Parent.names)
	}
	return 0
}

// SetPath changes the route's relative URL pattern, including the prefix used
// by its descendants. If validation fails, the current tree remains usable.
func (r *Route) SetPath(path string) error {
	next := newRoutePath(r, path)
	if r.Path != nil && r.Path.Resolver != nil && next.IsGlob {
		next.Resolver = r.Path.Resolver
	}
	return r.SetPathInfo(next)
}

// SetPathInfo replaces a route's pattern and resolver and refreshes its index.
func (r *Route) SetPathInfo(path *PathInfo) error {
	if path == nil {
		path = newRoutePath(r, "")
	}
	m := r.ParentMux
	registry := NewMatcherRegistry()
	if m != nil {
		registry = m.matchers
	}
	segments, err := path.compile(registry)
	if err != nil && (m != nil || !errors.Is(err, ErrMatcherNotFound)) {
		return err
	}
	if len(r.Children) != 0 && (path.IsGlob || isTerminal(segments)) {
		return ErrHasChildren
	}
	r.Path = path
	if m != nil {
		m.indexRoute(r, segments)
		for _, child := range r.Children {
			m.reindex(child)
		}
	} else {
		r.linkPaths()
	}
	return nil
}

func (m *Mux) reindex(r *Route) {
	m.indexRoute(r, r.segments)
	for _, child := range r.Children {
		m.reindex(child)
	}
}
