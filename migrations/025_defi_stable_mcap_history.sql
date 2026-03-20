-- ============================================================
-- 025_defi_stable_mcap_history.sql
-- 2-year market cap history for all tracked stablecoins.
-- Daily granularity from DefiLlama /stablecoins/chart/{asset}
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_stable_mcap_history (
    time            TIMESTAMPTZ NOT NULL,
    coin_id         TEXT        NOT NULL REFERENCES coins(coin_id) ON DELETE CASCADE,
    mcap_usd        NUMERIC(30, 2) NOT NULL,
    circulating     NUMERIC(30, 2),
    price_usd       NUMERIC(20, 10),                  -- for depeg history

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

-- Keep 2 years of stablecoin history
SELECT add_retention_policy('defi_stable_mcap_history', INTERVAL '730 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_defi_stable_mcap_hist_coin_time
    ON defi_stable_mcap_history (coin_id, time DESC);

COMMENT ON TABLE defi_stable_mcap_history IS '2-year stablecoin mcap + price history. Daily granularity. price_usd allows historical depeg analysis.';
