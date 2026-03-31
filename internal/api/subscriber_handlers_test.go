package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zapi-api/zapi/internal/cache"
	"github.com/zapi-api/zapi/internal/model"
)

func TestSubscribers_AdminOnly(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "user-key", IsActive: true, IsAdmin: false}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/subscribers", nil)
	req.Header.Set("X-API-Key", "user-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}
}

func TestSubscribers_AdminAllowed(t *testing.T) {
	s := &testStore{
		subscriber:  &model.Subscriber{APIKey: "admin-key", IsActive: true, IsAdmin: true},
		subscribers: []model.Subscriber{{ID: "1", Name: "App1"}},
	}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/subscribers", nil)
	req.Header.Set("X-API-Key", "admin-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSubscriber(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "admin-key", IsActive: true, IsAdmin: true}}
	router := NewRouter(cache.New(), s)

	body := `{"name":"Test App","webhook_url":"https://example.com/hook"}`
	req := httptest.NewRequest("POST", "/v1/subscribers", strings.NewReader(body))
	req.Header.Set("X-API-Key", "admin-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.Subscriber
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "Test App" {
		t.Errorf("expected name 'Test App', got %q", resp.Name)
	}
	if resp.APIKey == "" {
		t.Error("expected API key to be generated")
	}
}

func TestCreateSubscriber_MissingName(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "admin-key", IsAdmin: true}}
	router := NewRouter(cache.New(), s)

	body := `{"webhook_url":"https://example.com/hook"}`
	req := httptest.NewRequest("POST", "/v1/subscribers", strings.NewReader(body))
	req.Header.Set("X-API-Key", "admin-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListSubscribers(t *testing.T) {
	s := &testStore{
		subscriber: &model.Subscriber{APIKey: "admin-key", IsAdmin: true},
		subscribers: []model.Subscriber{
			{ID: "1", Name: "App1", APIKey: "key1", IsActive: true},
		},
	}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("GET", "/v1/subscribers", nil)
	req.Header.Set("X-API-Key", "admin-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []model.Subscriber
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 subscriber, got %d", len(resp))
	}
}

func TestUpdateSubscriber(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "admin-key", IsAdmin: true}}
	router := NewRouter(cache.New(), s)

	body := `{"name":"Updated App","webhook_url":"https://new.example.com/hook"}`
	req := httptest.NewRequest("PUT", "/v1/subscribers/some-id", strings.NewReader(body))
	req.Header.Set("X-API-Key", "admin-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSubscriber_MissingName(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "admin-key", IsAdmin: true}}
	router := NewRouter(cache.New(), s)

	body := `{"webhook_url":"https://example.com/hook"}`
	req := httptest.NewRequest("PUT", "/v1/subscribers/some-id", strings.NewReader(body))
	req.Header.Set("X-API-Key", "admin-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeactivateSubscriber(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "admin-key", IsAdmin: true}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("DELETE", "/v1/subscribers/some-id", nil)
	req.Header.Set("X-API-Key", "admin-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSubscriber_InvalidBody(t *testing.T) {
	s := &testStore{subscriber: &model.Subscriber{APIKey: "admin-key", IsAdmin: true}}
	router := NewRouter(cache.New(), s)

	req := httptest.NewRequest("POST", "/v1/subscribers", strings.NewReader("not json"))
	req.Header.Set("X-API-Key", "admin-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
