-- ============================================================
-- 031_news_price_impact.sql
-- Price snapshots at 1h, 4h, 24h after article publication.
-- Populated by engine/news/impact/price.go via scheduled job.
-- Enables news → price correlation analysis.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_price_impact (
    id              BIGSERIAL   PRIMARY KEY,
    article_id      BIGINT      NOT NULL REFERENCES news_articles(id) ON DELETE CASCADE,
    coin_id         TEXT        NOT NULL REFERENCES coins(coin_id) ON DELETE CASCADE,

    -- Price at publication
    price_at_publish    NUMERIC(30, 10),
    published_at        TIMESTAMPTZ NOT NULL,

    -- Snapshot windows
    price_1h            NUMERIC(30, 10),
    price_4h            NUMERIC(30, 10),
    price_24h           NUMERIC(30, 10),

    change_pct_1h       NUMERIC(10, 4) GENERATED ALWAYS AS (
        CASE WHEN price_at_publish > 0 THEN ((price_1h - price_at_publish) / price_at_publish) * 100 ELSE NULL END
    ) STORED,
    change_pct_4h       NUMERIC(10, 4) GENERATED ALWAYS AS (
        CASE WHEN price_at_publish > 0 THEN ((price_4h - price_at_publish) / price_at_publish) * 100 ELSE NULL END
    ) STORED,
    change_pct_24h      NUMERIC(10, 4) GENERATED ALWAYS AS (
        CASE WHEN price_at_publish > 0 THEN ((price_24h - price_at_publish) / price_at_publish) * 100 ELSE NULL END
    ) STORED,

    -- Snapshot completion flags
    snapshot_1h_done    BOOLEAN NOT NULL DEFAULT FALSE,
    snapshot_4h_done    BOOLEAN NOT NULL DEFAULT FALSE,
    snapshot_24h_done   BOOLEAN NOT NULL DEFAULT FALSE,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (article_id, coin_id)
);

-- Pending snapshot job queue
CREATE INDEX IF NOT EXISTS idx_price_impact_pending_1h
    ON news_price_impact (published_at)
    WHERE snapshot_1h_done = FALSE;

CREATE INDEX IF NOT EXISTS idx_price_impact_pending_24h
    ON news_price_impact (published_at)
    WHERE snapshot_24h_done = FALSE;

-- Per-coin impact analysis
CREATE INDEX IF NOT EXISTS idx_price_impact_coin
    ON news_price_impact (coin_id, published_at DESC);

COMMENT ON TABLE news_price_impact IS 'Price impact tracking at 1h/4h/24h post-publication. Scheduled snapshots fill in windows. Generated columns compute % change.';
