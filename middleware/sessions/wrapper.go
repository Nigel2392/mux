//go:build !js && !wasm
// +build !js,!wasm

package sessions

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
)

var _ Session = (*scsRequestSession)(nil)

type scsRequestSession struct {
	r          *http.Request
	store      *scs.SessionManager
	finalizers []func(r *http.Request, ctx context.Context) (context.Context, error)
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

func (s *scsRequestSession) AddFinalizer(finalizer func(r *http.Request, ctx context.Context) (context.Context, error)) {
	s.finalizers = append(s.finalizers, finalizer)
}

func (s *scsRequestSession) Store() *scs.SessionManager {
	return s.store
}
