-- ============================================================
-- 030_news_sources.sql
-- Registry of news sources with credibility scores and polling config.
-- Read by fetchers to determine polling interval and weight.
-- ============================================================

CREATE TABLE IF NOT EXISTS news_sources (
    source_name         news_source_name PRIMARY KEY,

    display_name        TEXT        NOT NULL,
    base_url            TEXT,
    rss_url             TEXT,
    logo_url            TEXT,

    -- Credibility score (0.0 - 1.0)
    -- Used in importance_score formula: credibility * base_score
    credibility_score   NUMERIC(4, 3) NOT NULL DEFAULT 0.5,

    -- Polling config
    poll_interval_seconds INTEGER   NOT NULL DEFAULT 600,  -- 10 min default
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    fetch_type          TEXT        NOT NULL DEFAULT 'rss', -- "rss", "api", "cryptopanic_filter"

    -- Stats
    articles_fetched_total BIGINT   NOT NULL DEFAULT 0,
    last_fetched_at     TIMESTAMPTZ,
    last_success_at     TIMESTAMPTZ,
    consecutive_failures INTEGER    NOT NULL DEFAULT 0,

    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default sources
INSERT INTO news_sources (source_name, display_name, base_url, rss_url, credibility_score, poll_interval_seconds, fetch_type)
VALUES
    ('cryptopanic',     'CryptoPanic',      'https://cryptopanic.com',          NULL,                                               0.70, 300,  'api'),
    ('coindesk',        'CoinDesk',         'https://coindesk.com',             'https://www.coindesk.com/arc/outboundfeeds/rss/',   0.90, 600,  'rss'),
    ('cointelegraph',   'CoinTelegraph',    'https://cointelegraph.com',        'https://cointelegraph.com/rss',                    0.85, 600,  'rss'),
    ('decrypt',         'Decrypt',          'https://decrypt.co',               'https://decrypt.co/feed',                          0.80, 600,  'rss'),
    ('coingecko',       'CoinGecko News',   'https://coingecko.com',            NULL,                                               0.75, 1800, 'api'),
    ('theblock',        'The Block',        'https://theblock.co',              'https://www.theblock.co/rss.xml',                  0.85, 600,  'rss'),
    ('blockworks',      'Blockworks',       'https://blockworks.co',            'https://blockworks.co/feed',                       0.80, 600,  'rss')
ON CONFLICT (source_name) DO NOTHING;

COMMENT ON TABLE news_sources IS 'Source registry with credibility weights. credibility_score feeds into importance_score calculation in AI processor.';
