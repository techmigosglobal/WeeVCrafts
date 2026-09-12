ALTER TABLE product_media
    DROP CONSTRAINT IF EXISTS product_media_status_check;

ALTER TABLE product_media
    ADD CONSTRAINT product_media_status_check
    CHECK (status IN ('pending', 'ready', 'rejected', 'deleted', 'deleting'));
