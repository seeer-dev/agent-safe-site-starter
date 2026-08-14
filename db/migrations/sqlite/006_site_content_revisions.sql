-- Published revision separation: keep draft and published copy in separate
-- columns on the same row. The draft fields (title body) hold the working
-- copy. The published_* fields hold the currently-live approved copy.
-- This lets an editor modify the draft without taking the published
-- version offline. Publish atomically copies draft to published.

ALTER TABLE site_content ADD COLUMN published_title TEXT NOT NULL DEFAULT '';
ALTER TABLE site_content ADD COLUMN published_body TEXT NOT NULL DEFAULT '';
ALTER TABLE site_content ADD COLUMN published_updated_unix INTEGER NOT NULL DEFAULT 0;

-- Backfill published_* from existing published rows so currently-live
-- content remains live after the migration.
UPDATE site_content
  SET published_title = title,
      published_body = body,
      published_updated_unix = updated_unix
  WHERE status = 'published' AND published_updated_unix = 0;
