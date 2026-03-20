-- ============================================================
-- 002_coins.sql
-- Canonical coin registry — sourced from CoinGecko
-- coin_id (CoinGecko id string) is the universal key
-- across all dwizzyOS subsystems.
-- ============================================================

CREATE TABLE IF NOT EXISTS coins (
    coin_id         TEXT        PRIMARY KEY,          -- "bitcoin", "ethereum", "matic-network"
    symbol          TEXT        NOT NULL,             -- "btc", "eth", "matic"
    name            TEXT        NOT NULL,             -- "Bitcoin", "Ethereum"
    image_url       TEXT,                             -- CoinGecko image CDN URL
    market_cap_rank INTEGER,                          -- 1-based rank, NULL for unranked
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast rank-ordered listing (primary market page query)
CREATE INDEX IF NOT EXISTS idx_coins_rank
    ON coins (market_cap_rank ASC NULLS LAST)
    WHERE is_active = TRUE;

-- Case-insensitive symbol lookup (symbol mapping auto-build)
CREATE INDEX IF NOT EXISTS idx_coins_symbol_lower
    ON coins (LOWER(symbol));

-- Trigram index for fuzzy name/symbol search
CREATE INDEX IF NOT EXISTS idx_coins_name_trgm
    ON coins USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_coins_symbol_trgm
    ON coins USING GIN (symbol gin_trgm_ops);

COMMENT ON TABLE coins IS 'Canonical coin registry. coin_id = CoinGecko id string. Universal FK across all tables.';