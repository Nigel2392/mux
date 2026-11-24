//go:build !js && !wasm

package mux_test

import (
	"maps"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	muxv1 "github.com/Nigel2392/mux/v1"
	muxv2 "github.com/Nigel2392/mux/v2"
)

type comparisonRoute struct {
	method, pattern string
	parent          int // 1-based route index, or zero for the mux
}

type comparisonRequest struct {
	method, path string
	route        int // 1-based route index, or zero for a miss
	vars         map[string][]string
}

type comparisonTraffic struct {
	name     string
	requests []comparisonRequest
}

func comparisonRouteName(route int) string {
	if route == 0 {
		return ""
	}
	return strconv.Itoa(route)
}

func buildComparisonV1(routes []comparisonRoute, names []string) *muxv1.Mux {
	router := muxv1.New()
	registered := make([]*muxv1.Route, len(routes))
	for i, route := range routes {
		if route.parent == 0 {
			registered[i] = router.Handle(route.method, route.pattern, handler, names[i])
		} else {
			registered[i] = registered[route.parent-1].Handle(route.method, route.pattern, handler, names[i])
		}
	}
	return router
}

func buildComparisonV2(routes []comparisonRoute, names []string) *muxv2.Mux {
	router := muxv2.New()
	registered := make([]*muxv2.Route, len(routes))
	for i, route := range routes {
		if route.parent == 0 {
			registered[i] = router.Handle(route.method, route.pattern, handler, names[i])
		} else {
			registered[i] = registered[route.parent-1].Handle(route.method, route.pattern, handler, names[i])
		}
	}
	return router
}

func comparisonRouteNames(routes []comparisonRoute) []string {
	names := make([]string, len(routes))
	for i := range names {
		names[i] = comparisonRouteName(i + 1)
	}
	return names
}

func checkComparisonMatch(b *testing.B, request comparisonRequest, name string, vars map[string][]string) {
	b.Helper()
	if want := comparisonRouteName(request.route); name != want || !maps.EqualFunc(vars, request.vars, slices.Equal) {
		b.Fatalf("%s %s matched route %q, captures %v, want route %q, captures %v",
			request.method, request.path, name, vars, want, request.vars)
	}
}

func compareMatch(b *testing.B, routes []comparisonRoute, traffic ...comparisonTraffic) {
	names := comparisonRouteNames(routes)
	b.Run("version=v1", func(b *testing.B) {
		router := buildComparisonV1(routes, names)
		for _, scenario := range traffic {
			b.Run(scenario.name, func(b *testing.B) {
				for _, request := range scenario.requests {
					route, vars := router.Match(request.method, request.path)
					name := ""
					if route != nil {
						name = route.Name
					}
					checkComparisonMatch(b, request, name, vars)
				}
				b.ReportAllocs()
				i := 0
				for b.Loop() {
					request := scenario.requests[i]
					router.Match(request.method, request.path)
					i++
					if i == len(scenario.requests) {
						i = 0
					}
				}
			})
		}
	})
	b.Run("version=v2", func(b *testing.B) {
		router := buildComparisonV2(routes, names)
		for _, scenario := range traffic {
			b.Run(scenario.name, func(b *testing.B) {
				for _, request := range scenario.requests {
					route, vars := router.Match(request.method, request.path)
					name := ""
					if route != nil {
						name = route.Name
					}
					checkComparisonMatch(b, request, name, vars)
				}
				b.ReportAllocs()
				i := 0
				for b.Loop() {
					request := scenario.requests[i]
					router.Match(request.method, request.path)
					i++
					if i == len(scenario.requests) {
						i = 0
					}
				}
			})
		}
	})
}

func comparisonGET(name, path string, route int, vars map[string][]string) comparisonTraffic {
	return comparisonTraffic{name, []comparisonRequest{{http.MethodGet, path, route, vars}}}
}

func comparisonFanout(count int, suffix string) []comparisonRoute {
	routes := make([]comparisonRoute, count)
	for i := range routes {
		routes[i] = comparisonRoute{http.MethodGet, "/items/" + strconv.Itoa(i) + suffix, 0}
	}
	return routes
}

