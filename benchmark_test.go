package mux_test

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	mux "github.com/Nigel2392/mux/v2"
)

func benchmarkMatch(b *testing.B, router *mux.Mux, method, path string, want *mux.Route) {
	b.Helper()
	if got, _ := router.Match(method, path); got != want {
		b.Fatalf("%s %s: got %p, want %p", method, path, got, want)
	}
	b.ReportAllocs()
	for b.Loop() {
		router.Match(method, path)
	}
}

func BenchmarkMatchLiteral(b *testing.B) {
	for _, count := range []int{1, 100, 10_000, 100_000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			router := mux.New()
			var last *mux.Route
			for i := range count {
				last = router.Get("/items/"+strconv.Itoa(i), handler)
			}
			b.Run("hit", func(b *testing.B) {
				benchmarkMatch(b, router, mux.GET, last.String(), last)
			})
			b.Run("miss", func(b *testing.B) {
				benchmarkMatch(b, router, mux.GET, "/items/missing", nil)
			})
		})
	}
}

func BenchmarkMatchDepth(b *testing.B) {
	for _, depth := range []int{1, 4, 16, 64, 256} {
		b.Run(strconv.Itoa(depth), func(b *testing.B) {
			router := mux.New()
			prefix := strings.Repeat("/segment", depth-1)
			route := router.Get(prefix+"/leaf", handler)
			b.Run("hit", func(b *testing.B) {
				benchmarkMatch(b, router, mux.GET, route.String(), route)
			})
			b.Run("late-miss", func(b *testing.B) {
				benchmarkMatch(b, router, mux.GET, prefix+"/missing", nil)
			})
		})
	}
}

func BenchmarkMatchMatchers(b *testing.B) {
	for _, tt := range []struct {
		name, pattern, path string
		miss                bool
	}{
		{"variable", "/items/<<id>>", "/items/widget", false},
		{"variable-miss", "/items/<<id>>", "/items/widget/more", true},
		{"int", "/items/<<int:id>>", "/items/12345678901234567890", false},
		{"int-miss", "/items/<<int:id>>", "/items/1234567890123456789x", true},
		{"regex", "/items/<<regex:code:[a-z]{3}-[0-9]{4}>>", "/items/abc-1234", false},
		{"regex-miss", "/items/<<regex:code:[a-z]{3}-[0-9]{4}>>", "/items/abc-123x", true},
		{"glob-empty", "/items/*", "/items", false},
		{"glob-short", "/items/*", "/items/image.png", false},
		{"glob-long", "/items/*", "/items" + strings.Repeat("/segment", 128), false},
	} {
		b.Run(tt.name, func(b *testing.B) {
			router := mux.New()
			route := router.Get(tt.pattern, handler)
			if tt.miss {
				route = nil
			}
			benchmarkMatch(b, router, mux.GET, tt.path, route)
		})
	}
}

func BenchmarkMatchCaptures(b *testing.B) {
	for _, count := range []int{1, 4, 8, 9, 16, 32} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			for _, names := range []string{"distinct", "repeated"} {
				b.Run(names, func(b *testing.B) {
					var pattern strings.Builder
					for i := range count {
						name := "id"
						if names == "distinct" {
							name += strconv.Itoa(i)
						}
						fmt.Fprintf(&pattern, "/<<%s>>", name)
					}
					router := mux.New()
					route := router.Get(pattern.String(), handler)
					benchmarkMatch(b, router, mux.GET, strings.Repeat("/value", count), route)
				})
			}
		})
	}
}

func BenchmarkMatchRegexSiblings(b *testing.B) {
	for _, count := range []int{1, 8, 32, 128} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			router := mux.New()
			routes := make([]*mux.Route, count)
			for i := range routes {
				routes[i] = router.Get(fmt.Sprintf("/items/<<regex:id:item-%03d-[a-z]+>>", i), handler)
			}
			b.Run("first", func(b *testing.B) {
				benchmarkMatch(b, router, mux.GET, "/items/item-000-blue", routes[0])
			})
			b.Run("last", func(b *testing.B) {
				benchmarkMatch(b, router, mux.GET, fmt.Sprintf("/items/item-%03d-blue", count-1), routes[count-1])
			})
			b.Run("miss", func(b *testing.B) {
				benchmarkMatch(b, router, mux.GET, "/items/item-999-blue", nil)
			})
		})
	}
}

func BenchmarkMatchBacktracking(b *testing.B) {
	router := mux.New()
	router.Get("/items/42/edit", handler)
	router.Get("/items/<<regex:code:[0-9]+>>/view", handler)
	router.Get("/items/<<int:id>>/edit", handler)
	variable := router.Get("/items/<<slug>>/show", handler)
	glob := router.Get("/items/*", handler)
	for _, tt := range []struct {
		name, method, path string
		route              *mux.Route
	}{
		{"capture-after-three-dead-ends", mux.GET, "/items/42/show", variable},
		{"glob-after-four-dead-ends", mux.GET, "/items/42/unknown", glob},
		{"wrong-method", mux.POST, "/items/42/edit", nil},
		{"extra-segment", mux.POST, "/items/42/edit/extra", nil},
	} {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkMatch(b, router, tt.method, tt.path, tt.route)
		})
	}
}

