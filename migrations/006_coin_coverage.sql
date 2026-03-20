-- ============================================================
-- 006_coin_coverage.sql
-- Coverage tier classification per coin (A/B/C/D)
-- Drives MarketDataService fetch strategy selection.
-- GapDetector assigns tiers during cold load.
-- ============================================================

CREATE TYPE coverage_tier AS ENUM ('A', 'B', 'C', 'D');

CREATE TABLE IF NOT EXISTS coin_coverage (
    coin_id             TEXT            PRIMARY KEY REFERENCES coins(coin_id) ON DELETE CASCADE,
    tier                coverage_tier   NOT NULL DEFAULT 'D',

    -- Exchange availability flags
    on_binance          BOOLEAN         NOT NULL DEFAULT FALSE,
    on_bybit            BOOLEAN         NOT NULL DEFAULT FALSE,
    on_okx              BOOLEAN         NOT NULL DEFAULT FALSE,
    on_kucoin           BOOLEAN         NOT NULL DEFAULT FALSE,
    on_gate             BOOLEAN         NOT NULL DEFAULT FALSE,
    on_kraken           BOOLEAN         NOT NULL DEFAULT FALSE,
    on_mexc             BOOLEAN         NOT NULL DEFAULT FALSE,
    on_htx              BOOLEAN         NOT NULL DEFAULT FALSE,
    on_coinpaprika      BOOLEAN         NOT NULL DEFAULT FALSE,

    -- DEX info
    is_dex_only         BOOLEAN         NOT NULL DEFAULT FALSE,
    dex_chain           TEXT,                         -- "ethereum", "bsc", "solana", "base"
    dex_contract_address TEXT,                        -- checksum address
    dex_pair_address    TEXT,                         -- DexScreener pair address

    -- Sync timestamps per exchange (for gap detection freshness)
    binance_verified_at TIMESTAMPTZ,
    bybit_verified_at   TIMESTAMPTZ,

    assigned_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- Tier-based batch queries (MarketDataService strategy selection)
CREATE INDEX IF NOT EXISTS idx_coverage_tier
    ON coin_coverage (tier);

-- DEX-only coins for DexScreener lookup
CREATE INDEX IF NOT EXISTS idx_coverage_dex_only
    ON coin_coverage (dex_chain)
    WHERE is_dex_only = TRUE;

-- Binance-listed coins (hot tier candidates)
CREATE INDEX IF NOT EXISTS idx_coverage_binance
    ON coin_coverage (coin_id)
    WHERE on_binance = TRUE;

COMMENT ON TABLE coin_coverage IS 'Tier A=top100 CEX, B=101-500 CEX, C=501-1000 DEX/cold, D=untracked. Drives fetch strategy in MarketDataService.';
