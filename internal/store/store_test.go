package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/papi-api/papi/internal/model"
)

// Compile-time interface compliance check.
var _ Store = (*memoryStore)(nil)

type memoryStore struct {
	mu          sync.Mutex
	rates       []model.ComputedRate
	subscribers []model.Subscriber
	nextID      int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{}
}

func (m *memoryStore) SaveRate(_ context.Context, rate model.ComputedRate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rates = append(m.rates, rate)
	return nil
}

func (m *memoryStore) GetHistory(_ context.Context, pair string, since time.Time, tier string, limit, offset int) ([]model.ComputedRate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []model.ComputedRate
	for _, r := range m.rates {
		if r.Pair == pair && r.AmountTier == tier && !r.Timestamp.Before(since) {
			filtered = append(filtered, r)
		}
	}

	// Apply offset.
	if offset > len(filtered) {
		return nil, nil
	}
	filtered = filtered[offset:]

	// Apply limit.
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

func (m *memoryStore) GetSubscribers(_ context.Context) ([]model.Subscriber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var active []model.Subscriber
	for _, s := range m.subscribers {
		if s.IsActive {
			active = append(active, s)
		}
	}
	return active, nil
}

func (m *memoryStore) GetSubscriberByKey(_ context.Context, apiKey string) (*model.Subscriber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.subscribers {
		if s.APIKey == apiKey && s.IsActive {
			cp := s
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("subscriber not found")
}

func (m *memoryStore) CreateSubscriber(_ context.Context, name, webhookURL string) (*model.Subscriber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	s := model.Subscriber{
		ID:         fmt.Sprintf("id-%d", m.nextID),
		Name:       name,
		APIKey:     fmt.Sprintf("key-%d", m.nextID),
		WebhookURL: webhookURL,
		IsActive:   true,
	}
	m.subscribers = append(m.subscribers, s)
	return &s, nil
}

func (m *memoryStore) UpdateSubscriber(_ context.Context, id string, name, webhookURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.subscribers {
		if m.subscribers[i].ID == id {
			m.subscribers[i].Name = name
			m.subscribers[i].WebhookURL = webhookURL
			return nil
		}
	}
	return fmt.Errorf("subscriber %s not found", id)
}

func (m *memoryStore) DeactivateSubscriber(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.subscribers {
		if m.subscribers[i].ID == id {
			m.subscribers[i].IsActive = false
			return nil
		}
	}
	return fmt.Errorf("subscriber %s not found", id)
}

func (m *memoryStore) Ping(_ context.Context) error {
	return nil
}

func (m *memoryStore) Close() error {
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMemoryStore_SaveAndGetHistory(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()
	now := time.Now().UTC()

	rate := model.ComputedRate{
		Pair:       "USDT-NGN",
		AmountTier: "aggregate",
		BuyRate:    1600,
		SellRate:   1580,
		MidRate:    1590,
		Timestamp:  now,
	}

	if err := s.SaveRate(ctx, rate); err != nil {
		t.Fatalf("SaveRate: %v", err)
	}

	got, err := s.GetHistory(ctx, "USDT-NGN", now.Add(-time.Minute), "aggregate", 10, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 rate, got %d", len(got))
	}
	if got[0].BuyRate != 1600 {
		t.Errorf("expected BuyRate 1600, got %f", got[0].BuyRate)
	}
}

func TestMemoryStore_GetHistory_Pagination(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()
	base := time.Now().UTC()

	for i := 0; i < 5; i++ {
		r := model.ComputedRate{
			Pair:       "USDT-NGN",
			AmountTier: "aggregate",
			BuyRate:    float64(1600 + i),
			Timestamp:  base.Add(time.Duration(i) * time.Second),
		}
		if err := s.SaveRate(ctx, r); err != nil {
			t.Fatalf("SaveRate: %v", err)
		}
	}

	got, err := s.GetHistory(ctx, "USDT-NGN", base.Add(-time.Minute), "aggregate", 2, 1)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rates, got %d", len(got))
	}
	// offset=1 skips the first rate (BuyRate 1600), so first result should be 1601.
	if got[0].BuyRate != 1601 {
		t.Errorf("expected BuyRate 1601, got %f", got[0].BuyRate)
	}
	if got[1].BuyRate != 1602 {
		t.Errorf("expected BuyRate 1602, got %f", got[1].BuyRate)
	}
}

func TestMemoryStore_GetHistory_FiltersByPair(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()
	now := time.Now().UTC()

	pairs := []string{"USDT-NGN", "USDT-GHS"}
	for _, pair := range pairs {
		r := model.ComputedRate{
			Pair:       pair,
			AmountTier: "aggregate",
			BuyRate:    1500,
			Timestamp:  now,
		}
		if err := s.SaveRate(ctx, r); err != nil {
			t.Fatalf("SaveRate: %v", err)
		}
	}

	got, err := s.GetHistory(ctx, "USDT-GHS", now.Add(-time.Minute), "aggregate", 10, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 rate, got %d", len(got))
	}
	if got[0].Pair != "USDT-GHS" {
		t.Errorf("expected pair USDT-GHS, got %s", got[0].Pair)
	}
}

func TestMemoryStore_Subscribers(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()

	sub, err := s.CreateSubscriber(ctx, "Acme", "https://acme.example.com/hook")
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}
	if sub.Name != "Acme" || sub.WebhookURL != "https://acme.example.com/hook" {
		t.Fatalf("unexpected subscriber: %+v", sub)
	}
	if sub.APIKey == "" || sub.ID == "" {
		t.Fatal("expected non-empty ID and APIKey")
	}

	// List subscribers.
	subs, err := s.GetSubscribers(ctx)
	if err != nil {
		t.Fatalf("GetSubscribers: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subscriber, got %d", len(subs))
	}

	// Get by key.
	found, err := s.GetSubscriberByKey(ctx, sub.APIKey)
	if err != nil {
		t.Fatalf("GetSubscriberByKey: %v", err)
	}
	if found.ID != sub.ID {
		t.Errorf("expected ID %s, got %s", sub.ID, found.ID)
	}
}

func TestMemoryStore_DeactivateSubscriber(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()

	sub, err := s.CreateSubscriber(ctx, "TestCo", "https://test.co/hook")
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}

	if err := s.DeactivateSubscriber(ctx, sub.ID); err != nil {
		t.Fatalf("DeactivateSubscriber: %v", err)
	}

	// Should not appear in list.
	subs, err := s.GetSubscribers(ctx)
	if err != nil {
		t.Fatalf("GetSubscribers: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("expected 0 subscribers, got %d", len(subs))
	}

	// Should not be found by key.
	_, err = s.GetSubscriberByKey(ctx, sub.APIKey)
	if err == nil {
		t.Fatal("expected error looking up deactivated subscriber")
	}
}

func TestMemoryStore_UpdateSubscriber(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()

	sub, err := s.CreateSubscriber(ctx, "OldName", "https://old.example.com/hook")
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}

	if err := s.UpdateSubscriber(ctx, sub.ID, "NewName", "https://new.example.com/hook"); err != nil {
		t.Fatalf("UpdateSubscriber: %v", err)
	}

	found, err := s.GetSubscriberByKey(ctx, sub.APIKey)
	if err != nil {
		t.Fatalf("GetSubscriberByKey: %v", err)
	}
	if found.Name != "NewName" {
		t.Errorf("expected name NewName, got %s", found.Name)
	}
	if found.WebhookURL != "https://new.example.com/hook" {
		t.Errorf("expected webhook https://new.example.com/hook, got %s", found.WebhookURL)
	}
}
