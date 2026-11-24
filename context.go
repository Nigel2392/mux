package mux

import (
	"context"
	"net/http"
)

// ContextKey identifies the context values used by the mux.
type ContextKey struct {
	K string
}

var (
	routeContextKey = ContextKey{"mux.route"}
	varsContextKey  = ContextKey{"mux.variables"}
)

func ContextWithRoute(ctx context.Context, route *Route) context.Context {
	return context.WithValue(ctx, routeContextKey, route)
}

func RouteFromContext(ctx context.Context) *Route {
	route, _ := ctx.Value(routeContextKey).(*Route)
	return route
}

func contextWithVars(ctx context.Context, vars Variables) context.Context {
	return context.WithValue(ctx, varsContextKey, vars)
}

func SetContextVars(r *http.Request, vars Variables) *http.Request {
	return r.WithContext(contextWithVars(r.Context(), vars))
}

func Vars(r *http.Request) Variables {
	vars, _ := r.Context().Value(varsContextKey).(Variables)
	return vars
}
