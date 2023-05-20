package mux_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/Nigel2392/mux"
)

var rt = mux.New()

func init() {
	rt.Handle(mux.GET, "/", index)
	rt.Handle(mux.GET, "/hello/world/<<name>>/<<age>>/", helloworldnameage)
	var route = rt.Handle(mux.GET, "/hello/", hello)
	route = route.Handle(mux.GET, "/world/", helloworld)
	route = route.Handle(mux.GET, "/<<name>>/", helloworldname)
	route.Handle(mux.GET, "/<<age>>/*/", helloworldnameageglob)
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "index")
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello")
}

func helloworld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "helloworld")
}

func helloworldname(w http.ResponseWriter, r *http.Request) {
	var v = mux.GetVariables(r)
	fmt.Fprintf(w, "helloworldname: %s", v.GetAll("name"))
}

func helloworldnameage(w http.ResponseWriter, r *http.Request) {
	var v = mux.GetVariables(r)
	fmt.Fprintf(w, "helloworldnameage: %s, %s", v.GetAll("name"), v.GetAll("age"))
}

func helloworldnameageglob(w http.ResponseWriter, r *http.Request) {
	var v = mux.GetVariables(r)
	fmt.Fprintf(w, "helloworldnameageglob: %s, %s, %s", v.GetAll("name"), v.GetAll("age"), v.GetAll(mux.GLOB))
}

type response_writer struct {
	bytes.Buffer
	headers http.Header
}

func (w response_writer) Header() http.Header {
	return w.headers
}

func (w response_writer) WriteHeader(statusCode int) {
	w.headers["Status"] = []string{fmt.Sprintf("%d", statusCode)}
}

func TestRouter(t *testing.T) {
	var tests = []struct {
		path     string
		expected string
	}{
		{
			path:     "/",
			expected: "index",
		},
		{
			path:     "/hello/",
			expected: "hello",
		},
		{
			path:     "/hello/world/",
			expected: "helloworld",
		},
		{
			path:     "/hello/world/john/",
			expected: "helloworldname: [john]",
		},
		{
			path:     "/hello/world/john/23/",
			expected: "helloworldnameage: [john], [23]",
		},
		{
			path:     "/hello/world/john/23/this/is/a/glob/",
			expected: "helloworldnameageglob: [john], [23], [this is a glob]",
		},
	}
	for _, test := range tests {
		var req, _ = http.NewRequest("GET", test.path, nil)
		var w = response_writer{}
		rt.ServeHTTP(&w, req)
		if w.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, w.String())
			return
		}
		t.Logf("%s ---> %s", test.path, w.String())
	}
}
