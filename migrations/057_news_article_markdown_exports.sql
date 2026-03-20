-- ============================================================
-- 057_news_article_markdown_exports.sql
-- Markdown exports for news articles stored in Google Drive.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_article_markdown_exports (
    article_id      BIGINT PRIMARY KEY REFERENCES news_articles(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    drive_url       TEXT NOT NULL,
    drive_path      TEXT NOT NULL,
    file_name       TEXT NOT NULL,
    exported_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_news_article_markdown_exports_exported_at
    ON news_article_markdown_exports (exported_at DESC);

COMMENT ON TABLE news_article_markdown_exports IS 'Google Drive markdown exports for news articles. Stores title plus shareable Drive URL for frontend rendering later.';
