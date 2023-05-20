package sessions

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
)

type scsRequestSession struct {
	r     *http.Request
	store *scs.SessionManager
}

func (s *scsRequestSession) Get(key string) interface{} {
	return s.store.Get(s.r.Context(), key)
}

func (s *scsRequestSession) Set(key string, value interface{}) {
	s.store.Put(s.r.Context(), key, value)
}

func (s *scsRequestSession) Destroy() error {
	return s.store.Destroy(s.r.Context())
}

func (s *scsRequestSession) Exists(key string) bool {
	return s.store.Exists(s.r.Context(), key)
}

func (s *scsRequestSession) Delete(key string) {
	s.store.Remove(s.r.Context(), key)
}

func (s *scsRequestSession) RenewToken() error {
	return s.store.RenewToken(s.r.Context())
}
