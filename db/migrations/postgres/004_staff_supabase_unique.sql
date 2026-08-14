-- Enforce uniqueness of non-empty supabase_user_id so that one Supabase
-- identity can be linked to at most one staff row. This prevents duplicate
-- linking and makes email-impersonation attempts fail at the DB layer.
CREATE UNIQUE INDEX IF NOT EXISTS idx_staff_supabase_user_unique
  ON staff_members (supabase_user_id)
  WHERE supabase_user_id != ''
