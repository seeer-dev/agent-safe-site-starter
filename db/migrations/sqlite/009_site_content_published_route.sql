-- Published route/placement/sort_order isolation: freeze key, placement,
-- and sort_order into published_* columns at publish time, so a draft edit
-- that changes route path, placement filter, or render order does NOT
-- affect the currently-live published snapshot. This closes the B5 gap
-- where publishedColumns aliased published_title/published_body but read
-- key/placement/sort_order directly from the draft columns.

ALTER TABLE site_content ADD COLUMN published_key TEXT NOT NULL DEFAULT '';
ALTER TABLE site_content ADD COLUMN published_placement TEXT NOT NULL DEFAULT '';
ALTER TABLE site_content ADD COLUMN published_sort_order INTEGER NOT NULL DEFAULT 0;

-- Backfill published_* from existing published rows so currently-live
-- content keeps its route/placement/order after the migration.
UPDATE site_content
  SET published_key = key,
      published_placement = placement,
      published_sort_order = sort_order
  WHERE status = 'published' AND published_updated_unix > 0;

-- Partial unique index: at most one live snapshot may use a given
-- published_key. Only applies to rows that actually have a published
-- snapshot (published_key != ''). Prevents two live routes from
-- colliding on the same path.
CREATE UNIQUE INDEX IF NOT EXISTS idx_site_content_published_key
  ON site_content (published_key)
  WHERE published_key != '';
