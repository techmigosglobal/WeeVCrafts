ALTER TABLE product_media
    ADD COLUMN IF NOT EXISTS public_url TEXT NOT NULL DEFAULT '';

ALTER TABLE product_media
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS product_media_pending_cleanup_idx
    ON product_media (status, created_at, id);
