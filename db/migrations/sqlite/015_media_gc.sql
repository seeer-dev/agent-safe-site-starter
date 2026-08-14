-- Canonical verified assets and durable media deletion jobs.
-- product_images uses a composite foreign key whose second column is
-- fixed to active so a product can never reference a verifying asset.

CREATE TABLE media_assets (
    object_key TEXT PRIMARY KEY,
    state TEXT NOT NULL CHECK (state IN ('verifying', 'active')),
    content_type TEXT NOT NULL,
    bytes INTEGER NOT NULL CHECK (bytes > 0 AND bytes <= 10485760),
    width INTEGER NOT NULL CHECK (width > 0 AND width <= 4096),
    height INTEGER NOT NULL CHECK (height > 0 AND height <= 4096),
    uploaded_by_user_id TEXT NOT NULL,
    verified_unix INTEGER NOT NULL,
    reservation_token TEXT NOT NULL DEFAULT '',
    reserved_unix INTEGER NOT NULL DEFAULT 0,
    unassociated_since_unix INTEGER NOT NULL DEFAULT 0,
    UNIQUE (object_key, state)
);

INSERT INTO media_assets
    (object_key, state, content_type, bytes, width, height, uploaded_by_user_id, verified_unix, reservation_token, reserved_unix, unassociated_since_unix)
SELECT mo.object_key, 'active', MIN(mo.content_type), MAX(mo.bytes), MAX(mo.width), MAX(mo.height),
       MIN(mo.uploaded_by_user_id), MAX(mo.verified_unix), '', 0,
       CASE WHEN EXISTS (SELECT 1 FROM product_images pi WHERE pi.object_key = mo.object_key)
            THEN 0 ELSE MAX(mo.verified_unix) END
FROM media_objects mo
GROUP BY mo.object_key;

ALTER TABLE product_images RENAME TO product_images_legacy_015;

CREATE TABLE product_images (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    object_key TEXT NOT NULL,
    asset_state TEXT NOT NULL DEFAULT 'active' CHECK (asset_state = 'active'),
    alt_text TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_unix INTEGER NOT NULL,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (object_key, asset_state) REFERENCES media_assets(object_key, state) ON DELETE RESTRICT
);

INSERT INTO product_images (id, product_id, object_key, asset_state, alt_text, sort_order, created_unix)
SELECT id, product_id, object_key, 'active', alt_text, sort_order, created_unix
FROM product_images_legacy_015;

DROP TABLE product_images_legacy_015;

CREATE INDEX idx_product_images_product ON product_images(product_id, sort_order);
CREATE INDEX idx_product_images_object_key ON product_images(object_key);
CREATE INDEX idx_media_assets_gc ON media_assets(state, unassociated_since_unix, reserved_unix);

CREATE TABLE media_gc_jobs (
    object_key TEXT PRIMARY KEY,
    created_unix INTEGER NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_attempt_unix INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_media_gc_jobs_created ON media_gc_jobs(created_unix, object_key);
