-- ============================================================
-- 035_signals_table.sql
-- Quant signal history from Python quant service.
-- Published to ch:signal:processed:{symbol}, consumed by
-- engine/agent and api/ws. Persisted here for history/backtest.
-- ============================================================

CREATE TABLE IF NOT EXISTS signals (
    id              BIGSERIAL   PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    coin_id         TEXT        NOT NULL REFERENCES coins(coin_id) ON DELETE CASCADE,
    exchange        TEXT        NOT NULL,
    symbol          TEXT        NOT NULL,
    timeframe       TEXT        NOT NULL,             -- "1m", "5m", "1h", "4h", "1d"

    -- Quant composite score
    quant_score     NUMERIC(5, 2),                   -- 0.0 – 100.0
    signal_type     TEXT        NOT NULL,             -- "buy", "sell", "hold", "watch"
    strength        TEXT,                             -- "strong", "moderate", "weak"

    -- Individual indicators
    rsi_14          NUMERIC(6, 2),
    macd_line       NUMERIC(20, 8),
    macd_signal     NUMERIC(20, 8),
    macd_hist       NUMERIC(20, 8),
    bb_upper        NUMERIC(30, 10),
    bb_lower        NUMERIC(30, 10),
    bb_mid          NUMERIC(30, 10),
    ema_9           NUMERIC(30, 10),
    ema_21          NUMERIC(30, 10),
    ema_200         NUMERIC(30, 10),

    -- Funding rate (futures only)
    funding_rate    NUMERIC(12, 8),
    funding_sentiment TEXT,                           -- "long_bias", "short_bias", "neutral"

    -- Anomaly flags
    volume_spike    BOOLEAN     NOT NULL DEFAULT FALSE,
    price_deviation BOOLEAN     NOT NULL DEFAULT FALSE,
    anomaly_score   NUMERIC(6, 3),

    -- Price context at signal time
    price_at_signal NUMERIC(30, 10)
);

-- Time-range + coin queries
CREATE INDEX IF NOT EXISTS idx_signals_coin_time
    ON signals (coin_id, created_at DESC);

-- Signal type filter
CREATE INDEX IF NOT EXISTS idx_signals_type_time
    ON signals (signal_type, created_at DESC);

-- Recent signals for live feed
CREATE INDEX IF NOT EXISTS idx_signals_time
    ON signals (created_at DESC);

COMMENT ON TABLE signals IS 'Quant signal history from Python service. RSI/MACD/BB/EMA + funding rate + anomaly. Drives AgentRouter for LLM analysis requests.';
