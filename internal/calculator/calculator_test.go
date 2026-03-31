package calculator

import (
	"math"
	"testing"
	"time"

	"github.com/papi-api/papi/internal/model"
)

func makeAd(side model.Side, price, volume, completionRate float64) model.Ad {
	return model.Ad{
		Source: "bybit", Pair: "USDT_NGN", Side: side,
		Price: price, Volume: volume, CompletionRate: completionRate,
		CapturedAt: time.Now(),
	}
}

func makeFloatingAd(side model.Side, price, premium float64) model.Ad {
	ad := makeAd(side, price, 100, 98)
	ad.PriceType = 1
	ad.Premium = premium
	return ad
}

func TestVWAP(t *testing.T) {
	ads := []model.Ad{
		makeAd(model.SideBuy, 1415, 100, 98),
		makeAd(model.SideBuy, 1420, 200, 95),
	}

	result := VWAP(ads)
	expected := (1415.0*100*98 + 1420.0*200*95) / (100.0*98 + 200.0*95)

	if math.Abs(result-expected) > 0.01 {
		t.Errorf("expected VWAP %f, got %f", expected, result)
	}
}

func TestVWAP_Empty(t *testing.T) {
	if VWAP(nil) != 0 {
		t.Error("expected 0 for empty ads")
	}
}

func TestDeriveBaseRate(t *testing.T) {
	ads := []model.Ad{
		makeFloatingAd(model.SideSell, 1521.38, 110),
		makeFloatingAd(model.SideSell, 1426.23, 103.12),
		makeAd(model.SideSell, 1518, 100, 98),
	}

	result := DeriveBaseRate(ads)
	if math.Abs(result-1383.07) > 0.1 {
		t.Errorf("expected base rate ~1383.07, got %f", result)
	}
}

func TestDeriveBaseRate_NoFloatingAds(t *testing.T) {
	ads := []model.Ad{makeAd(model.SideSell, 1518, 100, 98)}
	if DeriveBaseRate(ads) != 0 {
		t.Error("expected 0 for no floating ads")
	}
}

func TestCompute(t *testing.T) {
	buyAds := []model.Ad{
		makeAd(model.SideBuy, 1415, 100, 98),
		makeAd(model.SideBuy, 1416, 200, 95),
	}
	sellAds := []model.Ad{
		makeAd(model.SideSell, 1412, 150, 97),
		makeAd(model.SideSell, 1413, 100, 96),
	}

	rate := Compute("USDT_NGN", "500000", buyAds, sellAds)

	if rate.Pair != "USDT_NGN" {
		t.Errorf("expected pair USDT_NGN, got %s", rate.Pair)
	}
	if rate.BuyRate <= 0 || rate.SellRate <= 0 {
		t.Error("expected positive rates")
	}
	if rate.MidRate != (rate.BuyRate+rate.SellRate)/2 {
		t.Error("mid rate should be average of buy and sell")
	}
	if rate.Spread != rate.BuyRate-rate.SellRate {
		t.Error("spread should be buy - sell")
	}
}
