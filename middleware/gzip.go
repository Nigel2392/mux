//go:build !js && !wasm
// +build !js,!wasm

package middleware

import (
	"compress/gzip"
	"net/http"

	"github.com/Nigel2392/mux"
)

// GZIP compresses the response using gzip compression.
func GZIP(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		// Compress the response
		var gz = gzip.NewWriter(w)
		defer gz.Close()
		// Create gzip response writer
		var gzw = gzipResponseWriter{ResponseWriter: w, Writer: gz}
		next.ServeHTTP(gzw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	*gzip.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) { return w.Writer.Write(b) }
func (w gzipResponseWriter) Header() http.Header         { return w.ResponseWriter.Header() }
