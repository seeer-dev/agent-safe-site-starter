-- 012_site_content_approval.sql
-- Add approval/version/expiry gate to site_content. Each material draft edit
-- increments draft_version, which invalidates any prior approval (approved_version
-- must equal the current draft_version for publish to proceed). The approve
-- action records the approver identity, approval timestamp, and expiry timestamp.
-- Publish requires content.publish capability AND a current (non-stale,
-- non-expired) approval. This closes the B5/REQ-006/AC-011 gap where Publish
-- could promote any draft without an explicit approval step.
--
-- Snapshot-scoped governance: Publish also freezes the current approval
-- metadata (version, approver, timestamps) into published_* columns.
-- ListPublished/ListByPlacement filter on published_approval_expiry_unix > now,
-- so a published snapshot whose approval has expired is automatically absent
-- from public render. Existing unapproved published rows default to 0,
-- which is fail-closed absent (0 > now is false).
--
-- All mutations use atomic conditional UPDATE (WHERE draft_version = ?) to
-- prevent TOCTOU/lost-update races. No read-modify-upsert is used for
-- approve/publish/update.
ALTER TABLE site_content ADD COLUMN draft_version INTEGER NOT NULL DEFAULT 1 CHECK (draft_version >= 1);
ALTER TABLE site_content ADD COLUMN approved_version INTEGER NOT NULL DEFAULT 0 CHECK (approved_version >= 0);
ALTER TABLE site_content ADD COLUMN approver_user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE site_content ADD COLUMN approved_unix INTEGER NOT NULL DEFAULT 0 CHECK (approved_unix >= 0);
ALTER TABLE site_content ADD COLUMN approved_expiry_unix INTEGER NOT NULL DEFAULT 0 CHECK (approved_expiry_unix >= 0);
ALTER TABLE site_content ADD COLUMN published_version INTEGER NOT NULL DEFAULT 0 CHECK (published_version >= 0);
ALTER TABLE site_content ADD COLUMN published_approver_user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE site_content ADD COLUMN published_approved_unix INTEGER NOT NULL DEFAULT 0 CHECK (published_approved_unix >= 0);
ALTER TABLE site_content ADD COLUMN published_approval_expiry_unix INTEGER NOT NULL DEFAULT 0 CHECK (published_approval_expiry_unix >= 0);
