-- ============================================================
-- 066_cold_coin_data_market_snapshot.sql
-- Extend cold_coin_data with live market snapshot columns so the
-- market API can expose internal price/market-cap data without
-- depending on external frontend fallbacks.
-- ============================================================

ALTER TABLE cold_coin_data
    ADD COLUMN IF NOT EXISTS current_price_usd DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS market_cap_usd DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS total_volume_24h DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS price_change_24h DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS market_cap_change_24h DOUBLE PRECISION;

COMMENT ON COLUMN cold_coin_data.current_price_usd IS 'Latest market price in USD from the internal market bootstrap or cache.';
COMMENT ON COLUMN cold_coin_data.market_cap_usd IS 'Latest market cap in USD from the internal market bootstrap or cache.';
COMMENT ON COLUMN cold_coin_data.total_volume_24h IS 'Latest 24h traded volume in USD from the internal market bootstrap or cache.';
COMMENT ON COLUMN cold_coin_data.price_change_24h IS 'Latest 24h price change percentage from the internal market bootstrap or cache.';
COMMENT ON COLUMN cold_coin_data.market_cap_change_24h IS 'Latest 24h market cap change percentage from the internal market bootstrap or cache.';
