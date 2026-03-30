package normalizer

import (
	"sort"

	"github.com/zapi-api/zapi/internal/model"
)

const (
	minCompletionRate = 80.0
	minOrderCount     = 50
)

func Filter(ads []model.Ad) []model.Ad {
	result := make([]model.Ad, 0, len(ads))
	for _, ad := range ads {
		if ad.CompletionRate >= minCompletionRate && ad.OrderCount >= minOrderCount {
			result = append(result, ad)
		}
	}
	return result
}

func RemoveOutliers(ads []model.Ad) []model.Ad {
	if len(ads) < 4 {
		return ads
	}

	prices := make([]float64, len(ads))
	for i, ad := range ads {
		prices[i] = ad.Price
	}
	sort.Float64s(prices)

	q1 := percentile(prices, 25)
	q3 := percentile(prices, 75)
	iqr := q3 - q1
	lower := q1 - 1.5*iqr
	upper := q3 + 1.5*iqr

	result := make([]model.Ad, 0, len(ads))
	for _, ad := range ads {
		if ad.Price >= lower && ad.Price <= upper {
			result = append(result, ad)
		}
	}
	return result
}

func Normalize(ads []model.Ad) []model.Ad {
	return RemoveOutliers(Filter(ads))
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	k := (p / 100.0) * float64(len(sorted)-1)
	f := int(k)
	c := f + 1
	if c >= len(sorted) {
		return sorted[f]
	}
	d := k - float64(f)
	return sorted[f] + d*(sorted[c]-sorted[f])
}
