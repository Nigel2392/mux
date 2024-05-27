//go:build !js && !wasm
// +build !js,!wasm

package authentication

import (
	"context"
	"net/http"

	"github.com/Nigel2392/mux"
)

const context_user_key = "mux.middleware.authentication.User"

// Default request user interface.
//
// This interface can be used to check if the user
// is authenticated and if the user is an administrator.
//
// If you want to use these middlewares, you should implement this interface.
type User interface {
	// Check if the user is authenticated
	IsAuthenticated() bool

	// Check if the user is an administator
	IsAdmin() bool
}

// Add a user to the request.
func AddUserMiddleware(f func(*http.Request) User) mux.Middleware {

	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), context_user_key, f(r)))
			next.ServeHTTP(w, r)
		})
	}
}

// Helper function to get the user from the request.
func Retrieve(r *http.Request) User {
	var v = r.Context().Value(context_user_key)
	if v == nil {
		return nil
	}
	return v.(User)
}
