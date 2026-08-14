-- 011_order_version.sql
-- Add integer version column to orders for expected_version optimistic
-- concurrency. Each mutation increments version, clients send expected_version
-- and the store guards atomically with WHERE version = expected_version.
-- Existing orders default to version 1. This aligns with INTEGRATION_PLAN.md
-- section 6.2: "目標 status 與 expected_version 只存在 mutation request，
-- 不保存成 order column。Store 使用 UPDATE ... SET version = version + 1
-- WHERE id = ? AND version = ? with affected rows = 0 meaning 409 conflict."
ALTER TABLE orders ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
