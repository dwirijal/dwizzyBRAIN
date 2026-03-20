-- ============================================================
-- 036_ai_results_table.sql
-- LLM analysis output from AgentRouter.
-- Persisted for history, dedup, and premium API responses.
-- Published to ch:ai:result:{symbol} → api/ws + dwizzyBOT.
-- ============================================================

CREATE TABLE IF NOT EXISTS ai_results (
    id              BIGSERIAL   PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    coin_id         TEXT        NOT NULL REFERENCES coins(coin_id) ON DELETE CASCADE,
    symbol          TEXT        NOT NULL,

    -- Source signal that triggered this analysis
    signal_id       BIGINT      REFERENCES signals(id) ON DELETE SET NULL,

    -- Task type
    task_type       TEXT        NOT NULL,             -- "summarize", "analyze", "sentiment", "rag_query"

    -- LLM provider used (AgentRouter priority 1-7)
    provider        TEXT        NOT NULL,             -- "irag:groq", "groq_direct", "gemini_direct", "openrouter"
    model           TEXT,                             -- "llama3-8b-8192", "gemini-2.0-flash", etc.

    -- Output
    summary         TEXT,
    decision        TEXT,                             -- "BUY", "SELL", "HOLD", "WATCH", "NEUTRAL"
    confidence      NUMERIC(4, 3),                   -- 0.0 – 1.0
    reasoning       TEXT,
    key_factors     JSONB       DEFAULT '[]',         -- array of reasoning points
    risk_factors    JSONB       DEFAULT '[]',

    -- Price context at analysis time
    price_at_analysis NUMERIC(30, 10),

    -- Token usage
    prompt_tokens   INTEGER,
    completion_tokens INTEGER,
    latency_ms      INTEGER,

    -- Dedup: inflight lock cleared after write
    dedup_key       TEXT        UNIQUE                -- "{task_type}:{coin_id}:{signal_id}"
);

-- Latest analysis per coin
CREATE INDEX IF NOT EXISTS idx_ai_results_coin_time
    ON ai_results (coin_id, created_at DESC);

-- Task type filter (premium endpoint)
CREATE INDEX IF NOT EXISTS idx_ai_results_task
    ON ai_results (task_type, created_at DESC);

-- Recent for live feed
CREATE INDEX IF NOT EXISTS idx_ai_results_time
    ON ai_results (created_at DESC);

COMMENT ON TABLE ai_results IS 'LLM analysis output from AgentRouter. provider field tracks which priority was used. dedup_key prevents duplicate concurrent analysis.';
