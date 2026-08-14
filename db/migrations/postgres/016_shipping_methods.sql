-- Admin-managed shipping methods for public discovery (rev 10 slice 1).
-- No seed rows and no fee values. Quote/order remain fail-closed until
-- a later slice consumes these rows as the fee schedule.

CREATE TABLE IF NOT EXISTS shipping_methods (
  id TEXT PRIMARY KEY,
  method TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  fee INTEGER NOT NULL DEFAULT 0 CHECK (fee >= 0),
  free_threshold INTEGER CHECK (free_threshold IS NULL OR free_threshold > 0),
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
  version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
  updated_unix BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_shipping_methods_sort ON shipping_methods (sort_order, method);
