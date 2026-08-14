-- Staff lifecycle: link Supabase users to canonical staff rows and add
-- disable/enable status. Capabilities are derived server-side from the
-- active staff row, never from identity-provider claims alone.

ALTER TABLE staff_members ADD COLUMN IF NOT EXISTS supabase_user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE staff_members ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

CREATE INDEX IF NOT EXISTS idx_staff_supabase_user ON staff_members (supabase_user_id);
CREATE INDEX IF NOT EXISTS idx_staff_status ON staff_members (status);

-- Idempotency: a unique index on idempotency_key prevents concurrent
-- duplicate-key races. Empty keys are allowed (multiple rows can share
-- the empty string), so use a partial index that only enforces uniqueness
-- on non-empty keys.
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_idempotency_unique
  ON orders (idempotency_key)
  WHERE idempotency_key != ''
