package sitecontent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

var ErrNotFound = errors.New("site content not found")

// ErrStaleVersion is returned by conditional UPDATE methods when the
// expected_draft_version does not match the current draft_version in the
// database. This means another edit happened between the client's read
// and the write; the caller must re-read and retry.
var ErrStaleVersion = errors.New("stale draft version")

// ErrApprovalConflict is returned by Publish when the conditional UPDATE
// affected 0 rows due to a missing, stale, or expired approval (not due
// to a version mismatch, which is ErrStaleVersion).
var ErrApprovalConflict = errors.New("no current approval")

// ErrConflict is returned by DeleteDraftIfStillDraft when the row exists
// but is no longer a draft (was published between the read and the delete).
var ErrConflict = errors.New("row state changed")

type Store interface {
	ListAll(ctx context.Context) ([]SiteContent, error)
	ListPublished(ctx context.Context) ([]SiteContent, error)
	ListByPlacement(ctx context.Context, placement string) ([]SiteContent, error)
	GetByID(ctx context.Context, id string) (SiteContent, error)
	// Insert creates a new row. Only used by Create (always draft).
	Insert(ctx context.Context, sc SiteContent) error
	// UpdateDraft atomically updates draft fields and increments
	// draft_version, conditional on expectedDraftVersion matching the
	// current draft_version. Returns ErrStaleVersion if the version
	// does not match, ErrNotFound if the row does not exist.
	UpdateDraft(ctx context.Context, id string, expectedDraftVersion int, in SiteContentInput, updatedUnix int64) (SiteContent, error)
	// Approve atomically sets approval fields (approved_version =
	// draft_version, approver, timestamps) conditional on
	// expectedDraftVersion matching. Returns ErrStaleVersion on mismatch,
	// ErrNotFound if the row does not exist.
	Approve(ctx context.Context, id string, expectedDraftVersion int, approverUserID string, approvedUnix, approvedExpiryUnix int64) (SiteContent, error)
	// Publish atomically copies draft fields to published_* fields and
	// sets status='published', conditional on id, expectedDraftVersion
	// matching, approved_version = draft_version, and
	// approved_expiry_unix > now. Returns ErrStaleVersion on version
	// mismatch, ErrApprovalConflict if the approval is missing/stale/
	// expired, ErrNotFound if the row does not exist. On any failure the
	// published snapshot is unchanged.
	Publish(ctx context.Context, id string, expectedDraftVersion int, now int64) (SiteContent, error)
	Delete(ctx context.Context, id string) error
	// DeleteDraftIfStillDraft atomically deletes the row only if it is
	// still a draft (status='draft' and no published snapshot). This
	// prevents the TOCTOU where an update-only user reads a draft, a
	// concurrent publish promotes it, and the user then deletes a live
	// row. Returns ErrNotFound if the row does not exist, ErrConflict
	// if the row exists but is no longer a draft (was published).
	DeleteDraftIfStillDraft(ctx context.Context, id string) error
}

type SQLStore struct {
	db      *sql.DB
	dialect database.Dialect
}

func NewSQLStore(db *sql.DB, dialect database.Dialect) SQLStore {
	return SQLStore{db: db, dialect: dialect}
}

// adminColumns lists all columns for admin-facing queries (returns draft
// key/placement/title/body/sort_order plus the published_* fields and
// approval/version/expiry fields, including snapshot-scoped published
// approval metadata).
const adminColumns = `id, key, placement, title, body, status, sort_order, updated_unix, draft_version, approved_version, approver_user_id, approved_unix, approved_expiry_unix, published_title, published_body, published_key, published_placement, published_sort_order, published_updated_unix, published_version, published_approver_user_id, published_approved_unix, published_approval_expiry_unix`

// publishedColumns returns the published_* fields as
// key/placement/title/body/sort_order so the public API and renderer only
// see approved content. The draft key/placement/sort_order are NOT used —
// only the frozen published_* values. Governance metadata columns are
// NOT included here — they are only used in the WHERE clause for expiry
// filtering, never returned to the public API.
const publishedColumns = `id, published_key AS key, published_placement AS placement, published_title AS title, published_body AS body, status, published_sort_order AS sort_order, published_updated_unix AS updated_unix`

func (s SQLStore) ListAll(ctx context.Context) ([]SiteContent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+adminColumns+`
		FROM site_content ORDER BY sort_order ASC, updated_unix DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAdminRows(rows)
}

func (s SQLStore) ListPublished(ctx context.Context) ([]SiteContent, error) {
	now := time.Now().Unix()
	query := database.Bind(s.dialect, `SELECT `+publishedColumns+`
		FROM site_content WHERE status = 'published' AND published_updated_unix > 0 AND published_approval_expiry_unix > ? ORDER BY published_sort_order ASC, published_updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPublishedRows(rows)
}

