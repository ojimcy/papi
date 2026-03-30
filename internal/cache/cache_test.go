package cache

import (
	"testing"
	"time"

	"github.com/zapi-api/zapi/internal/model"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New()
	rate := model.ComputedRate{
		Pair: "USDT_NGN", AmountTier: "500000",
		BuyRate: 1416.38, SellRate: 1412.04, MidRate: 1414.21,
		Source: "bybit", Timestamp: time.Now(),
	}

	c.Set("USDT_NGN", "500000", rate)

	got, ok := c.Get("USDT_NGN", "500000")
	if !ok {
		t.Fatal("expected to find rate")
	}
	if got.BuyRate != 1416.38 {
		t.Errorf("expected 1416.38, got %f", got.BuyRate)
	}
}

func TestCache_Get_NotFound(t *testing.T) {
	c := New()
	_, ok := c.Get("USDT_NGN", "500000")
	if ok {
		t.Error("expected not found")
	}
}

func TestCache_GetAll(t *testing.T) {
	c := New()
	c.Set("USDT_NGN", "50000", model.ComputedRate{BuyRate: 1416})
	c.Set("USDT_NGN", "500000", model.ComputedRate{BuyRate: 1417})

	all := c.GetAll("USDT_NGN")
	if len(all) != 2 {
		t.Fatalf("expected 2 rates, got %d", len(all))
	}
}
