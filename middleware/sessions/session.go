//go:build !js && !wasm
// +build !js,!wasm

package sessions

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/Nigel2392/mux"
	"github.com/alexedwards/scs/v2"
)

const session_interface_key = "mux.middleware.sessions.Session"

// This interface will be set on the request's context.
type Session interface {
	Set(key string, value interface{})
	Get(key string) interface{}
	Exists(key string) bool
	Delete(key string)
	Destroy() error
	RenewToken() error
}

// Retrieve a session for the request.
//
// Makes the session globally available, and avoids import cycles.
func Retrieve(r *http.Request) Session {
	var v = r.Context().Value(session_interface_key)
	if v == nil {
		return nil
	}
	return v.(Session)
}

// Add a session to the request.
func SessionMiddleware(store *scs.SessionManager) mux.Middleware {
	return (func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			var token string
			cookie, err := r.Cookie(store.Cookie.Name)
			if err == nil {
				token = cookie.Value
			}

			ctx, err := store.Load(r.Context(), token)
			if err != nil {
				store.ErrorFunc(w, r, err)
				return
			}

			r = r.WithContext(ctx)
			bw := &bufferedResponseWriter{ResponseWriter: w}

			// ADDED CODE
			// 		ADDED CODE
			// 			ADDED CODE
			var scsSession = &scsRequestSession{
				r:     r,
				store: store,
			}
			r = r.WithContext(
				context.WithValue(r.Context(), session_interface_key, scsSession))
			// 			ADDED CODE
			// 		ADDED CODE
			// ADDED CODE

			next.ServeHTTP(bw, r)

			if r.MultipartForm != nil {
				r.MultipartForm.RemoveAll()
			}

			switch store.Status(ctx) {
			case scs.Modified:
				token, expiry, err := store.Commit(ctx)
				if err != nil {
					store.ErrorFunc(w, r, err)
					return
				}

				store.WriteSessionCookie(ctx, w, token, expiry)
			case scs.Destroyed:
				store.WriteSessionCookie(ctx, w, "", time.Time{})
			}

			w.Header().Add("Vary", "Cookie")

			if bw.code != 0 {
				w.WriteHeader(bw.code)
			}
			w.Write(bw.buf.Bytes())
		})
	})
}

// Oh yes, please derive your code from my package;
// Exporting? Oh, no of course not!
// Two implementations have to exist. Right?
type bufferedResponseWriter struct {
	http.ResponseWriter
	buf  bytes.Buffer
	code int
}

func (bw *bufferedResponseWriter) Write(b []byte) (int, error) {
	return bw.buf.Write(b)
}

func (bw *bufferedResponseWriter) WriteHeader(code int) {
	bw.code = code
}
