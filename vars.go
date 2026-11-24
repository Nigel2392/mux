package mux

import "strconv"

type Variables map[string][]string

func (v Variables) Get(key string) string {
	value, _ := v.GetOK(key)
	return value
}

func (v Variables) GetOK(key string) (string, bool) {
	values := v[key]
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

func (v Variables) GetAll(key string) []string { return v[key] }

func (v Variables) GetAllOK(key string) ([]string, bool) {
	values, ok := v[key]
	return values, ok
}

func (v Variables) GetInt(key string) int {
	number, _ := strconv.Atoi(v.Get(key))
	return number
}

func (v Variables) Len() int { return len(v) }

func (v Variables) Set(key, value string) bool {
	_, exists := v[key]
	v[key] = append(v[key], value)
	return !exists
}

func makeVariables(route *Route, values []string, offset int) Variables {
	if len(values) == 0 {
		return nil
	}
	vars := make(Variables, len(values))
	names := route.names[offset:]
	parts := route.Path.Path
	for i, value := range values {
		if i == len(names)-1 && len(parts) != 0 && parts[len(parts)-1].IsGlob &&
			!parts[len(parts)-1].IsVariable && route.segments[len(route.segments)-1].Flags().Has(M_CAPTURE) {
			// The v1 bare glob exposes the remaining segments under its own key.
			vars[names[i]] = route.Path.splitRemainder(value)
		} else {
			vars.Set(names[i], value)
		}
	}
	return vars
}
