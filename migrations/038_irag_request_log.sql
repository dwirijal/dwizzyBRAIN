-- ============================================================
-- 038_irag_request_log.sql
-- irag gateway request log + L2 cache hit/miss tracking.
-- Used for: debugging, rate limit monitoring, provider health,
-- and L2 cache warm-up analytics.
-- ============================================================

CREATE TYPE irag_request_status AS ENUM (
    'success',
    'cache_hit_l1',   -- served from Valkey L1
    'cache_hit_l2',   -- served from TimescaleDB L2
    'provider_error', -- upstream API returned error
    'circuit_open',   -- circuit breaker was open
    'timeout',
    'fallback_used'   -- primary failed, fallback chain succeeded
);

CREATE TABLE IF NOT EXISTS irag_request_log (
    id              BIGSERIAL   PRIMARY KEY,
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Request context
    endpoint        TEXT        NOT NULL,             -- "/v1/ai/text/groq", "/v1/search/google"
    category        TEXT        NOT NULL,             -- "ai", "search", "download", "tools", "stalk"
    provider_used   TEXT,                             -- "kanata", "nexure", "ryzumi", "chocomilk", "ytdlp"
    fallback_chain  TEXT[],                           -- providers attempted before success

    -- Response
    status          irag_request_status NOT NULL,
    http_status     INTEGER,
    latency_ms      INTEGER,
    response_size_bytes INTEGER,

    -- Cache info
    cache_key       TEXT,
    cache_ttl_seconds INTEGER,

    -- Error detail
    error_code      TEXT,
    error_message   TEXT,

    -- Client context
    client_id       TEXT,                             -- JWT sub or IP hash
    is_premium      BOOLEAN     NOT NULL DEFAULT FALSE
);

-- Convert to hypertable for efficient time-range queries
SELECT create_hypertable(
    'irag_request_log',
    'requested_at',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

ALTER TABLE irag_request_log SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'category, provider_used',
    timescaledb.compress_orderby   = 'requested_at DESC'
);

SELECT add_compression_policy('irag_request_log', INTERVAL '1 day', if_not_exists => TRUE);

-- Keep 7 days of raw logs (per storage plan)
SELECT add_retention_policy('irag_request_log', INTERVAL '7 days', if_not_exists => TRUE);

-- Error monitoring
CREATE INDEX IF NOT EXISTS idx_irag_log_errors
    ON irag_request_log (requested_at DESC)
    WHERE status IN ('provider_error', 'timeout', 'circuit_open');

-- Provider health queries
CREATE INDEX IF NOT EXISTS idx_irag_log_provider
    ON irag_request_log (provider_used, requested_at DESC);

-- Endpoint analytics
CREATE INDEX IF NOT EXISTS idx_irag_log_endpoint
    ON irag_request_log (endpoint, requested_at DESC);

COMMENT ON TABLE irag_request_log IS '7d retention request log for irag gateway. Hypertable. Tracks cache hits, fallback chains, provider health, and latency.';