func BenchmarkMatchMethods(b *testing.B) {
	for _, count := range []int{1, 8, 32, 128} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			router := mux.New()
			var last *mux.Route
			for i := range count {
				last = router.Handle("METHOD-"+strconv.Itoa(i), "/endpoint", handler)
			}
			b.Run("last", func(b *testing.B) {
				benchmarkMatch(b, router, last.Method, "/endpoint", last)
			})
			b.Run("miss", func(b *testing.B) {
				benchmarkMatch(b, router, mux.GET, "/endpoint", nil)
			})
		})
	}
}

func BenchmarkMatchScopedRoute(b *testing.B) {
	router := mux.New()
	parent := router.Get("/accounts/<<int:account>>", nil)
	route := parent.Get("/projects/<<project>>/settings", handler)
	router.Post("/accounts/<<id>>/projects/<<id>>/settings", handler)
	for _, tt := range []struct {
		name, method string
		want         *mux.Route
	}{
		{"hit", mux.GET, route},
		{"miss", mux.POST, nil},
	} {
		b.Run(tt.name, func(b *testing.B) {
			const path = "/accounts/123/projects/alpha/settings"
			if got, _, _ := parent.Match(tt.method, path); got != tt.want {
				b.Fatalf("got %p, want %p", got, tt.want)
			}
			b.ReportAllocs()
			for b.Loop() {
				parent.Match(tt.method, path)
			}
		})
	}
}

func BenchmarkMatchMixedTraffic(b *testing.B) {
	router := mux.New()
	status := router.Get("/health", handler)
	user := router.Get("/users/<<int:id>>", handler)
	avatar := router.Get("/users/<<int:id>>/avatar", handler)
	search := router.Post("/search", handler)
	asset := router.Any("/assets/*", handler)
	requests := [...]struct {
		method, path string
		route        *mux.Route
	}{
		{mux.GET, "/health", status},
		{mux.GET, "/users/42", user},
		{mux.GET, "/users/42/avatar", avatar},
		{mux.POST, "/search", search},
		{mux.GET, "/assets/icons/add.svg", asset},
		{mux.GET, "/health", status},
		{mux.GET, "/missing", nil},
		{mux.POST, "/users/42", nil},
	}
	for _, req := range requests {
		if got, _ := router.Match(req.method, req.path); got != req.route {
			b.Fatalf("%s %s: got %p, want %p", req.method, req.path, got, req.route)
		}
	}
	b.Run("serial", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			req := requests[i]
			router.Match(req.method, req.path)
			i = (i + 1) % len(requests)
		}
	})
	b.Run("parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				req := requests[i]
				router.Match(req.method, req.path)
				i = (i + 1) % len(requests)
			}
		})
	})
}

func BenchmarkMatchWorkingSet(b *testing.B) {
	for _, teams := range []int{100, 10_000} {
		b.Run(strconv.Itoa(teams)+"-teams", func(b *testing.B) {
			type teamRoutes struct {
				prefix             string
				settings, projects *mux.Route
			}
			router := mux.New()
			asset := router.Get("/assets/*", handler)
			routes := make([]teamRoutes, teams)
			for i := range routes {
				prefix := "/teams/" + strconv.Itoa(i)
				routes[i] = teamRoutes{
					prefix:   prefix,
					settings: router.Get(prefix+"/settings", handler),
					projects: router.Get(prefix+"/projects/<<int:id>>", handler),
				}
			}
			type request struct {
				path string
				want *mux.Route
			}
			random := rand.New(rand.NewPCG(1234, 5678))
			requests := make([]request, 4_096)
			for i := range requests {
				team := routes[random.IntN(teams)]
				switch random.IntN(8) {
				case 0:
					requests[i] = request{team.prefix + "/missing", nil}
				case 1:
					requests[i] = request{"/assets/icons/add.svg", asset}
				case 2, 3:
					requests[i] = request{team.prefix + "/projects/42", team.projects}
				default:
					requests[i] = request{team.prefix + "/settings", team.settings}
				}
			}
			for _, req := range requests {
				if got, _ := router.Match(mux.GET, req.path); got != req.want {
					b.Fatalf("%s: got %p, want %p", req.path, got, req.want)
				}
			}
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				router.Match(mux.GET, requests[i].path)
				i = (i + 1) % len(requests)
			}
		})
	}
}

func BenchmarkRegisterRoutes(b *testing.B) {
	for _, tt := range []struct {
		name   string
		count  int
		suffix string
	}{
		{"literal-100", 100, ""},
		{"literal-10000", 10_000, ""},
		{"variable-100", 100, "/<<id>>"},
		{"variable-10000", 10_000, "/<<id>>"},
		{"regex-100", 100, "/<<regex:id:[a-z]+-[0-9]+>>"},
	} {
		b.Run(tt.name, func(b *testing.B) {
			patterns := make([]string, tt.count)
			for i := range patterns {
				patterns[i] = "/items/" + strconv.Itoa(i) + tt.suffix
			}
			b.ReportAllocs()
			for b.Loop() {
				router := mux.New()
				for _, pattern := range patterns {
					router.Get(pattern, handler)
				}
			}
		})
	}
}

func BenchmarkRegisterSubtree(b *testing.B) {
	for _, count := range []int{1, 16, 128} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			patterns := make([]string, count)
			for i := range patterns {
				patterns[i] = "/projects/" + strconv.Itoa(i) + "/settings"
			}
			b.ReportAllocs()
			for b.Loop() {
				parent := mux.NewRoute(mux.GET, "/accounts/<<int:id>>", nil)
				for _, pattern := range patterns {
					parent.Get(pattern, handler)
				}
				mux.New().AddRoute(parent)
			}
		})
	}
}
