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
-- items_json is a JSON array of {"sku","name","price","quantity"} objects.
-- We use json_each to explode the array and INSERT one order_items row per
-- SKU per order. If an order has duplicate SKUs in items_json (which the
-- old CreateOrder contract did not reject), we aggregate by summing
-- quantity and line_total. The name and price are taken from the first
-- occurrence of each SKU (smallest array index). If items_json is not
-- valid JSON, json_each raises an error and the migration fails — a
-- corrupt items_json must be fixed before upgrade, not silently skipped.
INSERT INTO order_items (id, order_id, sku, name, price, quantity, line_total, returned_quantity, restocked_quantity, created_unix)
SELECT
  'oi_' || length(agg.order_id) || ':' || agg.order_id || '_' || length(agg.sku) || ':' || agg.sku,
  agg.order_id,
  agg.sku,
  COALESCE(agg.name, ''),
  agg.price,
  agg.total_qty,
  agg.total_line,
  0,
  0,
  agg.updated_unix
FROM (
  SELECT
    o.id AS order_id,
    o.updated_unix AS updated_unix,
    json_extract(je.value, '$.sku') AS sku,
    json_extract(
      (SELECT je2.value FROM json_each(o.items_json) AS je2
       WHERE json_extract(je2.value, '$.sku') = json_extract(je.value, '$.sku')
       ORDER BY je2.key LIMIT 1),
      '$.name'
    ) AS name,
    COALESCE(json_extract(
      (SELECT je2.value FROM json_each(o.items_json) AS je2
       WHERE json_extract(je2.value, '$.sku') = json_extract(je.value, '$.sku')
       ORDER BY je2.key LIMIT 1),
      '$.price'
    ), 0) AS price,
    SUM(COALESCE(json_extract(je.value, '$.quantity'), 0)) AS total_qty,
    SUM(COALESCE(json_extract(je.value, '$.price'), 0) * COALESCE(json_extract(je.value, '$.quantity'), 0)) AS total_line
  FROM orders o, json_each(o.items_json) AS je
  WHERE json_extract(je.value, '$.sku') IS NOT NULL
    AND COALESCE(json_extract(je.value, '$.quantity'), 0) > 0
  GROUP BY o.id, o.updated_unix, json_extract(je.value, '$.sku')
) agg;
