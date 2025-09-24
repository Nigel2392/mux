package mux_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Nigel2392/mux"
)

type pathInfoTest struct {
	path string

	ExpectedIsGlob bool
	ExpectedPath   []*mux.PathPart
}

func TestPathInfo(t *testing.T) {
	var pathInfoTests = []pathInfoTest{
		{
			path:           "",
			ExpectedIsGlob: false,
			ExpectedPath:   nil,
		},
		{
			path:           "/",
			ExpectedIsGlob: false,
			ExpectedPath:   nil,
		},
		{
			path:           "/hello",
			ExpectedIsGlob: false,
			ExpectedPath: []*mux.PathPart{
				{
					Part:       "hello",
					IsVariable: false,
				},
			},
		},
		{
			path:           "/hello/",
			ExpectedIsGlob: false,
			ExpectedPath: []*mux.PathPart{
				{
					Part:       "hello",
					IsVariable: false,
				},
			},
		},
		{
			path:           "/hello/world",
			ExpectedIsGlob: false,
			ExpectedPath: []*mux.PathPart{
				{
					Part:       "hello",
					IsVariable: false,
				},
				{
					Part:       "world",
					IsVariable: false,
				},
			},
		},
		{
			path:           "/hello/world/<<name>>",
			ExpectedIsGlob: false,
			ExpectedPath: []*mux.PathPart{
				{
					Part:       "hello",
					IsVariable: false,
				},
				{
					Part:       "world",
					IsVariable: false,
				},
				{
					Part:       "name",
					IsVariable: true,
				},
			},
		},
		{
			path:           "/hello/world/<<name>>/*/",
			ExpectedIsGlob: true,
			ExpectedPath: []*mux.PathPart{
				{
					Part:       "hello",
					IsVariable: false,
				},
				{
					Part:       "world",
					IsVariable: false,
				},
				{
					Part:       "name",
					IsVariable: true,
				},
				{
					Part:       "*",
					IsVariable: false,
				},
			},
		},
	}
	for _, test := range pathInfoTests {
		var info = mux.NewPathInfo(nil, test.path)
		if info.IsGlob != test.ExpectedIsGlob {
			t.Errorf("Expected %v, got %v", test.ExpectedIsGlob, info.IsGlob)
		}
		if len(info.Path) != len(test.ExpectedPath) {
			t.Errorf("Expected %v, got %v", len(test.ExpectedPath), len(info.Path))
		}
		for i, part := range info.Path {
			if part.Part != test.ExpectedPath[i].Part {
				t.Errorf("Expected %v, got %v", test.ExpectedPath[i].Part, part.Part)
			}
			if part.IsVariable != test.ExpectedPath[i].IsVariable {
				t.Errorf("Expected %v, got %v", test.ExpectedPath[i].IsVariable, part.IsVariable)
			}
			if part.IsVariable {
				if part.Part != test.ExpectedPath[i].Part {
					t.Errorf("Expected %v, got %v", test.ExpectedPath[i].Part, part.Part)
				}
			}
		}
		t.Log(info.String() + " == " + test.path)
	}
}

type matchTest struct {
	path string

	pathToMatch string

	data map[string][]string

	ExpectedMatched bool
}

func matchChain(chain []*mux.PathInfo, split []string) (bool, int, mux.Variables) {
	idx := 0
	var all mux.Variables
	for _, pi := range chain {
		var vars = make(mux.Variables)
		ok, next, vars := pi.Match(split, idx, vars)
		if !ok && next == -1 {
			return false, next, nil
		}
		// merge vars from this segment
		if len(vars) > 0 {
			if all == nil {
				all = make(mux.Variables, len(vars))
			}
			for k, v := range vars {
				all[k] = append(all[k], v...)
			}
		}
		idx = next
	}
	return idx == len(split), idx, all
}

