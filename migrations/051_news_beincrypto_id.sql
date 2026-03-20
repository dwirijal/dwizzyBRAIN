-- ============================================================
-- 051_news_beincrypto_id.sql
-- Add BeInCrypto Indonesia as a free Telegram-backed news source.
-- The site RSS is Cloudflare-blocked in this environment, so the
-- official Telegram channel is used as the ingest source.
-- ============================================================

DO $$
BEGIN
    ALTER TYPE news_source_name ADD VALUE IF NOT EXISTS 'beincrypto_id';
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

INSERT INTO news_sources (
    source_name, display_name, base_url, rss_url, credibility_score,
    poll_interval_seconds, is_active, fetch_type
)
VALUES
    (
        'beincrypto_id',
        'BeInCrypto Indonesia',
        'https://id.beincrypto.com',
        'https://t.me/s/BeInCryptoIDNews',
        0.780,
        900,
        TRUE,
        'telegram'
    )
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
