-- ============================================================
-- 004_developer_data.sql
-- GitHub stats and commit activity per coin
-- Sourced from CoinGecko /coins/{id} developer_data field
-- Refreshed every 24 hours alongside cold_coin_data
-- ============================================================

CREATE TABLE IF NOT EXISTS developer_data (
    coin_id                 TEXT        PRIMARY KEY REFERENCES coins(coin_id) ON DELETE CASCADE,

    forks                   INTEGER,
    stars                   INTEGER,
    subscribers             INTEGER,
    total_issues            INTEGER,
    closed_issues           INTEGER,
    pull_requests_merged    INTEGER,
    pull_request_contributors INTEGER,

    -- Rolling commit windows
    commits_4_weeks         INTEGER,
    commits_last_year       INTEGER,

    -- Weekly commit activity (52-week array from CoinGecko)
    commit_activity_4_weeks JSONB DEFAULT '[]',

    -- Code addition/deletion stats
    additions_4_weeks       BIGINT,
    deletions_4_weeks       BIGINT,

    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Find most active projects
CREATE INDEX IF NOT EXISTS idx_devdata_stars
    ON developer_data (stars DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_devdata_commits
    ON developer_data (commits_4_weeks DESC NULLS LAST);

COMMENT ON TABLE developer_data IS 'GitHub activity stats per coin. 24h refresh from CoinGecko developer_data.';
