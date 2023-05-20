package mux

import (
	"context"
	"net/http"
	"strconv"
)

var variablesKey = "mux.Variables"

type Variables map[string][]string

func SetVariables(r *http.Request, v Variables) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), variablesKey, v))
}

func Vars(r *http.Request) Variables {
	var v = r.Context().Value(variablesKey)
	if v == nil {
		return nil
	}
	return v.(Variables)
}

func (v Variables) Get(key string) string {
	if v == nil {
		return ""
	}
	if len(v[key]) == 0 {
		return ""
	}
	return v[key][0]
}

func (v Variables) GetAll(key string) []string {
	if v == nil {
		return nil
	}
	return v[key]
}

func (v Variables) GetInt(key string) int {
	var i, err = strconv.Atoi(v.Get(key))
	if err != nil {
		return 0
	}
	return i
}
