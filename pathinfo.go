package mux

import (
	"fmt"
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
func (p *PathInfo) Match(path []string) (matched bool, vars Variables) {

	var (
		variables = make(Variables)
		pathParts = make([]*PathPart, 0, len(p.Path))
	)
	for pt := p; pt != nil; pt = pt.Parent {
		pathParts = append(pt.Path, pathParts...)
	}

	if len(path) > len(pathParts) && !p.IsGlob {
		return false, nil
	}

	// Glob only allows for more parts, not less
	if len(path) < len(pathParts) && (len(path) != len(pathParts)-1 && !p.IsGlob) {
		return false, nil
	}

	for i, part := range pathParts {
		if part.IsGlob && i >= len(path) {
			variables[GLOB] = []string{}
			return true, variables
		}

		if i >= len(path) {
			return false, nil
		}

		var pathPart = path[i]
		switch {
		case part.IsVariable:
			if pathPart == "" {
				return false, nil
			}
			variables[part.Part] = append(variables[part.Part], pathPart)

		case part.Part != pathPart && part.IsGlob:
			if p.Resolver != nil {
				return p.Resolver.Match(variables, path[i:])
			}

			variables[GLOB] = append(variables[GLOB], path[i:]...)

		case part.Part != pathPart:
			return false, nil
		}
	}

	return true, variables
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
