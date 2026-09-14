CREATE TABLE IF NOT EXISTS seller_users (
    seller_id BIGINT NOT NULL REFERENCES sellers(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    permissions TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    added_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (seller_id, user_id)
);

CREATE INDEX IF NOT EXISTS seller_users_user_status_idx
    ON seller_users (user_id, status, seller_id);
