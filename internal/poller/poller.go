package poller

import (
	"context"
	"log/slog"
	"time"

	"github.com/papi-api/papi/internal/cache"
	"github.com/papi-api/papi/internal/calculator"
	"github.com/papi-api/papi/internal/model"
	"github.com/papi-api/papi/internal/normalizer"
	"github.com/papi-api/papi/internal/store"
	"github.com/papi-api/papi/internal/webhook"
)

type Poller struct {
	fetcher    Fetcher
	store      store.Store
	cache      *cache.Cache
	dispatcher *webhook.Dispatcher
	pair       string
}

func NewPoller(fetcher Fetcher, s store.Store, c *cache.Cache, dispatcher *webhook.Dispatcher, pair string) *Poller {
	return &Poller{
		fetcher:    fetcher,
		store:      s,
		cache:      c,
		dispatcher: dispatcher,
		pair:       pair,
	}
}

func (p *Poller) Start(ctx context.Context, interval time.Duration) {
	slog.Info("poller started", "pair", p.pair, "interval", interval)

	if err := p.RunOnce(ctx); err != nil {
		slog.Error("poll cycle failed", "error", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("poller stopped", "pair", p.pair)
			return
		case <-ticker.C:
			if err := p.RunOnce(ctx); err != nil {
				slog.Error("poll cycle failed", "error", err)
			}
		}
	}
}

func (p *Poller) RunOnce(ctx context.Context) error {
	var allRates []model.ComputedRate

	for _, tier := range model.AmountTiers {
		buyAds, err := p.fetcher.Fetch(ctx, p.pair, model.SideBuy, tier)
		if err != nil {
			slog.Error("fetch buy ads failed", "tier", tier, "error", err)
			continue
		}

		sellAds, err := p.fetcher.Fetch(ctx, p.pair, model.SideSell, tier)
		if err != nil {
			slog.Error("fetch sell ads failed", "tier", tier, "error", err)
			continue
		}

		cleanBuy := normalizer.Normalize(buyAds)
		cleanSell := normalizer.Normalize(sellAds)

		rate := calculator.Compute(p.pair, tier, cleanBuy, cleanSell)

		p.cache.Set(p.pair, tier, rate)

		if err := p.store.SaveRate(ctx, rate); err != nil {
			slog.Error("save rate failed", "tier", tier, "error", err)
		}

		allRates = append(allRates, rate)
	}

	if len(allRates) > 0 {
		aggregate := p.computeAggregate(allRates)

		oldAgg, hadOld := p.cache.Get(p.pair, "aggregate")
		p.cache.Set(p.pair, "aggregate", aggregate)

		if err := p.store.SaveRate(ctx, aggregate); err != nil {
			slog.Error("save aggregate rate failed", "error", err)
		}

		if p.dispatcher != nil && (!hadOld || p.dispatcher.ShouldNotify(oldAgg, aggregate)) {
			p.notifySubscribers(ctx, aggregate)
		}
	}

	return nil
}

func (p *Poller) computeAggregate(rates []model.ComputedRate) model.ComputedRate {
	var totalBuyWeight, totalSellWeight float64
	var weightedBuy, weightedSell float64
	var baseRateSum float64
	var baseRateCount int

	for _, r := range rates {
		w := 1.0
		if r.BuyRate > 0 {
			weightedBuy += r.BuyRate * w
			totalBuyWeight += w
		}
		if r.SellRate > 0 {
			weightedSell += r.SellRate * w
			totalSellWeight += w
		}
		if r.BaseRate > 0 {
			baseRateSum += r.BaseRate
			baseRateCount++
		}
	}

	buyRate := 0.0
	if totalBuyWeight > 0 {
		buyRate = weightedBuy / totalBuyWeight
	}
	sellRate := 0.0
	if totalSellWeight > 0 {
		sellRate = weightedSell / totalSellWeight
	}
	midRate := 0.0
	if buyRate > 0 && sellRate > 0 {
		midRate = (buyRate + sellRate) / 2
	}
	spread := buyRate - sellRate
	spreadPct := 0.0
	if midRate > 0 {
		spreadPct = spread / midRate * 100
	}
	baseRate := 0.0
	if baseRateCount > 0 {
		baseRate = baseRateSum / float64(baseRateCount)
	}

	return model.ComputedRate{
		Pair:       p.pair,
		AmountTier: "aggregate",
		BuyRate:    buyRate,
		SellRate:   sellRate,
		MidRate:    midRate,
		Spread:     spread,
		SpreadPct:  spreadPct,
		BaseRate:   baseRate,
		Source:     rates[0].Source,
		Timestamp:  time.Now(),
	}
}

func (p *Poller) notifySubscribers(ctx context.Context, rate model.ComputedRate) {
	subs, err := p.store.GetSubscribers(ctx)
	if err != nil {
		slog.Error("get subscribers failed", "error", err)
		return
	}

	for _, sub := range subs {
		if sub.WebhookURL == "" {
			continue
		}
		go func(s model.Subscriber) {
			if err := p.dispatcher.Deliver(s, rate); err != nil {
				slog.Error("webhook delivery failed", "subscriber", s.Name, "error", err)
			}
		}(sub)
	}
}
