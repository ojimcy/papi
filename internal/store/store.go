package store

import (
	"context"
	"time"

	"github.com/zapi-api/zapi/internal/model"
)

type Store interface {
	SaveRate(ctx context.Context, rate model.ComputedRate) error
	GetHistory(ctx context.Context, pair string, since time.Time, interval string) ([]model.ComputedRate, error)
	GetSubscribers(ctx context.Context) ([]model.Subscriber, error)
	GetSubscriberByKey(ctx context.Context, apiKey string) (*model.Subscriber, error)
	Close() error
}
