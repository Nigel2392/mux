package middleware

import (
	"net/http"
	"strconv"

	"github.com/Nigel2392/mux"
)

// Set the cache headers for the response.
// This will enable caching for the specified amount of seconds.
func Cache(maxAge int) func(next mux.Handler) mux.Handler {
	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			for _, header := range etagHeaders {
				w.Header().Del(header)
			}
			w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(maxAge))
			next.ServeHTTP(w, r)
		})
	}
}

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

// Set the cache headers for the response.
// This will disable caching.
func NoCache(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		for _, header := range etagHeaders {
			w.Header().Del(header)
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, no-transform, must-revalidate, private, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}
