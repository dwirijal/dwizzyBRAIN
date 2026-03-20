CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL,
    avatar_url      TEXT,
    timezone        TEXT NOT NULL DEFAULT 'UTC',
    locale          TEXT NOT NULL DEFAULT 'id-ID',
    plan_override   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auth_identities (
    id                UUID PRIMARY KEY,
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider          TEXT NOT NULL,
    provider_user_id  TEXT NOT NULL,
    metadata_json     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_user_id)
);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id                UUID PRIMARY KEY,
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status            TEXT NOT NULL DEFAULT 'active',
    session_family_id UUID NOT NULL,
    ip_hash           TEXT NOT NULL DEFAULT '',
    user_agent_hash   TEXT NOT NULL DEFAULT '',
    last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at        TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS auth_sessions_user_id_idx ON auth_sessions (user_id);
CREATE INDEX IF NOT EXISTS auth_sessions_status_idx ON auth_sessions (status);
CREATE INDEX IF NOT EXISTS auth_sessions_family_idx ON auth_sessions (session_family_id);

CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
    id                   UUID PRIMARY KEY,
    session_id           UUID NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    token_hash           TEXT NOT NULL UNIQUE,
    rotated_from_token_id UUID REFERENCES auth_refresh_tokens(id) ON DELETE SET NULL,
    consumed_at          TIMESTAMPTZ,
    expires_at           TIMESTAMPTZ NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS auth_refresh_tokens_session_id_idx ON auth_refresh_tokens (session_id);
CREATE INDEX IF NOT EXISTS auth_refresh_tokens_expires_at_idx ON auth_refresh_tokens (expires_at);

CREATE TABLE IF NOT EXISTS auth_nonces (
    id              UUID PRIMARY KEY,
    wallet_address   TEXT NOT NULL,
    nonce           TEXT NOT NULL,
    purpose         TEXT NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS auth_nonces_wallet_idx ON auth_nonces (wallet_address);
CREATE INDEX IF NOT EXISTS auth_nonces_nonce_idx ON auth_nonces (nonce);
