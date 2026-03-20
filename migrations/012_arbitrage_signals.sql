-- ============================================================
-- 012_arbitrage_signals.sql
-- Detected cross-exchange arbitrage opportunities.
-- Written by ArbitrageEngine every 5 seconds when threshold met.
-- ============================================================

CREATE TYPE arb_signal_status AS ENUM (
    'detected',     -- fresh signal, not yet acted on
    'alerted',      -- Discord/Telegram alert sent
    'expired',      -- spread closed before action
    'executed'      -- trade executed (future)
);

CREATE TABLE IF NOT EXISTS arbitrage_signals (
    id              BIGSERIAL   PRIMARY KEY,
    coin_id         TEXT        NOT NULL REFERENCES coins(coin_id) ON DELETE CASCADE,
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Opportunity definition
    buy_exchange    TEXT        NOT NULL,             -- "binance"
    sell_exchange   TEXT        NOT NULL,             -- "bybit"
    buy_symbol      TEXT        NOT NULL,             -- "BTCUSDT"
    sell_symbol     TEXT        NOT NULL,

    -- Prices at detection
    buy_price       NUMERIC(30, 10) NOT NULL,
    sell_price      NUMERIC(30, 10) NOT NULL,
    spread_pct      NUMERIC(10, 6)  NOT NULL
        GENERATED ALWAYS AS (
            ((sell_price - buy_price) / buy_price) * 100
        ) STORED,

    -- Depth at detection
    buy_depth_usd   NUMERIC(30, 2),
    sell_depth_usd  NUMERIC(30, 2),
    max_size_usd    NUMERIC(30, 2),                   -- min(buy_depth, sell_depth)

    -- Estimated net after fees
    estimated_fee_pct   NUMERIC(8, 4),
    net_spread_pct      NUMERIC(10, 6),

    status          arb_signal_status NOT NULL DEFAULT 'detected',
    alerted_at      TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,                      -- NULL = no expiry set yet

    -- Lifecycle tracking
    spread_at_close NUMERIC(10, 6),                   -- spread when signal expired
    duration_seconds INTEGER                           -- how long opportunity lasted
);

-- Active signal lookup
CREATE INDEX IF NOT EXISTS idx_arb_signals_status_time
    ON arbitrage_signals (status, detected_at DESC);

-- Per-coin history
CREATE INDEX IF NOT EXISTS idx_arb_signals_coin
    ON arbitrage_signals (coin_id, detected_at DESC);

-- Best opportunities
CREATE INDEX IF NOT EXISTS idx_arb_signals_spread
    ON arbitrage_signals (net_spread_pct DESC)
    WHERE status = 'detected';

COMMENT ON TABLE arbitrage_signals IS 'Cross-exchange arb signals from ArbitrageEngine. 5s scan interval. Triggers Discord + Telegram alert with cooldown.';
