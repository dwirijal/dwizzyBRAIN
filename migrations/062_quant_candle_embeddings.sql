-- ============================================================
-- 062_quant_candle_embeddings.sql
-- pgvector candle fingerprints for similarity search.
-- ============================================================

CREATE TABLE IF NOT EXISTS candle_embeddings (
    time        TIMESTAMPTZ NOT NULL,
    symbol      TEXT NOT NULL,
    timeframe   TEXT NOT NULL,
    embedding   vector(30),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (time, symbol, timeframe)
);

SELECT create_hypertable(
    'candle_embeddings',
    'time',
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_candle_embeddings_symbol_tf_time
    ON candle_embeddings (symbol, timeframe, time DESC);

CREATE INDEX IF NOT EXISTS idx_candle_embeddings_hnsw
    ON candle_embeddings
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

COMMENT ON TABLE candle_embeddings IS 'Quant fingerprint vectors for candle similarity search.';
