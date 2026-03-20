-- ============================================================
-- 023_defi_dex_volume_history.sql
-- Full volume history for top 30 DEXs.
-- Daily granularity from DefiLlama /summary/dexs/{slug}
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_dex_volume_history (
    time            TIMESTAMPTZ NOT NULL,
    slug            TEXT        NOT NULL REFERENCES defi_dex_latest(slug) ON DELETE CASCADE,
    volume_usd      NUMERIC(30, 2) NOT NULL,
    chain_volumes   JSONB       DEFAULT '{}',

    PRIMARY KEY (time, slug)
);

SELECT create_hypertable(
    'defi_dex_volume_history',
    'time',
    chunk_time_interval => INTERVAL '30 days',
    if_not_exists => TRUE
);

ALTER TABLE defi_dex_volume_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'slug',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('defi_dex_volume_history', INTERVAL '30 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_defi_dex_vol_hist_slug_time
    ON defi_dex_volume_history (slug, time DESC);

COMMENT ON TABLE defi_dex_volume_history IS 'Full volume history for top 30 DEXs. Daily granularity. Used for DEX volume chart on /v1/defi/dex page.';