func TestMatch(t *testing.T) {
	var matchTests = []matchTest{
		{
			path:            "/",
			pathToMatch:     "/",
			ExpectedMatched: true,
		},
		{
			path:            "/*",
			pathToMatch:     "/asd",
			ExpectedMatched: true,
		},
		{
			path:            "/hello",
			pathToMatch:     "/hello",
			ExpectedMatched: true,
		},
		{
			path:            "/hello",
			pathToMatch:     "/hello/",
			ExpectedMatched: true,
		},
		{
			path:            "/hello",
			pathToMatch:     "/hello/world",
			ExpectedMatched: false,
		},
		{
			path:            "/hello/world/",
			pathToMatch:     "/hello/world",
			ExpectedMatched: true,
		},
		{
			path:            "/hello/world/<<name>>",
			pathToMatch:     "/hello/world/john",
			ExpectedMatched: true,
		},
		{
			path:            "/hello//world/<<name>>",
			pathToMatch:     "/hello/world/john",
			ExpectedMatched: true,
		},
		{
			path:            "/hello//world/<<name>>//<<age>>",
			pathToMatch:     "/hello/world/john/20",
			ExpectedMatched: true,
		},
		{
			path:            "/hello//world/<<name>>//<<age>>",
			pathToMatch:     "/hello/world/john",
			ExpectedMatched: false,
		},
		{
			path:            "/hello/world/<<name>>//*/",
			pathToMatch:     "/hello/world/john/",
			ExpectedMatched: true,
		},
		{
			path:            "/hello/world/<<name>>//*/",
			pathToMatch:     "/hello/world/john/hello/world",
			ExpectedMatched: true,
		},
		{
			path:            "/hello/world/<<name>>/*/",
			pathToMatch:     "/hello/world/john/hello/world",
			ExpectedMatched: true,
		},
		{
			path:            "/hello/world/<<name>>/*",
			pathToMatch:     "/hello/world/john/hello/world/my/world",
			data:            map[string][]string{"name": {"john"}, mux.GLOB: {"hello", "world", "my", "world"}},
			ExpectedMatched: true,
		},
	}

	for _, test := range matchTests {
		var chain = strings.Split(test.path, mux.URL_DELIM+mux.URL_DELIM)
		var paths = make([]*mux.PathInfo, len(chain))
		for i, part := range chain {
			if !strings.HasPrefix(part, mux.URL_DELIM) {
				part = mux.URL_DELIM + part
			}
			if !strings.HasSuffix(part, mux.URL_DELIM) {
				part = part + mux.URL_DELIM
			}
			paths[i] = mux.NewPathInfo(nil, part)
		}

		for i := len(paths) - 1; i >= 0; i-- {
			if i > 0 {
				paths[i].Parent = paths[i-1]
			}
		}

		var info = paths[len(paths)-1]
		var split = mux.SplitPath(test.pathToMatch)
		var matched, _, vars = matchChain(paths, split)

		if matched != test.ExpectedMatched {
			t.Errorf("Expected %v, got %v: %s != %s (%d)", test.ExpectedMatched, matched, test.path, test.pathToMatch, len(split))
			t.Logf("%#v", info)
		}

		var expectedVars = make(mux.Variables)
		if test.data != nil {
			for k, v := range test.data {
				expectedVars[k] = v
			}

			if len(vars) != len(expectedVars) {
				t.Errorf("Expected %v, got %v: %s != %s", len(expectedVars), len(vars), test.path, test.pathToMatch)
			}
			for k, v := range vars {
				if len(v) != len(expectedVars[k]) {
					t.Errorf("Expected %v, got %v: %s != %s", len(expectedVars[k]), len(v), test.path, test.pathToMatch)
				}
				for i, val := range v {
					if val != expectedVars[k][i] {
						t.Errorf("Expected %v, got %v: %s != %s", expectedVars[k][i], val, test.path, test.pathToMatch)
					}

				}
				t.Logf("Variable %s: %s", k, v)
			}

		}

		t.Log(test.path, test.pathToMatch, test.ExpectedMatched, vars)
	}
}

type testBenchMark struct {
	router                  *mux.Mux
	routes_to_be_registered []string
	routes_to_be_checked    []testBenchmarkRoute
}

type testBenchmarkRoute struct {
	route string
	vars  map[string][]string
}

