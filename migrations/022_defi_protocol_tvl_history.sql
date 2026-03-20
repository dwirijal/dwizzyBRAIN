-- ============================================================
-- 022_defi_protocol_tvl_history.sql
-- Full TVL history for top 50 protocols.
-- TimescaleDB hypertable — daily granularity from DefiLlama.
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_protocol_tvl_history (
    time            TIMESTAMPTZ NOT NULL,
    slug            TEXT        NOT NULL REFERENCES defi_protocols(slug) ON DELETE CASCADE,
    tvl_usd         NUMERIC(30, 2) NOT NULL,
    chain_tvls      JSONB       DEFAULT '{}',          -- per-chain breakdown at this timestamp

    PRIMARY KEY (time, slug)
);

SELECT create_hypertable(
    'defi_protocol_tvl_history',
    'time',
    chunk_time_interval => INTERVAL '30 days',
    if_not_exists => TRUE
);

ALTER TABLE defi_protocol_tvl_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'slug',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('defi_protocol_tvl_history', INTERVAL '30 days', if_not_exists => TRUE);

-- No hard retention — keep full history for top 50
-- Other protocols: FullHistoryBackfill skips them per coverage tier

-- Protocol TVL chart query
CREATE INDEX IF NOT EXISTS idx_defi_protocol_tvl_hist_slug_time
    ON defi_protocol_tvl_history (slug, time DESC);

COMMENT ON TABLE defi_protocol_tvl_history IS 'Full TVL history for top 50 protocols. Daily granularity from DefiLlama /protocol/{slug}. No retention limit.';
