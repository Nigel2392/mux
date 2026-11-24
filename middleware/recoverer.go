//go:build !js && !wasm
// +build !js,!wasm

package middleware

import (
	"errors"
	"fmt"
	"net/http"
)

// Recoverer recovers from panics and logs the error,
// if the logger was set.
//
// Deprecated: Use [RecovererMiddleware] struct type instead.
func Recoverer(onError func(err error, w http.ResponseWriter, r *http.Request)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

type RecovererMiddleware struct {
	Next  func(next http.Handler, w http.ResponseWriter, r *http.Request)
	Error func(err error, w http.ResponseWriter, r *http.Request)
}

func (rm *RecovererMiddleware) HandleError(err error, w http.ResponseWriter, r *http.Request) {
	if rm.Error != nil {
		rm.Error(err, w, r)
	} else {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (rm *RecovererMiddleware) HandleNext(next http.Handler, w http.ResponseWriter, r *http.Request) {
	if rm.Next != nil {
		rm.Next(next, w, r)
	} else {
		next.ServeHTTP(w, r)
	}
}

func (rm *RecovererMiddleware) Handle(next http.Handler) http.Handler {
	return &_RecovererMiddleware{
		RecovererMiddleware: rm,
		next:                next,
	}
}

type _RecovererMiddleware struct {
	*RecovererMiddleware
	next http.Handler
}

func (rm *_RecovererMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			switch err := err.(type) {
			case error:
				rm.HandleError(err, w, r)
			case string:
				rm.HandleError(errors.New(err), w, r)
			default:
				rm.HandleError(fmt.Errorf("%v", err), w, r)
			}
		}
	}()

	rm.HandleNext(rm.next, w, r)
}
