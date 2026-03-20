-- ============================================================
-- 011_exchange_spread_history.sql
-- Bid/ask spread snapshots per exchange per coin.
-- Written by RecordSpread every 5 minutes.
-- Retention: 30 days raw. Compressed after 1 day.
-- ============================================================

CREATE TABLE IF NOT EXISTS exchange_spread_history (
    time            TIMESTAMPTZ NOT NULL,
    coin_id         TEXT        NOT NULL,
    exchange        TEXT        NOT NULL,
    symbol          TEXT        NOT NULL,

    bid_price       NUMERIC(30, 10) NOT NULL,
    ask_price       NUMERIC(30, 10) NOT NULL,
    spread_abs      NUMERIC(30, 10) NOT NULL,         -- ask - bid
    spread_pct      NUMERIC(10, 6)  NOT NULL,         -- (spread_abs / bid) * 100
    mid_price       NUMERIC(30, 10) GENERATED ALWAYS AS ((bid_price + ask_price) / 2) STORED,
    depth_bid_usd   NUMERIC(30, 2),                   -- USD depth at top 5 bid levels
    depth_ask_usd   NUMERIC(30, 2),

    PRIMARY KEY (time, coin_id, exchange)
);

SELECT create_hypertable(
    'exchange_spread_history',
    'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

ALTER TABLE exchange_spread_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'coin_id, exchange',
    timescaledb.compress_orderby   = 'time DESC'
);

SELECT add_compression_policy('exchange_spread_history', INTERVAL '1 day', if_not_exists => TRUE);
SELECT add_retention_policy('exchange_spread_history', INTERVAL '30 days', if_not_exists => TRUE);

-- Arbitrage engine: compare spread across exchanges for same coin
CREATE INDEX IF NOT EXISTS idx_spread_coin_time
    ON exchange_spread_history (coin_id, time DESC);

COMMENT ON TABLE exchange_spread_history IS 'Spread snapshots every 5 min. Used by ArbitrageEngine for opportunity detection. 30d retention.';
