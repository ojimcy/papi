package poller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/papi-api/papi/internal/model"
)

type BybitRequest struct {
	TokenID    string   `json:"tokenId"`
	CurrencyID string   `json:"currencyId"`
	Side       string   `json:"side"`
	Size       string   `json:"size"`
	Page       string   `json:"page"`
	Payment    []string `json:"payment"`
	Amount     string   `json:"amount"`
}

type BybitResponse struct {
	RetCode int        `json:"ret_code"`
	RetMsg  string     `json:"ret_msg"`
	Result  BybitResult `json:"result"`
}

type BybitResult struct {
	Count int         `json:"count"`
	Items []BybitItem `json:"items"`
}

type BybitItem struct {
	NickName          string `json:"nickName"`
	TokenID           string `json:"tokenId"`
	CurrencyID        string `json:"currencyId"`
	Side              int    `json:"side"`
	Price             string `json:"price"`
	LastQuantity      string `json:"lastQuantity"`
	MinAmount         string `json:"minAmount"`
	MaxAmount         string `json:"maxAmount"`
	RecentOrderNum    int    `json:"recentOrderNum"`
	RecentExecuteRate int    `json:"recentExecuteRate"`
	PriceType         int    `json:"priceType"`
	Premium           string `json:"premium"`
}

type Fetcher interface {
	Fetch(ctx context.Context, pair string, side model.Side, amount string) ([]model.Ad, error)
	Source() string
}

type BybitFetcher struct {
	baseURL string
	client  *http.Client
}

func NewBybitFetcher(baseURL string, client *http.Client) *BybitFetcher {
	return &BybitFetcher{baseURL: baseURL, client: client}
}

func (f *BybitFetcher) Source() string {
	return "bybit"
}

func (f *BybitFetcher) Fetch(ctx context.Context, pair string, side model.Side, amount string) ([]model.Ad, error) {
	tokenID, currencyID := parsePair(pair)

	bybitSide := "1"
	if side == model.SideSell {
		bybitSide = "0"
	}

	reqBody := BybitRequest{
		TokenID:    tokenID,
		CurrencyID: currencyID,
		Side:       bybitSide,
		Size:       "20",
		Page:       "1",
		Payment:    []string{},
		Amount:     amount,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.baseURL+"/fiat/otc/item/online", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Papi/1.0)")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bybit returned status %d", resp.StatusCode)
	}

	var bybitResp BybitResponse
	if err := json.NewDecoder(resp.Body).Decode(&bybitResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if bybitResp.RetCode != 0 {
		return nil, fmt.Errorf("bybit error: %s", bybitResp.RetMsg)
	}

	now := time.Now()
	ads := make([]model.Ad, 0, len(bybitResp.Result.Items))
	for _, item := range bybitResp.Result.Items {
		price, _ := strconv.ParseFloat(item.Price, 64)
		volume, _ := strconv.ParseFloat(item.LastQuantity, 64)
		minAmt, _ := strconv.ParseFloat(item.MinAmount, 64)
		maxAmt, _ := strconv.ParseFloat(item.MaxAmount, 64)
		premium, _ := strconv.ParseFloat(item.Premium, 64)

		ads = append(ads, model.Ad{
			Source:         "bybit",
			Pair:           pair,
			Side:           side,
			Price:          price,
			Volume:         volume,
			MinAmount:      minAmt,
			MaxAmount:      maxAmt,
			OrderCount:     item.RecentOrderNum,
			CompletionRate: float64(item.RecentExecuteRate),
			PriceType:      item.PriceType,
			Premium:        premium,
			CapturedAt:     now,
		})
	}

	return ads, nil
}

func parsePair(pair string) (token, currency string) {
	for i, c := range pair {
		if c == '_' {
			return pair[:i], pair[i+1:]
		}
	}
	return pair, ""
}
