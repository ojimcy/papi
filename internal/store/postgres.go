package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/papi-api/papi/internal/model"
)

func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

func (s *PostgresStore) SaveRate(ctx context.Context, rate model.ComputedRate) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO rate_snapshots (pair, amount_tier, buy_rate, sell_rate, mid_rate, spread, spread_pct, base_rate, source, captured_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		rate.Pair, rate.AmountTier, rate.BuyRate, rate.SellRate, rate.MidRate,
		rate.Spread, rate.SpreadPct, rate.BaseRate, rate.Source, rate.Timestamp,
	)
	return err
}

func (s *PostgresStore) GetHistory(ctx context.Context, pair string, since time.Time, tier string, limit, offset int) ([]model.ComputedRate, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT pair, amount_tier, buy_rate, sell_rate, mid_rate, spread, spread_pct, base_rate, source, captured_at
		 FROM rate_snapshots
		 WHERE pair = $1 AND amount_tier = $2 AND captured_at >= $3
		 ORDER BY captured_at DESC
		 LIMIT $4 OFFSET $5`,
		pair, tier, since, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []model.ComputedRate
	for rows.Next() {
		var r model.ComputedRate
		err := rows.Scan(&r.Pair, &r.AmountTier, &r.BuyRate, &r.SellRate, &r.MidRate,
			&r.Spread, &r.SpreadPct, &r.BaseRate, &r.Source, &r.Timestamp)
		if err != nil {
			return nil, err
		}
		rates = append(rates, r)
	}
	return rates, rows.Err()
}

func (s *PostgresStore) GetSubscribers(ctx context.Context) ([]model.Subscriber, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, api_key, webhook_url, is_active, is_admin FROM subscribers WHERE is_active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []model.Subscriber
	for rows.Next() {
		var sub model.Subscriber
		var webhookURL *string
		err := rows.Scan(&sub.ID, &sub.Name, &sub.APIKey, &webhookURL, &sub.IsActive, &sub.IsAdmin)
		if err != nil {
			return nil, err
		}
		if webhookURL != nil {
			sub.WebhookURL = *webhookURL
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (s *PostgresStore) GetSubscriberByKey(ctx context.Context, apiKey string) (*model.Subscriber, error) {
	var sub model.Subscriber
	var webhookURL *string
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, api_key, webhook_url, is_active, is_admin FROM subscribers WHERE api_key = $1 AND is_active = true`,
		apiKey).Scan(&sub.ID, &sub.Name, &sub.APIKey, &webhookURL, &sub.IsActive, &sub.IsAdmin)
	if err != nil {
		return nil, err
	}
	if webhookURL != nil {
		sub.WebhookURL = *webhookURL
	}
	return &sub, nil
}

func (s *PostgresStore) CreateSubscriber(ctx context.Context, name, webhookURL string) (*model.Subscriber, error) {
	apiKey := generateAPIKey()
	var sub model.Subscriber
	var whURL *string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO subscribers (name, api_key, webhook_url, is_active)
		 VALUES ($1, $2, $3, true)
		 RETURNING id, name, api_key, webhook_url, is_active, is_admin`,
		name, apiKey, webhookURL).Scan(&sub.ID, &sub.Name, &sub.APIKey, &whURL, &sub.IsActive, &sub.IsAdmin)
	if err != nil {
		return nil, err
	}
	if whURL != nil {
		sub.WebhookURL = *whURL
	}
	return &sub, nil
}

func (s *PostgresStore) UpdateSubscriber(ctx context.Context, id string, name, webhookURL string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE subscribers SET name = $1, webhook_url = $2 WHERE id = $3 AND is_active = true`,
		name, webhookURL, id)
	return err
}

func (s *PostgresStore) DeactivateSubscriber(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE subscribers SET is_active = false WHERE id = $1`,
		id)
	return err
}
