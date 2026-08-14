-- B4: Post-upload media verification registry + product images table.
-- media_objects is owned by the media module, product_images is owned by
-- the commerce module. Both tables are new - they do not exist in any
-- prior migration. The flat products.image / products.images columns
-- remain for backwards compatibility but are NOT backfilled as verified
-- metadata, the read path derives public URLs from product_images.
--
-- object_key is NOT unique: the same verified SHA-256 key can be
-- referenced by multiple source uploads (same user re-uploading the
-- same image to different temp keys). source_upload_key IS unique
-- to enforce one-registry-row-per-temp-upload idempotency. A lookup
-- index on object_key supports the commerce MediaVerifier adapter.

CREATE TABLE IF NOT EXISTS media_objects (
    id TEXT PRIMARY KEY,
    object_key TEXT NOT NULL,
    source_upload_key TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    bytes INTEGER NOT NULL CHECK (bytes > 0 AND bytes <= 10485760),
    width INTEGER NOT NULL CHECK (width > 0 AND width <= 4096),
    height INTEGER NOT NULL CHECK (height > 0 AND height <= 4096),
    uploaded_by_user_id TEXT NOT NULL,
    verified_unix INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_media_objects_object_key ON media_objects(object_key);

CREATE TABLE IF NOT EXISTS product_images (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    object_key TEXT NOT NULL,
    alt_text TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_unix INTEGER NOT NULL,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_product_images_product ON product_images(product_id, sort_order);
