package mux

import (
	"fmt"
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
	Path     []*PathPart
	Resolver Resolver
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

// Copies the slice, and append appends the other path to the end of this path.
//
// It will panic if the path on which this was called is a glob.
func (p *PathInfo) CopyAppend(other *PathInfo) *PathInfo {
	if p.IsGlob {
		panic("cannot append to a glob path")
	}
	var path = &PathInfo{
		IsGlob: other.IsGlob,
		Path:   make([]*PathPart, len(p.Path)+len(other.Path)),
	}
	copy(path.Path, p.Path)
	copy(path.Path[len(p.Path):], other.Path)
	//	var i = 0
	//	for {
	//		if i >= len(p.Path)+len(other.Path) {
	//			break
	//		}
	//		var part PathPart
	//		if i < len(p.Path) {
	//			part = *p.Path[i]
	//		} else {
	//			part = *other.Path[i-len(p.Path)]
	//		}
	//		path.Path[i] = &part
	//	}
	return path
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
func (p *PathInfo) Match(path []string) (bool, Variables) {
	// Glob only allows for more parts, not less
	if len(path) < len(p.Path) || (len(path) != len(p.Path) && !p.IsGlob) {
		return false, nil
	}
	var variables = make(Variables)
	for i, part := range p.Path {
		if part.IsVariable {
			var pathPart = path[i]
			if pathPart == "" {
				return false, nil
			}
			variables[part.Part] = append(variables[part.Part], pathPart)
		} else if part.Part != path[i] && part.IsGlob {

			if p.Resolver != nil {
				return p.Resolver.Match(variables, path[i:])
			}

			variables[GLOB] = append(variables[GLOB], path[i:]...)
		} else if part.Part != path[i] {
			return false, nil
		}
	}
	return true, variables
}

// Reverse returns the path with the variables replaced.
//
// If a variable is not found, this function will error.
func (p *PathInfo) Reverse(variables ...interface{}) (string, error) {
	var b strings.Builder
	var varIndex = 0
	b.WriteString(URL_DELIM)
	for i, part := range p.Path {
		if part.IsGlob && len(variables) >= varIndex {

			// The resolver can take over when it is a GLOB
			if p.Resolver != nil {
				return p.Resolver.Reverse(b.String(), variables[varIndex:]...)
			}

			for j, v := range variables[varIndex:] {
				b.WriteString(fmt.Sprint(v))

				if j+i < len(p.Path)-1 {
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

		if i < len(p.Path)-1 {
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
