-- ============================================================
-- 065_signals_symbol_timeframe_exchange_time.sql
-- Supports quant signal latest/history/summary reads.
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_signals_symbol_timeframe_exchange_time
    ON signals (symbol, timeframe, exchange, created_at DESC);

COMMENT ON INDEX idx_signals_symbol_timeframe_exchange_time IS
    'Supports quant signal latest/history/summary reads by symbol/timeframe/exchange.';
