package mux_test

import (
	"errors"
	"strings"
	"testing"
	"uuid"

	mux "github.com/Nigel2392/mux/v2"
)

func TestMatcherRegistryParseErrors(t *testing.T) {
	registry := mux.NewMatcherRegistry()
	for _, tt := range []struct {
		path string
		err  error
	}{
		{"/<<>>", mux.ErrInvalidMatcherArg},
		{"/<<var:>>", mux.ErrInvalidMatcherArg},
		{"/<<int:>>", mux.ErrInvalidMatcherArg},
		{"/<<uuid:>>", mux.ErrInvalidMatcherArg},
		{"/<<missing:id>>", mux.ErrMatcherNotFound},
		{"/<<regex:id>>", mux.ErrInvalidMatcherArg},
		{"/<<regex:id:>>", mux.ErrInvalidMatcherArg},
		{"/<<regex::[0-9]+>>", mux.ErrInvalidMatcherArg},
		{"/<<regex:id:[>>", mux.ErrInvalidMatcherArg},
		{"/one//two", mux.ErrInvalidMatcherArg},
		{"/one/*/two", mux.ErrTerminalNotLast},
		{"/one/<<glob:rest>>/two", mux.ErrTerminalNotLast},
	} {
		t.Run(tt.path, func(t *testing.T) {
			if parsed, err := registry.ParsePath(tt.path); parsed != nil || !errors.Is(err, tt.err) {
				t.Fatalf("ParsePath() = %v, %v; want nil, %v", parsed, err, tt.err)
			}
		})
	}
	for _, key := range []string{"var", "int", "uuid", "regex", "glob"} {
		if matcher, ok := registry.Get(key); !ok || matcher.Key() != key {
			t.Errorf("registry missing %q", key)
		}
	}
	if parsed, err := registry.ParsePath("///"); len(parsed) != 0 || err != nil {
		t.Fatalf("ParsePath(root) = %v, %v", parsed, err)
	}
}

func TestBoundMatcherIdentity(t *testing.T) {
	registry := mux.NewMatcherRegistry()
	for _, tt := range []struct {
		segment, name, key, literal, display string
		flags                                mux.MatcherFlag
		priority                             int
	}{
		{"items", "", "static:items", "items", "items", mux.M_LITERAL, 1000},
		{"<<id>>", "id", "var", "", "<<id>>", mux.M_CAPTURE, 100},
		{"<<int:id>>", "id", "int", "", "<<int:id>>", mux.M_CAPTURE, 200},
		{"<<uuid:id>>", "id", "uuid", "", "<<uuid:id>>", mux.M_CAPTURE, 250},
		{"<<regex:code:a:b>>", "code", "regex:a:b", "", "<<regex:code:a:b>>", mux.M_CAPTURE, 300},
		{"*", "*", "glob", "", "*", mux.M_CAPTURE | mux.M_TERMINAL, 0},
		{"<<glob:rest>>", "rest", "glob", "", "rest", mux.M_CAPTURE | mux.M_TERMINAL, 0},
		{"<<glob:>>", "*", "glob", "", "*", mux.M_CAPTURE | mux.M_TERMINAL, 0},
	} {
		t.Run(tt.segment, func(t *testing.T) {
			matcher, err := registry.ParseSegment(tt.segment)
			if err != nil {
				t.Fatal(err)
			}
			if matcher.Name() != tt.name || matcher.Key() != tt.key || matcher.Literal() != tt.literal ||
				matcher.String() != tt.display || matcher.Flags() != tt.flags || matcher.Priority() != tt.priority {
				t.Fatalf("unexpected matcher metadata for %q: %#v", tt.segment, matcher)
			}
		})
	}
}

const sampleUUID = "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"

