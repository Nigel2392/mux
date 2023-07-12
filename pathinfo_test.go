package mux_test

import (
	"fmt"
	"net/http"
	"strconv"
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
		var info = mux.NewPathInfo(test.path)
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

	info *mux.PathInfo

	ExpectedMatched bool
}

func TestMatch(t *testing.T) {
	var matchTests = []matchTest{
		{
			path:            "/",
			pathToMatch:     "/",
			info:            mux.NewPathInfo("/"),
			ExpectedMatched: true,
		},
		{
			path:            "/*",
			pathToMatch:     "/",
			info:            mux.NewPathInfo("/*"),
			ExpectedMatched: true,
		},
		{
			path:            "/*",
			pathToMatch:     "/asd",
			info:            mux.NewPathInfo("/*"),
			ExpectedMatched: true,
		},
		{
			path:            "/hello",
			pathToMatch:     "/hello",
			info:            mux.NewPathInfo("/hello"),
			ExpectedMatched: true,
		},
		{
			path:            "/hello",
			pathToMatch:     "/hello/",
			info:            mux.NewPathInfo("/hello"),
			ExpectedMatched: true,
		},
		{
			path:            "/hello",
			pathToMatch:     "/hello/world",
			info:            mux.NewPathInfo("/hello"),
			ExpectedMatched: false,
		},
		{
			path:            "/hello/world/",
			pathToMatch:     "/hello/world",
			info:            mux.NewPathInfo("/hello/world/"),
			ExpectedMatched: true,
		},
		{
			path:            "/hello/world/<<name>>",
			pathToMatch:     "/hello/world/john",
			info:            mux.NewPathInfo("/hello/world/<<name>>"),
			ExpectedMatched: true,
		},
		{
			path:            "/hello/world/<<name>>/*/",
			pathToMatch:     "/hello/world/john/hello/world",
			info:            mux.NewPathInfo("/hello/world/<<name>>/*/"),
			ExpectedMatched: true,
		},
		{
			path:            "/hello/123/asd/world/<<name>>/*/",
			pathToMatch:     "/hello/123/asd/world/john/",
			info:            mux.NewPathInfo("/hello/123/asd/world/<<name>>/*/"),
			ExpectedMatched: true,
		},
	}
	for _, test := range matchTests {
		var splitToMatch = mux.SplitPath(test.pathToMatch)
		fmt.Println(splitToMatch)
		fmt.Printf("%#v\n", test.info.Path)
		var matched, vars = test.info.Match(splitToMatch)
		if matched != test.ExpectedMatched {
			t.Errorf("Expected %v, got %v: %s != %s", test.ExpectedMatched, matched, test.path, test.pathToMatch)
			t.Logf("%#v", test.info)
			return
		}
		t.Log(test.path, test.pathToMatch, test.ExpectedMatched, vars)
	}

}

type testBenchMark struct {
	router                  *mux.Mux
	routes_to_be_registered []string
	routes_to_be_checked    []string
}

var testBenchMarks = []testBenchMark{
	{
		router: mux.New(),
		routes_to_be_registered: []string{
			"/",
			"/*",
			"/*",
			"/hello/",
			"/hello/world/",
			"/hello/world/<<name>>/",
			"/hello/world/<<name>>/<<age>>/",
			"/hello/world/<<name>>/<<age>>/*/",
			"/this/is/a/very/long/route/<<name>>/<<age>>/this/is/a/very/long/route/<<name>>/<<age>>/this/is/a/very/long/route/<<name>>/<<age>>/this/is/a/very/long/route/<<name>>/<<age>>/",
		},
		routes_to_be_checked: []string{
			"/",
			"/",
			"/asd",
			"/hello/",
			"/hello/world/",
			"/hello/world/john/",
			"/hello/world/john/20/",
			"/hello/world/john/20/hello/world/",
			"/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/this/is/a/very/long/route/john/20/",
		},
	},
}

func BenchmarkMatch(b *testing.B) {
	b.StopTimer()
	for _, test := range testBenchMarks {
		for _, route := range test.routes_to_be_registered {
			test.router.Handle(mux.GET, route, func(responsewriter http.ResponseWriter, request *http.Request) {})
		}
		for _, route := range test.routes_to_be_checked {
			b.StartTimer()
			b.Run("Match-"+strconv.Itoa(len(strings.Split(route, mux.URL_DELIM))), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var match, _ = test.router.Match(mux.ANY, route)
					if match == nil {
						b.Errorf("Expected %v, got %v: %s", true, match, route)
					}
				}
			})
			b.StopTimer()
		}
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
