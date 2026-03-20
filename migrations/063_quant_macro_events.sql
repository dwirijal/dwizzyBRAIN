-- ============================================================
-- 063_quant_macro_events.sql
-- Macro event series snapshots for quant event labeling.
-- ============================================================

CREATE TABLE IF NOT EXISTS macro_events (
    time            TIMESTAMPTZ NOT NULL,
    series_id       TEXT NOT NULL,
    series_name     TEXT NOT NULL,
    source          TEXT NOT NULL DEFAULT 'fred',
    event_type      TEXT NOT NULL DEFAULT 'macro',
    value           NUMERIC(30, 10) NOT NULL,
    importance      SMALLINT NOT NULL DEFAULT 1,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (time, series_id)
);

SELECT create_hypertable(
    'macro_events',
    'time',
    chunk_time_interval => INTERVAL '30 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_macro_events_series_time
    ON macro_events (series_id, time DESC);

COMMENT ON TABLE macro_events IS 'Macro event series snapshots used to derive candle event labels.';
