-- ============================================================
-- 018_defi_protocol_coverage.sql
-- Tier classification + sync metadata per protocol.
-- Drives history backfill depth and polling frequency.
-- ============================================================

CREATE TYPE defi_coverage_tier AS ENUM ('top50', 'top300', 'other');

CREATE TABLE IF NOT EXISTS defi_protocol_coverage (
    slug                TEXT        PRIMARY KEY REFERENCES defi_protocols(slug) ON DELETE CASCADE,
    tier                defi_coverage_tier NOT NULL DEFAULT 'other',

    -- Sync timestamps
    tvl_last_synced_at      TIMESTAMPTZ,
    fees_last_synced_at     TIMESTAMPTZ,
    history_backfilled      BOOLEAN     NOT NULL DEFAULT FALSE,
    history_backfilled_at   TIMESTAMPTZ,
    history_start_date      DATE,                     -- earliest data available in TimescaleDB

    -- Data quality
    has_fees_data           BOOLEAN     NOT NULL DEFAULT FALSE,
    has_revenue_data        BOOLEAN     NOT NULL DEFAULT FALSE,
    has_token_data          BOOLEAN     NOT NULL DEFAULT FALSE,

    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_defi_coverage_tier
    ON defi_protocol_coverage (tier);

CREATE INDEX IF NOT EXISTS idx_defi_coverage_backfill
    ON defi_protocol_coverage (slug)
    WHERE history_backfilled = FALSE;

COMMENT ON TABLE defi_protocol_coverage IS 'Sync state per protocol. top50 gets full history, top300 gets 90d, other gets latest only.';
