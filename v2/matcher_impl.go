package mux

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"uuid"
)

// --- Static ---

type StaticMatcher struct {
	text string
}

func (m StaticMatcher) Key() string        { return "static:" + m.text }
func (m StaticMatcher) Flags() MatcherFlag { return M_LITERAL }
func (m StaticMatcher) Priority() int      { return 1000 }
func (m StaticMatcher) String() string     { return m.text }
func (m StaticMatcher) Name() string       { return "" }
func (m StaticMatcher) Literal() string    { return m.text }

func (m StaticMatcher) Match(p *PathBuf) (string, bool) {
	seg := p.Consume()
	return "", seg == m.text
}

func (m StaticMatcher) Format(b io.StringWriter, _ []any) (int, error) {
	_, err := b.WriteString(m.text)
	return 0, err
}

// --- Var (stateless: inherits Key() == "var") ---

type VarMatcher struct{}

func (VarMatcher) Key() string { return "var" }

func (m VarMatcher) Bind(name string) (BoundMatcher, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: var requires a name", ErrInvalidMatcherArg)
	}
	return boundVar{VarMatcher: m, name: name}, nil
}

type boundVar struct {
	VarMatcher
	name string
}

func (b boundVar) Flags() MatcherFlag { return M_CAPTURE }
func (b boundVar) Priority() int      { return 100 }
func (b boundVar) String() string     { return "<<" + b.name + ">>" }
func (b boundVar) Name() string       { return b.name }
func (b boundVar) Literal() string    { return "" }

func (b boundVar) Match(p *PathBuf) (string, bool) {
	seg := p.Consume()
	return seg, seg != ""
}

func (b boundVar) Format(w io.StringWriter, args []any) (int, error) {
	if len(args) == 0 {
		return 0, ErrNotEnoughVariables
	}
	_, err := w.WriteString(fmt.Sprint(args[0]))
	return 1, err
}

// --- Int (stateless: inherits Key() == "int") ---

type IntMatcher struct{}

func (IntMatcher) Key() string { return "int" }

func (m IntMatcher) Bind(name string) (BoundMatcher, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: int requires a name", ErrInvalidMatcherArg)
	}
	return boundInt{IntMatcher: m, name: name}, nil
}

type boundInt struct {
	IntMatcher
	name string
}

func (b boundInt) Flags() MatcherFlag { return M_CAPTURE }
func (b boundInt) Priority() int      { return 200 }
func (b boundInt) String() string     { return "<<int:" + b.name + ">>" }
func (b boundInt) Name() string       { return b.name }
func (b boundInt) Literal() string    { return "" }

func (b boundInt) Match(p *PathBuf) (string, bool) {
	seg := p.Consume()
	if seg == "" {
		return "", false
	}
	for i := 0; i < len(seg); i++ {
		if seg[i] < '0' || seg[i] > '9' {
			return "", false
		}
	}
	return seg, true
}

func (b boundInt) Format(w io.StringWriter, args []any) (int, error) {
	if len(args) == 0 {
		return 0, ErrNotEnoughVariables
	}
	_, err := w.WriteString(fmt.Sprint(args[0]))
	return 1, err
}

// UUIDMatcher matches a single segment in any format accepted by uuid.Parse.
type UUIDMatcher struct{}

func (UUIDMatcher) Key() string { return "uuid" }

func (m UUIDMatcher) Bind(name string) (BoundMatcher, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: uuid requires a name", ErrInvalidMatcherArg)
	}
	return boundUUID{UUIDMatcher: m, name: name}, nil
}

type boundUUID struct {
	UUIDMatcher
	name string
}

func (b boundUUID) Flags() MatcherFlag { return M_CAPTURE }
func (b boundUUID) Priority() int      { return 250 }
func (b boundUUID) String() string     { return "<<uuid:" + b.name + ">>" }
func (b boundUUID) Name() string       { return b.name }
func (b boundUUID) Literal() string    { return "" }

