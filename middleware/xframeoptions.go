//go:build !js && !wasm
// +build !js,!wasm

package middleware

import (
	"net/http"

	"github.com/Nigel2392/mux"
)

// XFrameOption is the type for the XFrameOptions middleware.
type XFrameOption string

const (
	// XFrameDeny is the most restrictive option, and it tells the browser to not display the content in an iframe.
	XFrameDeny XFrameOption = "DENY"
	// XFrameSame is the default value for XFrameOptions.
	XFrameSame XFrameOption = "SAMEORIGIN"
	// XFrameAllow is a special case, and it should not be used.
	// It is obsolete and is only here for backwards compatibility.
	XFrameAllow XFrameOption = "ALLOW-FROM"
)

// X-Frame-Options is a header that can be used to indicate whether or not a browser should be allowed to render a page in a <frame>, <iframe> or <object>.
//
// Sites can use this to avoid clickjacking attacks, by ensuring that their content is not embedded into other sites.
func XFrameOptions(options XFrameOption) mux.Middleware {
	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", string(options))
			next.ServeHTTP(w, r)
		})
	}
}