func TestBoundMatcherMatch(t *testing.T) {
	registry := mux.NewMatcherRegistry()
	for _, tt := range []struct {
		segment, path, value, rest string
		ok                         bool
	}{
		{"items", "/items/next", "", "next", true},
		{"items", "/ITEMS/next", "", "next", false},
		{"<<id>>", "/café/next", "café", "next", true},
		{"<<id>>", "/", "", "", false},
		{"<<int:id>>", "/012345678901234567890/next", "012345678901234567890", "next", true},
		{"<<int:id>>", "/-12/next", "", "next", false},
		{"<<int:id>>", "/+12/next", "", "next", false},
		{"<<int:id>>", "/١٢/next", "", "next", false},
		{"<<int:id>>", "/", "", "", false},
		{"<<uuid:id>>", "/" + sampleUUID + "/next", sampleUUID, "next", true},
		{"<<uuid:id>>", "/" + strings.ToUpper(sampleUUID) + "/next", strings.ToUpper(sampleUUID), "next", true},
		{"<<uuid:id>>", "/{" + sampleUUID + "}/next", "{" + sampleUUID + "}", "next", true},
		{"<<uuid:id>>", "/urn:uuid:" + sampleUUID + "/next", "urn:uuid:" + sampleUUID, "next", true},
		{"<<uuid:id>>", "/f81d4fae7dec11d0a76500a0c91e6bf6/next", "f81d4fae7dec11d0a76500a0c91e6bf6", "next", true},
		{"<<uuid:id>>", "/f81d4fae-7dec-11d0-a765-00a0c91e6bfz/next", "", "next", false},
		{"<<uuid:id>>", "/123/next", "", "next", false},
		{"<<uuid:id>>", "/", "", "", false},
		{"<<regex:code:a:b>>", "/a:b/next", "a:b", "next", true},
		{"<<regex:code:ab|cd>>", "/abc/next", "", "next", false},
		{"<<regex:code:ab|cd>>", "/xcd/next", "", "next", false},
		{"<<regex:code:.*>>", "/", "", "", false},
		{"*", "/one//two/", "one//two", "", true},
		{"*", "/", "", "", true},
	} {
		t.Run(tt.segment+" "+tt.path, func(t *testing.T) {
			matcher, err := registry.ParseSegment(tt.segment)
			if err != nil {
				t.Fatal(err)
			}
			buf := mux.NewPathBuf(tt.path)
			value, ok := matcher.Match(&buf)
			if rest := buf.ConsumeRest(); value != tt.value || ok != tt.ok || rest != tt.rest {
				t.Fatalf("Match() = %q, %v, remaining %q; want %q, %v, remaining %q", value, ok, rest, tt.value, tt.ok, tt.rest)
			}
		})
	}
}

func TestBoundMatcherFormat(t *testing.T) {
	registry := mux.NewMatcherRegistry()
	for _, tt := range []struct {
		name, segment, output string
		args                  []any
		consumed              int
		err                   error
	}{
		{"literal", "items", "items", []any{1}, 0, nil},
		{"variable", "<<id>>", "8", []any{8, 9}, 1, nil},
		{"variable-missing", "<<id>>", "", nil, 0, mux.ErrNotEnoughVariables},
		{"int", "<<int:id>>", "123", []any{123}, 1, nil},
		{"int-missing", "<<int:id>>", "", nil, 0, mux.ErrNotEnoughVariables},
		{"uuid", "<<uuid:id>>", strings.ToUpper(sampleUUID), []any{strings.ToUpper(sampleUUID)}, 1, nil},
		{"uuid-typed", "<<uuid:id>>", sampleUUID, []any{uuid.MustParse(sampleUUID)}, 1, nil},
		{"uuid-invalid", "<<uuid:id>>", "", []any{"not-a-uuid"}, 0, mux.ErrInvalidMatcherArg},
		{"uuid-missing", "<<uuid:id>>", "", nil, 0, mux.ErrNotEnoughVariables},
		{"regex", "<<regex:code:[a-z]+>>", "abc", []any{"abc"}, 1, nil},
		{"regex-invalid", "<<regex:code:[a-z]+>>", "", []any{123}, 0, mux.ErrInvalidMatcherArg},
		{"regex-missing", "<<regex:code:[a-z]+>>", "", nil, 0, mux.ErrNotEnoughVariables},
		{"glob", "*", "one/2/three", []any{"one", 2, "three"}, 3, nil},
		{"glob-empty", "*", "", nil, 0, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := registry.ParseSegment(tt.segment)
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			n, err := matcher.Format(&out, tt.args)
			if n != tt.consumed || !errors.Is(err, tt.err) || out.String() != tt.output {
				t.Fatalf("Format() = %q, %d, %v; want %q, %d, %v", out.String(), n, err, tt.output, tt.consumed, tt.err)
			}
		})
	}
}

