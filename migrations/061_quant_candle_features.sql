-- ============================================================
-- 061_quant_candle_features.sql
-- Quant derived features and candlestick/SMC flags.
-- ============================================================

CREATE TABLE IF NOT EXISTS candle_features (
    time                TIMESTAMPTZ NOT NULL,
    symbol              TEXT NOT NULL,
    timeframe           TEXT NOT NULL,

    -- Price action
    candle_body_pct     NUMERIC(12, 6),
    upper_wick_pct      NUMERIC(12, 6),
    lower_wick_pct      NUMERIC(12, 6),
    dist_from_ema9      NUMERIC(12, 6),
    dist_from_ema21     NUMERIC(12, 6),
    dist_from_ema50     NUMERIC(12, 6),
    dist_from_ema200    NUMERIC(12, 6),
    dist_from_vwap      NUMERIC(12, 6),
    bb_position         NUMERIC(12, 6),
    atr_ratio           NUMERIC(12, 6),
    kc_position         NUMERIC(12, 6),

    -- Momentum slopes
    rsi_slope           NUMERIC(12, 6),
    macd_hist_slope     NUMERIC(12, 6),
    obv_slope           NUMERIC(12, 6),

    -- Price change windows
    change_1h           NUMERIC(12, 6),
    change_4h           NUMERIC(12, 6),
    change_1d           NUMERIC(12, 6),
    change_1w           NUMERIC(12, 6),

    -- Candlestick pattern flags
    pattern_doji            BOOLEAN,
    pattern_hammer          BOOLEAN,
    pattern_shooting_star    BOOLEAN,
    pattern_engulfing        BOOLEAN,
    pattern_morning_star     BOOLEAN,
    pattern_evening_star     BOOLEAN,
    pattern_marubozu         BOOLEAN,
    pattern_inside_bar       BOOLEAN,
    pattern_pinbar           BOOLEAN,

    -- SMC flags
    smc_order_block         BOOLEAN,
    smc_fvg                 BOOLEAN,
    smc_bos                 BOOLEAN,
    smc_choch               BOOLEAN,
    smc_liquidity_sweep     BOOLEAN,
    smc_premium_zone        BOOLEAN,
    smc_discount_zone       BOOLEAN,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (time, symbol, timeframe)
);

SELECT create_hypertable(
    'candle_features',
    'time',
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_candle_features_symbol_tf_time
    ON candle_features (symbol, timeframe, time DESC);

COMMENT ON TABLE candle_features IS 'Derived candle features, candlestick pattern flags, and SMC labels.';
