-- Durable short-lived state required by Netlify's stateless Go functions.
CREATE TABLE IF NOT EXISTS runtime_web_sessions (
    session_hash CHAR(64) PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_slugs TEXT[] NOT NULL DEFAULT '{}',
    csrf_token TEXT NOT NULL,
    user_agent TEXT NOT NULL DEFAULT '',
    ip_address TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS runtime_web_sessions_user_expiry_idx
    ON runtime_web_sessions (user_id, expires_at, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS runtime_web_sessions_expiry_idx
    ON runtime_web_sessions (expires_at);

CREATE TABLE IF NOT EXISTS runtime_rate_limits (
    bucket_key CHAR(64) PRIMARY KEY,
    hits INTEGER NOT NULL CHECK (hits > 0),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS runtime_rate_limits_expiry_idx
    ON runtime_rate_limits (expires_at);

CREATE TABLE IF NOT EXISTS runtime_cache (
    cache_key TEXT PRIMARY KEY,
    value BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS runtime_cache_expiry_idx
    ON runtime_cache (expires_at);

ALTER TABLE product_media ADD COLUMN IF NOT EXISTS object_bytes BYTEA;

INSERT INTO schema_migrations (version) VALUES (0022) ON CONFLICT (version) DO NOTHING;