func TestUUIDRoutePriorityAndFallback(t *testing.T) {
	router := mux.New()
	slugRoute := router.Get("/items/<<slug>>", handler)
	intRoute := router.Get("/items/<<int:number>>", handler)
	uuidRoute := router.Get("/items/<<uuid:id>>", handler)
	regexRoute := router.Get("/items/<<regex:code:F81D[0-9A-F-]+>>", handler)

	for _, tt := range []struct {
		segment string
		route   *mux.Route
		key     string
	}{
		{sampleUUID, uuidRoute, "id"},
		{"12345678123412341234123456789012", uuidRoute, "id"},
		{strings.ToUpper(sampleUUID), regexRoute, "code"},
		{"123", intRoute, "number"},
		{"not-a-uuid", slugRoute, "slug"},
	} {
		t.Run(tt.segment, func(t *testing.T) {
			route, vars := router.Match(mux.GET, "/items/"+tt.segment)
			if route != tt.route || len(vars) != 1 || vars.Get(tt.key) != tt.segment {
				t.Fatalf("Match() = (%p, %v), want (%p, %q: %q)", route, vars, tt.route, tt.key, tt.segment)
			}
		})
	}
}

func TestReverseUUIDRoute(t *testing.T) {
	router := mux.New()
	route := router.Get("/items/<<uuid:id>>", handler, "item")

	path, err := router.Reverse("item", uuid.MustParse(sampleUUID))
	if want := "/items/" + sampleUUID + "/"; err != nil || path != want {
		t.Fatalf("Reverse() = (%q, %v), want (%q, nil)", path, err, want)
	}
	matched, vars := router.Match(mux.GET, path)
	if matched != route || vars.Get("id") != sampleUUID {
		t.Fatalf("Match(%q) = (%p, %v), want (%p, id: %q)", path, matched, vars, route, sampleUUID)
	}
	if _, err := router.Reverse("item", "not-a-uuid"); !errors.Is(err, mux.ErrInvalidMatcherArg) {
		t.Fatalf("Reverse(invalid UUID) error = %v, want %v", err, mux.ErrInvalidMatcherArg)
	}
}

type failingWriter struct {
	writes int
	err    error
}

func (w *failingWriter) WriteString(s string) (int, error) {
	if w.writes == 0 {
		return 0, w.err
	}
	w.writes--
	return len(s), nil
}

func TestBoundMatcherFormatPropagatesWriterError(t *testing.T) {
	registry := mux.NewMatcherRegistry()
	writerErr := errors.New("write failed")
	for _, tt := range []struct {
		segment          string
		writes, consumed int
	}{
		{"items", 0, 0},
		{"<<id>>", 0, 1},
		{"<<int:id>>", 0, 1},
		{"<<regex:code:[0-9]+>>", 0, 1},
		{"*", 0, 0},
		{"*", 1, 1},
		{"*", 2, 1},
	} {
		matcher, err := registry.ParseSegment(tt.segment)
		if err != nil {
			t.Fatal(err)
		}
		writer := &failingWriter{writes: tt.writes, err: writerErr}
		if n, err := matcher.Format(writer, []any{123, 456}); n != tt.consumed || !errors.Is(err, writerErr) {
			t.Errorf("%s after %d writes: Format() = %d, %v; want %d, %v", tt.segment, tt.writes, n, err, tt.consumed, writerErr)
		}
	}
}
