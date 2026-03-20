-- ============================================================
-- 060_quant_candle_indicators.sql
-- Quant indicator snapshot table for per-candle technical analysis.
-- Extends the existing ohlcv hypertable with computed indicator state.
-- ============================================================

CREATE TABLE IF NOT EXISTS candle_indicators (
    time            TIMESTAMPTZ NOT NULL,
    symbol          TEXT NOT NULL,
    timeframe       TEXT NOT NULL,

    -- Trend
    ema_9           NUMERIC(30, 10),
    ema_21          NUMERIC(30, 10),
    ema_50          NUMERIC(30, 10),
    ema_200         NUMERIC(30, 10),
    sma_50          NUMERIC(30, 10),
    sma_200         NUMERIC(30, 10),
    vwap            NUMERIC(30, 10),
    supertrend      NUMERIC(30, 10),
    supertrend_dir  SMALLINT,                   -- 1 bullish, -1 bearish
    adx             NUMERIC(12, 6),
    ichimoku_tenkan NUMERIC(30, 10),
    ichimoku_kijun  NUMERIC(30, 10),
    ichimoku_senkou_a NUMERIC(30, 10),
    ichimoku_senkou_b NUMERIC(30, 10),

    -- Momentum
    rsi_14          NUMERIC(12, 6),
    rsi_2           NUMERIC(12, 6),
    macd            NUMERIC(30, 10),
    macd_signal     NUMERIC(30, 10),
    macd_hist       NUMERIC(30, 10),
    stoch_k         NUMERIC(12, 6),
    stoch_d         NUMERIC(12, 6),
    cci_20          NUMERIC(12, 6),
    roc_10          NUMERIC(12, 6),
    mfi_14          NUMERIC(12, 6),

    -- Volatility
    atr_14          NUMERIC(30, 10),
    bb_upper        NUMERIC(30, 10),
    bb_mid          NUMERIC(30, 10),
    bb_lower        NUMERIC(30, 10),
    bb_pct_b        NUMERIC(12, 6),
    bb_width        NUMERIC(12, 6),
    kc_upper        NUMERIC(30, 10),
    kc_lower        NUMERIC(30, 10),
    hist_vol_20     NUMERIC(12, 6),

    -- Volume
    obv             NUMERIC(30, 10),
    cmf_20          NUMERIC(12, 6),
    volume_sma20    NUMERIC(30, 10),
    volume_ratio    NUMERIC(12, 6),
    volume_trend    NUMERIC(12, 6),

    -- Support / resistance
    pivot_classic   NUMERIC(30, 10),
    pivot_r1        NUMERIC(30, 10),
    pivot_s1        NUMERIC(30, 10),
    fib_382         NUMERIC(30, 10),
    fib_500         NUMERIC(30, 10),
    fib_618         NUMERIC(30, 10),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (time, symbol, timeframe)
);

SELECT create_hypertable(
    'candle_indicators',
    'time',
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_candle_indicators_symbol_tf_time
    ON candle_indicators (symbol, timeframe, time DESC);

COMMENT ON TABLE candle_indicators IS 'Quant indicator snapshots for per-candle technical analysis.';
