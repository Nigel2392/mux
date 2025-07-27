package mux

import "context"

type routeContextKey struct{}

func ContextWithRoute(ctx context.Context, route *Route) context.Context {
	return context.WithValue(ctx, routeContextKey{}, route)
}

func RouteFromContext(ctx context.Context) *Route {
	route, ok := ctx.Value(routeContextKey{}).(*Route)
	if !ok {
		return nil
	}
	return route
}
