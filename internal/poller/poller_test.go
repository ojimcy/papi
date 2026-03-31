package poller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zapi-api/zapi/internal/cache"
	"github.com/zapi-api/zapi/internal/model"
	"github.com/zapi-api/zapi/internal/store"
	"github.com/zapi-api/zapi/internal/webhook"
)

type mockFetcher struct {
	ads []model.Ad
	err error
}

func (m *mockFetcher) Fetch(_ context.Context, _ string, _ model.Side, _ string) ([]model.Ad, error) {
	return m.ads, m.err
}
func (m *mockFetcher) Source() string { return "mock" }

type mockStore struct {
	saved       []model.ComputedRate
	saveErr     error
	subscribers []model.Subscriber
}

func (m *mockStore) SaveRate(_ context.Context, rate model.ComputedRate) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, rate)
	return nil
}
func (m *mockStore) GetHistory(_ context.Context, _ string, _ time.Time, _ string, _, _ int) ([]model.ComputedRate, error) {
	return nil, nil
}
func (m *mockStore) GetSubscribers(_ context.Context) ([]model.Subscriber, error) {
	return m.subscribers, nil
}
func (m *mockStore) GetSubscriberByKey(_ context.Context, _ string) (*model.Subscriber, error) {
	return nil, nil
}
func (m *mockStore) CreateSubscriber(_ context.Context, _ string, _ string) (*model.Subscriber, error) {
	return nil, nil
}
func (m *mockStore) UpdateSubscriber(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
func (m *mockStore) DeactivateSubscriber(_ context.Context, _ string) error { return nil }
func (m *mockStore) Ping(_ context.Context) error                          { return nil }
func (m *mockStore) Close() error                                          { return nil }

var _ store.Store = (*mockStore)(nil)

func TestPoller_RunOnce(t *testing.T) {
	ads := []model.Ad{
		{Source: "mock", Pair: "USDT_NGN", Side: model.SideBuy, Price: 1415, Volume: 100, CompletionRate: 98, OrderCount: 200},
		{Source: "mock", Pair: "USDT_NGN", Side: model.SideBuy, Price: 1416, Volume: 200, CompletionRate: 95, OrderCount: 150},
	}

	fetcher := &mockFetcher{ads: ads}
	s := &mockStore{}
	c := cache.New()

	p := NewPoller(fetcher, s, c, nil, "USDT_NGN")
	err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 4 tiers + 1 aggregate = 5 rates saved
	if len(s.saved) != 5 {
		t.Errorf("expected 5 saved rates, got %d", len(s.saved))
	}

	_, ok := c.Get("USDT_NGN", "aggregate")
	if !ok {
		t.Error("expected aggregate rate in cache")
	}
}

func TestPoller_RunOnce_FetchError(t *testing.T) {
	fetcher := &mockFetcher{err: fmt.Errorf("network error")}
	s := &mockStore{}
	c := cache.New()

	p := NewPoller(fetcher, s, c, nil, "USDT_NGN")
	err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce should not return error on fetch failures, got: %v", err)
	}

	// No rates saved when all fetches fail
	if len(s.saved) != 0 {
		t.Errorf("expected 0 saved rates, got %d", len(s.saved))
	}
}

func TestPoller_RunOnce_WithWebhook(t *testing.T) {
	var webhookCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ads := []model.Ad{
		{Source: "mock", Pair: "USDT_NGN", Side: model.SideBuy, Price: 1415, Volume: 100, CompletionRate: 98, OrderCount: 200},
	}

	fetcher := &mockFetcher{ads: ads}
	s := &mockStore{
		subscribers: []model.Subscriber{
			{APIKey: "key1", WebhookURL: server.URL, IsActive: true},
		},
	}
	c := cache.New()
	d := webhook.New(0.5, http.DefaultClient)

	p := NewPoller(fetcher, s, c, d, "USDT_NGN")
	p.RunOnce(context.Background())

	// Give webhook goroutine time to fire
	time.Sleep(100 * time.Millisecond)

	if !webhookCalled.Load() {
		t.Error("expected webhook to be called on first rate")
	}
}

func TestPoller_RunOnce_WebhookNotTriggeredBelowThreshold(t *testing.T) {
	var webhookCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ads := []model.Ad{
		{Source: "mock", Pair: "USDT_NGN", Side: model.SideBuy, Price: 1415, Volume: 100, CompletionRate: 98, OrderCount: 200},
	}

	fetcher := &mockFetcher{ads: ads}
	s := &mockStore{
		subscribers: []model.Subscriber{
			{APIKey: "key1", WebhookURL: server.URL},
		},
	}
	c := cache.New()
	d := webhook.New(0.5, http.DefaultClient)

	p := NewPoller(fetcher, s, c, d, "USDT_NGN")

	// First run sets the rate
	p.RunOnce(context.Background())
	time.Sleep(50 * time.Millisecond)
	webhookCalled.Store(false)

	// Second run with same data - no significant change
	p.RunOnce(context.Background())
	time.Sleep(50 * time.Millisecond)

	if webhookCalled.Load() {
		t.Error("webhook should not fire when rate hasn't changed significantly")
	}
}

func TestPoller_RunOnce_SkipsEmptyWebhookURL(t *testing.T) {
	ads := []model.Ad{
		{Source: "mock", Pair: "USDT_NGN", Side: model.SideBuy, Price: 1415, Volume: 100, CompletionRate: 98, OrderCount: 200},
	}

	fetcher := &mockFetcher{ads: ads}
	s := &mockStore{
		subscribers: []model.Subscriber{
			{APIKey: "key1", WebhookURL: "", IsActive: true},
		},
	}
	c := cache.New()
	d := webhook.New(0.5, http.DefaultClient)

	p := NewPoller(fetcher, s, c, d, "USDT_NGN")
	// Should not panic or error with empty webhook URL
	p.RunOnce(context.Background())
}

func TestPoller_RunOnce_CachesAllTiers(t *testing.T) {
	ads := []model.Ad{
		{Source: "mock", Pair: "USDT_NGN", Side: model.SideBuy, Price: 1415, Volume: 100, CompletionRate: 98, OrderCount: 200},
	}

	fetcher := &mockFetcher{ads: ads}
	s := &mockStore{}
	c := cache.New()

	p := NewPoller(fetcher, s, c, nil, "USDT_NGN")
	p.RunOnce(context.Background())

	for _, tier := range model.AmountTiers {
		if _, ok := c.Get("USDT_NGN", tier); !ok {
			t.Errorf("expected tier %s in cache", tier)
		}
	}
}

func TestPoller_Start_CancelsOnContext(t *testing.T) {
	fetcher := &mockFetcher{ads: []model.Ad{
		{Source: "mock", Pair: "USDT_NGN", Side: model.SideBuy, Price: 1415, Volume: 100, CompletionRate: 98, OrderCount: 200},
	}}
	s := &mockStore{}
	c := cache.New()

	p := NewPoller(fetcher, s, c, nil, "USDT_NGN")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		p.Start(ctx, 100*time.Millisecond)
		close(done)
	}()

	// Let it run one cycle
	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK, stopped
	case <-time.After(2 * time.Second):
		t.Fatal("poller did not stop after context cancel")
	}

	if len(s.saved) == 0 {
		t.Error("expected at least one save from the poller")
	}
}
