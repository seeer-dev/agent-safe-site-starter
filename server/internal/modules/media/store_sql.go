package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// SQLRegistryStore owns both the verified-media registry and its GC queue.
type SQLRegistryStore struct {
	db      *sql.DB
	dialect database.Dialect
}

func NewSQLRegistryStore(db *sql.DB, dialect database.Dialect) SQLRegistryStore {
	return SQLRegistryStore{db: db, dialect: dialect}
}

func (s SQLRegistryStore) ReserveVerified(ctx context.Context, obj MediaObject, token string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin media reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var pending int
	jobQuery := database.Bind(s.dialect, `SELECT COUNT(*) FROM media_gc_jobs WHERE object_key = ?`)
	if err := tx.QueryRowContext(ctx, jobQuery, obj.ObjectKey).Scan(&pending); err != nil {
		return false, fmt.Errorf("check media deletion job: %w", err)
	}
	if pending != 0 {
		return false, ErrDeletionPending
	}

	query := database.Bind(s.dialect, `SELECT state, content_type, bytes, width, height, uploaded_by_user_id
		FROM media_assets WHERE object_key = ?`)
	if s.dialect == database.Postgres {
		query += ` FOR UPDATE`
	}
	var state, contentType, owner string
	var bytes int64
	var width, height int
	err = tx.QueryRowContext(ctx, query, obj.ObjectKey).Scan(&state, &contentType, &bytes, &width, &height, &owner)
	switch {
	case err == nil:
		if state != "active" {
			return false, ErrVerificationInProgress
		}
		if contentType != obj.ContentType || bytes != obj.Bytes || width != obj.Width || height != obj.Height || owner != obj.UploadedByUserID {
			return false, fmt.Errorf("media asset metadata mismatch")
		}
		update := database.Bind(s.dialect, `UPDATE media_assets
			SET verified_unix = ?, unassociated_since_unix = CASE
				WHEN EXISTS (SELECT 1 FROM product_images WHERE object_key = ?) THEN 0 ELSE ? END
			WHERE object_key = ? AND state = 'active'`)
		if _, err := tx.ExecContext(ctx, update, obj.VerifiedUnix, obj.ObjectKey, obj.VerifiedUnix, obj.ObjectKey); err != nil {
			return false, fmt.Errorf("renew media asset: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit media renewal: %w", err)
		}
		return false, nil
	case !errors.Is(err, sql.ErrNoRows):
		return false, fmt.Errorf("read media asset: %w", err)
	}

	insert := database.Bind(s.dialect, `INSERT INTO media_assets
		(object_key, state, content_type, bytes, width, height, uploaded_by_user_id, verified_unix, reservation_token, reserved_unix, unassociated_since_unix)
		VALUES (?, 'verifying', ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
	if _, err := tx.ExecContext(ctx, insert, obj.ObjectKey, obj.ContentType, obj.Bytes, obj.Width, obj.Height, obj.UploadedByUserID, obj.VerifiedUnix, token, obj.VerifiedUnix); err != nil {
		if database.IsUniqueViolation(err) {
			return false, ErrVerificationInProgress
		}
		return false, fmt.Errorf("reserve media asset: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit media reservation: %w", err)
	}
	return true, nil
}

func (s SQLRegistryStore) CompleteVerified(ctx context.Context, obj MediaObject, token string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin media completion: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var pending int
	jobQuery := database.Bind(s.dialect, `SELECT COUNT(*) FROM media_gc_jobs WHERE object_key = ?`)
	if err := tx.QueryRowContext(ctx, jobQuery, obj.ObjectKey).Scan(&pending); err != nil {
		return fmt.Errorf("check media deletion job: %w", err)
	}
	if pending != 0 {
		return ErrDeletionPending
	}

	query := database.Bind(s.dialect, `SELECT state, reservation_token FROM media_assets WHERE object_key = ?`)
	if s.dialect == database.Postgres {
		query += ` FOR UPDATE`
	}
	var state, storedToken string
	if err := tx.QueryRowContext(ctx, query, obj.ObjectKey).Scan(&state, &storedToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDeletionPending
		}
		return fmt.Errorf("read media reservation: %w", err)
	}
	if state == "verifying" {
		if storedToken == "" || storedToken != token {
			return ErrVerificationInProgress
		}
		activate := database.Bind(s.dialect, `UPDATE media_assets
			SET state = 'active', reservation_token = '', reserved_unix = 0,
				verified_unix = ?, unassociated_since_unix = ?
			WHERE object_key = ? AND state = 'verifying' AND reservation_token = ?`)
		if _, err := tx.ExecContext(ctx, activate, obj.VerifiedUnix, obj.VerifiedUnix, obj.ObjectKey, token); err != nil {
			return fmt.Errorf("activate media asset: %w", err)
		}
	} else if state != "active" {
		return fmt.Errorf("unsupported media asset state %q", state)
	}

	insert := database.Bind(s.dialect, `INSERT INTO media_objects
		(id, object_key, source_upload_key, content_type, bytes, width, height, uploaded_by_user_id, verified_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if _, err := tx.ExecContext(ctx, insert, obj.ID, obj.ObjectKey, obj.SourceUploadKey, obj.ContentType, obj.Bytes, obj.Width, obj.Height, obj.UploadedByUserID, obj.VerifiedUnix); err != nil {
		if database.IsUniqueViolation(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("insert media_objects: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit media completion: %w", err)
	}
	return nil
}

func (s SQLRegistryStore) AbortVerification(ctx context.Context, objectKey, token string) error {
	query := database.Bind(s.dialect, `DELETE FROM media_assets
		WHERE object_key = ? AND state = 'verifying' AND reservation_token = ?`)
	_, err := s.db.ExecContext(ctx, query, objectKey, token)
	if err != nil {
		return fmt.Errorf("abort media reservation: %w", err)
	}
	return nil
}

func (s SQLRegistryStore) GetBySourceKey(ctx context.Context, sourceUploadKey string) (MediaObject, error) {
	query := database.Bind(s.dialect, `SELECT mo.id, mo.object_key, mo.source_upload_key, mo.content_type, mo.bytes, mo.width, mo.height, mo.uploaded_by_user_id, mo.verified_unix
		FROM media_objects mo JOIN media_assets ma ON ma.object_key = mo.object_key AND ma.state = 'active'
		WHERE mo.source_upload_key = ? LIMIT 1`)
	return scanMediaObjectRow(s.db.QueryRowContext(ctx, query, sourceUploadKey))
}

func (s SQLRegistryStore) GetByObjectKey(ctx context.Context, objectKey string) (MediaObject, error) {
	query := database.Bind(s.dialect, `SELECT mo.id, mo.object_key, mo.source_upload_key, mo.content_type, mo.bytes, mo.width, mo.height, mo.uploaded_by_user_id, mo.verified_unix
		FROM media_objects mo JOIN media_assets ma ON ma.object_key = mo.object_key AND ma.state = 'active'
		WHERE mo.object_key = ? LIMIT 1`)
	return scanMediaObjectRow(s.db.QueryRowContext(ctx, query, objectKey))
}

func scanMediaObjectRow(row *sql.Row) (MediaObject, error) {
	var obj MediaObject
	err := row.Scan(&obj.ID, &obj.ObjectKey, &obj.SourceUploadKey, &obj.ContentType, &obj.Bytes, &obj.Width, &obj.Height, &obj.UploadedByUserID, &obj.VerifiedUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MediaObject{}, ErrObjectNotFound
		}
		return MediaObject{}, err
	}
	return obj, nil
}

func (s SQLRegistryStore) ListEligible(ctx context.Context, nowUnix int64, limit int) ([]string, error) {
	return s.listEligible(ctx, s.db, nowUnix, limit, false)
}

type rowQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (s SQLRegistryStore) listEligible(ctx context.Context, q rowQueryer, nowUnix int64, limit int, lock bool) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("media gc limit must be positive")
	}
	query := database.Bind(s.dialect, `SELECT ma.object_key FROM media_assets ma
		WHERE ((ma.state = 'active' AND ma.unassociated_since_unix > 0 AND ma.unassociated_since_unix <= ?)
			OR (ma.state = 'verifying' AND ma.reserved_unix > 0 AND ma.reserved_unix <= ?))
		AND NOT EXISTS (SELECT 1 FROM product_images pi WHERE pi.object_key = ma.object_key)
		AND NOT EXISTS (SELECT 1 FROM media_gc_jobs job WHERE job.object_key = ma.object_key)
		ORDER BY ma.object_key LIMIT ?`)
	if lock && s.dialect == database.Postgres {
		query += ` FOR UPDATE SKIP LOCKED`
	}
	rows, err := q.QueryContext(ctx, query, nowUnix-int64(verifiedMediaRetention.Seconds()), nowUnix-int64(staleVerificationRetention.Seconds()), limit)
	if err != nil {
		return nil, fmt.Errorf("list media gc candidates: %w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (s SQLRegistryStore) ClaimEligible(ctx context.Context, nowUnix int64, limit int) ([]GCJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin media gc claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	keys, err := s.listEligible(ctx, tx, nowUnix, limit, true)
	if err != nil {
		return nil, err
	}
	jobs := make([]GCJob, 0, len(keys))
	for _, key := range keys {
		insert := database.Bind(s.dialect, `INSERT INTO media_gc_jobs (object_key, created_unix, attempts, last_attempt_unix)
			VALUES (?, ?, 0, 0) ON CONFLICT(object_key) DO NOTHING`)
		res, err := tx.ExecContext(ctx, insert, key, nowUnix)
		if err != nil {
			return nil, fmt.Errorf("create media gc job: %w", err)
		}
		inserted, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if inserted == 0 {
			continue
		}
		deleteAsset := database.Bind(s.dialect, `DELETE FROM media_assets
			WHERE object_key = ?
			AND ((state = 'active' AND unassociated_since_unix > 0 AND unassociated_since_unix <= ?)
				OR (state = 'verifying' AND reserved_unix > 0 AND reserved_unix <= ?))
			AND NOT EXISTS (SELECT 1 FROM product_images pi WHERE pi.object_key = media_assets.object_key)`)
		res, err = tx.ExecContext(ctx, deleteAsset, key, nowUnix-int64(verifiedMediaRetention.Seconds()), nowUnix-int64(staleVerificationRetention.Seconds()))
		if err != nil {
			return nil, fmt.Errorf("claim media asset: %w", err)
		}
		deleted, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if deleted != 1 {
			return nil, fmt.Errorf("media asset changed during gc claim")
		}
		deleteSources := database.Bind(s.dialect, `DELETE FROM media_objects WHERE object_key = ?`)
		if _, err := tx.ExecContext(ctx, deleteSources, key); err != nil {
			return nil, fmt.Errorf("delete media source rows: %w", err)
		}
		jobs = append(jobs, GCJob{ObjectKey: key, CreatedUnix: nowUnix})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit media gc claim: %w", err)
	}
	return jobs, nil
}

func (s SQLRegistryStore) ListGCJobs(ctx context.Context, limit int) ([]GCJob, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("media gc limit must be positive")
	}
	query := database.Bind(s.dialect, `SELECT object_key, created_unix, attempts, last_attempt_unix
		FROM media_gc_jobs
		ORDER BY CASE WHEN attempts = 0 THEN 0 ELSE 1 END, last_attempt_unix, created_unix, object_key
		LIMIT ?`)
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list media gc jobs: %w", err)
	}
	defer rows.Close()
	var jobs []GCJob
	for rows.Next() {
		var job GCJob
		if err := rows.Scan(&job.ObjectKey, &job.CreatedUnix, &job.Attempts, &job.LastAttemptUnix); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s SQLRegistryStore) MarkGCSucceeded(ctx context.Context, objectKey string) error {
	query := database.Bind(s.dialect, `DELETE FROM media_gc_jobs WHERE object_key = ?`)
	_, err := s.db.ExecContext(ctx, query, objectKey)
	if err != nil {
		return fmt.Errorf("complete media gc job: %w", err)
	}
	return nil
}

func (s SQLRegistryStore) MarkGCFailed(ctx context.Context, objectKey string, attemptedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE media_gc_jobs
		SET attempts = attempts + 1, last_attempt_unix = ? WHERE object_key = ?`)
	_, err := s.db.ExecContext(ctx, query, attemptedUnix, objectKey)
	if err != nil {
		return fmt.Errorf("record media gc failure: %w", err)
	}
	return nil
}
