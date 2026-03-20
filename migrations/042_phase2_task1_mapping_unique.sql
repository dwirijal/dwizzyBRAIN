-- ============================================================
-- 042_phase2_task1_mapping_unique.sql
-- Enforce the exchange + exchange_symbol conflict target used by
-- the mapping builder upsert path.
-- ============================================================

UPDATE coin_exchange_mappings
SET exchange = LOWER(BTRIM(exchange)),
    exchange_symbol = UPPER(BTRIM(exchange_symbol)),
    base_asset = UPPER(BTRIM(COALESCE(base_asset, ''))),
    quote_asset = UPPER(BTRIM(COALESCE(quote_asset, '')))
WHERE exchange IS NOT NULL
   OR exchange_symbol IS NOT NULL
   OR base_asset IS NOT NULL
   OR quote_asset IS NOT NULL;

WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY exchange, exchange_symbol
            ORDER BY
                is_primary DESC,
                verified_at DESC NULLS LAST,
                updated_at DESC,
                id DESC
        ) AS rn
    FROM coin_exchange_mappings
)
DELETE FROM coin_exchange_mappings m
USING ranked r
WHERE m.id = r.id
  AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mappings_exchange_symbol_unique
    ON coin_exchange_mappings (exchange, exchange_symbol);
