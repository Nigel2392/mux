//go:build !js && !wasm
// +build !js,!wasm

package authentication

import (
	"context"
	"net/http"

	"github.com/Nigel2392/mux"
)

const (
	context_user_key = "mux.middleware.authentication.User"
)

type userContextValue struct {
	load func() User
	user User
}

func (u *userContextValue) User() User {
	if u.user == nil {
		u.user = u.load()
	}
	return u.user
}

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

func UserFromContext(ctx context.Context) User {
	var v = ctx.Value(context_user_key)
	if v == nil {
		return nil
	}
	return v.(*userContextValue).User()
}

// Helper function to get the user from the request.
func Retrieve(r *http.Request) User {
	return UserFromContext(r.Context())
}

// Add a user to the request.
func AddUserMiddleware(f func(*http.Request) User) mux.Middleware {
	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {

			r = r.WithContext(context.WithValue(r.Context(), context_user_key, &userContextValue{
				load: func() User {
					return f(r)
				},
			}))

			next.ServeHTTP(w, r)
		})
	}
}

// Middleware that only allows users who are authenticated to continue.
// By default, will call the notAuth function.
// Configure the AddUserMiddleware to change the default behavior.
func LoginRequiredMiddleware(notAuth func(w http.ResponseWriter, r *http.Request)) mux.Middleware {
	if notAuth == nil {
		panic("LoginRequiredMiddleware: notAuth function is nil")
	}
	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			var user = Retrieve(r)
			if user == nil || !user.IsAuthenticated() {
				notAuth(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

//
//// Middleware that only allows users who are not authenticated to continue
//// By default, will never call the isAuth function.
//// Set the following function to change the default behavior:
//// Configure the AddUserMiddleware to change the default behavior.
//func LogoutRequiredMiddleware(isAuth func(r *request.Request)) func(next router.Handler) router.Handler {
//	if isAuth == nil {
//		panic("LogoutRequiredMiddleware: isAuth function is nil")
//	}
//	return func(next router.Handler) router.Handler {
//		return router.HandleFunc(func(r *request.Request) {
//			if r.User != nil && r.User.IsAuthenticated() {
//				isAuth(r)
//			} else {
//				next.ServeHTTP(r)
//			}
//		})
//	}
//}
//
//// Middleware that only allows users who are not authenticated to continue
//// By default, will never call the isAuth function.
//// Set the following function to change the default behavior:
//// Configure the AddUserMiddleware to change the default behavior.
//func LogoutRequiredRedirectMiddleware(nextURL string) func(next router.Handler) router.Handler {
//	return LogoutRequiredMiddleware(func(r *request.Request) {
//		router.RedirectWithNextURL(r, nextURL)
//	})
//}
//
