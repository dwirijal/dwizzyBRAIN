-- ============================================================
-- 024_defi_chain_tvl_history.sql
-- Full TVL history for top 15 chains.
-- Daily granularity from DefiLlama /v1/historicalChainTvl/{chain}
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_chain_tvl_history (
    time            TIMESTAMPTZ NOT NULL,
    chain_id        TEXT        NOT NULL REFERENCES defi_chain_tvl_latest(chain_id) ON DELETE CASCADE,
    tvl_usd         NUMERIC(30, 2) NOT NULL,

    PRIMARY KEY (time, chain_id)
);

SELECT create_hypertable(
    'defi_chain_tvl_history',
    'time',
    chunk_time_interval => INTERVAL '30 days',
    if_not_exists => TRUE
);

ALTER TABLE defi_chain_tvl_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'chain_id',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('defi_chain_tvl_history', INTERVAL '30 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_defi_chain_tvl_hist_chain_time
    ON defi_chain_tvl_history (chain_id, time DESC);

COMMENT ON TABLE defi_chain_tvl_history IS 'Full TVL history for top 15 chains. Daily granularity. No retention limit.';
