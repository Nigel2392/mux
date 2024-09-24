package mux

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrRouteNotFound    = Error("route not found")
	ErrTooManyVariables = Error("too many variables provided to replace in path")
)
