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
	subscriber  *model.Subscriber
	subscribers []model.Subscriber
	history     []model.ComputedRate
	historyErr  error
}

func (m *testStore) SaveRate(_ context.Context, _ model.ComputedRate) error { return nil }
func (m *testStore) GetHistory(_ context.Context, _ string, _ time.Time, _ string, limit, offset int) ([]model.ComputedRate, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	h := m.history
	if offset > len(h) {
		offset = len(h)
	}
	h = h[offset:]
	if limit > len(h) {
		limit = len(h)
	}
	return h[:limit], nil
}
func (m *testStore) GetSubscribers(_ context.Context) ([]model.Subscriber, error) {
	return m.subscribers, nil
}
func (m *testStore) CreateSubscriber(_ context.Context, name, webhookURL string) (*model.Subscriber, error) {
	return &model.Subscriber{ID: "generated-id", Name: name, APIKey: "generated-key", WebhookURL: webhookURL, IsActive: true}, nil
}
func (m *testStore) UpdateSubscriber(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
func (m *testStore) DeactivateSubscriber(_ context.Context, _ string) error { return nil }
func (m *testStore) GetSubscriberByKey(_ context.Context, apiKey string) (*model.Subscriber, error) {
	if m.subscriber != nil && m.subscriber.APIKey == apiKey {
		return m.subscriber, nil
	}
	return nil, fmt.Errorf("not found")
}
func (m *testStore) Ping(_ context.Context) error { return nil }
func (m *testStore) Close() error                 { return nil }

var _ store.Store = (*testStore)(nil)

func TestGetPairs(t *testing.T) {
	c := cache.New()
	c.Set("USDT_NGN", "aggregate", model.ComputedRate{Pair: "USDT_NGN"})
	c.Set("USDT_NGN", "500000", model.ComputedRate{Pair: "USDT_NGN"})

	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(c, s)

	req := httptest.NewRequest("GET", "/v1/pairs", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(resp))
	}
	if resp[0]["pair"] != "USDT_NGN" {
		t.Errorf("expected USDT_NGN, got %v", resp[0]["pair"])
	}
	tiers, ok := resp[0]["tiers"].([]any)
	if !ok || len(tiers) != 2 {
		t.Errorf("expected 2 tiers, got %v", resp[0]["tiers"])
	}
}

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

func TestGetRates_InvalidPairFormat(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates?pair=invalid", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid pair format, got %d", w.Code)
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

func TestGetHistory_Success(t *testing.T) {
	history := []model.ComputedRate{
		{Pair: "USDT_NGN", MidRate: 1414, Timestamp: time.Now().Add(-1 * time.Hour)},
		{Pair: "USDT_NGN", MidRate: 1415, Timestamp: time.Now()},
	}
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}, history: history}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN&period=24h", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["pair"] != "USDT_NGN" {
		t.Errorf("expected pair USDT_NGN, got %v", resp["pair"])
	}
	if resp["period"] != "24h" {
		t.Errorf("expected period 24h, got %v", resp["period"])
	}
	data, ok := resp["data"].([]any)
	if !ok || len(data) != 2 {
		t.Errorf("expected 2 data points, got %v", resp["data"])
	}
}

func TestGetHistory_DefaultPeriod(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["period"] != "24h" {
		t.Errorf("expected default period 24h, got %v", resp["period"])
	}
}

func TestGetHistory_7dPeriod(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN&period=7d", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetHistory_MissingPair(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetHistory_InvalidPeriod(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN&period=99x", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetHistory_StoreError(t *testing.T) {
	s := &testStore{
		subscriber: &model.Subscriber{APIKey: "test-key"},
		historyErr: fmt.Errorf("db error"),
	}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestGetRates_InvalidAPIKey(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "real-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates?pair=USDT_NGN", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetHistory_Pagination(t *testing.T) {
	history := []model.ComputedRate{
		{Pair: "USDT_NGN", MidRate: 1413, Timestamp: time.Now().Add(-2 * time.Hour)},
		{Pair: "USDT_NGN", MidRate: 1414, Timestamp: time.Now().Add(-1 * time.Hour)},
		{Pair: "USDT_NGN", MidRate: 1415, Timestamp: time.Now()},
	}
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}, history: history}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN&limit=2&offset=1", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data, ok := resp["data"].([]any)
	if !ok || len(data) != 2 {
		t.Errorf("expected 2 data points, got %v", resp["data"])
	}
	if resp["limit"] != float64(2) {
		t.Errorf("expected limit 2, got %v", resp["limit"])
	}
	if resp["offset"] != float64(1) {
		t.Errorf("expected offset 1, got %v", resp["offset"])
	}
}

func TestGetHistory_DefaultLimit(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["limit"] != float64(100) {
		t.Errorf("expected default limit 100, got %v", resp["limit"])
	}
	if resp["offset"] != float64(0) {
		t.Errorf("expected default offset 0, got %v", resp["offset"])
	}
}

func TestGetHistory_InvalidLimit(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN&limit=-5", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHealth(t *testing.T) {
	router := NewRouter(cache.New(), &testStore{})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	router := NewRouter(cache.New(), &testStore{})

	req := httptest.NewRequest("OPTIONS", "/v1/rates", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected Access-Control-Allow-Origin header")
	}
}

func TestCORS_RegularRequest(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected Access-Control-Allow-Origin header on regular request")
	}
}

func TestAuthMiddleware_CacheExpiry(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key", IsActive: true}}
	auth := NewAuthMiddleware(s)
	auth.ttl = 50 * time.Millisecond

	handler := auth.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request caches
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Remove subscriber from store
	s.subscriber = nil

	// Immediate request still uses cache
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "test-key")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 from cache, got %d", w.Code)
	}

	// Wait for TTL
	time.Sleep(60 * time.Millisecond)

	// Now should fail
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "test-key")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401 after TTL, got %d", w.Code)
	}
}

func TestRequestID_Present(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Request-Id") == "" {
		t.Error("expected X-Request-Id header in response")
	}
}

func TestGetHistory_WithTier(t *testing.T) {
	history := []model.ComputedRate{
		{Pair: "USDT_NGN", AmountTier: "500000", MidRate: 1414, Timestamp: time.Now()},
	}
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}, history: history}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN&tier=500000", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["tier"] != "500000" {
		t.Errorf("expected tier 500000, got %v", resp["tier"])
	}
}

func TestGetHistory_DefaultTier(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "test-key"}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/rates/history?pair=USDT_NGN", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["tier"] != "aggregate" {
		t.Errorf("expected default tier aggregate, got %v", resp["tier"])
	}
}
