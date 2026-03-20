-- ============================================================
-- 013_arbitrage_config.sql
-- Per-coin alert thresholds for ArbitrageEngine.
-- Allows fine-tuning sensitivity per asset class.
-- ============================================================

CREATE TABLE IF NOT EXISTS arbitrage_config (
    coin_id             TEXT        PRIMARY KEY REFERENCES coins(coin_id) ON DELETE CASCADE,

    -- Spread thresholds
    min_spread_pct      NUMERIC(8, 4)   NOT NULL DEFAULT 0.30,  -- minimum gross spread %
    min_net_spread_pct  NUMERIC(8, 4)   NOT NULL DEFAULT 0.10,  -- after estimated fees

    -- Depth requirements (USD)
    min_depth_usd       NUMERIC(20, 2)  NOT NULL DEFAULT 10000, -- both sides must have this depth

    -- Cooldown: don't re-alert same pair within N seconds
    cooldown_seconds    INTEGER         NOT NULL DEFAULT 300,

    -- Exchange pair restrictions (NULL = allow all)
    allowed_buy_exchanges  TEXT[]       DEFAULT NULL,
    allowed_sell_exchanges TEXT[]       DEFAULT NULL,

    -- Enable/disable per coin
    is_enabled          BOOLEAN         NOT NULL DEFAULT TRUE,

    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- Seed default config for top 20 coins (engine will upsert on startup)
-- INSERT handled by engine/market/arbitrage/config.go at startup

CREATE INDEX IF NOT EXISTS idx_arb_config_enabled
    ON arbitrage_config (coin_id)
    WHERE is_enabled = TRUE;

COMMENT ON TABLE arbitrage_config IS 'Per-coin ArbitrageEngine thresholds: min spread, depth, cooldown. Upserted by engine at startup with defaults.';
