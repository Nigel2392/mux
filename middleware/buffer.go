package middleware

import (
	"bytes"
	"maps"
	"net/http"

	"github.com/Nigel2392/mux"
)

type BufferedResponse interface {
	http.ResponseWriter
	FlushBuffer()
}

func BufferMiddleware(newFn func(w http.ResponseWriter, r *http.Request) http.ResponseWriter) mux.Middleware {
	if newFn == nil {
		newFn = func(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
			return &bufferedResponseWriter{
				w:   w,
				hdr: w.Header().Clone(),
			}
		}
	}

	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			var bw = newFn(w, r)

			next.ServeHTTP(bw, r)

			if buffered, ok := bw.(BufferedResponse); ok {
				buffered.FlushBuffer()
			}
		})
	}
}

type bufferedResponseWriter struct {
	code int
	hdr  http.Header
	buf  bytes.Buffer
	w    http.ResponseWriter
}

func (bw *bufferedResponseWriter) WriteHeader(code int) {
	bw.code = code
}

func (bw *bufferedResponseWriter) Header() http.Header {
	return bw.hdr
}

func (bw *bufferedResponseWriter) Write(b []byte) (int, error) {
	return bw.buf.Write(b)
}

func (bw *bufferedResponseWriter) FlushBuffer() {
	if bw.code != 0 {
		bw.w.WriteHeader(bw.code)
	}

	for k := range bw.w.Header() {
		if _, ok := bw.hdr[k]; !ok {
			delete(bw.hdr, k)
		}
	}

	maps.Copy(bw.w.Header(), bw.hdr)

	bw.buf.WriteTo(bw.w)
}
