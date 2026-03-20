-- ============================================================
-- 050_news_entities_compat.sql
-- Compatibility entity tagging table for the live schema.
-- Stores extracted coin/protocol references as text labels.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_entities (
    id              BIGSERIAL PRIMARY KEY,
    article_id      BIGINT NOT NULL REFERENCES news_articles(id) ON DELETE CASCADE,
    coin_id         TEXT,
    llama_slug      TEXT,
    entity_type     TEXT NOT NULL DEFAULT 'coin',
    entity_name     TEXT,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    mention_count    INTEGER NOT NULL DEFAULT 1,
    confidence      NUMERIC(4, 3),
    CHECK (coin_id IS NOT NULL OR llama_slug IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_news_entities_coin
    ON news_entities (coin_id, article_id DESC)
    WHERE coin_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_news_entities_slug
    ON news_entities (llama_slug, article_id DESC)
    WHERE llama_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_news_entities_primary
    ON news_entities (article_id)
    WHERE is_primary = TRUE;

COMMENT ON TABLE news_entities IS 'Heuristic coin/protocol tags per article for the live news pipeline.';