func (s SQLStore) ListByPlacement(ctx context.Context, placement string) ([]SiteContent, error) {
	now := time.Now().Unix()
	query := database.Bind(s.dialect, `SELECT `+publishedColumns+`
		FROM site_content WHERE status = 'published' AND published_updated_unix > 0 AND published_approval_expiry_unix > ? AND published_placement = ? ORDER BY published_sort_order ASC, published_updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query, now, placement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPublishedRows(rows)
}

func (s SQLStore) GetByID(ctx context.Context, id string) (SiteContent, error) {
	query := database.Bind(s.dialect, `SELECT `+adminColumns+`
		FROM site_content WHERE id = ? LIMIT 1`)
	var sc SiteContent
	if err := s.db.QueryRowContext(ctx, query, id).Scan(
		&sc.ID, &sc.Key, &sc.Placement, &sc.Title, &sc.Body, &sc.Status, &sc.SortOrder, &sc.UpdatedUnix,
		&sc.DraftVersion, &sc.ApprovedVersion, &sc.ApproverUserID, &sc.ApprovedUnix, &sc.ApprovedExpiryUnix,
		&sc.PublishedTitle, &sc.PublishedBody, &sc.PublishedKey, &sc.PublishedPlacement, &sc.PublishedSortOrder, &sc.PublishedUpdatedUnix,
		&sc.PublishedVersion, &sc.PublishedApproverUserID, &sc.PublishedApprovedUnix, &sc.PublishedApprovalExpiryUnix,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SiteContent{}, ErrNotFound
		}
		return SiteContent{}, err
	}
	return sc, nil
}

// Insert creates a new site_content row. Only used by Create (always
// draft with draft_version=1). Not used for updates.
func (s SQLStore) Insert(ctx context.Context, sc SiteContent) error {
	query := database.Bind(s.dialect, `INSERT INTO site_content
		(id, key, placement, title, body, status, sort_order, updated_unix, draft_version, approved_version, approver_user_id, approved_unix, approved_expiry_unix, published_title, published_body, published_key, published_placement, published_sort_order, published_updated_unix, published_version, published_approver_user_id, published_approved_unix, published_approval_expiry_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if _, err := s.db.ExecContext(ctx, query, sc.ID, sc.Key, sc.Placement, sc.Title, sc.Body, sc.Status, sc.SortOrder, sc.UpdatedUnix,
		sc.DraftVersion, sc.ApprovedVersion, sc.ApproverUserID, sc.ApprovedUnix, sc.ApprovedExpiryUnix,
		sc.PublishedTitle, sc.PublishedBody, sc.PublishedKey, sc.PublishedPlacement, sc.PublishedSortOrder, sc.PublishedUpdatedUnix,
		sc.PublishedVersion, sc.PublishedApproverUserID, sc.PublishedApprovedUnix, sc.PublishedApprovalExpiryUnix); err != nil {
		return fmt.Errorf("insert site content: %w", err)
	}
	return nil
}

// UpdateDraft atomically updates the draft fields and increments
// draft_version, conditional on expectedDraftVersion matching the current
// draft_version. This prevents TOCTOU/lost-update: if another edit happened
// between the client's read and this write, the WHERE clause fails to match
// and no rows are updated. The published_* fields are never touched.
func (s SQLStore) UpdateDraft(ctx context.Context, id string, expectedDraftVersion int, in SiteContentInput, updatedUnix int64) (SiteContent, error) {
	query := database.Bind(s.dialect, `UPDATE site_content SET
		key = ?, placement = ?, title = ?, body = ?, sort_order = ?, updated_unix = ?,
		draft_version = draft_version + 1
		WHERE id = ? AND draft_version = ?`)
	res, err := s.db.ExecContext(ctx, query, in.Key, in.Placement, in.Title, in.Body, in.SortOrder, updatedUnix, id, expectedDraftVersion)
	if err != nil {
		return SiteContent{}, fmt.Errorf("update draft: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return SiteContent{}, fmt.Errorf("update draft rows affected: %w", err)
	}
	if affected == 0 {
		// Distinguish not-found from stale-version.
		if _, err := s.GetByID(ctx, id); err != nil {
			if errors.Is(err, ErrNotFound) {
				return SiteContent{}, ErrNotFound
			}
			return SiteContent{}, err
		}
		return SiteContent{}, ErrStaleVersion
	}
	return s.GetByID(ctx, id)
}

// Approve atomically sets the approval fields, conditional on
// expectedDraftVersion matching the current draft_version. The
// approved_version is set to the current draft_version, making the draft
// eligible for Publish until either the draft is edited (incrementing
// draft_version) or the expiry passes. The draft and published fields are
// never touched.
func (s SQLStore) Approve(ctx context.Context, id string, expectedDraftVersion int, approverUserID string, approvedUnix, approvedExpiryUnix int64) (SiteContent, error) {
	query := database.Bind(s.dialect, `UPDATE site_content SET
		approved_version = draft_version,
		approver_user_id = ?,
		approved_unix = ?,
		approved_expiry_unix = ?,
		updated_unix = ?
		WHERE id = ? AND draft_version = ?`)
	res, err := s.db.ExecContext(ctx, query, approverUserID, approvedUnix, approvedExpiryUnix, approvedUnix, id, expectedDraftVersion)
	if err != nil {
		return SiteContent{}, fmt.Errorf("approve: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return SiteContent{}, fmt.Errorf("approve rows affected: %w", err)
	}
	if affected == 0 {
		if _, err := s.GetByID(ctx, id); err != nil {
			if errors.Is(err, ErrNotFound) {
				return SiteContent{}, ErrNotFound
			}
			return SiteContent{}, err
		}
		return SiteContent{}, ErrStaleVersion
	}
	return s.GetByID(ctx, id)
}

// Publish atomically copies draft fields to published_* fields and sets
// status='published', conditional on:
//   - id matching
//   - draft_version = expectedDraftVersion (optimistic concurrency)
//   - approved_version = draft_version (approval not stale)
//   - approved_expiry_unix > now (approval not expired)
//
// It also freezes the current approval metadata (version, approver,
// timestamps) into published_version, published_approver_user_id,
// published_approved_unix, published_approval_expiry_unix. This is the
// snapshot-scoped governance: ListPublished/ListByPlacement filter on
// published_approval_expiry_unix > now, so the published snapshot
// disappears from public render when its frozen approval expires.
//
// If any condition fails, 0 rows are updated and the published snapshot
// is unchanged. The caller distinguishes stale-version from approval-
// conflict by re-reading the row.
func (s SQLStore) Publish(ctx context.Context, id string, expectedDraftVersion int, now int64) (SiteContent, error) {
	query := database.Bind(s.dialect, `UPDATE site_content SET
		status = 'published',
		published_title = title,
		published_body = body,
		published_key = TRIM(key),
		published_placement = placement,
		published_sort_order = sort_order,
		published_updated_unix = ?,
		published_version = draft_version,
		published_approver_user_id = approver_user_id,
		published_approved_unix = approved_unix,
		published_approval_expiry_unix = approved_expiry_unix,
		updated_unix = ?
		WHERE id = ? AND draft_version = ? AND approved_version = draft_version AND approved_expiry_unix > ?`)
	res, err := s.db.ExecContext(ctx, query, now, now, id, expectedDraftVersion, now)
	if err != nil {
		return SiteContent{}, fmt.Errorf("publish: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return SiteContent{}, fmt.Errorf("publish rows affected: %w", err)
	}
	if affected == 0 {
		// Distinguish not-found, stale-version, and approval-conflict.
		existing, err := s.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return SiteContent{}, ErrNotFound
			}
			return SiteContent{}, err
		}
		if existing.DraftVersion != expectedDraftVersion {
			return SiteContent{}, ErrStaleVersion
		}
		// Version matched but approval is missing/stale/expired.
		return SiteContent{}, ErrApprovalConflict
	}
	return s.GetByID(ctx, id)
}

func (s SQLStore) Delete(ctx context.Context, id string) error {
	query := database.Bind(s.dialect, `DELETE FROM site_content WHERE id = ?`)
	if _, err := s.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("delete site content: %w", err)
	}
	return nil
}

// DeleteDraftIfStillDraft atomically deletes the row only if it is still
// a draft with no published snapshot. This prevents the TOCTOU where an
// update-only user reads a draft, a concurrent publish promotes it, and
// the user then deletes a live row. The WHERE clause ensures the delete
// only succeeds if the row is still a draft; if 0 rows are affected, we
// re-read to distinguish not-found from state-changed.
func (s SQLStore) DeleteDraftIfStillDraft(ctx context.Context, id string) error {
	query := database.Bind(s.dialect, `DELETE FROM site_content WHERE id = ? AND status = 'draft' AND published_updated_unix = 0`)
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete draft: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete draft rows affected: %w", err)
	}
	if affected == 0 {
		// Distinguish not-found from state-changed (was published).
		if _, err := s.GetByID(ctx, id); err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		return ErrConflict
	}
	return nil
}

func scanAdminRows(rows *sql.Rows) ([]SiteContent, error) {
	var out []SiteContent
	for rows.Next() {
		var sc SiteContent
		if err := rows.Scan(
			&sc.ID, &sc.Key, &sc.Placement, &sc.Title, &sc.Body, &sc.Status, &sc.SortOrder, &sc.UpdatedUnix,
			&sc.DraftVersion, &sc.ApprovedVersion, &sc.ApproverUserID, &sc.ApprovedUnix, &sc.ApprovedExpiryUnix,
			&sc.PublishedTitle, &sc.PublishedBody, &sc.PublishedKey, &sc.PublishedPlacement, &sc.PublishedSortOrder, &sc.PublishedUpdatedUnix,
			&sc.PublishedVersion, &sc.PublishedApproverUserID, &sc.PublishedApprovedUnix, &sc.PublishedApprovalExpiryUnix,
		); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func scanPublishedRows(rows *sql.Rows) ([]SiteContent, error) {
	var out []SiteContent
	for rows.Next() {
		var sc SiteContent
		if err := rows.Scan(
			&sc.ID, &sc.Key, &sc.Placement, &sc.Title, &sc.Body, &sc.Status, &sc.SortOrder, &sc.UpdatedUnix,
		); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}
