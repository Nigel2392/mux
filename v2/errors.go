package mux

import "errors"

type Error string

func (e Error) Error() string { return string(e) }

const ErrRouteNotFound = Error("route not found")

var (
	ErrMatcherNotFound    = errors.New("matcher not found")
	ErrInvalidMatcherArg  = errors.New("invalid matcher argument")
	ErrNotEnoughVariables = errors.New("not enough variables")
	ErrTooManyVariables   = errors.New("too many variables")
	ErrTerminalNotLast    = errors.New("terminal matcher must be the last segment")
	ErrParentIsTerminal   = errors.New("parent node ends with a terminal matcher")
	ErrHasChildren        = errors.New("node with children cannot end with a terminal matcher")
	ErrCyclicParent       = errors.New("cannot set parent to self or descendant")
	ErrDetachedNode       = errors.New("node is not attached to a tree")
	ErrRouteAttached      = errors.New("route already has a parent or mux")
)
