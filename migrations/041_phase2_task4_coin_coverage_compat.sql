-- ============================================================
-- 041_phase2_task4_coin_coverage_compat.sql
-- Compatibility upgrade for the legacy coin_coverage table so the
-- GapDetector can write the full Phase 2 coverage model.
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'coverage_tier'
    ) THEN
        CREATE TYPE coverage_tier AS ENUM ('A', 'B', 'C', 'D');
    END IF;
END $$;

ALTER TABLE coin_coverage
    ADD COLUMN IF NOT EXISTS on_okx BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS on_kucoin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS on_gate BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS on_kraken BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS on_mexc BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS on_htx BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS on_coinpaprika BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS dex_contract_address TEXT,
    ADD COLUMN IF NOT EXISTS dex_pair_address TEXT,
    ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS binance_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS bybit_verified_at TIMESTAMPTZ;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'coin_coverage'
          AND column_name = 'dex_contract'
    ) THEN
        UPDATE coin_coverage
        SET dex_contract_address = COALESCE(dex_contract_address, dex_contract)
        WHERE dex_contract IS NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_coverage_tier
    ON coin_coverage (tier);

CREATE INDEX IF NOT EXISTS idx_coverage_binance
    ON coin_coverage (coin_id)
    WHERE on_binance = TRUE;
