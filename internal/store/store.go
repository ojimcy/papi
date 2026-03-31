package store

import (
	"context"
	"time"

	"github.com/zapi-api/zapi/internal/model"
)

type Store interface {
	SaveRate(ctx context.Context, rate model.ComputedRate) error
	GetHistory(ctx context.Context, pair string, since time.Time, tier string, limit, offset int) ([]model.ComputedRate, error)
	GetSubscribers(ctx context.Context) ([]model.Subscriber, error)
	GetSubscriberByKey(ctx context.Context, apiKey string) (*model.Subscriber, error)
	CreateSubscriber(ctx context.Context, name, webhookURL string) (*model.Subscriber, error)
	UpdateSubscriber(ctx context.Context, id string, name, webhookURL string) error
	DeactivateSubscriber(ctx context.Context, id string) error
	Ping(ctx context.Context) error
	Close() error
}
