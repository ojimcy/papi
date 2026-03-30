package cache

import (
	"sync"

	"github.com/zapi-api/zapi/internal/model"
)

type Cache struct {
	mu    sync.RWMutex
	rates map[string]map[string]model.ComputedRate
}

func New() *Cache {
	return &Cache{
		rates: make(map[string]map[string]model.ComputedRate),
	}
}

func (c *Cache) Set(pair, amountTier string, rate model.ComputedRate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.rates[pair]; !ok {
		c.rates[pair] = make(map[string]model.ComputedRate)
	}
	c.rates[pair][amountTier] = rate
}

func (c *Cache) Get(pair, amountTier string) (model.ComputedRate, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tiers, ok := c.rates[pair]
	if !ok {
		return model.ComputedRate{}, false
	}
	rate, ok := tiers[amountTier]
	return rate, ok
}

func (c *Cache) GetAll(pair string) []model.ComputedRate {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tiers, ok := c.rates[pair]
	if !ok {
		return nil
	}
	rates := make([]model.ComputedRate, 0, len(tiers))
	for _, rate := range tiers {
		rates = append(rates, rate)
	}
	return rates
}
