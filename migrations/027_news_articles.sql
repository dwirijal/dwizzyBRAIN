-- ============================================================
-- 027_news_articles.sql
-- Raw news articles from all sources.
-- Retention: 90 days. AI metadata stored in separate table.
-- Sources: CryptoPanic, RSS (CoinDesk/CT/Decrypt), CoinGecko /news
-- ============================================================

CREATE TYPE news_source_name AS ENUM (
    'cryptopanic',
    'coindesk',
    'cointelegraph',
    'decrypt',
    'coingecko',
    'theblock',
    'blockworks',
    'other'
);

CREATE TABLE IF NOT EXISTS news_articles (
    id              BIGSERIAL   PRIMARY KEY,
    external_id     TEXT        NOT NULL,             -- source's own ID or URL hash
    source          news_source_name NOT NULL,
    source_url      TEXT        NOT NULL,
    title           TEXT        NOT NULL,
    body_preview    TEXT,                             -- first ~500 chars (RSS excerpt or CryptoPanic snippet)
    full_url        TEXT,                             -- canonical article URL
    image_url       TEXT,
    author          TEXT,
    published_at    TIMESTAMPTZ NOT NULL,
    fetched_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CryptoPanic specific
    cp_kind         TEXT,                             -- "news", "media", "analysis"
    cp_votes_positive INTEGER DEFAULT 0,
    cp_votes_negative INTEGER DEFAULT 0,
    cp_votes_important INTEGER DEFAULT 0,

    -- Dedup
    url_hash        TEXT        GENERATED ALWAYS AS (MD5(source_url)) STORED,

    is_processed    BOOLEAN     NOT NULL DEFAULT FALSE,  -- AI processor flag
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,

    UNIQUE (external_id, source)
);

-- AI processor queue: unprocessed articles ordered by publish time
CREATE INDEX IF NOT EXISTS idx_news_articles_unprocessed
    ON news_articles (published_at DESC)
    WHERE is_processed = FALSE AND is_active = TRUE;

-- Feed queries: recent articles per source
CREATE INDEX IF NOT EXISTS idx_news_articles_source_time
    ON news_articles (source, published_at DESC)
    WHERE is_active = TRUE;

-- Dedup by URL
CREATE UNIQUE INDEX IF NOT EXISTS idx_news_articles_url_hash
    ON news_articles (url_hash);

-- published_at for time-range queries
CREATE INDEX IF NOT EXISTS idx_news_articles_published
    ON news_articles (published_at DESC)
    WHERE is_active = TRUE;

COMMENT ON TABLE news_articles IS 'Raw news from CryptoPanic, RSS, CoinGecko. 90d retention. is_processed flag drives AI batch processor queue.';
