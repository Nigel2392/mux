package mux

import (
	"strconv"
)

type Variables map[string][]string

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
