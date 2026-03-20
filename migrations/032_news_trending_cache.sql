-- ============================================================
-- 032_news_trending_cache.sql
-- Pre-computed trending topics refreshed every 1 hour.
-- Written by engine/news/trending/compute.go.
-- Serves /v1/news/trending endpoint directly.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_trending_cache (
    id              SERIAL      PRIMARY KEY,
    window          TEXT        NOT NULL DEFAULT '24h',  -- "1h", "6h", "24h"

    -- Trending coins (by mention count + importance weight)
    trending_coins  JSONB       NOT NULL DEFAULT '[]',
    -- shape: [{ coin_id, symbol, name, mention_count, avg_sentiment, avg_importance }, ...]

    -- Trending keywords (extracted by AI processor)
    trending_keywords JSONB     NOT NULL DEFAULT '[]',
    -- shape: [{ keyword, count, sentiment }, ...]

    -- Trending protocols
    trending_protocols JSONB    NOT NULL DEFAULT '[]',
    -- shape: [{ slug, name, mention_count }, ...]

    -- Top articles by importance_score in window
    top_article_ids BIGINT[]    DEFAULT '{}',

    -- Stats
    articles_processed INTEGER  NOT NULL DEFAULT 0,
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (window)
);

-- Seed default rows so upsert works on first run
INSERT INTO news_trending_cache (window, trending_coins, trending_keywords, trending_protocols)
VALUES ('1h', '[]', '[]', '[]'), ('6h', '[]', '[]', '[]'), ('24h', '[]', '[]', '[]')
ON CONFLICT (window) DO NOTHING;

COMMENT ON TABLE news_trending_cache IS 'Pre-computed trending data per time window. Upserted every 1h by TrendingCompute. Serves /v1/news/trending with zero query cost.';
