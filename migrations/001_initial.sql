CREATE TABLE rate_snapshots (
    id          BIGSERIAL PRIMARY KEY,
    pair        VARCHAR(16) NOT NULL,
    amount_tier VARCHAR(16) NOT NULL,
    buy_rate    DECIMAL(18, 4) NOT NULL,
    sell_rate   DECIMAL(18, 4) NOT NULL,
    mid_rate    DECIMAL(18, 4) NOT NULL,
    spread      DECIMAL(18, 4) NOT NULL,
    spread_pct  DECIMAL(8, 4) NOT NULL,
    base_rate   DECIMAL(18, 4),
    source      VARCHAR(32) NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_snapshots_pair_time ON rate_snapshots (pair, captured_at DESC);
CREATE INDEX idx_snapshots_pair_tier_time ON rate_snapshots (pair, amount_tier, captured_at DESC);

CREATE TABLE subscribers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    api_key     VARCHAR(64) NOT NULL UNIQUE,
    webhook_url TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscribers_api_key ON subscribers (api_key) WHERE is_active = true;
