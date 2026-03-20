-- ============================================================
-- 014_defi_protocols.sql
-- Registry of top 500 DeFi protocols from DefiLlama.
-- coin_id FK allows linking to market data.
-- ============================================================

CREATE TYPE defi_protocol_category AS ENUM (
    'dex', 'lending', 'bridge', 'yield', 'liquid_staking',
    'derivatives', 'options', 'insurance', 'rwa', 'launchpad',
    'gaming', 'nft', 'other'
);

CREATE TABLE IF NOT EXISTS defi_protocols (
    slug                TEXT        PRIMARY KEY,      -- DefiLlama slug: "uniswap", "aave-v3"
    name                TEXT        NOT NULL,
    coin_id             TEXT        REFERENCES coins(coin_id) ON DELETE SET NULL,
    logo_url            TEXT,
    category            defi_protocol_category NOT NULL DEFAULT 'other',
    chains              TEXT[]      NOT NULL DEFAULT '{}',  -- ["ethereum", "arbitrum", "optimism"]
    description         TEXT,
    website_url         TEXT,
    twitter_handle      TEXT,
    github_url          TEXT,
    audit_links         TEXT[]      DEFAULT '{}',
    is_multi_chain      BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    defillama_id        TEXT,                         -- DefiLlama internal id (if different from slug)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Category filter (DeFi page)
CREATE INDEX IF NOT EXISTS idx_defi_protocols_category
    ON defi_protocols (category)
    WHERE is_active = TRUE;

-- Chain filter
CREATE INDEX IF NOT EXISTS idx_defi_protocols_chains
    ON defi_protocols USING GIN (chains);

-- Coin linkage
CREATE INDEX IF NOT EXISTS idx_defi_protocols_coin
    ON defi_protocols (coin_id)
    WHERE coin_id IS NOT NULL;

COMMENT ON TABLE defi_protocols IS 'Top 500 DeFi protocol registry from DefiLlama. slug is universal DeFi key. Linked to coins via coin_id where applicable.';
