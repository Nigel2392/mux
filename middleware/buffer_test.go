package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware"
)

func serve(multiplexer http.Handler) *httptest.Server {
	return httptest.NewServer(multiplexer)
}

func TestBufferMiddleware(t *testing.T) {
	var rt = mux.New()
	rt.Use(middleware.BufferMiddleware(nil))
	rt.Handle(mux.POST, "/buffer/", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Buffered Response"))
	}), "buffered")

	var server = serve(rt)
	defer server.Close()

	resp, err := http.Post(server.URL+"/buffer/", "text/plain", nil)
	if err != nil {
		t.Fatalf("Failed to make POST request: %v", err)
	}

	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "Buffered Response" {
		t.Errorf("Expected 'Buffered Response', got '%s'", buf.String())
	}
}

func TestBufferMiddlewarePanic(t *testing.T) {
	var rt = mux.New()
	rt.Use(middleware.Recoverer(func(err error, w http.ResponseWriter, r *http.Request) {
		http.Error(w, "--Internal Server Error--", http.StatusInternalServerError)
	}))
	rt.Use(middleware.BufferMiddleware(nil))
	rt.Handle(mux.GET, "/panic/", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {

		w.Write([]byte("Before panic"))

		panic("intentional panic for testing")
	}), "panic")

	var server = serve(rt)
	defer server.Close()

	resp, err := http.Get(server.URL + "/panic/")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500, got %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "--Internal Server Error--\n" {
		t.Errorf("Expected 'Internal Server Error', got '%s'", buf.String())
	}

	t.Logf("Panic handled successfully with Recoverer and BufferMiddleware, response: %s", buf.String())
}

func TestBufferMiddlewarePanic2(t *testing.T) {
	var rt = mux.New()
	rt.Use(middleware.BufferMiddleware(nil))
	rt.Use(middleware.Recoverer(func(err error, w http.ResponseWriter, r *http.Request) {
		http.Error(w, "--Internal Server Error--", http.StatusInternalServerError)
	}))
	rt.Handle(mux.GET, "/panic/", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {

		w.Write([]byte("Before panic"))

		panic("intentional panic for testing")
	}), "panic")

	var server = serve(rt)
	defer server.Close()

	resp, err := http.Get(server.URL + "/panic/")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500, got %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "Before panic--Internal Server Error--\n" {
		t.Errorf("Expected 'Before panic--Internal Server Error--', got '%s'", buf.String())
	}

	t.Logf("Panic handled successfully with BufferMiddleware and Recoverer, response: %s", buf.String())
}
