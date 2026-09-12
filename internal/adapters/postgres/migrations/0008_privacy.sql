CREATE TABLE IF NOT EXISTS consent_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose TEXT NOT NULL CHECK (purpose IN ('marketing', 'analytics', 'necessary')),
    policy_version TEXT NOT NULL,
    granted BOOLEAN NOT NULL,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS consent_records_user_purpose_idx ON consent_records (user_id, purpose, created_at DESC);

CREATE TABLE IF NOT EXISTS privacy_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    request_type TEXT NOT NULL CHECK (request_type IN ('access', 'correction', 'deletion', 'grievance')),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_review', 'completed', 'rejected')),
    details JSONB NOT NULL DEFAULT '{}'::JSONB,
    due_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS privacy_requests_user_idx ON privacy_requests (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS marketing_preferences (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_marketing BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
