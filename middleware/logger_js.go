package middleware

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Nigel2392/mux"
)

type Logger struct {
	Output io.Writer
}

func (l *Logger) Intercept(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(v mux.Variables) {
		var startTime = time.Now()
		next.ServeHTTP(v)
		var timeTaken = time.Since(startTime)
		fmt.Fprintf(l.Output, "[%s] %v\n", timeTaken, v)
	})
}
