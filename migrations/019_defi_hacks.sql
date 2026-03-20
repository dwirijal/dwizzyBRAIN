-- ============================================================
-- 019_defi_hacks.sql
-- Historical hack/exploit records from banterous/defihack-db.
-- Synced daily via GitHub raw content.
-- ============================================================

CREATE TYPE hack_type AS ENUM (
    'flash_loan',
    'rug_pull',
    'exploit',
    'oracle_manipulation',
    'bridge_hack',
    'private_key_compromise',
    'social_engineering',
    'other'
);

CREATE TABLE IF NOT EXISTS defi_hacks (
    id              BIGSERIAL   PRIMARY KEY,
    protocol_slug   TEXT        REFERENCES defi_protocols(slug) ON DELETE SET NULL,
    protocol_name   TEXT        NOT NULL,             -- raw name from defihack-db (may not match slug)
    hack_date       DATE        NOT NULL,
    hack_type       hack_type   NOT NULL DEFAULT 'other',

    -- Financials
    funds_lost_usd  NUMERIC(30, 2),
    funds_returned_usd NUMERIC(30, 2),
    net_loss_usd    NUMERIC(30, 2) GENERATED ALWAYS AS (
        COALESCE(funds_lost_usd, 0) - COALESCE(funds_returned_usd, 0)
    ) STORED,

    -- Detail
    description     TEXT,
    chain           TEXT,
    tx_hash         TEXT,
    audit_company   TEXT,                             -- auditor at time of hack (if known)
    source_url      TEXT,

    -- Dedup
    external_id     TEXT        UNIQUE,               -- defihack-db row identifier

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Recent hacks (DeFi page hack feed)
CREATE INDEX IF NOT EXISTS idx_hacks_date
    ON defi_hacks (hack_date DESC);

-- Protocol history
CREATE INDEX IF NOT EXISTS idx_hacks_protocol
    ON defi_hacks (protocol_slug, hack_date DESC)
    WHERE protocol_slug IS NOT NULL;

-- Largest losses
CREATE INDEX IF NOT EXISTS idx_hacks_loss
    ON defi_hacks (net_loss_usd DESC NULLS LAST);

COMMENT ON TABLE defi_hacks IS 'DeFi exploit history from banterous/defihack-db. Daily sync. Used for protocol risk display and alert context.';
