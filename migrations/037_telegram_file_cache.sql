-- ============================================================
-- 037_telegram_file_cache.sql
-- Telegram file_id cache for generated charts, CSV exports,
-- and backup files. file_id is permanent — no re-upload needed.
-- Mirrors Valkey key: telegram:file:{file_key} (no TTL).
-- ============================================================

CREATE TYPE telegram_file_type AS ENUM (
    'chart',        -- generated price/indicator chart image
    'csv_export',   -- OHLCV or signal CSV export
    'backup_db',    -- pg_dump gzip backup
    'backup_valkey',-- Valkey snapshot
    'report',       -- generated PDF/HTML report
    'other'
);

CREATE TABLE IF NOT EXISTS telegram_file_cache (
    id              BIGSERIAL   PRIMARY KEY,

    -- Lookup key used by engine + Valkey mirror
    file_key        TEXT        NOT NULL UNIQUE,
    -- naming convention: "{type}:{coin_id_or_context}:{date_or_hash}"
    -- e.g. "chart:bitcoin:2026-03-18", "csv:BTCUSDT:1h:20260318", "backup:pg:20260318"

    -- Telegram identifiers
    file_id         TEXT        NOT NULL,             -- Telegram file_id (permanent)
    file_unique_id  TEXT,                             -- Telegram file_unique_id
    message_id      BIGINT,                           -- message ID in DISCORD_FILES_CHANNEL
    channel_id      TEXT,                             -- Telegram channel where file lives

    -- File metadata
    file_type       telegram_file_type NOT NULL DEFAULT 'other',
    file_name       TEXT,
    file_size_bytes BIGINT,
    mime_type       TEXT,

    -- Context
    coin_id         TEXT        REFERENCES coins(coin_id) ON DELETE SET NULL,
    timeframe       TEXT,
    date_context    DATE,                             -- date the chart/export represents

    -- Lifecycle
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ,
    access_count    INTEGER     NOT NULL DEFAULT 0
);

-- Lookup by file_key (primary access pattern)
CREATE INDEX IF NOT EXISTS idx_tg_file_cache_key
    ON telegram_file_cache (file_key);

-- Coin-based chart lookup
CREATE INDEX IF NOT EXISTS idx_tg_file_cache_coin
    ON telegram_file_cache (coin_id, file_type, date_context DESC)
    WHERE coin_id IS NOT NULL;

-- Backup files
CREATE INDEX IF NOT EXISTS idx_tg_file_cache_backup
    ON telegram_file_cache (file_type, uploaded_at DESC)
    WHERE file_type IN ('backup_db', 'backup_valkey');

COMMENT ON TABLE telegram_file_cache IS 'Permanent Telegram file_id store. Charts + CSVs + backups uploaded once, referenced forever. Mirrored in Valkey (no TTL).';
