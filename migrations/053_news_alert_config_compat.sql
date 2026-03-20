-- ============================================================
-- 053_news_alert_config_compat.sql
-- Compatibility migration for live DBs missing the news alert config table.
-- ============================================================

DO $$
BEGIN
    CREATE TYPE news_alert_type AS ENUM (
        'breaking_news',
        'regulation',
        'exploit_hack',
        'whale_movement',
        'high_importance',
        'coin_mention_spike',
        'depeg_news'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS news_alert_config (
    id                   SERIAL          PRIMARY KEY,
    alert_type           news_alert_type NOT NULL,
    target_coin_id       TEXT,
    target_slug          TEXT,
    min_importance_score NUMERIC(6, 3) DEFAULT 70.0,
    min_mention_count    INTEGER DEFAULT NULL,
    discord_webhook_url  TEXT,
    telegram_channel_id  TEXT,
    cooldown_seconds     INTEGER NOT NULL DEFAULT 1800,
    is_enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO news_alert_config (alert_type, min_importance_score, cooldown_seconds)
VALUES
    ('breaking_news',   80.0, 900),
    ('regulation',      70.0, 3600),
    ('exploit_hack',    60.0, 600),
    ('high_importance', 75.0, 1800)
ON CONFLICT DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_news_alert_config_type
    ON news_alert_config (alert_type)
    WHERE is_enabled = TRUE;

COMMENT ON TABLE news_alert_config IS 'News alert thresholds per type. Cooldown in Valkey. Seeded with global defaults; per-coin overrides can be added.';
