package poller

import (
	"context"
	"testing"
	"time"

	"github.com/zapi-api/zapi/internal/cache"
	"github.com/zapi-api/zapi/internal/model"
	"github.com/zapi-api/zapi/internal/store"
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
	saved []model.ComputedRate
}

func (m *mockStore) SaveRate(_ context.Context, rate model.ComputedRate) error {
	m.saved = append(m.saved, rate)
	return nil
}
func (m *mockStore) GetHistory(_ context.Context, _ string, _ time.Time, _ string) ([]model.ComputedRate, error) {
	return nil, nil
}
func (m *mockStore) GetSubscribers(_ context.Context) ([]model.Subscriber, error) { return nil, nil }
func (m *mockStore) GetSubscriberByKey(_ context.Context, _ string) (*model.Subscriber, error) {
	return nil, nil
}
func (m *mockStore) Close() error { return nil }

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
