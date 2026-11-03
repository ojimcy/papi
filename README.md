# papi

A Go service that aggregates peer-to-peer exchange rates (currently USDT/NGN from Bybit P2P) and serves them over a small REST API, with history, subscriber webhooks and rate limiting.

## Features
- Polls Bybit P2P on an interval and normalises the offers into buy/sell rates per amount tier.
- Real-time and historical rates (`/v1/rates`, `/v1/rates/history`) with pagination and staleness flags.
- Subscribers can register webhooks that fire when a rate moves past a threshold.
- API-key auth with an admin role, per-client rate limiting and request IDs.
- Postgres storage with versioned SQL migrations, an in-memory cache and a health endpoint.
- OpenAPI 3 spec in [`docs/openapi.yaml`](docs/openapi.yaml).

## Layout
```
cmd/server        entrypoint
internal/poller   Bybit polling
internal/normalizer, calculator   offer cleaning and tier rates
internal/api      handlers, middleware, router, rate limiter
internal/store    Postgres access and migrations
internal/webhook  webhook dispatcher
migrations/       SQL migrations
```

## Run it
```bash
cp .env.example .env      # set DATABASE_URL
make migrate
make run                  # listens on :8080
make test
```

Configuration (`.env`): `PORT`, `DATABASE_URL`, `POLL_INTERVAL`, `WEBHOOK_THRESHOLD`, `BYBIT_BASE_URL`, `PAIRS`, `LOG_LEVEL`.

## Stack
Go, PostgreSQL, Docker. Unit tests cover the poller, normalizer, calculator, cache, store, webhooks, rate limiter and handlers.
