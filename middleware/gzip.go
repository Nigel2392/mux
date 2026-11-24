//go:build !js && !wasm
// +build !js,!wasm

package middleware

import (
	"compress/gzip"
	"net/http"
)

// GZIP compresses the response using gzip compression.
func GZIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		// Compress the response
		var gz = gzip.NewWriter(w)
		defer gz.Close()

		// Create gzip response writer
		var gzw = gzipResponseWriter{ResponseWriter: w, Writer: gz}
		next.ServeHTTP(gzw, r)

		// Flush any remaining data to the underlying writer
		gz.Flush()
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	*gzip.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) { return w.Writer.Write(b) }
func (w gzipResponseWriter) Header() http.Header         { return w.ResponseWriter.Header() }
