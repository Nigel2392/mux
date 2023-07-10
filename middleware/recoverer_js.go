//go:build js && wasm
// +build js,wasm

package middleware

import (
	"errors"
	"fmt"

	"github.com/Nigel2392/mux"
)

// Recoverer recovers from panics and logs the error,
// if the logger was set.
func Recoverer(onError func(err error, v mux.Variables)) mux.Middleware {
	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(v mux.Variables) {
			defer func() {
				if err := recover(); err != nil {
					switch err := err.(type) {
					case error:
						onError(err, v)
					case string:
						onError(errors.New(err), v)
					default:
						onError(fmt.Errorf("%v", err), v)
					}
				}
			}()
			next.ServeHTTP(v)
		})
	}
}
