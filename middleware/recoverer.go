package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Nigel2392/mux"
)

// Recoverer recovers from panics and logs the error,
// if the logger was set.
func Recoverer(onError func(err error, w http.ResponseWriter, r *http.Request)) mux.Middleware {
	return func(next http.Handler) http.Handler {
		return mux.Handler(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					var newHandler = mux.Handler(func(w http.ResponseWriter, r *http.Request) {
						switch err := err.(type) {
						case error:
							onError(err, w, r)
						case string:
							onError(errors.New(err), w, r)
						default:
							onError(fmt.Errorf("%v", err), w, r)
						}
					})
					newHandler.ServeHTTP(w, r)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
