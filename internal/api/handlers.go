package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/zapi-api/zapi/internal/cache"
	"github.com/zapi-api/zapi/internal/model"
	"github.com/zapi-api/zapi/internal/store"
)

type Handlers struct {
	cache *cache.Cache
	store store.Store
}

func NewHandlers(c *cache.Cache, s store.Store) *Handlers {
	return &Handlers{cache: c, store: s}
}

func (h *Handlers) GetRates(w http.ResponseWriter, r *http.Request) {
	pair := r.URL.Query().Get("pair")
	if pair == "" {
		writeError(w, http.StatusBadRequest, "missing pair parameter")
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

	writeJSON(w, http.StatusOK, rate)
}

func (h *Handlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	pair := r.URL.Query().Get("pair")
	if pair == "" {
		writeError(w, http.StatusBadRequest, "missing pair parameter")
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

	since := time.Now().Add(-duration)
	interval := "minute"
	if period == "7d" {
		interval = "hour"
	}

	rates, err := h.store.GetHistory(r.Context(), pair, since, interval)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pair":   pair,
		"period": period,
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
