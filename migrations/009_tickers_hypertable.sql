-- ============================================================
-- 009_tickers_hypertable.sql
-- Realtime ticker snapshots per coin per exchange.
-- Written by BinanceWSFetcher / BybitWSFetcher every tick.
-- TimescaleDB hypertable — partitioned by time (1 day chunks).
-- Retention: 30 days (raw), 1 year (compressed daily aggregate).
-- ============================================================

CREATE TABLE IF NOT EXISTS tickers (
    time            TIMESTAMPTZ NOT NULL,
    coin_id         TEXT        NOT NULL,             -- CoinGecko universal key
    exchange        TEXT        NOT NULL,             -- "binance", "bybit", "okx"
    symbol          TEXT        NOT NULL,             -- "BTCUSDT"

    -- Price
    price           NUMERIC(30, 10) NOT NULL,
    price_open_24h  NUMERIC(30, 10),
    price_high_24h  NUMERIC(30, 10),
    price_low_24h   NUMERIC(30, 10),
    price_change_24h NUMERIC(30, 10),
    price_change_pct_24h NUMERIC(10, 4),

    -- Volume
    volume_base_24h  NUMERIC(30, 4),
    volume_quote_24h NUMERIC(30, 4),

    -- Order book best
    bid_price       NUMERIC(30, 10),
    ask_price       NUMERIC(30, 10),
    spread_pct      NUMERIC(10, 6)  GENERATED ALWAYS AS (
        CASE
            WHEN bid_price > 0 THEN ((ask_price - bid_price) / bid_price) * 100
            ELSE NULL
        END
    ) STORED,

    -- Last trade
    last_trade_time TIMESTAMPTZ,

    PRIMARY KEY (time, coin_id, exchange)
);

-- Convert to hypertable
SELECT create_hypertable(
    'tickers',
    'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Compression: compress chunks older than 1 day
ALTER TABLE tickers SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'coin_id, exchange',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('tickers', INTERVAL '1 day', if_not_exists => TRUE);

-- Retention: drop raw data older than 30 days
SELECT add_retention_policy('tickers', INTERVAL '30 days', if_not_exists => TRUE);

-- Fast latest-per-coin queries (detail page tickers section)
CREATE INDEX IF NOT EXISTS idx_tickers_coin_time
    ON tickers (coin_id, time DESC);

-- Exchange-level queries
CREATE INDEX IF NOT EXISTS idx_tickers_exchange_symbol_time
    ON tickers (exchange, symbol, time DESC);

COMMENT ON TABLE tickers IS 'Realtime ticker time-series. 30d raw retention + compression. Segmented by coin_id+exchange for fast latest lookup.';
