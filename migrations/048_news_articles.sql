-- ============================================================
-- 048_news_articles.sql
-- Raw news articles from all sources.
-- Retention handled at the application layer for now.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_articles (
    id                  BIGSERIAL PRIMARY KEY,
    external_id         TEXT NOT NULL,
    source              news_source_name NOT NULL,
    source_url          TEXT NOT NULL,
    title               TEXT NOT NULL,
    body_preview        TEXT,
    full_url            TEXT,
    image_url           TEXT,
    author              TEXT,
    published_at        TIMESTAMPTZ NOT NULL,
    fetched_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cp_kind             TEXT,
    cp_votes_positive   INTEGER NOT NULL DEFAULT 0,
    cp_votes_negative   INTEGER NOT NULL DEFAULT 0,
    cp_votes_important  INTEGER NOT NULL DEFAULT 0,
    url_hash            TEXT GENERATED ALWAYS AS (MD5(source_url)) STORED,
    is_processed        BOOLEAN NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (external_id, source)
);

CREATE INDEX IF NOT EXISTS idx_news_articles_unprocessed
    ON news_articles (published_at DESC)
    WHERE is_processed = FALSE AND is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_news_articles_source_time
    ON news_articles (source, published_at DESC)
    WHERE is_active = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_news_articles_url_hash
    ON news_articles (url_hash);

CREATE INDEX IF NOT EXISTS idx_news_articles_published
    ON news_articles (published_at DESC)
    WHERE is_active = TRUE;

COMMENT ON TABLE news_articles IS 'Raw news from RSS sources. AI metadata is stored separately in later phases.';