var testBenchMarks = []testBenchMark{
	{
		router: mux.New(),
		routes_to_be_registered: []string{
			"/",
			"/hello/",
			"/hello/world/",
			"/hello/world/<<name>>/",
			"/hello/world/<<name>>/<<age>>/",
			"/hello/world/<<name>>/<<age>>/*/",
			"/this/is/a/very/long/route/<<name>>/<<age>>/this/is/a/very/long/route/<<name>>/<<age>>/this/is/a/very/long/route/<<name>>/<<age>>/this/is/a/very/long/route/<<name>>/<<age>>/",
			"/that/is/a//very/long/route/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/a//very/long/a/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/a//very/long/b/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/a//very/long/c/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/a//very/long/d/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/a//very/long/e/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/was//very/long/a/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/were//very/long/b/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/how//very/long/c/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/when//very/long/d/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/that/is/why//very/long/e/<<name>>/<<age>>/this/is/a//very/long/route/<<name>>/<<age>>//this/is/a/very/long/route/<<name>>/<<age>>//this/is/a/very/long//route/<<name>>/<<age>>/",
			"/*",
		},
		routes_to_be_checked: []testBenchmarkRoute{
			{
				route: "/",
				vars:  nil,
			},
			{
				route: "/",
				vars:  nil,
			},
			{
				route: "/asd",
				vars:  mux.Variables{mux.GLOB: {"asd"}},
			},
			{
				route: "/hello/",
				vars:  nil,
			},
			{
				route: "/hello/world/",
				vars:  nil,
			},
			{
				route: "/hello/world/john/",
				vars:  mux.Variables{"name": {"john"}},
			},
			{
				route: "/hello/world/john/20/",
				vars:  mux.Variables{"name": {"john"}, "age": {"20"}},
			},
			{
				route: "/hello/world/john/20/hello/world/",
				vars:  mux.Variables{"name": {"john"}, "age": {"20"}, mux.GLOB: {"hello", "world"}},
			},
			{
				route: "/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/a/very/long/a/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/a/very/long/b/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/a/very/long/c/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/a/very/long/d/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/a/very/long/e/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/a/very/long/f/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{mux.GLOB: {"that", "is", "a", "very", "long", "f", "john", "20", "this", "is", "a", "very", "long", "route", "john", "20", "this", "is", "a", "very", "long", "route", "john", "20", "this", "is", "a", "very", "long", "route", "john", "20"}},
			},
			{
				route: "/that/is/was/very/long/a/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/were/very/long/b/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/how/very/long/c/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/when/very/long/d/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
			{
				route: "/that/is/why/very/long/e/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
				vars:  mux.Variables{"name": {"john", "john", "john", "john"}, "age": {"20", "20", "20", "20"}},
			},
		},
	},
}

func BenchmarkMatch(b *testing.B) {
	b.StopTimer()
	for _, test := range testBenchMarks {
		for _, route := range test.routes_to_be_registered {

			var split = strings.Split(route, mux.URL_DELIM+mux.URL_DELIM)
			if len(split) == 1 {
				test.router.Handle(mux.GET, route, mux.NewHandler(func(responsewriter http.ResponseWriter, request *http.Request) {}))
				continue
			}

			var curr *mux.Route
			for i, part := range split {
				strings.TrimPrefix(part, mux.URL_DELIM)
				if !strings.HasSuffix(part, mux.URL_DELIM) {
					part = part + mux.URL_DELIM
				}

				if i == 0 {
					curr = test.router.Handle(mux.GET, part, mux.NewHandler(func(responsewriter http.ResponseWriter, request *http.Request) {}))
				} else {
					curr = curr.Handle(mux.GET, part, mux.NewHandler(func(responsewriter http.ResponseWriter, request *http.Request) {}))
				}
			}
		}

		for _, route := range test.routes_to_be_checked {
			b.Run("Match-"+route.route, func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					var match, vars = test.router.Match(mux.ANY, route.route)
					if match == nil {
						b.Fatalf("Expected %v, got %v: %s", true, match, route)
					}

					if len(route.vars) != len(vars) {
						b.Fatalf("Expected %v, got %v: %s", len(route.vars), len(vars), route)
					}
				}
			})

			b.Log("Matched ", route)
		}
	}
}

