package calculator

import (
	"time"

	"github.com/papi-api/papi/internal/model"
)

func VWAP(ads []model.Ad) float64 {
	if len(ads) == 0 {
		return 0
	}

	var totalWeight, weightedSum float64
	for _, ad := range ads {
		weight := ad.Volume * ad.CompletionRate
		weightedSum += ad.Price * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0
	}
	return weightedSum / totalWeight
}

func DeriveBaseRate(ads []model.Ad) float64 {
	var sum float64
	var count int

	for _, ad := range ads {
		if ad.PriceType == 1 && ad.Premium > 0 {
			base := ad.Price * 100 / ad.Premium
			sum += base
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func Compute(pair, amountTier string, buyAds, sellAds []model.Ad) model.ComputedRate {
	buyRate := VWAP(buyAds)
	sellRate := VWAP(sellAds)

	midRate := 0.0
	if buyRate > 0 && sellRate > 0 {
		midRate = (buyRate + sellRate) / 2
	}

	spread := buyRate - sellRate
	spreadPct := 0.0
	if midRate > 0 {
		spreadPct = spread / midRate * 100
	}

	allAds := make([]model.Ad, 0, len(buyAds)+len(sellAds))
	allAds = append(allAds, buyAds...)
	allAds = append(allAds, sellAds...)
	baseRate := DeriveBaseRate(allAds)

	return model.ComputedRate{
		Pair:       pair,
		AmountTier: amountTier,
		BuyRate:    buyRate,
		SellRate:   sellRate,
		MidRate:    midRate,
		Spread:     spread,
		SpreadPct:  spreadPct,
		BaseRate:   baseRate,
		Source:     "bybit",
		Timestamp:  time.Now(),
	}
}
