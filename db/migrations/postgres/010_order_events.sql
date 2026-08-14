CREATE TABLE IF NOT EXISTS order_events (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  actor_user_id TEXT NOT NULL DEFAULT '',
  from_status TEXT NOT NULL DEFAULT '',
  to_status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_unix BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_events_order_sequence
  ON order_events (order_id, sequence);
