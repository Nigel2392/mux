package mux

import (
	"slices"
	"sort"
)

// Literal children are indexed by segment. Dynamic children share an edge
// when their matcher keys agree and are visited in descending priority order.
type branch struct {
	literals map[string]*branch
	dynamic  []edge
	routes   []*Route
	parent   *branch
	literal  string
}

type edge struct {
	key      string
	matcher  BoundMatcher
	flags    MatcherFlag
	priority int
	child    *branch
}

func (b *branch) add(matcher BoundMatcher) *branch {
	flags := matcher.Flags()
	if flags.Has(M_LITERAL) && flags&(M_CAPTURE|M_TERMINAL) == 0 {
		literal := matcher.Literal()
		child := b.literals[literal]
		if child == nil {
			if b.literals == nil {
				b.literals = make(map[string]*branch)
			}
			child = &branch{parent: b, literal: literal}
			b.literals[literal] = child
		}
		return child
	}

	key := matcher.Key()
	for _, existing := range b.dynamic {
		if existing.key == key {
			return existing.child
		}
	}
	next := edge{key: key, matcher: matcher, flags: flags, priority: matcher.Priority(), child: &branch{parent: b}}
	i := sort.Search(len(b.dynamic), func(i int) bool {
		return b.dynamic[i].priority < next.priority
	})
	b.dynamic = append(b.dynamic, edge{})
	copy(b.dynamic[i+1:], b.dynamic[i:])
	b.dynamic[i] = next
	return next.child
}

func (b *branch) removeRoute(route *Route) {
	i := slices.Index(b.routes, route)
	b.routes = slices.Delete(b.routes, i, i+1)
	for node := b; node.parent != nil && len(node.routes) == 0 && len(node.literals) == 0 && len(node.dynamic) == 0; node = node.parent {
		parent := node.parent
		if parent.literals[node.literal] == node {
			delete(parent.literals, node.literal)
		} else {
			i := slices.IndexFunc(parent.dynamic, func(edge edge) bool { return edge.child == node })
			parent.dynamic = slices.Delete(parent.dynamic, i, i+1)
		}
	}
}

// Match returns raw capture values or variables from a custom resolver.
// Separate results allow the caller's inline capture storage to stay on the stack.
func (b *branch) match(method string, path PathBuf, scratch *PathBuf, scope *Route, captures []string) (*Route, []string, Variables) {
	done := path.Done()
	if done {
		for _, route := range b.routes {
			if route.acceptsMethod(method) && route.within(scope) {
				if route.hasResolver() {
					if vars, ok := route.resolveCapture(captures, scope.captureOffset()); ok {
						return route, nil, vars
					}
					continue
				}
				return route, captures, nil
			}
		}
	} else if len(b.literals) != 0 {
		next := path
		if child := b.literals[next.Consume()]; child != nil {
			if route, values, vars := child.match(method, next, scratch, scope, captures); route != nil {
				return route, values, vars
			}
		}
	}

	for _, edge := range b.dynamic {
		if done && !edge.flags.Has(M_TERMINAL) {
			continue
		}
		if scratch == nil {
			scratch = new(PathBuf)
		}
		*scratch = path
		value, ok := edge.matcher.Match(scratch)
		if !ok {
			continue
		}
		values := captures
		if edge.flags.Has(M_CAPTURE) {
			values = append(values, value)
		}
		if route, matched, vars := edge.child.match(method, *scratch, scratch, scope, values); route != nil {
			return route, matched, vars
		}
	}
	return nil, nil, nil
}

func (r *Route) acceptsMethod(method string) bool {
	return r.Handler != nil && (r.Method == method || r.Method == ANY || method == ANY)
}

func (r *Route) within(scope *Route) bool {
	if scope == nil {
		return true
	}
	for current := r; current != nil; current = current.Parent {
		if current == scope {
			return true
		}
	}
	return false
}
