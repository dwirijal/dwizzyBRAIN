-- ============================================================
-- 046_defi_stablecoin_compat.sql
-- Compatibility layer for the live coins table shape.
-- The current database uses coins(id) as the primary key and
-- a generated coin_id column. Stablecoin tables reference id.
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_stablecoin_backing (
    id              BIGSERIAL   PRIMARY KEY,
    coin_id         TEXT        NOT NULL REFERENCES coins(id) ON DELETE CASCADE,
    snapshot_date   DATE        NOT NULL,

    peg_type        TEXT        NOT NULL DEFAULT 'USD',
    peg_mechanism   TEXT,

    price_usd       NUMERIC(20, 10),
    depeg_pct       NUMERIC(10, 6) GENERATED ALWAYS AS (
        ABS(COALESCE(price_usd, 1.0) - 1.0) * 100
    ) STORED,

    mcap_usd        NUMERIC(30, 2),
    circulating     NUMERIC(30, 2),

    backing_composition JSONB   NOT NULL DEFAULT '{}',

    attestation_url TEXT,
    attested_at     TIMESTAMPTZ,

    synced_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (coin_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_stablecoin_backing_coin_date
    ON defi_stablecoin_backing (coin_id, snapshot_date DESC);

CREATE INDEX IF NOT EXISTS idx_stablecoin_backing_depeg
    ON defi_stablecoin_backing (coin_id, synced_at DESC);

COMMENT ON TABLE defi_stablecoin_backing IS 'Stablecoin backing composition + depeg tracking. Daily snapshots. depeg_pct generated column drives DepegScanner alerts.';

CREATE TABLE IF NOT EXISTS defi_stable_mcap_history (
    time            TIMESTAMPTZ NOT NULL,
    coin_id         TEXT        NOT NULL REFERENCES coins(id) ON DELETE CASCADE,
    mcap_usd        NUMERIC(30, 2) NOT NULL,
    circulating     NUMERIC(30, 2),
    price_usd       NUMERIC(20, 10),

    PRIMARY KEY (time, coin_id)
);

SELECT create_hypertable(
    'defi_stable_mcap_history',
    'time',
    chunk_time_interval => INTERVAL '30 days',
    if_not_exists => TRUE
);

ALTER TABLE defi_stable_mcap_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'coin_id',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('defi_stable_mcap_history', INTERVAL '30 days', if_not_exists => TRUE);
SELECT add_retention_policy('defi_stable_mcap_history', INTERVAL '730 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_defi_stable_mcap_hist_coin_time
    ON defi_stable_mcap_history (coin_id, time DESC);

COMMENT ON TABLE defi_stable_mcap_history IS '2-year stablecoin mcap + price history. Daily granularity. price_usd allows historical depeg analysis.';
