package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/zapi-api/zapi/internal/model"
	"github.com/zapi-api/zapi/internal/store"
)

type contextKey string

const subscriberKey contextKey = "subscriber"

type AuthMiddleware struct {
	store store.Store
	mu    sync.RWMutex
	cache map[string]*model.Subscriber
}

func NewAuthMiddleware(s store.Store) *AuthMiddleware {
	return &AuthMiddleware{
		store: s,
		cache: make(map[string]*model.Subscriber),
	}
}

func (a *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			writeError(w, http.StatusUnauthorized, "missing API key")
			return
		}

		sub, err := a.lookup(r.Context(), apiKey)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		ctx := context.WithValue(r.Context(), subscriberKey, sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *AuthMiddleware) lookup(ctx context.Context, apiKey string) (*model.Subscriber, error) {
	a.mu.RLock()
	if sub, ok := a.cache[apiKey]; ok {
		a.mu.RUnlock()
		return sub, nil
	}
	a.mu.RUnlock()

	sub, err := a.store.GetSubscriberByKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.cache[apiKey] = sub
	a.mu.Unlock()

	return sub, nil
}
