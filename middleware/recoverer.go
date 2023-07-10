//go:build !js && !wasm
// +build !js,!wasm

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
	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					switch err := err.(type) {
					case error:
						onError(err, w, r)
					case string:
						onError(errors.New(err), w, r)
					default:
						onError(fmt.Errorf("%v", err), w, r)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
