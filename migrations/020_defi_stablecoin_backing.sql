-- ============================================================
-- 020_defi_stablecoin_backing.sql
-- Backing composition per stablecoin.
-- Sources: Circle/Tether attestations + DefiLlama stablecoins API.
-- ============================================================

CREATE TABLE IF NOT EXISTS defi_stablecoin_backing (
    id              BIGSERIAL   PRIMARY KEY,
    coin_id         TEXT        NOT NULL REFERENCES coins(coin_id) ON DELETE CASCADE,
    snapshot_date   DATE        NOT NULL,

    -- Peg info
    peg_type        TEXT        NOT NULL DEFAULT 'USD',  -- "USD", "EUR", "BTC"
    peg_mechanism   TEXT,                               -- "fiat-backed", "crypto-backed", "algorithmic", "hybrid"

    -- Current price (depeg detection)
    price_usd       NUMERIC(20, 10),
    depeg_pct       NUMERIC(10, 6) GENERATED ALWAYS AS (
        ABS(COALESCE(price_usd, 1.0) - 1.0) * 100
    ) STORED,

    -- Market metrics
    mcap_usd        NUMERIC(30, 2),
    circulating     NUMERIC(30, 2),

    -- Backing breakdown (JSONB — flexible per issuer)
    backing_composition JSONB   NOT NULL DEFAULT '{}',
    -- shape: { "cash": 45.2, "treasuries": 30.1, "commercial_paper": 10.5, "other": 14.2 }

    -- Reserve attestation
    attestation_url TEXT,
    attested_at     TIMESTAMPTZ,

    synced_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (coin_id, snapshot_date)
);

-- Latest backing per coin
CREATE INDEX IF NOT EXISTS idx_stablecoin_backing_coin_date
    ON defi_stablecoin_backing (coin_id, snapshot_date DESC);

-- Depeg monitor
CREATE INDEX IF NOT EXISTS idx_stablecoin_backing_depeg
    ON defi_stablecoin_backing (coin_id, synced_at DESC);

COMMENT ON TABLE defi_stablecoin_backing IS 'Stablecoin backing composition + depeg tracking. Daily snapshots. depeg_pct generated column drives DepegScanner alerts.';
