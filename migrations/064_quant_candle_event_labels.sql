-- ============================================================
-- 064_quant_candle_event_labels.sql
-- Per-candle macro event labels for quant backfill and pattern filtering.
-- ============================================================

CREATE TABLE IF NOT EXISTS candle_event_labels (
    time                    TIMESTAMPTZ NOT NULL,
    symbol                  TEXT NOT NULL,
    timeframe               TEXT NOT NULL,
    macro_environment       TEXT NOT NULL,
    proximity_label         TEXT NOT NULL,
    rate_direction          TEXT NOT NULL,
    rate_regime             TEXT NOT NULL,
    cpi_trend               TEXT NOT NULL,
    last_surprise_label     TEXT NOT NULL,
    last_surprise_value     NUMERIC(20, 8) NOT NULL DEFAULT 0,
    hours_to_event          NUMERIC(20, 8) NOT NULL DEFAULT 0,
    hours_from_event        NUMERIC(20, 8) NOT NULL DEFAULT 0,
    vol_context             TEXT NOT NULL,
    nearest_event_series_id TEXT NOT NULL DEFAULT '',
    nearest_event_time      TIMESTAMPTZ NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (time, symbol, timeframe)
);

SELECT create_hypertable(
    'candle_event_labels',
    'time',
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_candle_event_labels_symbol_tf_time
    ON candle_event_labels (symbol, timeframe, time DESC);

COMMENT ON TABLE candle_event_labels IS 'Macro event labels attached to each candle for filtering and pattern analysis.';
