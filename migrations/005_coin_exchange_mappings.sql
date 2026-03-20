-- ============================================================
-- 005_coin_exchange_mappings.sql
-- Bidirectional mapping: coin_id (CoinGecko) ↔ exchange symbol
-- Supports Binance, Bybit, OKX, KuCoin, Gate, Kraken, MEXC, HTX
-- CoinPaprika id also tracked here.
-- ============================================================

CREATE TYPE exchange_symbol_status AS ENUM (
    'active',
    'delisted',
    'not_listed',
    'dex_only',
    'unknown'
);

CREATE TABLE IF NOT EXISTS coin_exchange_mappings (
    id              BIGSERIAL   PRIMARY KEY,
    coin_id         TEXT        NOT NULL REFERENCES coins(coin_id) ON DELETE CASCADE,
    exchange        TEXT        NOT NULL,             -- "binance", "bybit", "okx", "kucoin", "gate", "kraken", "mexc", "htx", "coinpaprika"
    exchange_symbol TEXT        NOT NULL,             -- "BTCUSDT", "btc-bitcoin", "BTC-USDT"
    base_asset      TEXT        NOT NULL,             -- "BTC", "ETH"
    quote_asset     TEXT        NOT NULL DEFAULT 'USDT', -- "USDT", "USDC", "BUSD", "BTC"
    status          exchange_symbol_status NOT NULL DEFAULT 'active',
    is_primary      BOOLEAN     NOT NULL DEFAULT FALSE, -- preferred pair for this exchange (USDT preferred)
    verified_at     TIMESTAMPTZ,                      -- last confirmed still live on exchange
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (exchange, exchange_symbol)
);

-- Primary lookup: coin_id → exchange symbol
CREATE INDEX IF NOT EXISTS idx_cem_coin_exchange
    ON coin_exchange_mappings (coin_id, exchange)
    WHERE status = 'active';

-- Reverse lookup: exchange symbol → coin_id
CREATE INDEX IF NOT EXISTS idx_cem_symbol_exchange
    ON coin_exchange_mappings (UPPER(exchange_symbol), exchange)
    WHERE status = 'active';

-- Find all active primaries for a coin
CREATE INDEX IF NOT EXISTS idx_cem_primary
    ON coin_exchange_mappings (coin_id)
    WHERE is_primary = TRUE AND status = 'active';

COMMENT ON TABLE coin_exchange_mappings IS 'Symbol mapping layer. coin_id is universal key. MappingBuilder auto-builds from exchange info. Manual overrides for rebrands (MATIC→POL).';
COMMENT ON COLUMN coin_exchange_mappings.is_primary IS 'TRUE = preferred pair for this exchange. Quote priority: USDT > USDC > BUSD > BTC.';
