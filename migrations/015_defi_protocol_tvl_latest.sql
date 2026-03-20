-- ============================================================
-- 015_defi_protocol_tvl_latest.sql
-- Latest TVL snapshot per protocol.
-- Upserted every 1 hour by engine/defi/protocols/tvl.go
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_protocol_tvl_latest (
    slug                TEXT        PRIMARY KEY REFERENCES defi_protocols(slug) ON DELETE CASCADE,

    tvl_usd             NUMERIC(30, 2) NOT NULL DEFAULT 0,
    tvl_change_1d_pct   NUMERIC(10, 4),
    tvl_change_7d_pct   NUMERIC(10, 4),

    -- Per-chain TVL breakdown
    chain_tvls          JSONB       NOT NULL DEFAULT '{}',
    -- shape: { "ethereum": 1234567890.12, "arbitrum": 987654321.00, ... }

    -- Token price context
    token_price_usd     NUMERIC(30, 10),
    mcap_tvl_ratio      NUMERIC(10, 4),              -- market cap / TVL

    -- Fees & revenue (populated by fees.go if available)
    fees_24h_usd        NUMERIC(30, 2),
    revenue_24h_usd     NUMERIC(30, 2),
    fees_7d_usd         NUMERIC(30, 2),
    revenue_7d_usd      NUMERIC(30, 2),

    synced_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- TVL ranking (DeFi overview page)
CREATE INDEX IF NOT EXISTS idx_protocol_tvl_latest_rank
    ON defi_protocol_tvl_latest (tvl_usd DESC);

-- Recently synced check
CREATE INDEX IF NOT EXISTS idx_protocol_tvl_latest_synced
    ON defi_protocol_tvl_latest (synced_at DESC);

COMMENT ON TABLE defi_protocol_tvl_latest IS 'Latest TVL per protocol. 1h refresh. chain_tvls JSONB has per-chain breakdown. Merged with defi_protocols for API response.';