// BenchmarkCompareMatch uses only route syntax shared by both versions.
// Each pair uses the same registration order, paths, methods and captures.
func BenchmarkCompareMatch(b *testing.B) {
	b.Run("root", func(b *testing.B) {
		compareMatch(b, []comparisonRoute{{http.MethodGet, "/", 0}}, comparisonGET("hit", "/", 1, nil))
	})
	for _, count := range []int{1, 100, 10_000, 100_000} {
		b.Run("literal-fanout/routes="+strconv.Itoa(count), func(b *testing.B) {
			routes := comparisonFanout(count, "")
			traffic := []comparisonTraffic{
				comparisonGET("first-hit", routes[0].pattern, 1, nil),
				comparisonGET("miss", "/items/missing", 0, nil),
			}
			if count > 1 {
				traffic = append(traffic, comparisonGET("last-hit", routes[count-1].pattern, count, nil))
			}
			compareMatch(b, routes, traffic...)
		})
	}
	for _, depth := range []int{1, 4, 16, 64} {
		b.Run("literal-depth/segments="+strconv.Itoa(depth), func(b *testing.B) {
			prefix := strings.Repeat("/segment", depth-1)
			path := prefix + "/leaf"
			compareMatch(b, []comparisonRoute{{http.MethodGet, path, 0}},
				comparisonGET("hit", path, 1, nil),
				comparisonGET("late-miss", prefix+"/missing", 0, nil))
		})
	}
	b.Run("captures", func(b *testing.B) {
		routes := []comparisonRoute{
			{http.MethodGet, "/items/<<id>>", 0},
			{http.MethodGet, "/distinct/<<org>>/<<team>>/<<repo>>/<<file>>", 0},
			{http.MethodGet, "/repeated/<<id>>/<<id>>/<<id>>/<<id>>", 0},
		}
		compareMatch(b, routes,
			comparisonGET("one-hit", "/items/42", 1, map[string][]string{"id": {"42"}}),
			comparisonGET("one-miss", "/items/42/extra", 0, nil),
			comparisonGET("distinct-hit", "/distinct/one/two/three/four", 2,
				map[string][]string{"org": {"one"}, "team": {"two"}, "repo": {"three"}, "file": {"four"}}),
			comparisonGET("repeated-hit", "/repeated/one/two/three/four", 3,
				map[string][]string{"id": {"one", "two", "three", "four"}}))
	})
	b.Run("nested", func(b *testing.B) {
		compareMatch(b, comparisonNestedRoutes(),
			comparisonGET("parent-hit", "/orgs/acme", 1, map[string][]string{"org": {"acme"}}),
			comparisonGET("deep-hit", "/orgs/acme/teams/blue/repos/demo/files/css/app.css", 4,
				map[string][]string{"org": {"acme"}, "team": {"blue"}, "repo": {"demo"}, "*": {"css", "app.css"}}),
			comparisonGET("deep-miss", "/orgs/acme/teams/blue/repos/demo/missing", 0, nil))
	})
	b.Run("glob", func(b *testing.B) {
		longTail := strings.Repeat("/asset", 64)
		compareMatch(b, []comparisonRoute{{http.MethodGet, "/assets/*", 0}},
			comparisonGET("empty", "/assets", 1, map[string][]string{"*": nil}),
			comparisonGET("short", "/assets/icons/add.svg", 1, map[string][]string{"*": {"icons", "add.svg"}}),
			comparisonGET("long", "/assets"+longTail, 1, map[string][]string{"*": strings.Split(longTail[1:], "/")}))
	})
	b.Run("fallback", func(b *testing.B) {
		routes := []comparisonRoute{
			{http.MethodGet, "/items/new", 0},
			{http.MethodGet, "/items/<<id>>/view", 0},
			{http.MethodGet, "/items/*", 0},
		}
		compareMatch(b, routes,
			comparisonGET("literal", "/items/new", 1, nil),
			comparisonGET("variable", "/items/new/view", 2, map[string][]string{"id": {"new"}}),
			comparisonGET("glob-after-variable-dead-end", "/items/new/edit", 3, map[string][]string{"*": {"new", "edit"}}))
	})
	for _, count := range []int{1, 8, 32, 128} {
		b.Run("methods/count="+strconv.Itoa(count), func(b *testing.B) {
			routes := make([]comparisonRoute, count)
			for i := range routes {
				routes[i] = comparisonRoute{"METHOD-" + strconv.Itoa(i), "/endpoint", 0}
			}
			compareMatch(b, routes, comparisonTraffic{"last-hit", []comparisonRequest{
				{routes[count-1].method, "/endpoint", count, nil},
			}}, comparisonGET("miss", "/endpoint", 0, nil))
		})
	}
	for _, teams := range []int{100, 10_000} {
		b.Run("mixed/teams="+strconv.Itoa(teams), func(b *testing.B) {
			routes, traffic := comparisonMixedTraffic(teams)
			compareMatch(b, routes, traffic)
		})
	}
}

