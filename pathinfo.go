package mux

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
)

// VARIABLE_DELIMS, URL_DELIM and GLOB configure newly created routes and
// matcher registries. Set them before creating or registering routes.
var (
	VARIABLE_DELIMS = []string{"<<", ">>"}
	URL_DELIM       = "/"
	GLOB            = "*"
)

type pathSyntax struct {
	open, close, separator, glob string
}

func defaultPathSyntax() pathSyntax {
	syntax := pathSyntax{open: "<<", close: ">>", separator: URL_DELIM, glob: GLOB}
	if len(VARIABLE_DELIMS) >= 2 && VARIABLE_DELIMS[0] != "" && VARIABLE_DELIMS[1] != "" {
		syntax.open, syntax.close = VARIABLE_DELIMS[0], VARIABLE_DELIMS[1]
	}
	if syntax.separator == "" {
		syntax.separator = "/"
	}
	return syntax
}

func (s pathSyntax) variable(segment string) (string, bool) {
	if len(segment) >= len(s.open)+len(s.close) && strings.HasPrefix(segment, s.open) && strings.HasSuffix(segment, s.close) {
		return segment[len(s.open) : len(segment)-len(s.close)], true
	}
	return "", false
}

// SplitPath splits a path into segments while retaining empty interior segments.
func SplitPath(path string) []string {
	separator := defaultPathSyntax().separator
	path = strings.TrimPrefix(path, separator)
	path = strings.TrimSuffix(path, separator)
	if path == "" {
		return []string{}
	}
	return strings.Split(path, separator)
}

// Resolver can handle matching and reversal for the remainder of a glob route.
type Resolver interface {
	Reverse(baseURL string, variables ...any) (string, error)
	Match(vars Variables, path []string) (bool, Variables)
}

// PathInfo describes the local route pattern and its parent.
type PathInfo struct {
	IsGlob   bool
	Parent   *PathInfo
	Path     []*PathPart
	Resolver Resolver

	syntax   pathSyntax
	registry *MatcherRegistry
	segments []BoundMatcher
}

type PathPart struct {
	Part       string
	IsVariable bool
	IsGlob     bool
}

// NewPathInfo parses a route pattern using the default matcher registry. It
// panics when the pattern is invalid.
func NewPathInfo(route *Route, path string) *PathInfo {
	info := newRoutePath(route, path)
	var registry *MatcherRegistry
	if route != nil && route.ParentMux != nil && route.ParentMux.matchers != nil {
		registry = route.ParentMux.matchers
	}
	if registry == nil {
		registry = newMatcherRegistry(info.syntax)
	}
	segments, err := info.compile(registry)
	if err != nil {
		panic(err)
	}
	info.refresh(nil, segments, registry)
	if route != nil {
		info.inferResolver(route.Handler)
	}
	return info
}

func newRoutePath(route *Route, path string) *PathInfo {
	syntax := defaultPathSyntax()
	if route != nil && route.ParentMux != nil && route.ParentMux.matchers != nil {
		syntax = route.ParentMux.matchers.syntax
	}
	info := &PathInfo{syntax: syntax}
	buf := newPathBuf(path, syntax.separator)
	for !buf.Done() {
		segment := buf.Consume()
		part := &PathPart{Part: segment}
		if segment == syntax.glob && segment != "" {
			part.IsGlob = true
		} else if inner, ok := syntax.variable(segment); ok {
			part.Part, part.IsVariable = inner, true
			part.IsGlob = strings.HasPrefix(inner, "glob:")
		}
		info.IsGlob = part.IsGlob
		info.Path = append(info.Path, part)
	}
	if route != nil {
		info.inferResolver(route.Handler)
	}
	return info
}

func (p *PathInfo) compile(registry *MatcherRegistry) ([]BoundMatcher, error) {
	if registry == nil {
		registry = newMatcherRegistry(p.pathSyntax())
	}
	segments := make([]BoundMatcher, 0, len(p.Path))
	for i, part := range p.Path {
		var matcher BoundMatcher
		var err error
		switch {
		case part == nil || part.Part == "" && !part.IsVariable:
			return nil, fmt.Errorf("%w: empty path segment", ErrInvalidMatcherArg)
		case part.IsVariable:
			matcher, err = registry.bindVariable(part.Part)
		case part.IsGlob:
			matcher, err = registry.Bind("glob", part.Part)
		default:
			matcher = StaticMatcher{text: part.Part}
		}
		if err != nil {
			return nil, err
		}
		if matcher.Flags().Has(M_TERMINAL) && i != len(p.Path)-1 {
			return nil, ErrTerminalNotLast
		}
		segments = append(segments, matcher)
	}
	return segments, nil
}

func (p *PathInfo) refresh(parent *PathInfo, segments []BoundMatcher, registry *MatcherRegistry) {
	p.Parent, p.segments, p.registry = parent, segments, registry
	p.IsGlob = false
	if len(segments) != 0 {
		last := len(segments) - 1
		p.IsGlob = isGlobSegment(p.Path[last], segments[last])
	}
	if registry != nil {
		p.syntax = registry.syntax
	}
}

func isGlobSegment(part *PathPart, matcher BoundMatcher) bool {
	return matcher.Flags().Has(M_TERMINAL) &&
		(part.IsGlob || matcher.Key() == "glob" || part.IsVariable && strings.HasPrefix(part.Part, "glob:"))
}

func (p *PathInfo) inferResolver(handler http.Handler) {
	if p.IsGlob && p.Resolver == nil {
		p.Resolver, _ = handler.(Resolver)
	}
}

