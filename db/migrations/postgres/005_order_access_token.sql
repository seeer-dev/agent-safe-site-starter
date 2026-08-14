-- Customer order access: replace enumerable email-based lookup with an
-- opaque per-order access token. The token is generated at order creation
-- time and required for guest order access. This prevents enumeration of
-- orders by sequential ID or email guessing.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS access_token TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_orders_access_token ON orders (access_token);
