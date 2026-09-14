-- 018_curatory_storefront.sql
-- Curatory storefront port data plane: category entities, product variants,
-- moderated product comments, coupon-capable promos, extended order fields,
-- notification templates/logs, and a governed single-row store settings
-- record. All columns are additive or new tables, existing rows keep working.

CREATE TABLE IF NOT EXISTS categories (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
  updated_unix INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_categories_active_sort
  ON categories (is_active, sort_order);

CREATE TABLE IF NOT EXISTS product_variants (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  sku TEXT NOT NULL DEFAULT '',
  price_delta INTEGER NOT NULL DEFAULT 0,
  stock INTEGER NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0,
  updated_unix INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_variants_product_name
  ON product_variants (product_id, name);
CREATE INDEX IF NOT EXISTS idx_product_variants_sku
  ON product_variants (sku);

CREATE TABLE IF NOT EXISTS product_comments (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  nickname TEXT NOT NULL,
  content TEXT NOT NULL,
  rating INTEGER CHECK (rating IS NULL OR (rating BETWEEN 1 AND 5)),
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
  admin_reply TEXT NOT NULL DEFAULT '',
  replied_unix INTEGER NOT NULL DEFAULT 0,
  created_unix INTEGER NOT NULL,
  updated_unix INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_product_comments_product_status
  ON product_comments (product_id, status);
CREATE INDEX IF NOT EXISTS idx_product_comments_status_created
  ON product_comments (status, created_unix DESC);

ALTER TABLE payment_methods ADD COLUMN fee INTEGER NOT NULL DEFAULT 0;

ALTER TABLE promos ADD COLUMN min_subtotal INTEGER NOT NULL DEFAULT 0;
ALTER TABLE promos ADD COLUMN usage_limit INTEGER;
ALTER TABLE promos ADD COLUMN used_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE products ADD COLUMN is_featured INTEGER NOT NULL DEFAULT 0 CHECK (is_featured IN (0, 1));
ALTER TABLE products ADD COLUMN sold_count INTEGER NOT NULL DEFAULT 0 CHECK (sold_count >= 0);

ALTER TABLE orders ADD COLUMN recipient_name TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN recipient_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN city TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN district TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN cvs_store_name TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN cvs_store_address TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN invoice_type TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN invoice_tax_id TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN payment_fee INTEGER NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN coupon_code TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN buyer_note TEXT NOT NULL DEFAULT '';

ALTER TABLE articles ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1));
ALTER TABLE articles ADD COLUMN publish_at_unix INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS notification_templates (
  id TEXT PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL DEFAULT '',
  subject TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  is_enabled INTEGER NOT NULL DEFAULT 1 CHECK (is_enabled IN (0, 1)),
  updated_unix INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_logs (
  id TEXT PRIMARY KEY,
  code TEXT NOT NULL DEFAULT '',
  order_id TEXT NOT NULL DEFAULT '',
  recipient TEXT NOT NULL DEFAULT '',
  subject TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'skipped' CHECK (status IN ('sent', 'skipped', 'failed')),
  provider TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  created_unix INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_logs_order ON notification_logs (order_id);
CREATE INDEX IF NOT EXISTS idx_notification_logs_created ON notification_logs (created_unix DESC);

-- Governed single-row store settings: admin edits draft_json, publish copies
-- draft to published. The storefront bootstrap and the renderer read only
-- published_json, so an unpublished draft never leaks to the public site.
CREATE TABLE IF NOT EXISTS store_settings (
  id TEXT PRIMARY KEY,
  draft_json TEXT NOT NULL DEFAULT '{}',
  published_json TEXT NOT NULL DEFAULT '{}',
  draft_updated_unix INTEGER NOT NULL DEFAULT 0,
  published_unix INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 1,
  updated_unix INTEGER NOT NULL
);
