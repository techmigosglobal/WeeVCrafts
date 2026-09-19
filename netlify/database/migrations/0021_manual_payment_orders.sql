ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS payment_method TEXT NOT NULL DEFAULT 'gateway'
        CHECK (payment_method IN ('gateway', 'cash_on_delivery'));

CREATE INDEX IF NOT EXISTS orders_payment_method_created_idx
    ON orders (payment_method, created_at DESC, id DESC);

INSERT INTO schema_migrations (version) VALUES (0021) ON CONFLICT (version) DO NOTHING;
