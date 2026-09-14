CREATE TABLE IF NOT EXISTS inventory_transactions (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES sellers(id) ON DELETE RESTRICT,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id) ON DELETE RESTRICT,
    actor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    delta INTEGER NOT NULL CHECK (delta <> 0),
    before_quantity INTEGER NOT NULL CHECK (before_quantity >= 0),
    after_quantity INTEGER NOT NULL CHECK (after_quantity >= 0),
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS inventory_transactions_seller_created_idx
    ON inventory_transactions (seller_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS inventory_transactions_variant_created_idx
    ON inventory_transactions (variant_id, created_at DESC, id DESC);
