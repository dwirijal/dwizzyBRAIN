-- ============================================================
-- 049_news_ai_compat.sql
-- Compatibility version of AI news tables for the live schema.
-- Uses text identifiers for coin/protocol entities to avoid FK
-- coupling to older live coin PK shapes.
-- ============================================================

DO $$
BEGIN
    CREATE TYPE news_sentiment AS ENUM ('bullish', 'bearish', 'neutral', 'mixed');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE news_category AS ENUM (
        'regulation', 'hack_exploit', 'partnership', 'listing_delisting',
        'market_analysis', 'macro', 'technology', 'adoption', 'fundraising',
        'whale_movement', 'defi', 'nft', 'layer2', 'other'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS news_ai_metadata (
    article_id              BIGINT PRIMARY KEY REFERENCES news_articles(id) ON DELETE CASCADE,
    summary_short           TEXT,
    summary_long            TEXT,
    key_points              TEXT[] DEFAULT '{}',
    sentiment               news_sentiment,
    sentiment_score         NUMERIC(4, 3),
    category                news_category,
    subcategory             TEXT,
    importance_score        NUMERIC(6, 3),
    is_breaking             BOOLEAN NOT NULL DEFAULT FALSE,
    breaking_type           TEXT,
    model_used              TEXT,
    processing_latency_ms   INTEGER,
    prompt_tokens           INTEGER,
    completion_tokens       INTEGER,
    processed_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_news_ai_breaking
    ON news_ai_metadata (processed_at DESC)
    WHERE is_breaking = TRUE;

CREATE INDEX IF NOT EXISTS idx_news_ai_category
    ON news_ai_metadata (category, processed_at DESC);

CREATE INDEX IF NOT EXISTS idx_news_ai_importance
    ON news_ai_metadata (importance_score DESC NULLS LAST);

COMMENT ON TABLE news_ai_metadata IS 'Heuristic AI metadata per article for the live news pipeline.';
