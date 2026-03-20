-- ============================================================
-- 028_news_ai_metadata.sql
-- AI-generated metadata per article: summary, sentiment, categories,
-- importance score. One-to-one with news_articles.
-- Written by engine/news/ai/processor.go in 5-min batches.
-- ============================================================

CREATE TYPE news_sentiment AS ENUM ('bullish', 'bearish', 'neutral', 'mixed');
CREATE TYPE news_category  AS ENUM (
    'regulation', 'hack_exploit', 'partnership', 'listing_delisting',
    'market_analysis', 'macro', 'technology', 'adoption', 'fundraising',
    'whale_movement', 'defi', 'nft', 'layer2', 'other'
);

CREATE TABLE IF NOT EXISTS news_ai_metadata (
    article_id          BIGINT      PRIMARY KEY REFERENCES news_articles(id) ON DELETE CASCADE,

    -- Content
    summary_short       TEXT,                         -- 1-2 sentence summary
    summary_long        TEXT,                         -- paragraph summary for detail view
    key_points          TEXT[]      DEFAULT '{}',     -- bullet list of key facts

    -- Classification
    sentiment           news_sentiment,
    sentiment_score     NUMERIC(4, 3),                -- -1.0 (bearish) to +1.0 (bullish)
    category            news_category,
    subcategory         TEXT,                         -- free-form subcategory

    -- Importance scoring
    -- Formula: credibility_weight * source_score + (votes_positive - votes_negative) * vote_weight
    importance_score    NUMERIC(6, 3),               -- 0.0 to 100.0
    is_breaking         BOOLEAN     NOT NULL DEFAULT FALSE,
    breaking_type       TEXT,                         -- "regulation", "exploit_hack", "whale", "macro"

    -- LLM provider used
    model_used          TEXT,                         -- "groq/llama3-8b", "gemini-flash", etc.
    processing_latency_ms INTEGER,
    prompt_tokens       INTEGER,
    completion_tokens   INTEGER,

    processed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Breaking news filter
CREATE INDEX IF NOT EXISTS idx_news_ai_breaking
    ON news_ai_metadata (processed_at DESC)
    WHERE is_breaking = TRUE;

-- Category feed
CREATE INDEX IF NOT EXISTS idx_news_ai_category
    ON news_ai_metadata (category, processed_at DESC);

-- Importance ranking
CREATE INDEX IF NOT EXISTS idx_news_ai_importance
    ON news_ai_metadata (importance_score DESC NULLS LAST);

COMMENT ON TABLE news_ai_metadata IS 'AI-generated metadata per article. 5-min batch processor via irag/groq/gemini. importance_score drives alert + trending computation.';
