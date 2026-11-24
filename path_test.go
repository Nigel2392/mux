package mux_test

import (
	"strings"
	"testing"

	mux "github.com/Nigel2392/mux/v2"
)

func TestPathBufSegments(t *testing.T) {
	for _, tt := range []struct {
		path     string
		segments []string
	}{
		{"", nil},
		{"/", nil},
		{"///", nil},
		{"one", []string{"one"}},
		{"/one/", []string{"one"}},
		{"//one/two///", []string{"one", "two"}},
		{"/one//two/", []string{"one", "", "two"}},
		{"/café/東京/%2F/", []string{"café", "東京", "%2F"}},
		{"/./../one+two/", []string{".", "..", "one+two"}},
	} {
		t.Run(tt.path, func(t *testing.T) {
			buf := mux.NewPathBuf(tt.path)
			for _, want := range tt.segments {
				if buf.Done() {
					t.Fatalf("path ended before segment %q", want)
				}
				pos := buf.Pos
				if got := buf.Peek(); got != want || buf.Pos != pos {
					t.Fatalf("Peek() = %q at %d, want %q at %d", got, buf.Pos, want, pos)
				}
				if got := buf.Consume(); got != want {
					t.Fatalf("Consume() = %q, want %q", got, want)
				}
			}
			if !buf.Done() || buf.Peek() != "" || buf.Consume() != "" || buf.ConsumeRest() != "" {
				t.Fatalf("unexpected data after final segment: %+v", buf)
			}
		})
	}
}

func TestPathBufConsumeRest(t *testing.T) {
	buf := mux.NewPathBuf("///one//two/three///")
	if buf.Consume() != "one" {
		t.Fatal("expected first segment")
	}
	copy := buf
	if got := buf.ConsumeRest(); got != "/two/three" || !buf.Done() {
		t.Fatalf("ConsumeRest() = %q at position %d", got, buf.Pos)
	}
	if copy.Done() || copy.Consume() != "" || copy.Consume() != "two" {
		t.Fatalf("advancing a copy changed the original cursor state: %+v", copy)
	}
}

func FuzzPathBuf(f *testing.F) {
	for _, raw := range []string{"", "/", "///", "/one/two/", "/one//two/", "/café/東京", "/%2F/+", "one\x00/two"} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		clean := strings.Trim(raw, "/")
		buf := mux.NewPathBuf(raw)
		for clean != "" {
			first, rest, _ := strings.Cut(clean, "/")
			if buf.Done() || buf.Peek() != first || buf.Consume() != first {
				t.Fatalf("failed to consume segment %q from %q", first, raw)
			}
			remaining := buf
			if got := remaining.ConsumeRest(); got != rest || !remaining.Done() {
				t.Fatalf("ConsumeRest() = %q, want %q", got, rest)
			}
			clean = rest
		}
		if !buf.Done() {
			t.Fatalf("path still has data: %+v", buf)
		}
	})
}
