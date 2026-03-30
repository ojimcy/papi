package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zapi-api/zapi/internal/cache"
	"github.com/zapi-api/zapi/internal/model"
	"github.com/zapi-api/zapi/internal/store"
)

type testStore struct {
	subscriber *model.Subscriber
}

func (m *testStore) SaveRate(_ context.Context, _ model.ComputedRate) error { return nil }
func (m *testStore) GetHistory(_ context.Context, _ string, _ time.Time, _ string) ([]model.ComputedRate, error) {
	return nil, nil
}
func (m *testStore) GetSubscribers(_ context.Context) ([]model.Subscriber, error) { return nil, nil }
func (m *testStore) GetSubscriberByKey(_ context.Context, apiKey string) (*model.Subscriber, error) {
	if m.subscriber != nil && m.subscriber.APIKey == apiKey {
		return m.subscriber, nil
	}
	return nil, fmt.Errorf("not found")
}
func (m *testStore) Close() error { return nil }

var _ store.Store = (*testStore)(nil)

func TestGetRates_Aggregate(t *testing.T) {
	c := cache.New()
	c.Set("USDT_NGN", "aggregate", model.ComputedRate{
		Pair: "USDT_NGN", AmountTier: "aggregate",
		BuyRate: 1416, SellRate: 1412, MidRate: 1414,
		Source: "bybit", Timestamp: time.Now(),
	})

	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key", IsActive: true}}
	router := NewRouter(c, s)

	req := httptest.NewRequest("GET", "/v1/rates?pair=USDT_NGN", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var rate model.ComputedRate
	json.NewDecoder(w.Body).Decode(&rate)
	if rate.BuyRate != 1416 {
		t.Errorf("expected 1416, got %f", rate.BuyRate)
	}
}

func TestGetRates_WithAmount(t *testing.T) {
	c := cache.New()
	c.Set("USDT_NGN", "500000", model.ComputedRate{
		Pair: "USDT_NGN", AmountTier: "500000",
		BuyRate: 1416.38, Source: "bybit", Timestamp: time.Now(),
	})

	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(c, s)

	req := httptest.NewRequest("GET", "/v1/rates?pair=USDT_NGN&amount=500000", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetRates_MissingPair(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetRates_Unauthorized(t *testing.T) {
	router := NewRouter(cache.New(), &testStore{})

	req := httptest.NewRequest("GET", "/v1/rates?pair=USDT_NGN", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetRates_NotFound(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates?pair=USDT_GHS", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