// WithParent returns a copy that uses parent when rendering or reversing.
func (p *PathInfo) WithParent(parent *PathInfo) *PathInfo {
	if parent == nil {
		return p
	}
	if parent.IsGlob || isTerminal(parent.segments) {
		panic(ErrParentIsTerminal)
	}
	copy := *p
	copy.Parent = parent
	return &copy
}

func (p *PathInfo) pathSyntax() pathSyntax {
	if p.syntax.separator != "" {
		return p.syntax
	}
	return defaultPathSyntax()
}

func (p *PathInfo) pattern() string {
	var result strings.Builder
	syntax := p.pathSyntax()
	for i, part := range p.Path {
		if i > 0 {
			result.WriteString(syntax.separator)
		}
		if part.IsVariable {
			result.WriteString(syntax.open)
			result.WriteString(part.Part)
			result.WriteString(syntax.close)
		} else {
			result.WriteString(part.Part)
		}
	}
	return result.String()
}

func (p *PathInfo) ancestors() []*PathInfo {
	var paths []*PathInfo
	for path := p; path != nil; path = path.Parent {
		paths = append(paths, path)
	}
	slices.Reverse(paths)
	return paths
}

func (p *PathInfo) String() string {
	separator := p.pathSyntax().separator
	var result strings.Builder
	result.WriteString(separator)
	for _, path := range p.ancestors() {
		if len(path.Path) != 0 {
			if result.Len() > len(separator) {
				result.WriteString(separator)
			}
			result.WriteString(path.pattern())
		}
	}
	return result.String()
}

func (p *PathInfo) boundMatchers() ([]BoundMatcher, error) {
	if p.segments != nil {
		return p.segments, nil
	}
	return p.compile(p.registry)
}

// Match matches the local pattern against the supplied segments starting at
// from. A partial match returns the index where the next route should start.
func (p *PathInfo) Match(path []string, from int, variables Variables) (bool, int, Variables) {
	if from < 0 || from > len(path) {
		return false, -1, nil
	}
	segments, err := p.boundMatchers()
	if err != nil {
		return false, -1, nil
	}
	separator := p.pathSyntax().separator
	raw := strings.Join(path[from:], separator)
	buf := newPathBuf("", separator)
	buf.Path = raw
	index := from
	for i, matcher := range segments {
		flags := matcher.Flags()
		glob := isGlobSegment(p.Path[i], matcher)
		if index == len(path) && !flags.Has(M_TERMINAL) {
			return false, -1, nil
		}
		if glob && p.Resolver != nil {
			ok, resolved := matchResolver(p.Resolver, variables, path[index:])
			if !ok {
				return false, -1, nil
			}
			return true, len(path), resolved
		}
		start := buf.Pos
		value, ok := matcher.Match(&buf)
		if !ok {
			return false, -1, nil
		}
		if flags.Has(M_CAPTURE) {
			if variables == nil {
				variables = make(Variables)
			}
			if glob && !p.Path[i].IsVariable {
				variables[matcher.Name()] = path[index:]
			} else {
				variables[matcher.Name()] = append(variables[matcher.Name()], value)
			}
		}
		if glob && buf.Done() {
			index = len(path)
		} else if buf.Pos > start {
			consumed := raw[start:buf.Pos]
			index += strings.Count(consumed, separator)
			if !strings.HasSuffix(consumed, separator) {
				index++
			}
		}
	}
	return index == len(path), index, variables
}

// Reverse renders the full path using positional variables. A glob consumes
// the remaining arguments or hands them to its resolver.
func (p *PathInfo) Reverse(variables ...any) (string, error) {
	separator := p.pathSyntax().separator
	var result strings.Builder
	result.WriteString(separator)
	for _, path := range p.ancestors() {
		segments, err := path.boundMatchers()
		if err != nil {
			return "", err
		}
		for i, matcher := range segments {
			if path.Resolver != nil && isGlobSegment(path.Path[i], matcher) {
				return path.Resolver.Reverse(result.String(), variables...)
			}
			consumed, err := matcher.Format(&result, variables)
			if err != nil {
				return "", err
			}
			if consumed < 0 || consumed > len(variables) {
				return "", ErrInvalidMatcherArg
			}
			variables = variables[consumed:]
			if !matcher.Flags().Has(M_TERMINAL) {
				result.WriteString(separator)
			}
		}
	}
	if len(variables) != 0 {
		return "", ErrTooManyVariables
	}
	return result.String(), nil
}

func matchResolver(resolver Resolver, variables Variables, remainder []string) (bool, Variables) {
	inherited := variables
	if variables == nil {
		variables = make(Variables)
	}
	ok, resolved := resolver.Match(variables, remainder)
	if !ok {
		return false, nil
	}
	if inherited == nil {
		return true, resolved
	}
	maps.Copy(inherited, resolved)
	return true, inherited
}

func (p *PathInfo) splitRemainder(remainder string) []string {
	if remainder == "" {
		return nil
	}
	return strings.Split(remainder, p.pathSyntax().separator)
}

func (r *Route) pathResolver() Resolver {
	if r.Path == nil || !r.Path.IsGlob {
		return nil
	}
	return r.Path.Resolver
}

func (r *Route) hasResolver() bool {
	if r.pathResolver() == nil || len(r.Path.segments) == 0 {
		return false
	}
	flags := r.Path.segments[len(r.Path.segments)-1].Flags()
	return flags.Has(M_TERMINAL) && flags.Has(M_CAPTURE)
}

func (r *Route) resolveCapture(values []string, offset int) (Variables, bool) {
	last := len(values) - 1
	resolver := r.pathResolver()
	variables := makeVariables(r, values[:last], offset)
	ok, variables := matchResolver(resolver, variables, r.Path.splitRemainder(values[last]))
	return variables, ok
}
