-- ============================================================
-- 044_defi_yield_latest.sql
-- Latest snapshot per yield pool from DefiLlama /pools.
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_yield_latest (
    pool                TEXT        PRIMARY KEY,
    chain               TEXT        NOT NULL,
    project             TEXT        NOT NULL,
    symbol              TEXT,
    protocol_slug       TEXT        REFERENCES defi_protocols(slug) ON DELETE SET NULL,

    tvl_usd             NUMERIC(30, 2) NOT NULL DEFAULT 0,
    apy                 NUMERIC(20, 10),
    apy_base            NUMERIC(20, 10),
    apy_reward          NUMERIC(20, 10),
    apy_pct_1d          NUMERIC(20, 10),
    apy_pct_7d          NUMERIC(20, 10),
    apy_pct_30d         NUMERIC(20, 10),
    apy_mean_30d        NUMERIC(20, 10),
    volume_usd_1d       NUMERIC(30, 2),
    volume_usd_7d       NUMERIC(30, 2),

    stablecoin          BOOLEAN     NOT NULL DEFAULT FALSE,
    il_risk             TEXT,
    exposure            TEXT,
    reward_tokens       TEXT[]      NOT NULL DEFAULT '{}',
    underlying_tokens   TEXT[]      NOT NULL DEFAULT '{}',
    predictions         JSONB       NOT NULL DEFAULT '{}',
    pool_meta           JSONB       NOT NULL DEFAULT '{}',
    outlier             BOOLEAN     NOT NULL DEFAULT FALSE,
    count               INTEGER,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    synced_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_defi_yield_latest_chain
    ON defi_yield_latest (chain, tvl_usd DESC);

CREATE INDEX IF NOT EXISTS idx_defi_yield_latest_project
    ON defi_yield_latest (project, tvl_usd DESC);

COMMENT ON TABLE defi_yield_latest IS 'Latest yield pool snapshot from DefiLlama /pools.';
