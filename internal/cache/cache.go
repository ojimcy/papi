package cache

import (
	"sync"

	"github.com/papi-api/papi/internal/model"
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

func (c *Cache) Pairs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pairs := make([]string, 0, len(c.rates))
	for pair := range c.rates {
		pairs = append(pairs, pair)
	}
	return pairs
}

func (c *Cache) Tiers(pair string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tiers, ok := c.rates[pair]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(tiers))
	for tier := range tiers {
		names = append(names, tier)
	}
	return names
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
