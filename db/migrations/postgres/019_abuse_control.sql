-- 019_abuse_control.sql
-- Deployment-wide abuse control: shared fixed-window counters for public
-- mutation and expensive quote endpoints. The authoritative counter lives in
-- the application database so every replica observes the same state.
--
-- client_key is an opaque HMAC-derived identifier stamped by the edge (or a
-- peer-derived digest in no-edge local runs). Raw client addresses are never
-- stored here.

CREATE TABLE IF NOT EXISTS abuse_buckets (
  bucket TEXT NOT NULL,
  client_key TEXT NOT NULL,
  window_start_unix BIGINT NOT NULL,
  count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (bucket, client_key)
);

-- Expired-bucket cleanup deletes by window_start_unix, so keep it indexed
-- and the bounded cleanup pass never scans live rows.
CREATE INDEX IF NOT EXISTS idx_abuse_buckets_window
  ON abuse_buckets (window_start_unix);
