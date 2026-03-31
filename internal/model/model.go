package model

import "time"

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type Ad struct {
	Source         string
	Pair           string
	Side           Side
	Price          float64
	Volume         float64
	MinAmount      float64
	MaxAmount      float64
	OrderCount     int
	CompletionRate float64
	PriceType      int
	Premium        float64
	CapturedAt     time.Time
}

type ComputedRate struct {
	Pair       string    `json:"pair"`
	AmountTier string    `json:"amount_tier"`
	BuyRate    float64   `json:"buy_rate"`
	SellRate   float64   `json:"sell_rate"`
	MidRate    float64   `json:"mid_rate"`
	Spread     float64   `json:"spread"`
	SpreadPct  float64   `json:"spread_pct"`
	BaseRate   float64   `json:"base_rate"`
	Source     string    `json:"source"`
	Timestamp  time.Time `json:"timestamp"`
}

type Subscriber struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	APIKey     string `json:"api_key"`
	WebhookURL string `json:"webhook_url,omitempty"`
	IsActive   bool   `json:"is_active"`
	IsAdmin    bool   `json:"is_admin"`
}

var AmountTiers = []string{"50000", "500000", "5000000", "20000000"}
