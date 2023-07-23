package middleware

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Nigel2392/mux"
)

type Logger struct {
	Output         io.Writer
	RequestProxied bool
}

var DefaultLogger = &Logger{
	Output: os.Stdout,
}

func (l *Logger) Intercept(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var (
			remoteAddr = mux.GetIP(r, l.RequestProxied)
			startTime  = time.Now()
			method     = r.Method
			path       = r.URL.Path
		)

		next.ServeHTTP(w, r)

		var timeTaken = time.Since(startTime)
		fmt.Fprintf(l.Output, "[%s %s] %s %s\n", timeTaken, method, remoteAddr, path)
	})
}
