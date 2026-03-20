-- ============================================================
-- 010_ohlcv_hypertable.sql
-- OHLCV candles per coin per exchange per timeframe.
-- Retention policy varies by timeframe (shorter TF = less history).
-- ============================================================

CREATE TABLE IF NOT EXISTS ohlcv (
    time        TIMESTAMPTZ NOT NULL,
    coin_id     TEXT        NOT NULL,
    exchange    TEXT        NOT NULL,                 -- "binance", "bybit", "okx"
    symbol      TEXT        NOT NULL,                 -- "BTCUSDT"
    timeframe   TEXT        NOT NULL,                 -- "1m", "5m", "15m", "1h", "4h", "1d"
    open        NUMERIC(30, 10) NOT NULL,
    high        NUMERIC(30, 10) NOT NULL,
    low         NUMERIC(30, 10) NOT NULL,
    close       NUMERIC(30, 10) NOT NULL,
    volume      NUMERIC(30, 4)  NOT NULL,
    quote_volume NUMERIC(30, 4),                      -- volume in quote currency (USDT)
    trades      INTEGER,                              -- number of trades in candle
    is_closed   BOOLEAN     NOT NULL DEFAULT TRUE,    -- FALSE = current forming candle

    PRIMARY KEY (time, coin_id, exchange, timeframe)
);

SELECT create_hypertable(
    'ohlcv',
    'time',
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

-- Compression: compress chunks older than 7 days
ALTER TABLE ohlcv SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'coin_id, exchange, timeframe',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('ohlcv', INTERVAL '7 days', if_not_exists => TRUE);

-- Retention per timeframe (enforced by OHLCVService scheduler)
-- 1m  → 7 days
-- 5m  → 14 days
-- 15m → 30 days
-- 1h  → 90 days
-- 4h  → 180 days
-- 1d  → 2 years (730 days)
-- Retention policy set at 2 years; fine-grained cleanup via scheduled job.
SELECT add_retention_policy('ohlcv', INTERVAL '730 days', if_not_exists => TRUE);

-- Primary query pattern: coin + timeframe + time range
CREATE INDEX IF NOT EXISTS idx_ohlcv_coin_tf_time
    ON ohlcv (coin_id, timeframe, time DESC);

-- Exchange-specific backfill check
CREATE INDEX IF NOT EXISTS idx_ohlcv_exchange_symbol_tf
    ON ohlcv (exchange, symbol, timeframe, time DESC);

COMMENT ON TABLE ohlcv IS 'OHLCV candles. Multi-timeframe: 1m/5m/15m/1h/4h/1d. Retention: 7d (1m) to 730d (1d). BackfillOHLCV fills history for top 100 coins.';
