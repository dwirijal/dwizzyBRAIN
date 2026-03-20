-- ============================================================
-- 026_defi_fees_history.sql
-- 1-year fees + revenue history for top 100 protocols.
-- Daily granularity from DefiLlama /summary/fees/{slug}
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_fees_history (
    time            TIMESTAMPTZ NOT NULL,
    slug            TEXT        NOT NULL REFERENCES defi_protocols(slug) ON DELETE CASCADE,
    fees_usd        NUMERIC(30, 2),
    revenue_usd     NUMERIC(30, 2),                   -- protocol revenue (subset of fees)
    holder_revenue_usd NUMERIC(30, 2),                -- revenue distributed to token holders

    PRIMARY KEY (time, slug)
);

SELECT create_hypertable(
    'defi_fees_history',
    'time',
    chunk_time_interval => INTERVAL '30 days',
    if_not_exists => TRUE
);

ALTER TABLE defi_fees_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'slug',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('defi_fees_history', INTERVAL '30 days', if_not_exists => TRUE);

-- Keep 1 year of fees history
SELECT add_retention_policy('defi_fees_history', INTERVAL '365 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_defi_fees_hist_slug_time
    ON defi_fees_history (slug, time DESC);

COMMENT ON TABLE defi_fees_history IS '1-year fees/revenue history for top 100 protocols. Daily from DefiLlama. Used for protocol detail page revenue chart.';