func comparisonNestedRoutes() []comparisonRoute {
	return []comparisonRoute{
		{http.MethodGet, "/orgs/<<org>>", 0},
		{http.MethodGet, "/teams/<<team>>", 1},
		{http.MethodGet, "/repos/<<repo>>", 2},
		{http.MethodGet, "/files/*", 3},
		{http.MethodPost, "/actions/<<action>>", 3},
	}
}

func comparisonMixedTraffic(teams int) ([]comparisonRoute, comparisonTraffic) {
	routes := make([]comparisonRoute, 0, 2*teams+2)
	routes = append(routes, comparisonRoute{http.MethodGet, "/health", 0}, comparisonRoute{http.MethodGet, "/assets/*", 0})
	for i := range teams {
		prefix := "/teams/" + strconv.Itoa(i)
		routes = append(routes,
			comparisonRoute{http.MethodGet, prefix + "/settings", 0},
			comparisonRoute{http.MethodPost, prefix + "/projects/<<project>>", 0})
	}

	random := rand.New(rand.NewPCG(1234, 5678))
	requests := make([]comparisonRequest, 256)
	for i := range requests {
		team := random.IntN(teams)
		settings, project := 2+2*team, 3+2*team
		prefix := "/teams/" + strconv.Itoa(team)
		switch random.IntN(8) {
		case 0:
			requests[i] = comparisonRequest{http.MethodGet, prefix + "/missing", 0, nil}
		case 1:
			requests[i] = comparisonRequest{http.MethodGet, "/assets/icons/add.svg", 2, map[string][]string{"*": {"icons", "add.svg"}}}
		case 2:
			requests[i] = comparisonRequest{http.MethodGet, "/health", 1, nil}
		case 3, 4:
			requests[i] = comparisonRequest{http.MethodPost, prefix + "/projects/alpha", project + 1, map[string][]string{"project": {"alpha"}}}
		default:
			requests[i] = comparisonRequest{http.MethodGet, routes[settings].pattern, settings + 1, nil}
		}
	}
	return routes, comparisonTraffic{"request-stream", requests}
}

// BenchmarkCompareRegister measures building a fresh mux through Handle,
// including each version's route construction and ID generation.
func BenchmarkCompareRegister(b *testing.B) {
	for _, scenario := range []struct {
		name   string
		routes []comparisonRoute
	}{
		{"literal/routes=100", comparisonFanout(100, "")},
		{"literal/routes=10000", comparisonFanout(10_000, "")},
		{"variable/routes=10000", comparisonFanout(10_000, "/<<id>>")},
		{"nested/routes=5", comparisonNestedRoutes()},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			names := comparisonRouteNames(scenario.routes)
			b.Run("version=v1", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					buildComparisonV1(scenario.routes, names)
				}
			})
			b.Run("version=v2", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					buildComparisonV2(scenario.routes, names)
				}
			})
		})
	}
}
