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

func BufferMiddleware(new func(w http.ResponseWriter, r *http.Request) http.ResponseWriter) mux.Middleware {
	if new == nil {
		new = func(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
			return &bufferedResponseWriter{
				ResponseWriter: w,
				hdr:            w.Header().Clone(),
			}
		}
	}

	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			var bw = new(w, r)

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
	http.ResponseWriter
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
		bw.ResponseWriter.WriteHeader(bw.code)
	}

	for k := range bw.ResponseWriter.Header() {
		if _, ok := bw.hdr[k]; !ok {
			delete(bw.hdr, k)
		}
	}

	maps.Copy(bw.ResponseWriter.Header(), bw.hdr)

	bw.ResponseWriter.Write(bw.buf.Bytes())
}
