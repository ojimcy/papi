package poller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zapi-api/zapi/internal/model"
)

func TestBybitFetcher_Fetch(t *testing.T) {
	response := BybitResponse{
		RetCode: 0,
		RetMsg:  "SUCCESS",
		Result: BybitResult{
			Count: 2,
			Items: []BybitItem{
				{
					NickName:          "Trader1",
					TokenID:           "USDT",
					CurrencyID:        "NGN",
					Side:              1,
					Price:             "1415.00",
					LastQuantity:      "100.50",
					MinAmount:         "5000",
					MaxAmount:         "100000",
					RecentOrderNum:    500,
					RecentExecuteRate: 98,
					PriceType:         0,
					Premium:           "0",
				},
				{
					NickName:          "Trader2",
					TokenID:           "USDT",
					CurrencyID:        "NGN",
					Side:              1,
					Price:             "1420.00",
					LastQuantity:      "200.00",
					MinAmount:         "10000",
					MaxAmount:         "500000",
					RecentOrderNum:    300,
					RecentExecuteRate: 95,
					PriceType:         1,
					Premium:           "110",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/fiat/otc/item/online" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body BybitRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.TokenID != "USDT" {
			t.Errorf("expected tokenId USDT, got %s", body.TokenID)
		}
		if body.CurrencyID != "NGN" {
			t.Errorf("expected currencyId NGN, got %s", body.CurrencyID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	fetcher := NewBybitFetcher(server.URL, http.DefaultClient)
	ads, err := fetcher.Fetch(context.Background(), "USDT_NGN", model.SideBuy, "50000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ads) != 2 {
		t.Fatalf("expected 2 ads, got %d", len(ads))
	}

	if ads[0].Price != 1415.00 {
		t.Errorf("expected price 1415.00, got %f", ads[0].Price)
	}
	if ads[0].Volume != 100.50 {
		t.Errorf("expected volume 100.50, got %f", ads[0].Volume)
	}
	if ads[0].Side != model.SideBuy {
		t.Errorf("expected side buy, got %s", ads[0].Side)
	}
	if ads[0].Source != "bybit" {
		t.Errorf("expected source bybit, got %s", ads[0].Source)
	}
	if ads[1].PriceType != 1 {
		t.Errorf("expected priceType 1, got %d", ads[1].PriceType)
	}
	if ads[1].Premium != 110.0 {
		t.Errorf("expected premium 110, got %f", ads[1].Premium)
	}
}

func TestBybitFetcher_Fetch_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BybitResponse{RetCode: 0, RetMsg: "SUCCESS", Result: BybitResult{}})
	}))
	defer server.Close()

	fetcher := NewBybitFetcher(server.URL, http.DefaultClient)
	ads, err := fetcher.Fetch(context.Background(), "USDT_NGN", model.SideBuy, "50000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ads) != 0 {
		t.Errorf("expected 0 ads, got %d", len(ads))
	}
}

func TestBybitFetcher_Fetch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := NewBybitFetcher(server.URL, http.DefaultClient)
	_, err := fetcher.Fetch(context.Background(), "USDT_NGN", model.SideBuy, "50000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
