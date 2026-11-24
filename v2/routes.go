package mux

import (
	"slices"
	"strings"
)

func (r *Route) String() string {
	if r.Path == nil {
		return "/"
	}
	return r.Path.String()
}

// PathName returns the names of the route and its ancestors, omitting empty names.
func (r *Route) PathName() []string {
	var names []string
	for current := r; current != nil; current = current.Parent {
		if current.Name != "" {
			names = append(names, current.Name)
		}
	}
	slices.Reverse(names)
	return names
}

// Find follows a chain of names through this subtree. Unnamed routes are
// transparent to the lookup.
func (r *Route) Find(names []string) (*Route, bool) {
	for len(names) != 0 && names[0] == "" {
		names = names[1:]
	}
	if len(names) == 0 {
		return nil, false
	}
	if r.Name != "" {
		if r.Name != names[0] {
			return nil, false
		}
		names = names[1:]
		for len(names) != 0 && names[0] == "" {
			names = names[1:]
		}
		if len(names) == 0 {
			return r, true
		}
	}
	for _, child := range r.Children {
		if found, ok := child.Find(names); ok {
			return found, true
		}
	}
	return nil, false
}

func (m *Mux) Find(name string) *Route {
	names := strings.Split(name, NAME_SEPARATOR)
	for _, route := range m.routes {
		if found, ok := route.Find(names); ok {
			return found
		}
	}
	return nil
}

func (m *Mux) Reverse(name string, variables ...any) (string, error) {
	route := m.Find(name)
	if route == nil {
		return "", ErrRouteNotFound
	}
	return route.Path.Reverse(variables...)
}

// RemoveRoute detaches a top-level route and its subtree.
func (m *Mux) RemoveRoute(route *Route) {
	if route == nil || route.Parent != nil || route.ParentMux != m {
		return
	}
	if i := slices.Index(m.routes, route); i >= 0 {
		m.routes = slices.Delete(m.routes, i, i+1)
		route.detach()
	}
}

// RemoveChild detaches a direct child and keeps its descendants with it.
func (r *Route) RemoveChild(child *Route) {
	if i := slices.Index(r.Children, child); i >= 0 {
		r.Children = slices.Delete(r.Children, i, i+1)
		child.detach()
	}
}

func (r *Route) detach() {
	r.unindex()
	r.Parent = nil
	r.linkPaths()
}

func (r *Route) unindex() {
	for _, child := range r.Children {
		child.unindex()
	}
	if r.branch != nil {
		r.branch.removeRoute(r)
	}
	r.ParentMux, r.branch, r.segments, r.names = nil, nil, nil, nil
}

func (r *Route) linkPaths() {
	if r.Path == nil {
		r.Path = newRoutePath(r, "")
	}
	r.Path.Parent = nil
	if r.Parent != nil {
		r.Path.Parent = r.Parent.Path
	}
	for _, child := range r.Children {
		child.linkPaths()
	}
}

func (m *Mux) ResetRoutes() {
	for _, route := range m.routes {
		route.detach()
	}
	m.routes = nil
	m.root = branch{}
}

// RemoveByPath removes routes registered with this full pattern.
func (m *Mux) RemoveByPath(path string) {
	path = m.matchers.pathBuf(path).Path
	for i := len(m.routes) - 1; i >= 0; i-- {
		root := m.routes[i]
		if root.hasPath(path) {
			m.RemoveRoute(root)
		} else {
			root.removeChildrenByPath(path)
		}
	}
}

func (r *Route) removeChildrenByPath(path string) {
	for i := len(r.Children) - 1; i >= 0; i-- {
		child := r.Children[i]
		if child.hasPath(path) {
			r.RemoveChild(child)
		} else {
			child.removeChildrenByPath(path)
		}
	}
}

// RemoveByPath removes the first matching route in this subtree.
func (r *Route) RemoveByPath(path string) bool {
	return r.removeByPath(newPathBuf(path, r.pathSeparator()).Path)
}

func (r *Route) removeByPath(path string) bool {
	if r.hasPath(path) {
		if r.Parent != nil {
			r.Parent.RemoveChild(r)
			return true
		}
		if r.ParentMux != nil {
			r.ParentMux.RemoveRoute(r)
			return true
		}
	}
	for _, child := range r.Children {
		if child.removeByPath(path) {
			return true
		}
	}
	return false
}

func (r *Route) hasPath(path string) bool {
	return newPathBuf(r.String(), r.pathSeparator()).Path == path
}

func (r *Route) pathSeparator() string {
	if r.Path != nil {
		return r.Path.pathSyntax().separator
	}
	return defaultPathSyntax().separator
}

// A detached subtree has no shared index. Build an isolated one so matching
// cannot attach or modify the caller's routes.
func (r *Route) matchDetached(method, path string) (*Route, bool, Variables) {
	mux := New()
	registry := mux.Matchers()
	parsed, err := registry.parseRoute(r, r.Parent, nil, nil)
	if err != nil {
		return nil, false, nil
	}
	clones := make(map[*Route]*Route, len(parsed))
	for _, item := range parsed {
		clone := *item.route
		path := *item.path
		clone.Path = &path
		clone.Parent = clones[item.parent]
		clone.Children = nil
		mux.indexRoute(&clone, item.segments)
		clones[item.route] = &clone
	}
	match, vars := mux.Match(method, path)
	if match != nil {
		for route, clone := range clones {
			if clone == match {
				return route, true, vars
			}
		}
	}
	return nil, false, nil
}
