package mux

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrRouteNotFound = Error("route not found")
)
