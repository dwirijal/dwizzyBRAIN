-- ============================================================
-- 021_defi_alert_config.sql
-- Alert threshold configuration per type + target.
-- Read by TVLDropScanner, NewTop50Scanner, FeeSpikeScanner,
-- DepegScanner at startup and cached in Valkey.
-- ============================================================

CREATE TYPE defi_alert_type AS ENUM (
    'tvl_drop',
    'tvl_spike',
    'new_top50',
    'fee_spike',
    'depeg',
    'hack_detected',
    'whale_movement'
);

CREATE TABLE IF NOT EXISTS defi_alert_config (
    id                  SERIAL      PRIMARY KEY,
    alert_type          defi_alert_type NOT NULL,

    -- Target scope (NULL = global/all)
    target_slug         TEXT,                           -- specific protocol slug
    target_chain        TEXT,                           -- specific chain
    target_coin_id      TEXT        REFERENCES coins(coin_id) ON DELETE CASCADE,

    -- Threshold values (meaning varies by type)
    threshold_pct       NUMERIC(10, 4),                 -- e.g. TVL drop > 20%
    threshold_usd       NUMERIC(30, 2),                 -- e.g. TVL drop > $1M absolute
    threshold_abs       NUMERIC(20, 6),                 -- e.g. depeg > 0.01

    -- Alert targets
    discord_webhook_url TEXT,
    telegram_channel_id TEXT,

    -- Cooldown
    cooldown_seconds    INTEGER     NOT NULL DEFAULT 3600,

    is_enabled          BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (alert_type, COALESCE(target_slug, ''), COALESCE(target_chain, ''), COALESCE(target_coin_id::TEXT, ''))
);

CREATE INDEX IF NOT EXISTS idx_defi_alert_type
    ON defi_alert_config (alert_type)
    WHERE is_enabled = TRUE;

COMMENT ON TABLE defi_alert_config IS 'Alert thresholds for DeFi scanners. Cooldown tracked in Valkey key alert:defi:cooldown:{type}:{id}.';
