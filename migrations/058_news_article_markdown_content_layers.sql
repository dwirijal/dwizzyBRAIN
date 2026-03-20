-- ============================================================
-- 058_news_article_markdown_content_layers.sql
-- Adds JSON and folder pointers for Drive-hosted news article archives.
-- ============================================================

ALTER TABLE news_article_markdown_exports
    ADD COLUMN IF NOT EXISTS content_folder_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_json_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_json_url TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN news_article_markdown_exports.content_folder_path IS 'Drive folder containing the markdown and JSON content package.';
COMMENT ON COLUMN news_article_markdown_exports.content_json_path IS 'Drive path to the JSON renderable copy of the article.';
COMMENT ON COLUMN news_article_markdown_exports.content_json_url IS 'Shareable Drive link to the JSON renderable copy of the article.';
