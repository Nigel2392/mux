//go:build !js && !wasm
// +build !js,!wasm

package middleware

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Nigel2392/mux/hosts"
)

type Logger struct {
	Output         io.Writer
	RequestProxied bool
}

var DefaultLogger = &Logger{
	Output: os.Stdout,
}

func (l *Logger) Intercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			remoteAddr = hosts.GetIP(r, l.RequestProxied)
			startTime  = time.Now()
			method     = r.Method
			path       = r.URL.Path
		)

		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}

		next.ServeHTTP(w, r)

		var timeTaken = time.Since(startTime)
		fmt.Fprintf(l.Output, "[%s %s] %s %s\n", timeTaken, method, remoteAddr, path)
	})
}
