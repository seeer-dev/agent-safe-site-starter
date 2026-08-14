-- 014_order_items.sql
-- B7: Per-item returned/restocked quantity model.
-- order_items persists the order line-item snapshot with returned_quantity
-- and restocked_quantity columns. The CHECK constraint enforces
-- 0 <= restocked_quantity <= returned_quantity <= quantity so that
-- restock can never exceed what was ordered or what was confirmed returned.
-- restock_idempotency stores the idempotency key + request fingerprint
-- for the per-item restock action, matching the CreateOrder idempotency pattern.
CREATE TABLE IF NOT EXISTS order_items (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  sku TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  price INTEGER NOT NULL DEFAULT 0,
  quantity INTEGER NOT NULL,
  line_total INTEGER NOT NULL DEFAULT 0,
  returned_quantity INTEGER NOT NULL DEFAULT 0,
  restocked_quantity INTEGER NOT NULL DEFAULT 0,
  created_unix INTEGER NOT NULL,
  CHECK (restocked_quantity >= 0),
  CHECK (returned_quantity >= 0),
  CHECK (restocked_quantity <= returned_quantity),
  CHECK (returned_quantity <= quantity)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_items_order_sku
  ON order_items (order_id, sku);

CREATE TABLE IF NOT EXISTS restock_idempotency (
  idempotency_key TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  request_fingerprint TEXT NOT NULL,
  response_json TEXT NOT NULL DEFAULT '',
  created_unix INTEGER NOT NULL,
  CHECK (response_json <> '')
);

-- Backfill order_items from existing orders.items_json.
-- items_json is a TEXT column holding a JSON array of
-- {"sku","name","price","quantity"} objects.
-- We cast to jsonb and use CROSS JOIN LATERAL WITH ORDINALITY to explode
-- the array, then aggregate per (order_id, sku). Duplicate SKUs in the
-- same order (which the old CreateOrder contract did not reject) are
-- merged: quantity and line_total are summed, name and price are taken
-- from the first occurrence (smallest ordinality) via array_agg.
-- If items_json is not valid JSON, jsonb_array_elements raises an error
-- and the migration fails — a corrupt items_json must be fixed before
-- upgrade, not silently skipped.
-- NOTE: Not live-verified against PostgreSQL (no live PG in this env).
-- SQLite parity is covered by migration upgrade tests.
WITH exploded AS (
  SELECT
    o.id AS order_id,
    o.updated_unix AS updated_unix,
    je.elem->>'sku' AS sku,
    COALESCE(je.elem->>'name', '') AS name,
    COALESCE((je.elem->>'price')::int, 0) AS price,
    COALESCE((je.elem->>'quantity')::int, 0) AS quantity,
    je.ord AS ord
  FROM orders o
  CROSS JOIN LATERAL jsonb_array_elements(o.items_json::jsonb) WITH ORDINALITY AS je(elem, ord)
  WHERE je.elem->>'sku' IS NOT NULL
    AND COALESCE((je.elem->>'quantity')::int, 0) > 0
)
INSERT INTO order_items (id, order_id, sku, name, price, quantity, line_total, returned_quantity, restocked_quantity, created_unix)
SELECT
  'oi_' || length(order_id) || ':' || order_id || '_' || length(sku) || ':' || sku,
  order_id,
  sku,
  COALESCE((array_agg(name ORDER BY ord))[1], ''),
  COALESCE((array_agg(price ORDER BY ord))[1], 0),
  SUM(quantity),
  SUM(price * quantity),
  0,
  0,
  MAX(updated_unix)
FROM exploded
GROUP BY order_id, sku;
