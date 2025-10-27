//go:build !js && !wasm
// +build !js,!wasm

package sessions

import (
	"context"
	"net/http"

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
	AddFinalizer(finalizer func(r *http.Request, ctx context.Context) (context.Context, error))
}

// Retrieve a session for the request.
//
// Makes the session globally available, and avoids import cycles.
func Retrieve(r *http.Request) Session {
	return SessionFromContext(r.Context())
}

// SessionFromContext retrieves the session from the context.
func SessionFromContext(ctx context.Context) Session {
	var v = ctx.Value(session_interface_key)
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

			bw := &sessionResponseWriter{
				ResponseWriter: w,
				request:        r,
				sessionManager: store,
			}

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

			ctx = r.Context()

			for _, finalizer := range scsSession.finalizers {
				ctx, err = finalizer(r, ctx)
				if err != nil {
					store.ErrorFunc(w, r, err)
					return
				}
			}

			if !bw.written {
				commitAndWriteSessionCookie(store, w, r)
			}
		})
	})
}
