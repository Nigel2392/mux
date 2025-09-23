package mux

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// SplitPath splits a path into its parts.
func SplitPath(path string) []string {
	// path = strings.ToLower(path)
	path = strings.Trim(path, URL_DELIM)
	if path == "" {
		return []string{}
	}
	return strings.Split(path, URL_DELIM)
}

// Global variables for use in the package.
//
// These are exported so that they can be changed if needed.
var (
	VARIABLE_DELIMS = []string{"<<", ">>"}
	URL_DELIM       = "/"
	GLOB            = "*"
)

type Resolver interface {
	Reverse(baseURL string, variables ...interface{}) (string, error)
	Match(vars Variables, path []string) (bool, Variables)
}

// PathInfo contains information about a path.
//
// It can be used to match a path, and retrieve variables from it.
type PathInfo struct {
	IsGlob   bool
	Parent   *PathInfo
	Path     []*PathPart
	Resolver Resolver
}

// WithParent returns a new PathInfo with the given parent.
func (p *PathInfo) WithParent(parent *PathInfo) *PathInfo {
	if parent == nil {
		return p
	}

	if parent.IsGlob {
		panic(fmt.Sprintf("parent path of %q cannot be a glob", p.String()))
	}

	return &PathInfo{
		IsGlob:   p.IsGlob,
		Parent:   parent,
		Path:     p.Path,
		Resolver: p.Resolver,
	}
}

// String returns a string representation of the path.
func (p *PathInfo) String() string {
	var b strings.Builder
	var totalLen int
	var delimLen = len(VARIABLE_DELIMS[0]) + len(VARIABLE_DELIMS[1])
	for _, part := range p.Path {
		totalLen += len(part.Part) + 1
		if part.IsVariable {
			totalLen += delimLen
		}
	}

	if p.Parent != nil {
		b.WriteString(p.Parent.String())
	}

	b.Grow(totalLen)
	b.WriteString(URL_DELIM)
	for i, part := range p.Path {
		if part.IsVariable {
			b.WriteString(VARIABLE_DELIMS[0])
		}
		b.WriteString(part.Part)
		if part.IsVariable {
			b.WriteString(VARIABLE_DELIMS[1])
		}
		if i < len(p.Path)-1 {
			b.WriteString(URL_DELIM)
		}
	}

	return b.String()
}

// PathPart contains information about a part of a path.
type PathPart struct {
	Part       string
	IsVariable bool
	IsGlob     bool
	// Validators []func(string) bool
}

// Match matches a path to this path.
//
// It returns whether the path matched, and the variables in the path.
//
// If the path does not match, the variables will be nil.
func (p *PathInfo) Match(path []string, from int, variables Variables) (matched bool, nextFrom int) {
	var i = from

	// Match each part defined on this PathInfo only (no parent traversal here).
	for idx, part := range p.Path {
		// If this part is a terminal glob, it eats the rest (possibly zero)
		if part.IsGlob {
			if p.Resolver != nil {
				// If you have a resolver, delegate the remainder
				ok, v := p.Resolver.Match(variables, path[i:])
				if !ok {
					return false, -1
				}
				maps.Copy(variables, v)
				return ok, len(path)
			} else {
				// Capture the remainder as GLOB
				variables[GLOB] = append(variables[GLOB], path[i:]...)
			}
			// Glob ends the pattern
			i = len(path)

			// If glob is not the last declared part (shouldn't happen if enforced elsewhere),
			// treat as full since it's terminal by contract.
			if idx != len(p.Path)-1 {
				// defensive: refuse unexpected extra parts after a glob
				return false, -1
			}
			break
		}

		// Need a segment to match this literal/variable
		if i >= len(path) {
			// Not enough segments -> not even a partial for children,
			// because the parent hasn't matched its own pattern fully.
			return false, -1
		}

		seg := path[i]
		switch {
		case part.IsVariable:
			if seg == "" {
				return false, -1
			}
			variables[part.Part] = append(variables[part.Part], seg)
		default:
			// literal
			if part.Part != seg {
				return false, -1
			}
		}
		i++
	}

	// At this point, this PathInfo has fully matched its own pattern.
	// If we consumed the whole path (or a glob consumed the rest), it's a full match.
	if i == len(path) {
		return true, i
	}

	// Otherwise, it's a partial match: children should continue from i.
	return false, i
}

// Reverse returns the path with the variables replaced.
//
// If a variable is not found, this function will error.
func (p *PathInfo) Reverse(variables ...interface{}) (string, error) {
	var (
		b          strings.Builder
		varIndex   int
		seenParts  int
		totalParts int
	)

	// create path slice
	var path = make([]*PathInfo, 0)
	for pt := p; pt != nil; pt = pt.Parent {
		path = append(path, pt)
		totalParts += len(pt.Path)
	}

	// reverse the path slice
	slices.Reverse(path)

	b.WriteString(URL_DELIM)

	for pathIndex, pathObject := range path {

		for _, part := range pathObject.Path {
			seenParts++

			if part.IsGlob && len(variables) >= varIndex && pathIndex == len(path)-1 {
				// The resolver can take over when it is a GLOB
				if pathObject.Resolver != nil && pathIndex == len(path)-1 {
					return pathObject.Resolver.Reverse(b.String(), variables[varIndex:]...)
				}

				for _, v := range variables[varIndex:] {
					varIndex++

					b.WriteString(fmt.Sprint(v))

					if varIndex < len(variables) {
						b.WriteString(URL_DELIM)
					}
				}

				break
			}

			if part.IsVariable && varIndex >= len(variables) {
				return "", ErrNotEnoughVariables
			}

			if part.IsVariable {
				b.WriteString(fmt.Sprint(variables[varIndex]))
				varIndex++
			} else {
				b.WriteString(part.Part)
			}

			b.WriteString(URL_DELIM)
		}
	}

	if !p.IsGlob && len(variables) > varIndex {
		return "", ErrTooManyVariables
	}

	return b.String(), nil
}

// NewPathInfo creates a new PathInfo object from a path string.
//
// The path string can contain variables,
// which are defined by the text between the VARIABLE_DELIMS.
//
// This function will panic if the GLOB is not the last part of the path.
func NewPathInfo(rt *Route, path string) *PathInfo {
	var parts = SplitPath(path)
	var info = &PathInfo{
		Path: make([]*PathPart, 0, len(parts)),
	}

	for i, part := range parts {
		var pathPart = &PathPart{Part: part}

		// Check if this part is a variable
		if strings.HasPrefix(part, VARIABLE_DELIMS[0]) &&
			strings.HasSuffix(part, VARIABLE_DELIMS[1]) {

			pathPart.IsVariable = true
			pathPart.Part = part[len(VARIABLE_DELIMS[0]) : len(part)-len(VARIABLE_DELIMS[1])]
		} else if part == GLOB && i == len(parts)-1 {
			info.IsGlob = true
			pathPart.IsGlob = true
		} else if part == GLOB {
			panic("glob must be the last part of the path, using glob specifies an unknown path length")
		}
		info.Path = append(info.Path, pathPart)
	}

	if rt != nil && info.IsGlob {
		if resolver, ok := rt.Handler.(Resolver); ok {
			info.Resolver = resolver
		}
	}

	return info
}
