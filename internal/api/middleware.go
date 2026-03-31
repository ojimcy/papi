package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/zapi-api/zapi/internal/model"
	"github.com/zapi-api/zapi/internal/store"
)

type cacheEntry struct {
	subscriber *model.Subscriber
	cachedAt   time.Time
}

type contextKey string

const subscriberKey contextKey = "subscriber"

type AuthMiddleware struct {
	store store.Store
	mu    sync.RWMutex
	cache map[string]cacheEntry
	ttl   time.Duration
}

func NewAuthMiddleware(s store.Store) *AuthMiddleware {
	return &AuthMiddleware{
		store: s,
		cache: make(map[string]cacheEntry),
		ttl:   5 * time.Minute,
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
	if entry, ok := a.cache[apiKey]; ok && time.Since(entry.cachedAt) < a.ttl {
		a.mu.RUnlock()
		return entry.subscriber, nil
	}
	a.mu.RUnlock()

	sub, err := a.store.GetSubscriberByKey(ctx, apiKey)
	if err != nil {
		a.mu.Lock()
		delete(a.cache, apiKey)
		a.mu.Unlock()
		return nil, err
	}

	a.mu.Lock()
	a.cache[apiKey] = cacheEntry{subscriber: sub, cachedAt: time.Now()}
	a.mu.Unlock()

	return sub, nil
}
