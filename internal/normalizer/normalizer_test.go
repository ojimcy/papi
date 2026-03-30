package normalizer

import (
	"testing"
	"time"

	"github.com/zapi-api/zapi/internal/model"
)

func makeAd(price, volume, completionRate float64, orderCount int) model.Ad {
	return model.Ad{
		Source:         "bybit",
		Pair:           "USDT_NGN",
		Side:           model.SideBuy,
		Price:          price,
		Volume:         volume,
		MinAmount:      5000,
		MaxAmount:      100000,
		OrderCount:     orderCount,
		CompletionRate: completionRate,
		CapturedAt:     time.Now(),
	}
}

func TestFilter_RemovesLowCompletionRate(t *testing.T) {
	ads := []model.Ad{
		makeAd(1415, 100, 98, 200),
		makeAd(1416, 100, 50, 200),
		makeAd(1414, 100, 85, 200),
	}

	result := Filter(ads)
	if len(result) != 2 {
		t.Fatalf("expected 2 ads, got %d", len(result))
	}
}

func TestFilter_RemovesLowOrderCount(t *testing.T) {
	ads := []model.Ad{
		makeAd(1415, 100, 98, 200),
		makeAd(1416, 100, 98, 10),
		makeAd(1414, 100, 98, 50),
	}

	result := Filter(ads)
	if len(result) != 2 {
		t.Fatalf("expected 2 ads, got %d", len(result))
	}
}

func TestRemoveOutliers_IQR(t *testing.T) {
	ads := []model.Ad{
		makeAd(1410, 100, 98, 200),
		makeAd(1412, 100, 98, 200),
		makeAd(1413, 100, 98, 200),
		makeAd(1414, 100, 98, 200),
		makeAd(1415, 100, 98, 200),
		makeAd(1416, 100, 98, 200),
		makeAd(1418, 100, 98, 200),
		makeAd(9999, 100, 98, 200),
		makeAd(100, 100, 98, 200),
	}

	result := RemoveOutliers(ads)
	for _, ad := range result {
		if ad.Price == 9999 || ad.Price == 100 {
			t.Errorf("outlier price %f should have been removed", ad.Price)
		}
	}
	if len(result) != 7 {
		t.Errorf("expected 7 ads, got %d", len(result))
	}
}

func TestRemoveOutliers_TooFewAds(t *testing.T) {
	ads := []model.Ad{makeAd(1415, 100, 98, 200), makeAd(1416, 100, 98, 200)}
	result := RemoveOutliers(ads)
	if len(result) != 2 {
		t.Errorf("expected 2 ads, got %d", len(result))
	}
}

func TestNormalize_FullPipeline(t *testing.T) {
	ads := []model.Ad{
		makeAd(1415, 100, 98, 200),
		makeAd(1416, 100, 50, 200),
		makeAd(1414, 100, 98, 10),
		makeAd(9999, 100, 98, 200),
		makeAd(1413, 100, 90, 100),
	}

	result := Normalize(ads)
	// After filter: 1415 (98%, 200), 9999 (98%, 200), 1413 (90%, 100) = 3 ads
	// After IQR on 3 ads: too few to remove outliers, so all 3 remain
	if len(result) != 3 {
		t.Fatalf("expected 3 ads, got %d", len(result))
	}
}
