-- ============================================================
-- 045_defi_yield_history.sql
-- Daily history per yield pool from DefiLlama /chart/{pool}.
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_yield_history (
    time            TIMESTAMPTZ NOT NULL,
    pool            TEXT        NOT NULL REFERENCES defi_yield_latest(pool) ON DELETE CASCADE,
    chain           TEXT        NOT NULL,
    project         TEXT        NOT NULL,
    symbol          TEXT,
    tvl_usd         NUMERIC(30, 2) NOT NULL,
    apy             NUMERIC(20, 10),
    apy_base        NUMERIC(20, 10),
    apy_reward      NUMERIC(20, 10),
    metadata        JSONB       NOT NULL DEFAULT '{}',

    PRIMARY KEY (time, pool)
);

SELECT create_hypertable(
    'defi_yield_history',
    'time',
    chunk_time_interval => INTERVAL '30 days',
    if_not_exists => TRUE
);

ALTER TABLE defi_yield_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'pool',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('defi_yield_history', INTERVAL '30 days', if_not_exists => TRUE);
SELECT add_retention_policy('defi_yield_history', INTERVAL '365 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_defi_yield_hist_pool_time
    ON defi_yield_history (pool, time DESC);

COMMENT ON TABLE defi_yield_history IS 'Daily yield history per pool from DefiLlama /chart/{pool}.';
