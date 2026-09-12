ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS idempotency_request_hash TEXT;
