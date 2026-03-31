package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/papi-api/papi/internal/model"
)

type WebhookPayload struct {
	Event string             `json:"event"`
	Data  model.ComputedRate `json:"data"`
}

type Dispatcher struct {
	threshold float64
	client    *http.Client
}

func New(threshold float64, client *http.Client) *Dispatcher {
	return &Dispatcher{threshold: threshold, client: client}
}

func (d *Dispatcher) ShouldNotify(old, new_ model.ComputedRate) bool {
	if old.MidRate == 0 {
		return true
	}
	change := math.Abs(new_.MidRate-old.MidRate) / old.MidRate * 100
	return change >= d.threshold
}

func (d *Dispatcher) Deliver(sub model.Subscriber, rate model.ComputedRate) error {
	payload := WebhookPayload{
		Event: "rate.updated",
		Data:  rate,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	signature := sign(body, sub.APIKey)

	var lastErr error
	delays := []time.Duration{0, 1 * time.Second, 5 * time.Second}

	for i, delay := range delays {
		if i > 0 {
			time.Sleep(delay)
		}

		req, err := http.NewRequest(http.MethodPost, sub.WebhookURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", signature)

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return fmt.Errorf("webhook delivery failed after 3 attempts: %w", lastErr)
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
