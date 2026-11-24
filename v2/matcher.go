package mux

import (
	"fmt"
	"io"
	"strings"
)

type MatcherFlag uint8

const M_NONE MatcherFlag = 0

const (
	M_LITERAL MatcherFlag = 1 << iota
	M_TERMINAL
	M_CAPTURE
)

func (f MatcherFlag) Has(flag MatcherFlag) bool {
	return f&flag != 0
}

type Matcher interface {
	Key() string
	Bind(arg string) (BoundMatcher, error)
}

type BoundMatcher interface {
	// Key identifies equivalent matching behavior, including flags and priority.
	// Equivalent matchers may bind different capture names.
	Key() string
	Flags() MatcherFlag
	Priority() int
	String() string
	Name() string
	Literal() string
	Format(b io.StringWriter, args []any) (consumed int, err error)
	Match(path *PathBuf) (val string, ok bool)
}

type MatcherRegistry struct {
	matchers   map[string]Matcher
	syntax     pathSyntax
	defaultKey string
}

func NewMatcherRegistry() *MatcherRegistry {
	return newMatcherRegistry(defaultPathSyntax())
}

func newMatcherRegistry(syntax pathSyntax) *MatcherRegistry {
	r := &MatcherRegistry{
		matchers:   make(map[string]Matcher),
		syntax:     syntax,
		defaultKey: "var",
	}
	r.Register(VarMatcher{})
	r.Register(IntMatcher{})
	r.Register(UUIDMatcher{})
	r.Register(RegexMatcher{})
	r.Register(GlobMatcher{symbol: r.syntax.glob, separator: r.syntax.separator})
	return r
}

func (r *MatcherRegistry) Register(m Matcher) {
	r.matchers[m.Key()] = m
}

func (r *MatcherRegistry) Get(key string) (Matcher, bool) {
	m, ok := r.matchers[key]
	return m, ok
}

func (r *MatcherRegistry) Bind(key, arg string) (BoundMatcher, error) {
	m, ok := r.Get(key)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrMatcherNotFound, key)
	}
	return m.Bind(arg)
}

// ParseSegment converts a single raw path segment into a BoundMatcher.
//
//   - "*"                         -> glob matcher
//   - "<<id>>"                    -> default ("var") matcher with arg "id"
//   - "<<int:id>>"                -> "int" matcher with arg "id"
//   - "<<uuid:id>>"               -> "uuid" matcher with arg "id"
//   - "<<regex:post_id:[0-9]+>>"  -> "regex" matcher with arg "post_id:[0-9]+"
//   - "users"                     -> static literal matcher
func (r *MatcherRegistry) ParseSegment(seg string) (BoundMatcher, error) {
	if seg == r.syntax.glob {
		return r.Bind("glob", seg)
	}
	if inner, ok := r.syntax.variable(seg); ok {
		return r.bindVariable(inner)
	}
	return StaticMatcher{text: seg}, nil
}

func (r *MatcherRegistry) bindVariable(inner string) (BoundMatcher, error) {
	if inner == "" {
		return nil, fmt.Errorf("%w: empty variable segment", ErrInvalidMatcherArg)
	}
	if key, arg, ok := strings.Cut(inner, ":"); ok {
		return r.Bind(key, arg)
	}
	return r.Bind(r.defaultKey, inner)
}

func (r *MatcherRegistry) ParsePath(pattern string) ([]BoundMatcher, error) {
	buf := r.pathBuf(pattern)
	if buf.Done() {
		return nil, nil
	}
	var out []BoundMatcher
	for !buf.Done() {
		seg := buf.Consume()
		if seg == "" {
			return nil, fmt.Errorf("%w: empty segment in %q", ErrInvalidMatcherArg, pattern)
		}
		bm, err := r.ParseSegment(seg)
		if err != nil {
			return nil, err
		}
		if bm.Flags().Has(M_TERMINAL) && !buf.Done() {
			return nil, ErrTerminalNotLast
		}
		out = append(out, bm)
	}
	return out, nil
}

func (r *MatcherRegistry) pathBuf(path string) PathBuf {
	if r == nil {
		return NewPathBuf(path)
	}
	return newPathBuf(path, r.syntax.separator)
}
