-- ============================================================
-- 029_news_entities.sql
-- Entity tagging results per article.
-- coin_id + llama_slug per article extracted by AI processor.
-- Enables /v1/news/coin/{coin_id} and /v1/news/protocol/{slug} feeds.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_entities (
    id              BIGSERIAL   PRIMARY KEY,
    article_id      BIGINT      NOT NULL REFERENCES news_articles(id) ON DELETE CASCADE,

    -- Coin entity
    coin_id         TEXT        REFERENCES coins(coin_id) ON DELETE CASCADE,

    -- DeFi protocol entity
    llama_slug      TEXT        REFERENCES defi_protocols(slug) ON DELETE CASCADE,

    -- Entity type context
    entity_type     TEXT        NOT NULL DEFAULT 'coin',  -- "coin", "protocol", "chain", "person", "exchange"
    entity_name     TEXT,                                  -- raw name extracted by AI
    is_primary      BOOLEAN     NOT NULL DEFAULT FALSE,    -- is this the main subject of the article?
    mention_count   INTEGER     NOT NULL DEFAULT 1,
    confidence      NUMERIC(4, 3),                         -- AI extraction confidence 0.0-1.0

    CHECK (coin_id IS NOT NULL OR llama_slug IS NOT NULL)
);

-- Feed per coin: all articles mentioning this coin
CREATE INDEX IF NOT EXISTS idx_news_entities_coin
    ON news_entities (coin_id, article_id DESC)
    WHERE coin_id IS NOT NULL;

-- Feed per protocol
CREATE INDEX IF NOT EXISTS idx_news_entities_slug
    ON news_entities (llama_slug, article_id DESC)
    WHERE llama_slug IS NOT NULL;

-- Primary entity per article (most relevant coin/protocol)
CREATE INDEX IF NOT EXISTS idx_news_entities_primary
    ON news_entities (article_id)
    WHERE is_primary = TRUE;

COMMENT ON TABLE news_entities IS 'Coin + protocol entity tags per article from AI processor. Enables per-coin and per-protocol news feeds.';
