-- ============================================================
-- 034_news_price_impact_history.sql
-- Permanent time-series of news price impact events.
-- Populated from news_price_impact once all 3 snapshots complete.
-- TimescaleDB hypertable for long-term correlation analysis.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_price_impact_history (
    time                TIMESTAMPTZ NOT NULL,         -- = published_at of the article
    article_id          BIGINT      NOT NULL REFERENCES news_articles(id) ON DELETE CASCADE,
    coin_id             TEXT        NOT NULL REFERENCES coins(coin_id) ON DELETE CASCADE,

    -- Sentiment context at time of publish
    sentiment           news_sentiment,
    importance_score    NUMERIC(6, 3),
    category            news_category,
    is_breaking         BOOLEAN,

    -- Price impact
    change_pct_1h       NUMERIC(10, 4),
    change_pct_4h       NUMERIC(10, 4),
    change_pct_24h      NUMERIC(10, 4),

    PRIMARY KEY (time, article_id, coin_id)
);

SELECT create_hypertable(
    'news_price_impact_history',
    'time',
    chunk_time_interval => INTERVAL '30 days',
    if_not_exists => TRUE
);

ALTER TABLE news_price_impact_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'coin_id',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('news_price_impact_history', INTERVAL '30 days', if_not_exists => TRUE);

-- No retention limit — permanent record for ML/quant use

-- Sentiment + impact correlation queries
CREATE INDEX IF NOT EXISTS idx_news_impact_hist_coin_time
    ON news_price_impact_history (coin_id, time DESC);

CREATE INDEX IF NOT EXISTS idx_news_impact_hist_sentiment
    ON news_price_impact_history (sentiment, time DESC);

COMMENT ON TABLE news_price_impact_history IS 'Permanent time-series of completed news price impact records. No retention. Used for quant model training and sentiment-price correlation.';
