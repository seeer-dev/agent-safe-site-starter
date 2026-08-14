CREATE TABLE IF NOT EXISTS products (
  id TEXT PRIMARY KEY,
  sku TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  long_description TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  images TEXT NOT NULL DEFAULT '[]',
  category TEXT NOT NULL DEFAULT 'apparel',
  status TEXT NOT NULL DEFAULT 'draft',
  material TEXT NOT NULL DEFAULT '',
  origin TEXT NOT NULL DEFAULT '',
  price INTEGER NOT NULL DEFAULT 0,
  original_price INTEGER NOT NULL DEFAULT 0,
  stock INTEGER NOT NULL DEFAULT 0,
  tag TEXT NOT NULL DEFAULT '',
  rating DOUBLE PRECISION NOT NULL DEFAULT 0,
  reviews_count INTEGER NOT NULL DEFAULT 0,
  updated_unix BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_products_status ON products (status);
CREATE INDEX IF NOT EXISTS idx_products_category ON products (category);
CREATE INDEX IF NOT EXISTS idx_products_slug ON products (slug);

CREATE TABLE IF NOT EXISTS members (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active',
  tier TEXT NOT NULL DEFAULT 'regular',
  tags TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  total_orders INTEGER NOT NULL DEFAULT 0,
  total_spent INTEGER NOT NULL DEFAULT 0,
  updated_unix BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
  id TEXT PRIMARY KEY,
  member_id TEXT NOT NULL DEFAULT '',
  customer_name TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  items_json TEXT NOT NULL DEFAULT '[]',
  shipping_address TEXT NOT NULL DEFAULT '',
  shipping_method TEXT NOT NULL DEFAULT '',
  tracking_number TEXT NOT NULL DEFAULT '',
  subtotal INTEGER NOT NULL DEFAULT 0,
  discount INTEGER NOT NULL DEFAULT 0,
  shipping INTEGER NOT NULL DEFAULT 0,
  total INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'pending',
  payment_status TEXT NOT NULL DEFAULT 'unpaid',
  return_request_status TEXT NOT NULL DEFAULT '',
  payment_intent_id TEXT NOT NULL DEFAULT '',
  idempotency_key TEXT NOT NULL DEFAULT '',
  timeline_json TEXT NOT NULL DEFAULT '[]',
  expected_status TEXT NOT NULL DEFAULT '',
  updated_unix BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_member ON orders (member_id);
CREATE INDEX IF NOT EXISTS idx_orders_idempotency ON orders (idempotency_key);

CREATE TABLE IF NOT EXISTS promos (
  id TEXT PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL DEFAULT 'percent',
  value INTEGER NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  starts_unix BIGINT NOT NULL DEFAULT 0,
  expires_unix BIGINT NOT NULL DEFAULT 0,
  updated_unix BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS payment_methods (
  id TEXT PRIMARY KEY,
  method TEXT NOT NULL UNIQUE,
  provider_label TEXT NOT NULL DEFAULT '',
  environment TEXT NOT NULL DEFAULT 'sandbox',
  readiness_status TEXT NOT NULL DEFAULT 'pending_setup',
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  updated_unix BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS site_content (
  id TEXT PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  placement TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'draft',
  sort_order INTEGER NOT NULL DEFAULT 0,
  updated_unix BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_site_content_placement ON site_content (placement);
CREATE INDEX IF NOT EXISTS idx_site_content_status ON site_content (status);

CREATE TABLE IF NOT EXISTS staff_members (
  id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  email TEXT NOT NULL,
  role_label TEXT NOT NULL DEFAULT 'readonly',
  updated_unix BIGINT NOT NULL
);
