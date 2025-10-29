//go:build !js && !wasm
// +build !js,!wasm

package middleware

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/mux"
)

// Check if the request.Host is in the allowed hosts list
func AllowedHosts(allowed_hosts ...string) func(next mux.Handler) mux.Handler {
	return AllowedHostsFunc(allowed_hosts, func(w http.ResponseWriter, r *http.Request, host string) {
		http.Error(w, fmt.Sprintf("Host not allowed: %s", host), http.StatusForbidden)
	})
}

// Check if the request.Host is in the allowed hosts list
//
// notAllowed is called when the host is not allowed.
func AllowedHostsFunc(allowed_hosts []string, notAllowed func(w http.ResponseWriter, r *http.Request, host string)) func(next mux.Handler) mux.Handler {
	if len(allowed_hosts) == 0 {
		panic("AllowedHosts: No hosts provided.")
	}

	var unique_hosts = make(map[string]struct{})
	for _, host := range allowed_hosts {
		switch {
		case host == "":
			panic("AllowedHosts: Empty host not allowed.")
		case host == "*":
			return func(next mux.Handler) mux.Handler {
				return next
			}
		}
		unique_hosts[host] = struct{}{}
	}

	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			// Check if ALLOWED_HOSTS is set and if the request host is allowed
			var requestHost = mux.GetHost(r)
			if _, allowed := unique_hosts[requestHost]; !allowed {
				notAllowed(w, r, requestHost)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