type reverseTest struct {
	name     string
	args     []interface{}
	expected string
}

func TestReverse(t *testing.T) {
	var reverseTests = []reverseTest{
		{
			name:     "/",
			args:     nil,
			expected: "/",
		},
		{
			name:     "/hello",
			args:     nil,
			expected: "/hello/",
		},
		{
			name:     "/hello/world",
			args:     nil,
			expected: "/hello/world/",
		},
		{
			name:     "/hello/world/<<name>>",
			args:     []interface{}{"john"},
			expected: "/hello/world/john/",
		},
		{
			name:     "/hello/world/<<name>>/<<age>>",
			args:     []interface{}{"john", 20},
			expected: "/hello/world/john/20/",
		},
		{
			name:     "/hello/world/<<name>>/<<age>>/*/",
			args:     []interface{}{"john", 20, "hello/world"},
			expected: "/hello/world/john/20/hello/world",
		},
		{
			name:     "/hello/world/<<name>>/*/",
			args:     []interface{}{"john", 20, "hello/world"},
			expected: "/hello/world/john/20/hello/world",
		},
		{
			name:     "/hello/world/<<name>>/*/",
			args:     []interface{}{"john", 20, "hello/world/"},
			expected: "/hello/world/john/20/hello/world/",
		},
		{
			name:     "/hello/world/<<name>>/*",
			args:     []interface{}{"john", 20, "hello/world/my/world"},
			expected: "/hello/world/john/20/hello/world/my/world",
		},
		{
			name:     "/hello/world/<<name>>/<<age>>/*/",
			args:     []interface{}{"john", 20},
			expected: "/hello/world/john/20/",
		},
	}

	for _, test := range reverseTests {
		t.Run(fmt.Sprintf("Reverse-(%s)", test.name), func(t *testing.T) {
			var info = mux.NewPathInfo(nil, test.name)
			var result, err = info.Reverse(test.args...)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

//	type routevarsBenchmark struct {
//		router          *router_v3.Router
//		routes          []string
//		routes_to_match []string
//	}
//
//	var routevarsBenchmarks = []routevarsBenchmark{
//		{
//			router: router_v3.NewRouter(true),
//			routes: []string{"/",
//				"/hello",
//				"/hello/world",
//				"/hello/world/<<name:string>>",
//				"/hello/world/<<name:string>>/<<age:int>>",
//				"/hello/world/<<name:string>>/<<age:int>>/<<any>>",
//				"/this/is/a/very/long/route/<<name:string>>/<<age:int>>/this/is/a/very/long/route/<<name:string>>/<<age:int>>/this/is/a/very/long/route/<<name:string>>/<<age:int>>/this/is/a/very/long/route/<<name:string>>/<<age:int>>",
//			},
//			routes_to_match: []string{
//				"/",
//				"/hello",
//				"/hello/world",
//				"/hello/world/john",
//				"/hello/world/john/20",
//				"/hello/world/john/20/hello/world",
//				"/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20",
//			},
//		},
//	}
//
//	func BenchmarkRouteVars(b *testing.B) {
//		for _, test := range routevarsBenchmarks {
//			b.Run("RouteVars", routeVars(test))
//		}
//	}
//
//	func routeVars(test routevarsBenchmark) func(b *testing.B) {
//		return func(b *testing.B) {
//			b.StopTimer()
//			for _, route := range test.routes {
//				test.router.Get(route, router_v3.HandleFunc(func(r *request.Request) {}))
//			}
//			for _, route := range test.routes_to_match {
//				b.StartTimer()
//				b.Run("match-"+strconv.Itoa(len(strings.Split(route, mux.URL_DELIM))), func(b *testing.B) {
//					for i := 0; i < b.N; i++ {
//						var match, _, _ = test.router.Match(router_v3.GET, route)
//						if !match {
//							b.Errorf("Expected %v, got %v: %s", true, match, route)
//						}
//					}
//				})
//				b.StopTimer()
//			}
//		}
//	}
//
