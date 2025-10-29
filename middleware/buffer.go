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
			w = newFn(w, r)
			next.ServeHTTP(w, r)

			if buffered, ok := w.(BufferedResponse); ok {
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

func NewBufferedWriter(w http.ResponseWriter) http.ResponseWriter {
	return &bufferedResponseWriter{
		w:   w,
		hdr: w.Header().Clone(),
	}
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

	var dst = bw.w.Header()
	for k := range dst {
		if _, ok := bw.hdr[k]; !ok {
			delete(dst, k)
		}
	}

	maps.Copy(dst, bw.hdr)

	bw.buf.WriteTo(bw.w)
}
