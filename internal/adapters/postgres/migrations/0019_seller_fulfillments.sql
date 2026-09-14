CREATE TABLE IF NOT EXISTS seller_fulfillments (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES sellers(id) ON DELETE RESTRICT,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'shipped', 'delivered')),
    carrier TEXT NOT NULL DEFAULT '',
    tracking_number TEXT NOT NULL DEFAULT '',
    last_note TEXT NOT NULL DEFAULT '',
    updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (seller_id, order_id)
);

CREATE INDEX IF NOT EXISTS seller_fulfillments_seller_status_idx
    ON seller_fulfillments (seller_id, status, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS seller_fulfillments_order_idx
    ON seller_fulfillments (order_id, seller_id);
