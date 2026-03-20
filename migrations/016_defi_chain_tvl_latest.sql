-- ============================================================
-- 016_defi_chain_tvl_latest.sql
-- Latest TVL snapshot per blockchain.
-- Upserted every 1 hour from DefiLlama /v1/chains
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_chain_tvl_latest (
    chain_id            TEXT        PRIMARY KEY,      -- "ethereum", "bsc", "arbitrum", "solana"
    name                TEXT        NOT NULL,         -- display name
    coin_id             TEXT        REFERENCES coins(coin_id) ON DELETE SET NULL,
    logo_url            TEXT,

    tvl_usd             NUMERIC(30, 2) NOT NULL DEFAULT 0,
    tvl_change_1d_pct   NUMERIC(10, 4),
    tvl_change_7d_pct   NUMERIC(10, 4),

    -- Protocol count on this chain
    protocol_count      INTEGER,

    -- Token market data context
    token_price_usd     NUMERIC(30, 10),
    token_symbol        TEXT,

    synced_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chain_tvl_latest_rank
    ON defi_chain_tvl_latest (tvl_usd DESC);

COMMENT ON TABLE defi_chain_tvl_latest IS 'Latest TVL per chain. 1h refresh. Used by /v1/defi/chain endpoint.';
