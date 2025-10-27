package sessions

import (
	"net/http"

	_ "unsafe"

	"github.com/alexedwards/scs/v2"
)

//go:linkname _commitAndWriteSessionCookie github.com/alexedwards/scs/v2.(*SessionManager).commitAndWriteSessionCookie
func _commitAndWriteSessionCookie(s *scs.SessionManager, w http.ResponseWriter, r *http.Request)

func commitAndWriteSessionCookie(s *scs.SessionManager, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Vary", "Cookie")
	_commitAndWriteSessionCookie(s, w, r)
}

// Oh yes, please derive your code from my package;
// Exporting? Oh, no of course not!
// Two implementations have to exist. Right?
type sessionResponseWriter struct {
	http.ResponseWriter
	request        *http.Request
	sessionManager *scs.SessionManager
	written        bool
}

func (sw *sessionResponseWriter) Write(b []byte) (int, error) {
	if !sw.written {
		commitAndWriteSessionCookie(sw.sessionManager, sw.ResponseWriter, sw.request)
		sw.written = true
	}

	return sw.ResponseWriter.Write(b)
}

func (sw *sessionResponseWriter) WriteHeader(code int) {
	if !sw.written {
		commitAndWriteSessionCookie(sw.sessionManager, sw.ResponseWriter, sw.request)
		sw.written = true
	}

	sw.ResponseWriter.WriteHeader(code)
}

func (sw *sessionResponseWriter) Unwrap() http.ResponseWriter {
	return sw.ResponseWriter
}
