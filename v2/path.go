package mux

import "strings"

// PathBuf tracks traversal progress over a trimmed path with zero allocations.
type PathBuf struct {
	Path string
	Pos  int

	separator string
}

func NewPathBuf(rawPath string) PathBuf {
	return newPathBuf(rawPath, URL_DELIM)
}

func newPathBuf(rawPath, separator string) PathBuf {
	if separator == "/" || separator == "" {
		return PathBuf{Path: strings.Trim(rawPath, "/")}
	}
	path := rawPath
	for strings.HasPrefix(path, separator) {
		path = path[len(separator):]
	}
	for strings.HasSuffix(path, separator) {
		path = path[:len(path)-len(separator)]
	}
	return PathBuf{Path: path, separator: separator}
}

func (p PathBuf) Done() bool {
	return p.Pos >= len(p.Path)
}

// Peek returns the next segment without advancing Pos.
func (p PathBuf) Peek() string {
	if p.Pos >= len(p.Path) {
		return ""
	}
	if idx, _ := p.nextSeparator(); idx >= 0 {
		return p.Path[p.Pos : p.Pos+idx]
	}
	return p.Path[p.Pos:]
}

// Consume returns the next segment and advances Pos past the following '/'.
func (p *PathBuf) Consume() string {
	if p.Pos >= len(p.Path) {
		return ""
	}
	if idx, width := p.nextSeparator(); idx >= 0 {
		seg := p.Path[p.Pos : p.Pos+idx]
		p.Pos += idx + width
		return seg
	}
	seg := p.Path[p.Pos:]
	p.Pos = len(p.Path)
	return seg
}

func (p PathBuf) nextSeparator() (index, width int) {
	if p.separator == "" {
		return strings.IndexByte(p.Path[p.Pos:], '/'), 1
	}
	return strings.Index(p.Path[p.Pos:], p.separator), len(p.separator)
}

// ConsumeRest consumes and returns everything remaining (for terminal/glob matchers).
func (p *PathBuf) ConsumeRest() string {
	if p.Pos >= len(p.Path) {
		return ""
	}
	rest := p.Path[p.Pos:]
	p.Pos = len(p.Path)
	return rest
}
