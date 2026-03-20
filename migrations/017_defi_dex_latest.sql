-- ============================================================
-- 017_defi_dex_latest.sql
-- Latest volume snapshot per DEX.
-- Upserted every 15 minutes from DefiLlama /overview/dexs
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_dex_latest (
    slug                TEXT        PRIMARY KEY,      -- "uniswap-v3", "pancakeswap"
    name                TEXT        NOT NULL,
    logo_url            TEXT,
    chains              TEXT[]      NOT NULL DEFAULT '{}',
    protocol_slug       TEXT        REFERENCES defi_protocols(slug) ON DELETE SET NULL,

    -- Volume
    volume_24h_usd      NUMERIC(30, 2) NOT NULL DEFAULT 0,
    volume_7d_usd       NUMERIC(30, 2),
    volume_change_1d_pct NUMERIC(10, 4),
    volume_change_7d_pct NUMERIC(10, 4),

    -- Per-chain breakdown
    chain_volumes       JSONB       NOT NULL DEFAULT '{}',

    -- Market share
    market_share_pct    NUMERIC(8, 4),

    synced_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dex_latest_volume
    ON defi_dex_latest (volume_24h_usd DESC);

CREATE INDEX IF NOT EXISTS idx_dex_latest_chains
    ON defi_dex_latest USING GIN (chains);

COMMENT ON TABLE defi_dex_latest IS 'DEX volume snapshot. 15m refresh. Used by /v1/defi/dex endpoint.';
