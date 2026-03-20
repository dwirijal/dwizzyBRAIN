-- ============================================================
-- 052_news_trending_cache_compat.sql
-- Compatibility migration for live DBs missing the news trending cache table.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_trending_cache (
    id                SERIAL      PRIMARY KEY,
    time_window       TEXT        NOT NULL DEFAULT '24h',
    trending_coins    JSONB       NOT NULL DEFAULT '[]',
    trending_keywords JSONB       NOT NULL DEFAULT '[]',
    trending_protocols JSONB      NOT NULL DEFAULT '[]',
    top_article_ids   BIGINT[]    DEFAULT '{}',
    articles_processed INTEGER    NOT NULL DEFAULT 0,
    computed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (time_window)
);

INSERT INTO news_trending_cache (time_window, trending_coins, trending_keywords, trending_protocols)
VALUES ('1h', '[]', '[]', '[]'), ('6h', '[]', '[]', '[]'), ('24h', '[]', '[]', '[]')
ON CONFLICT (time_window) DO NOTHING;

COMMENT ON TABLE news_trending_cache IS 'Pre-computed trending data per time window. Upserted every 1h by TrendingCompute. Serves /v1/news/trending with zero query cost.';
