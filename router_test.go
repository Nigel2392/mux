package mux_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Nigel2392/mux"
)

var rt = mux.New()

func init() {
	rt.Handle(mux.GET, "/", mux.NewHandler(index), "index")
	rt.Handle(mux.GET, "/hello/world/<<name>>/<<age>>/", mux.NewHandler(helloworldnameage), "numbered")
	var route = rt.Handle(mux.GET, "/hello/", mux.NewHandler(hello), "hello")
	route = route.Handle(mux.GET, "/world/", mux.NewHandler(helloworld), "world")
	route = route.Handle(mux.GET, "/<<name>>/", mux.NewHandler(helloworldname), "named")
	route.Handle(mux.GET, "/<<age>>/asd/*/", mux.NewHandler(helloworldnameageglob), "gobbed")
	rt.Handle(mux.GET, "/*", mux.NewHandler(index), "catchall")
}

func index(w http.ResponseWriter, r *http.Request) {
	var vars = mux.Vars(r)
	if len(vars) > 0 {
		fmt.Fprintf(w, "index: %v", vars[mux.GLOB])
	} else {
		fmt.Fprintf(w, "index")
	}
}

func hello(w http.ResponseWriter, r *http.Request) {
	var rt = mux.RouteFromContext(r.Context())
	fmt.Fprintf(w, "hello from %s", rt.Name)
}

func helloworld(w http.ResponseWriter, r *http.Request) {
	var rt = mux.RouteFromContext(r.Context())
	fmt.Fprintf(w, "helloworld from %s", rt.Name)
}

func helloworldname(w http.ResponseWriter, r *http.Request) {
	var v = mux.Vars(r)
	fmt.Fprintf(w, "helloworldname: %s", v.GetAll("name"))
}

func helloworldnameage(w http.ResponseWriter, r *http.Request) {
	var v = mux.Vars(r)
	fmt.Fprintf(w, "helloworldnameage: %s, %s", v.GetAll("name"), v.GetAll("age"))
}

func helloworldnameageglob(w http.ResponseWriter, r *http.Request) {
	var v = mux.Vars(r)
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
			path:     "/catchme",
			expected: "index: [catchme]",
		},
		{
			path:     "/hello/",
			expected: "hello from hello",
		},
		{
			path:     "/hello/world/",
			expected: "helloworld from world",
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
			path:     "/hello/world/john/23/asd/this/is/a/glob/",
			expected: "helloworldnameageglob: [john], [23], [this is a glob]",
		},
	}
	for _, test := range tests {
		var req, _ = http.NewRequest("GET", test.path, nil)
		var w = response_writer{}
		w.headers = make(http.Header)
		rt.ServeHTTP(&w, req)
		if w.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, w.String())
			return
		}
		t.Logf("%s ---> %s", test.path, w.String())
	}
}

type findTest struct {
	name     string
	expected string // path
}

var findTests = []findTest{
	{
		name:     "index",
		expected: "/",
	},
	{
		name:     "hello",
		expected: "/hello",
	},
	{
		name:     "hello:world",
		expected: "/hello/world",
	},
	{
		name:     "hello:world:named",
		expected: "/hello/world/<<name>>",
	},
	{
		name:     "numbered",
		expected: "/hello/world/<<name>>/<<age>>",
	},
	{
		name:     "hello:world:named:gobbed",
		expected: "/hello/world/<<name>>/<<age>>/asd/*",
	},
}

func TestFind(t *testing.T) {
	for _, test := range findTests {
		var route = rt.Find(test.name)
		if route == nil {
			t.Errorf("Expected to find route %s", test.name)
			return
		}
		if route.Path.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, route.Path.String())
			return
		}
		t.Logf("%s ---> %s", test.name, route.Path.String())
	}
}

var reverseTests = []reverseTest{
	{
		name:     "index",
		args:     nil,
		expected: "/",
	},
	{
		name:     "hello",
		args:     nil,
		expected: "/hello/",
	},
	{
		name:     "hello:world",
		args:     nil,
		expected: "/hello/world/",
	},
	{
		name:     "hello:world:named",
		args:     []interface{}{"john"},
		expected: "/hello/world/john/",
	},
	{
		name:     "numbered",
		args:     []interface{}{"john", 20},
		expected: "/hello/world/john/20/",
	},
	{
		name:     "hello:world:named:gobbed",
		args:     []interface{}{"john", 20, "hello/world"},
		expected: "/hello/world/john/20/asd/hello/world",
	},
}

func TestMuxReverse(t *testing.T) {
	for _, test := range reverseTests {
		var path, err = rt.Reverse(test.name, test.args...)
		if err != nil {
			t.Errorf("Expected to reverse %s, got error %v", test.name, err)
			return
		}
		if path != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, path)
			return
		}
		t.Logf("%s ---> %s", test.name, path)
	}
}

const nsContextKey = "namespace"

func TestMuxNamespace(t *testing.T) {
	var m = mux.New()
	var ns = m.Namespace(mux.NamespaceOptions{
		OnRouteAdded: func(route *mux.Route) {
			route.Name = fmt.Sprintf("%s:%s", nsContextKey, route.Name)
		},
		OnRouteServe: func(r *http.Request) *http.Request {
			return r.WithContext(context.WithValue(
				r.Context(), nsContextKey, "namespaceValue",
			))
		},
	})
	ns.Handle("GET", "/hello", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var rt = mux.RouteFromContext(r.Context())
		fmt.Fprintf(w, "Hello from namespace: %s", rt.Name)
	}), "hello")

	ns.Handle("GET", "/root", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var rt = mux.RouteFromContext(r.Context())
		var namespaceValue = r.Context().Value(nsContextKey)
		fmt.Fprintf(w, "Hello from root: %s / %s", rt.Name, namespaceValue)
	}), "root")

	var req, _ = http.NewRequest("GET", "/hello", nil)
	var w = response_writer{
		headers: make(http.Header),
	}
	m.ServeHTTP(&w, req)

	if w.String() != "Hello from namespace: namespace:hello" {
		t.Errorf("Expected 'Hello from namespace: namespace:hello', got '%s'", w.String())
		return
	}

	req, _ = http.NewRequest("GET", "/root", nil)
	w = response_writer{}
	m.ServeHTTP(&w, req)

	if w.String() != "Hello from root: namespace:root / namespaceValue" {
		t.Errorf("Expected 'Hello from root: namespace:root / namespaceValue', got '%s'", w.String())
		return
	}
}
