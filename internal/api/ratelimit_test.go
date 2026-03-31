package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zapi-api/zapi/internal/cache"
	"github.com/zapi-api/zapi/internal/model"
)

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	defer rl.Close()
	for i := 0; i < 5; i++ {
		if !rl.allow("test-key") {
			t.Errorf("request %d should be allowed within burst", i)
		}
	}
}

func TestRateLimiter_BlocksAfterBurst(t *testing.T) {
	rl := NewRateLimiter(10, 3)
	defer rl.Close()
	for i := 0; i < 3; i++ {
		rl.allow("test-key")
	}
	if rl.allow("test-key") {
		t.Error("should be blocked after burst exhausted")
	}
}

func TestRateLimiter_IndependentKeys(t *testing.T) {
	rl := NewRateLimiter(10, 2)
	defer rl.Close()
	rl.allow("key-a")
	rl.allow("key-a")
	// key-a exhausted

	if !rl.allow("key-b") {
		t.Error("key-b should not be affected by key-a limits")
	}
}

func TestRateLimiter_Close(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	rl.Close() // should not panic or block
}

func TestRateLimiter_Returns429(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	// Exhaust rate limit (burst is 20)
	for i := 0; i < 25; i++ {
		req := httptest.NewRequest("GET", "/v1/rates?pair=USDT_NGN", nil)
		req.Header.Set("X-API-Key", "test-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Next request should be 429
	req := httptest.NewRequest("GET", "/v1/rates?pair=USDT_NGN", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}
