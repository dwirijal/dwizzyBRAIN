-- ============================================================
-- 047_news_sources.sql
-- Registry of news sources with credibility scores and polling config.
-- This first slice supports RSS-first ingestion.
-- ============================================================

DO $$
BEGIN
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
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS news_sources (
    source_name              news_source_name PRIMARY KEY,
    display_name             TEXT NOT NULL,
    base_url                 TEXT,
    rss_url                  TEXT,
    logo_url                 TEXT,
    credibility_score        NUMERIC(4, 3) NOT NULL DEFAULT 0.5,
    poll_interval_seconds    INTEGER NOT NULL DEFAULT 600,
    is_active                BOOLEAN NOT NULL DEFAULT TRUE,
    fetch_type               TEXT NOT NULL DEFAULT 'rss',
    articles_fetched_total   BIGINT NOT NULL DEFAULT 0,
    last_fetched_at          TIMESTAMPTZ,
    last_success_at          TIMESTAMPTZ,
    consecutive_failures     INTEGER NOT NULL DEFAULT 0,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO news_sources (
    source_name, display_name, base_url, rss_url, credibility_score,
    poll_interval_seconds, is_active, fetch_type
)
VALUES
    ('cryptopanic',   'CryptoPanic',    'https://cryptopanic.com',      NULL,                                            0.700,  300, FALSE, 'api'),
    ('coindesk',      'CoinDesk',       'https://www.coindesk.com',     'https://www.coindesk.com/arc/outboundfeeds/rss/', 0.900, 600, TRUE,  'rss'),
    ('cointelegraph', 'CoinTelegraph',   'https://cointelegraph.com',    'https://cointelegraph.com/rss',                0.850, 600, TRUE,  'rss'),
    ('decrypt',       'Decrypt',        'https://decrypt.co',           'https://decrypt.co/feed',                      0.800, 600, TRUE,  'rss'),
    ('coingecko',     'CoinGecko News',  'https://www.coingecko.com',    NULL,                                            0.750, 1800, FALSE, 'api'),
    ('theblock',      'The Block',      'https://www.theblock.co',      'https://www.theblock.co/rss.xml',              0.850, 600, TRUE,  'rss'),
    ('blockworks',    'Blockworks',     'https://blockworks.co',        'https://blockworks.co/feed',                   0.800, 600, TRUE,  'rss')
ON CONFLICT (source_name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    base_url = EXCLUDED.base_url,
    rss_url = EXCLUDED.rss_url,
    credibility_score = EXCLUDED.credibility_score,
    poll_interval_seconds = EXCLUDED.poll_interval_seconds,
    is_active = EXCLUDED.is_active,
    fetch_type = EXCLUDED.fetch_type,
    updated_at = NOW();

COMMENT ON TABLE news_sources IS 'Source registry with credibility weights and polling configuration.';
