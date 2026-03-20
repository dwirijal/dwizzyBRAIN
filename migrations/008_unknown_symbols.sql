-- ============================================================
-- 008_unknown_symbols.sql
-- Symbols from WebSocket streams that have no mapping yet.
-- UnknownSymbolResolver runs hourly to auto-resolve via
-- fuzzy match against coins table.
-- ============================================================

CREATE TYPE unknown_symbol_status AS ENUM (
    'pending',      -- not yet processed
    'resolved',     -- successfully mapped to coin_id
    'unresolvable', -- tried, no match found
    'ignored'       -- manually marked to skip (noise/test symbols)
);

CREATE TABLE IF NOT EXISTS unknown_symbols (
    id              BIGSERIAL   PRIMARY KEY,
    exchange        TEXT        NOT NULL,             -- "binance", "bybit", "okx"
    raw_symbol      TEXT        NOT NULL,             -- "NEWTKUSDT", "XYZBTC"
    base_asset      TEXT,                             -- stripped base: "NEWTK", "XYZ"
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    seen_count      INTEGER     NOT NULL DEFAULT 1,
    status          unknown_symbol_status NOT NULL DEFAULT 'pending',
    resolved_coin_id TEXT       REFERENCES coins(coin_id) ON DELETE SET NULL,
    resolve_notes   TEXT,                             -- e.g. "fuzzy matched to 'newtok-finance'"
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (exchange, raw_symbol)
);

-- Hourly resolver job: find pending symbols
CREATE INDEX IF NOT EXISTS idx_unknown_pending
    ON unknown_symbols (first_seen_at ASC)
    WHERE status = 'pending';

-- Frequently seen unknowns = higher priority
CREATE INDEX IF NOT EXISTS idx_unknown_seen_count
    ON unknown_symbols (seen_count DESC)
    WHERE status = 'pending';

COMMENT ON TABLE unknown_symbols IS 'WS symbols without mapping. UnknownSymbolResolver does hourly fuzzy match. Never dropped — accumulate for manual review fallback.';
