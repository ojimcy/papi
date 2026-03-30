package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zapi-api/zapi/internal/model"
)

func TestDispatcher_ShouldNotify_BelowThreshold(t *testing.T) {
	d := New(0.5, http.DefaultClient)
	old := model.ComputedRate{MidRate: 1414.00}
	new_ := model.ComputedRate{MidRate: 1414.50}

	if d.ShouldNotify(old, new_) {
		t.Error("should not notify for change below threshold")
	}
}

func TestDispatcher_ShouldNotify_AboveThreshold(t *testing.T) {
	d := New(0.5, http.DefaultClient)
	old := model.ComputedRate{MidRate: 1414.00}
	new_ := model.ComputedRate{MidRate: 1425.00}

	if !d.ShouldNotify(old, new_) {
		t.Error("should notify for change above threshold")
	}
}

func TestDispatcher_ShouldNotify_FirstRate(t *testing.T) {
	d := New(0.5, http.DefaultClient)
	if !d.ShouldNotify(model.ComputedRate{}, model.ComputedRate{MidRate: 1414}) {
		t.Error("should notify for first rate")
	}
}

func TestDispatcher_Deliver(t *testing.T) {
	var received atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Signature") == "" {
			t.Error("expected X-Signature header")
		}
		var payload WebhookPayload
		json.NewDecoder(r.Body).Decode(&payload)
		if payload.Event != "rate.updated" {
			t.Errorf("expected rate.updated, got %s", payload.Event)
		}
		received.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := New(0.5, http.DefaultClient)
	err := d.Deliver(
		model.Subscriber{APIKey: "test-key", WebhookURL: server.URL},
		model.ComputedRate{Pair: "USDT_NGN", MidRate: 1414, Timestamp: time.Now()},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !received.Load() {
		t.Error("webhook was not received")
	}
}

func TestDispatcher_Deliver_Retries(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := New(0.5, http.DefaultClient)
	err := d.Deliver(
		model.Subscriber{APIKey: "key", WebhookURL: server.URL},
		model.ComputedRate{Pair: "USDT_NGN", MidRate: 1414},
	)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}
