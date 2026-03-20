-- ============================================================
-- 007_coin_data_completeness.sql
-- Field-level availability flags + completeness score per coin
-- Used by availability map in /market/{id} response.
-- Updated by Enricher after each fetch attempt.
-- ============================================================

CREATE TABLE IF NOT EXISTS coin_data_completeness (
    coin_id             TEXT        PRIMARY KEY REFERENCES coins(coin_id) ON DELETE CASCADE,

    -- Field availability booleans
    has_price           BOOLEAN     NOT NULL DEFAULT FALSE,
    has_market_cap      BOOLEAN     NOT NULL DEFAULT FALSE,
    has_volume          BOOLEAN     NOT NULL DEFAULT FALSE,
    has_ohlcv           BOOLEAN     NOT NULL DEFAULT FALSE,
    has_orderbook       BOOLEAN     NOT NULL DEFAULT FALSE,
    has_tickers         BOOLEAN     NOT NULL DEFAULT FALSE,
    has_description     BOOLEAN     NOT NULL DEFAULT FALSE,
    has_links           BOOLEAN     NOT NULL DEFAULT FALSE,
    has_ath_atl         BOOLEAN     NOT NULL DEFAULT FALSE,
    has_dev_data        BOOLEAN     NOT NULL DEFAULT FALSE,
    has_sentiment       BOOLEAN     NOT NULL DEFAULT FALSE,
    has_categories      BOOLEAN     NOT NULL DEFAULT FALSE,

    -- Composite score 0.0 - 1.0
    -- Computed as: (count of TRUE flags) / (total flags = 12)
    completeness_score  NUMERIC(4, 3) NOT NULL DEFAULT 0.0
        GENERATED ALWAYS AS (
            (
                has_price::int + has_market_cap::int + has_volume::int +
                has_ohlcv::int + has_orderbook::int + has_tickers::int +
                has_description::int + has_links::int + has_ath_atl::int +
                has_dev_data::int + has_sentiment::int + has_categories::int
            )::numeric / 12.0
        ) STORED,

    last_enrichment_attempt TIMESTAMPTZ,
    last_enrichment_success TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Find coins needing enrichment (score < threshold, not recently attempted)
CREATE INDEX IF NOT EXISTS idx_completeness_score
    ON coin_data_completeness (completeness_score ASC, last_enrichment_attempt ASC NULLS FIRST);

-- Coins with no price yet — priority for Enricher
CREATE INDEX IF NOT EXISTS idx_completeness_no_price
    ON coin_data_completeness (coin_id)
    WHERE has_price = FALSE;

COMMENT ON TABLE coin_data_completeness IS 'Field-level availability + computed score 0.0-1.0. Drives availability map in /market/{id} and Enricher prioritization.';
