package mux

import "net/http"

func (m *Mux) HandleFunc(method, path string, handler func(http.ResponseWriter, *http.Request), name ...string) *Route {
	return m.Handle(method, path, http.HandlerFunc(handler), name...)
}

func (m *Mux) Get(path string, handler http.Handler, name ...string) *Route {
	return m.Handle(GET, path, handler, name...)
}

func (m *Mux) Post(path string, handler http.Handler, name ...string) *Route {
	return m.Handle(POST, path, handler, name...)
}

func (m *Mux) Put(path string, handler http.Handler, name ...string) *Route {
	return m.Handle(PUT, path, handler, name...)
}

func (m *Mux) Delete(path string, handler http.Handler, name ...string) *Route {
	return m.Handle(DELETE, path, handler, name...)
}

func (m *Mux) Head(path string, handler http.Handler, name ...string) *Route {
	return m.Handle(HEAD, path, handler, name...)
}

func (m *Mux) Patch(path string, handler http.Handler, name ...string) *Route {
	return m.Handle(PATCH, path, handler, name...)
}

func (m *Mux) Options(path string, handler http.Handler, name ...string) *Route {
	return m.Handle(OPTIONS, path, handler, name...)
}

func (m *Mux) Any(path string, handler http.Handler, name ...string) *Route {
	return m.Handle(ANY, path, handler, name...)
}

func (r *Route) HandleFunc(method, path string, handler func(http.ResponseWriter, *http.Request), name ...string) *Route {
	return r.Handle(method, path, http.HandlerFunc(handler), name...)
}

func (r *Route) Get(path string, handler http.Handler, name ...string) *Route {
	return r.Handle(GET, path, handler, name...)
}

func (r *Route) Post(path string, handler http.Handler, name ...string) *Route {
	return r.Handle(POST, path, handler, name...)
}

func (r *Route) Put(path string, handler http.Handler, name ...string) *Route {
	return r.Handle(PUT, path, handler, name...)
}

func (r *Route) Delete(path string, handler http.Handler, name ...string) *Route {
	return r.Handle(DELETE, path, handler, name...)
}

func (r *Route) Head(path string, handler http.Handler, name ...string) *Route {
	return r.Handle(HEAD, path, handler, name...)
}

func (r *Route) Patch(path string, handler http.Handler, name ...string) *Route {
	return r.Handle(PATCH, path, handler, name...)
}

func (r *Route) Options(path string, handler http.Handler, name ...string) *Route {
	return r.Handle(OPTIONS, path, handler, name...)
}

func (r *Route) Any(path string, handler http.Handler, name ...string) *Route {
	return r.Handle(ANY, path, handler, name...)
}
