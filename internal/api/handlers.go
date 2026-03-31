package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/papi-api/papi/internal/cache"
	"github.com/papi-api/papi/internal/model"
	"github.com/papi-api/papi/internal/store"
)

type Handlers struct {
	cache *cache.Cache
	store store.Store
}

func NewHandlers(c *cache.Cache, s store.Store) *Handlers {
	return &Handlers{cache: c, store: s}
}

func (h *Handlers) GetPairs(w http.ResponseWriter, r *http.Request) {
	pairs := h.cache.Pairs()
	result := make([]map[string]any, 0, len(pairs))
	for _, pair := range pairs {
		result = append(result, map[string]any{
			"pair":  pair,
			"tiers": h.cache.Tiers(pair),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) GetRates(w http.ResponseWriter, r *http.Request) {
	pair := r.URL.Query().Get("pair")
	if pair == "" {
		writeError(w, http.StatusBadRequest, "missing pair parameter")
		return
	}
	if !validPair(pair) {
		writeError(w, http.StatusBadRequest, "invalid pair format: expected TOKEN_CURRENCY (e.g. USDT_NGN)")
		return
	}

	amount := r.URL.Query().Get("amount")
	tier := "aggregate"
	if amount != "" {
		tier = matchTier(amount)
	}

	rate, ok := h.cache.Get(pair, tier)
	if !ok {
		writeError(w, http.StatusNotFound, "rate not found for pair")
		return
	}

	resp := map[string]any{
		"pair":        rate.Pair,
		"amount_tier": rate.AmountTier,
		"buy_rate":    rate.BuyRate,
		"sell_rate":   rate.SellRate,
		"mid_rate":    rate.MidRate,
		"spread":      rate.Spread,
		"spread_pct":  rate.SpreadPct,
		"base_rate":   rate.BaseRate,
		"source":      rate.Source,
		"timestamp":   rate.Timestamp,
		"stale":       time.Since(rate.Timestamp) > 5*time.Minute,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	pair := r.URL.Query().Get("pair")
	if pair == "" {
		writeError(w, http.StatusBadRequest, "missing pair parameter")
		return
	}
	if !validPair(pair) {
		writeError(w, http.StatusBadRequest, "invalid pair format: expected TOKEN_CURRENCY (e.g. USDT_NGN)")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	duration, err := parsePeriod(period)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid period: use 1h, 6h, 24h, or 7d")
		return
	}

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		l, err := strconv.Atoi(v)
		if err != nil || l < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit: must be a positive integer")
			return
		}
		if l > 1000 {
			l = 1000
		}
		limit = l
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		o, err := strconv.Atoi(v)
		if err != nil || o < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset: must be a non-negative integer")
			return
		}
		offset = o
	}

	tier := r.URL.Query().Get("tier")
	if tier == "" {
		tier = "aggregate"
	}

	since := time.Now().Add(-duration)

	rates, err := h.store.GetHistory(r.Context(), pair, since, tier, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pair":   pair,
		"tier":   tier,
		"period": period,
		"limit":  limit,
		"offset": offset,
		"data":   rates,
	})
}

func matchTier(amount string) string {
	val, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return "aggregate"
	}

	tiers := []float64{50000, 500000, 5000000, 20000000}
	tierNames := model.AmountTiers

	closest := tierNames[0]
	minDiff := abs(val - tiers[0])
	for i := 1; i < len(tiers); i++ {
		diff := abs(val - tiers[i])
		if diff < minDiff {
			minDiff = diff
			closest = tierNames[i]
		}
	}
	return closest
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func parsePeriod(period string) (time.Duration, error) {
	switch period {
	case "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "24h":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported period: %s", period)
	}
}

func validPair(pair string) bool {
	parts := strings.SplitN(pair, "_", 2)
	return len(parts) == 2 && len(parts[0]) >= 2 && len(parts[1]) >= 2
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": message,
		"code":  status,
	})
}
