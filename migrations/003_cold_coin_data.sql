-- ============================================================
-- 003_cold_coin_data.sql
-- ATH/ATL, description, links, categories, genesis_date
-- Refreshed every 24 hours from CoinGecko /coins/{id}
-- ============================================================

CREATE TABLE IF NOT EXISTS cold_coin_data (
    coin_id                 TEXT        PRIMARY KEY REFERENCES coins(coin_id) ON DELETE CASCADE,

    -- Price extremes
    ath_usd                 NUMERIC(30, 10),
    ath_date                TIMESTAMPTZ,
    atl_usd                 NUMERIC(30, 10),
    atl_date                TIMESTAMPTZ,
    ath_change_percent      NUMERIC(10, 4),           -- current price % from ATH
    atl_change_percent      NUMERIC(10, 4),

    -- Supply
    circulating_supply      NUMERIC(30, 2),
    total_supply            NUMERIC(30, 2),
    max_supply              NUMERIC(30, 2),
    fully_diluted_valuation NUMERIC(30, 2),

    -- Market data snapshot (warm fallback when Valkey TTL expired)
    current_price_usd       NUMERIC(30, 10),
    market_cap_usd          NUMERIC(30, 2),
    total_volume_24h        NUMERIC(30, 2),
    price_change_24h        NUMERIC(10, 4),
    market_cap_change_24h   NUMERIC(10, 4),

    -- Identity / metadata
    description_en          TEXT,
    genesis_date            DATE,
    hashing_algorithm       TEXT,
    country_origin          TEXT,

    -- Links (JSONB for flexibility)
    links                   JSONB DEFAULT '{}',
    -- expected shape: { homepage: [], whitepaper: "", repos: [], twitter: "", telegram: "", subreddit: "" }

    -- Categories array
    categories              TEXT[] DEFAULT '{}',

    -- Sentiment
    sentiment_votes_up      NUMERIC(6, 2),
    sentiment_votes_down    NUMERIC(6, 2),
    watchlist_portfolio_users BIGINT,

    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Category search
CREATE INDEX IF NOT EXISTS idx_cold_coin_categories
    ON cold_coin_data USING GIN (categories);

-- Freshness check
CREATE INDEX IF NOT EXISTS idx_cold_coin_updated
    ON cold_coin_data (updated_at DESC);

COMMENT ON TABLE cold_coin_data IS '24h refresh cold data: ATH/ATL, supply, description, links, categories. Fallback price source when hot/warm unavailable.';
