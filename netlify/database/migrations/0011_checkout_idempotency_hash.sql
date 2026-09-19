ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS idempotency_request_hash TEXT;

INSERT INTO schema_migrations (version) VALUES (0011) ON CONFLICT (version) DO NOTHING;
