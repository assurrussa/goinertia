package main

import (
	"context"
	"crypto/rand"
	"maps"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v3"
)

const sessionCookie = "goinertia_browser_session"

type sessionKey struct{}

// sessions is fixture-only process memory. Each server and each cookie owns
// distinct data; consume-once reads and writes are atomic across requests.
// The bounded browser suite discards all sessions when its server exits.
type sessions struct {
	mu     sync.Mutex
	values map[string]map[string]any
}

func newSessions() *sessions { return &sessions{values: make(map[string]map[string]any)} }

func (s *sessions) identify(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.values[id]; ok && id != "" {
		return id, false
	}
	id = rand.Text()
	s.values[id] = make(map[string]any)
	return id, true
}

func (s *sessions) get(id, key string, consume bool) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	value := cloneValue(s.values[id][key])
	if consume {
		delete(s.values[id], key)
	}
	return value
}

func (s *sessions) set(id, key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[id][key] = cloneValue(value)
}

func (s *sessions) remove(id, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values[id], key)
}

// Clone the map types used by fixture props so callbacks cannot share mutable
// session values with another in-flight request. Leaf values are immutable.
func cloneValue(value any) any {
	switch data := value.(type) {
	case map[string]string:
		return maps.Clone(data)
	case map[string]any:
		cloned := make(map[string]any, len(data))
		for key, value := range data {
			cloned[key] = cloneValue(value)
		}
		return cloned
	default:
		return value
	}
}

type fiberSessions struct{ *sessions }

func (s fiberSessions) middleware(c fiber.Ctx) error {
	id, created := s.identify(c.Cookies(sessionCookie))
	if created {
		c.Cookie(&fiber.Cookie{Name: sessionCookie, Value: id, Path: "/", HTTPOnly: true, SameSite: "Lax"})
	}
	c.Locals(sessionKey{}, id)
	return c.Next()
}

func fiberSessionID(c fiber.Ctx) string {
	id, _ := c.Locals(sessionKey{}).(string)
	return id
}

func (s fiberSessions) Get(c fiber.Ctx, key string) (any, error) {
	return s.get(fiberSessionID(c), key, false), nil
}

func (s fiberSessions) Set(c fiber.Ctx, key string, value any) error {
	s.set(fiberSessionID(c), key, value)
	return nil
}

func (s fiberSessions) Delete(c fiber.Ctx, key string) error {
	s.remove(fiberSessionID(c), key)
	return nil
}

func (s fiberSessions) Flash(c fiber.Ctx, key string, value any) error { return s.Set(c, key, value) }

func (s fiberSessions) GetFlash(c fiber.Ctx, key string) (any, error) {
	return s.get(fiberSessionID(c), key, true), nil
}

type httpSessions struct{ *sessions }

func (s httpSessions) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var supplied string
		if cookie, err := r.Cookie(sessionCookie); err == nil {
			supplied = cookie.Value
		}
		id, created := s.identify(supplied)
		if created {
			//nolint:gosec // The fixture serves plain HTTP exclusively on loopback.
			http.SetCookie(w, &http.Cookie{
				Name: sessionCookie, Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
			})
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, id)))
	})
}

func httpSessionID(r *http.Request) string {
	id, _ := r.Context().Value(sessionKey{}).(string)
	return id
}

func (s httpSessions) Flash(_ http.ResponseWriter, r *http.Request, key string, value any) error {
	s.set(httpSessionID(r), key, value)
	return nil
}

func (s httpSessions) GetFlash(_ http.ResponseWriter, r *http.Request, key string) (any, error) {
	return s.get(httpSessionID(r), key, true), nil
}