func (b boundUUID) Match(p *PathBuf) (string, bool) {
	seg := p.Consume()
	if _, err := uuid.Parse(seg); err != nil {
		return "", false
	}
	return seg, true
}

func (b boundUUID) Format(w io.StringWriter, args []any) (int, error) {
	if len(args) == 0 {
		return 0, ErrNotEnoughVariables
	}
	value := fmt.Sprint(args[0])
	if _, err := uuid.Parse(value); err != nil {
		return 0, fmt.Errorf("%w: value %q is not a UUID", ErrInvalidMatcherArg, value)
	}
	_, err := w.WriteString(value)
	return 1, err
}

// --- Regex (stateful: overrides Key() to "regex:<expr>") ---

type RegexMatcher struct{}

func (RegexMatcher) Key() string { return "regex" }

// Bind expects arg in the form "<name>:<pattern>" (e.g. "post_id:[0-9]+").
func (RegexMatcher) Bind(arg string) (BoundMatcher, error) {
	name, expr, ok := strings.Cut(arg, ":")
	if !ok || name == "" || expr == "" {
		return nil, fmt.Errorf("%w: regex expects name:pattern, got %q", ErrInvalidMatcherArg, arg)
	}
	re, err := regexp.Compile("^(?:" + expr + ")$")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMatcherArg, err)
	}
	return boundRegex{name: name, expr: expr, re: re}, nil
}

type boundRegex struct {
	name string
	expr string
	re   *regexp.Regexp
}

func (b boundRegex) Key() string        { return "regex:" + b.expr } // ignores b.name, distinguishes b.expr
func (b boundRegex) Flags() MatcherFlag { return M_CAPTURE }
func (b boundRegex) Priority() int      { return 300 }
func (b boundRegex) String() string     { return "<<regex:" + b.name + ":" + b.expr + ">>" }
func (b boundRegex) Name() string       { return b.name }
func (b boundRegex) Literal() string    { return "" }

func (b boundRegex) Match(p *PathBuf) (string, bool) {
	seg := p.Consume()
	if seg == "" || !b.re.MatchString(seg) {
		return "", false
	}
	return seg, true
}

func (b boundRegex) Format(w io.StringWriter, args []any) (int, error) {
	if len(args) == 0 {
		return 0, ErrNotEnoughVariables
	}
	s := fmt.Sprint(args[0])
	if !b.re.MatchString(s) {
		return 0, fmt.Errorf("%w: value %q does not match %q", ErrInvalidMatcherArg, s, b.expr)
	}
	_, err := w.WriteString(s)
	return 1, err
}

// --- Glob (stateless: inherits Key() == "glob") ---

type GlobMatcher struct {
	symbol    string
	separator string
}

func (GlobMatcher) Key() string { return "glob" }

func (m GlobMatcher) Bind(name string) (BoundMatcher, error) {
	if name == "" {
		name = m.symbol
	}
	return boundGlob{GlobMatcher: m, name: name}, nil
}

type boundGlob struct {
	GlobMatcher
	name string
}

func (b boundGlob) Flags() MatcherFlag { return M_TERMINAL | M_CAPTURE }
func (b boundGlob) Priority() int      { return 0 }
func (b boundGlob) String() string     { return b.name }
func (b boundGlob) Name() string       { return b.name }
func (b boundGlob) Literal() string    { return "" }

func (b boundGlob) Match(p *PathBuf) (string, bool) {
	return p.ConsumeRest(), true
}

func (b boundGlob) Format(w io.StringWriter, args []any) (int, error) {
	separator := b.separator
	if separator == "" {
		separator = "/"
	}
	for i, a := range args {
		if i > 0 {
			if _, err := w.WriteString(separator); err != nil {
				return i, err
			}
		}
		if _, err := w.WriteString(fmt.Sprint(a)); err != nil {
			return i, err
		}
	}
	return len(args), nil
}
