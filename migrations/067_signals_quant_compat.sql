-- ============================================================
-- 067_signals_quant_compat.sql
-- Compatibility layer for the quant signal API.
-- The live signals table may predate the current quant contract,
-- so we add missing columns without changing the existing FK shape.
-- ============================================================

ALTER TABLE signals
    ADD COLUMN IF NOT EXISTS coin_id TEXT,
    ADD COLUMN IF NOT EXISTS exchange TEXT,
    ADD COLUMN IF NOT EXISTS symbol TEXT,
    ADD COLUMN IF NOT EXISTS quant_score NUMERIC(5, 2),
    ADD COLUMN IF NOT EXISTS rsi_14 NUMERIC(6, 2),
    ADD COLUMN IF NOT EXISTS macd_line NUMERIC(20, 8),
    ADD COLUMN IF NOT EXISTS macd_signal NUMERIC(20, 8),
    ADD COLUMN IF NOT EXISTS macd_hist NUMERIC(20, 8),
    ADD COLUMN IF NOT EXISTS bb_upper NUMERIC(30, 10),
    ADD COLUMN IF NOT EXISTS bb_lower NUMERIC(30, 10),
    ADD COLUMN IF NOT EXISTS bb_mid NUMERIC(30, 10),
    ADD COLUMN IF NOT EXISTS ema_9 NUMERIC(30, 10),
    ADD COLUMN IF NOT EXISTS ema_21 NUMERIC(30, 10),
    ADD COLUMN IF NOT EXISTS ema_200 NUMERIC(30, 10),
    ADD COLUMN IF NOT EXISTS funding_rate NUMERIC(12, 8),
    ADD COLUMN IF NOT EXISTS funding_sentiment TEXT,
    ADD COLUMN IF NOT EXISTS volume_spike BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS price_deviation BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS anomaly_score NUMERIC(6, 3),
    ADD COLUMN IF NOT EXISTS price_at_signal NUMERIC(30, 10);

ALTER TABLE signals
    DROP CONSTRAINT IF EXISTS signals_strength_check;

ALTER TABLE signals
    ALTER COLUMN strength TYPE TEXT USING strength::text;

CREATE INDEX IF NOT EXISTS idx_signals_symbol_time_exchange
    ON signals (symbol, timeframe, exchange, created_at DESC);
