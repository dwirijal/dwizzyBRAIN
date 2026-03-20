-- ============================================================
-- 040_phase2_task3_ohlcv_compat.sql
-- Compatibility upgrade for the legacy ohlcv hypertable so it can
-- support the Phase 2 OHLCV service without dropping old data.
-- ============================================================

ALTER TABLE ohlcv
    ADD COLUMN IF NOT EXISTS symbol TEXT,
    ADD COLUMN IF NOT EXISTS timeframe TEXT,
    ADD COLUMN IF NOT EXISTS quote_volume NUMERIC(30, 4),
    ADD COLUMN IF NOT EXISTS is_closed BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE ohlcv
SET timeframe = "interval"
WHERE timeframe IS NULL AND "interval" IS NOT NULL;

UPDATE ohlcv
SET symbol = COALESCE(symbol, coin_id)
WHERE symbol IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_ohlcv_time_coin_exchange_timeframe
    ON ohlcv ("time", coin_id, exchange, timeframe);

CREATE INDEX IF NOT EXISTS idx_ohlcv_coin_tf_time
    ON ohlcv (coin_id, timeframe, "time" DESC);

CREATE INDEX IF NOT EXISTS idx_ohlcv_exchange_symbol_tf
    ON ohlcv (exchange, symbol, timeframe, "time" DESC);
