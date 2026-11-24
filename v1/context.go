package mux

import (
	"context"
	"net/http"
)

var (
	routeContextKey = ContextKey{"mux.route"}
	varsContextKey  = ContextKey{"mux.variables"}
)

// Exported ContextKey struct allows third party packages
// to work with the context more freely.
type ContextKey struct {
	K string
}

func ContextWithRoute(ctx context.Context, route *Route) context.Context {
	return context.WithValue(ctx, routeContextKey, route)
}

func RouteFromContext(ctx context.Context) *Route {
	route, ok := ctx.Value(routeContextKey).(*Route)
	if !ok {
		return nil
	}
	return route
}

func SetContextVars(r *http.Request, v Variables) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), varsContextKey, v))
}

func Vars(r *http.Request) Variables {
	var v = r.Context().Value(varsContextKey)
	if v == nil {
		return nil
	}
	return v.(Variables)
}
