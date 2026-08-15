package staff

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

var ErrNotFound = errors.New("staff member not found")

// ErrGuardedBlocked is returned by guarded store methods when the
// last-owner invariant blocks the operation. The caller checks GetByID
// to distinguish "not found" from "blocked by guard".
var ErrGuardedBlocked = errors.New("staff: operation blocked by last-owner guard")

type Store interface {
	List(ctx context.Context) ([]StaffMember, error)
	ListActive(ctx context.Context) ([]StaffMember, error)
	GetByID(ctx context.Context, id string) (StaffMember, error)
	GetBySupabaseUserID(ctx context.Context, supabaseUserID string) (StaffMember, error)
	GetByEmail(ctx context.Context, email string) (StaffMember, error)
	Upsert(ctx context.Context, sm StaffMember) error
	// UpsertGuarded performs an upsert with an atomic last-owner guard
	// using a transaction that locks all active owner rows (FOR UPDATE
	// on PostgreSQL, no-op on SQLite which serializes via
	// SetMaxOpenConns(1)). This prevents the cross-row race where two
	// concurrent transactions demote/disable different active owners and
	// both see count=2, both proceed, leaving 0 active owners.
	UpsertGuarded(ctx context.Context, sm StaffMember) error
	UpdateStatus(ctx context.Context, id, status string, updatedUnix int64) error
	// UpdateStatusGuarded performs a status update with an atomic
	// last-owner guard using the same transaction-based locking approach.
	UpdateStatusGuarded(ctx context.Context, id, status string, updatedUnix int64) error
	Delete(ctx context.Context, id string) error
	// DeleteGuarded performs a delete with an atomic last-owner guard
	// using the same transaction-based locking approach.
	DeleteGuarded(ctx context.Context, id string) error
	CountActiveByRole(ctx context.Context, roleLabel string) (int, error)
}

type SQLStore struct {
	db                  *sql.DB
	dialect             database.Dialect
	testAfterLockHook   func(ctx context.Context, tx *sql.Tx)
	testLockAttemptHook func(ctx context.Context, tx *sql.Tx)
}

func NewSQLStore(db *sql.DB, dialect database.Dialect) SQLStore {
	return SQLStore{db: db, dialect: dialect}
}

func (s SQLStore) List(ctx context.Context) ([]StaffMember, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, display_name, email, role_label, supabase_user_id, status, updated_unix
		FROM staff_members ORDER BY updated_unix DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StaffMember
	for rows.Next() {
		var sm StaffMember
		if err := rows.Scan(&sm.ID, &sm.DisplayName, &sm.Email, &sm.RoleLabel, &sm.SupabaseUserID, &sm.Status, &sm.UpdatedUnix); err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

func (s SQLStore) ListActive(ctx context.Context) ([]StaffMember, error) {
	query := database.Bind(s.dialect, `SELECT id, display_name, email, role_label, supabase_user_id, status, updated_unix
		FROM staff_members WHERE status = 'active' ORDER BY updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StaffMember
	for rows.Next() {
		var sm StaffMember
		if err := rows.Scan(&sm.ID, &sm.DisplayName, &sm.Email, &sm.RoleLabel, &sm.SupabaseUserID, &sm.Status, &sm.UpdatedUnix); err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

func (s SQLStore) GetByID(ctx context.Context, id string) (StaffMember, error) {
	query := database.Bind(s.dialect, `SELECT id, display_name, email, role_label, supabase_user_id, status, updated_unix
		FROM staff_members WHERE id = ? LIMIT 1`)
	var sm StaffMember
	if err := s.db.QueryRowContext(ctx, query, id).Scan(&sm.ID, &sm.DisplayName, &sm.Email, &sm.RoleLabel, &sm.SupabaseUserID, &sm.Status, &sm.UpdatedUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StaffMember{}, ErrNotFound
		}
		return StaffMember{}, err
	}
	return sm, nil
}

func (s SQLStore) GetBySupabaseUserID(ctx context.Context, supabaseUserID string) (StaffMember, error) {
	if supabaseUserID == "" {
		return StaffMember{}, ErrNotFound
	}
	query := database.Bind(s.dialect, `SELECT id, display_name, email, role_label, supabase_user_id, status, updated_unix
		FROM staff_members WHERE supabase_user_id = ? LIMIT 1`)
	var sm StaffMember
	if err := s.db.QueryRowContext(ctx, query, supabaseUserID).Scan(&sm.ID, &sm.DisplayName, &sm.Email, &sm.RoleLabel, &sm.SupabaseUserID, &sm.Status, &sm.UpdatedUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StaffMember{}, ErrNotFound
		}
		return StaffMember{}, err
	}
	return sm, nil
}

func (s SQLStore) GetByEmail(ctx context.Context, email string) (StaffMember, error) {
	if email == "" {
		return StaffMember{}, ErrNotFound
	}
	query := database.Bind(s.dialect, `SELECT id, display_name, email, role_label, supabase_user_id, status, updated_unix
		FROM staff_members WHERE email = ? LIMIT 1`)
	var sm StaffMember
	if err := s.db.QueryRowContext(ctx, query, email).Scan(&sm.ID, &sm.DisplayName, &sm.Email, &sm.RoleLabel, &sm.SupabaseUserID, &sm.Status, &sm.UpdatedUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StaffMember{}, ErrNotFound
		}
		return StaffMember{}, err
	}
	return sm, nil
}

func (s SQLStore) Upsert(ctx context.Context, sm StaffMember) error {
	query := database.Bind(s.dialect, `INSERT INTO staff_members
		(id, display_name, email, role_label, supabase_user_id, status, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			display_name = excluded.display_name,
			email = excluded.email,
			role_label = excluded.role_label,
			supabase_user_id = excluded.supabase_user_id,
			status = excluded.status,
			updated_unix = excluded.updated_unix`)
	if _, err := s.db.ExecContext(ctx, query, sm.ID, sm.DisplayName, sm.Email, sm.RoleLabel, sm.SupabaseUserID, sm.Status, sm.UpdatedUnix); err != nil {
		return fmt.Errorf("upsert staff member: %w", err)
	}
	return nil
}

func (s SQLStore) UpdateStatus(ctx context.Context, id, status string, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE staff_members SET status = ?, updated_unix = ? WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, status, updatedUnix, id)
	if err != nil {
		return fmt.Errorf("update staff status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// lockActiveOwners locks all active owner rows within the transaction.
// On PostgreSQL this uses SELECT ... FOR UPDATE which blocks concurrent
// transactions from modifying or locking these rows until the current
// transaction commits or rolls back. This serializes concurrent
// last-owner checks across different owner rows, preventing the race
// where two concurrent demotes both see count=2 and both proceed.
// On SQLite, FOR UPDATE is not supported but SetMaxOpenConns(1)
// serializes all access, so the lock is a no-op.
func (s SQLStore) lockActiveOwners(ctx context.Context, tx *sql.Tx) error {
	if s.testLockAttemptHook != nil {
		s.testLockAttemptHook(ctx, tx)
	}
	if s.dialect == database.Postgres {
		_, err := tx.ExecContext(ctx, `SELECT id FROM staff_members WHERE status = 'active' AND role_label = 'owner' FOR UPDATE`)
		return err
	}
	// SQLite: no-op. SetMaxOpenConns(1) serializes all transactions.
	return nil
}

// countActiveOwnersInTx counts active owners within the transaction,
// after the lock has been acquired. This count reflects all committed
// changes from concurrent transactions that committed before this
// transaction acquired the lock.
func (s SQLStore) countActiveOwnersInTx(ctx context.Context, tx *sql.Tx) (int, error) {
	query := database.Bind(s.dialect, `SELECT COUNT(*) FROM staff_members WHERE status = 'active' AND role_label = 'owner'`)
	var n int
	err := tx.QueryRowContext(ctx, query).Scan(&n)
	return n, err
}

// getByIDInTx reads a staff member within the transaction.
func (s SQLStore) getByIDInTx(ctx context.Context, tx *sql.Tx, id string) (StaffMember, error) {
	query := database.Bind(s.dialect, `SELECT id, display_name, email, role_label, supabase_user_id, status, updated_unix
		FROM staff_members WHERE id = ? LIMIT 1`)
	var sm StaffMember
	if err := tx.QueryRowContext(ctx, query, id).Scan(&sm.ID, &sm.DisplayName, &sm.Email, &sm.RoleLabel, &sm.SupabaseUserID, &sm.Status, &sm.UpdatedUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StaffMember{}, ErrNotFound
		}
		return StaffMember{}, err
	}
	return sm, nil
}

// isDemotingActiveOwner returns true if the update changes an active
// owner to a non-owner or disabled state.
func isDemotingActiveOwner(existing StaffMember, newRoleLabel, newStatus string) bool {
	return existing.RoleLabel == "owner" && existing.Status == "active" &&
		(newRoleLabel != "owner" || newStatus == "disabled")
}

// UpsertGuarded atomically upserts with a last-owner guard using a
// transaction that locks all active owner rows. If the existing row is
// an active owner and the new values would demote or disable it, and it
// is the last active owner, the update is blocked.
func (s SQLStore) UpsertGuarded(ctx context.Context, sm StaffMember) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert guarded tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.lockActiveOwners(ctx, tx); err != nil {
		return fmt.Errorf("lock active owners: %w", err)
	}
	if s.testAfterLockHook != nil {
		s.testAfterLockHook(ctx, tx)
	}

	existing, err := s.getByIDInTx(ctx, tx, sm.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	// If the row exists and is an active owner being demoted/disabled,
	// check the last-owner invariant.
	if !errors.Is(err, ErrNotFound) && isDemotingActiveOwner(existing, sm.RoleLabel, sm.Status) {
		count, err := s.countActiveOwnersInTx(ctx, tx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrGuardedBlocked
		}
	}

	query := database.Bind(s.dialect, `INSERT INTO staff_members
		(id, display_name, email, role_label, supabase_user_id, status, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			display_name = excluded.display_name,
			email = excluded.email,
			role_label = excluded.role_label,
			supabase_user_id = excluded.supabase_user_id,
			status = excluded.status,
			updated_unix = excluded.updated_unix`)
	if _, err := tx.ExecContext(ctx, query, sm.ID, sm.DisplayName, sm.Email, sm.RoleLabel, sm.SupabaseUserID, sm.Status, sm.UpdatedUnix); err != nil {
		return fmt.Errorf("upsert staff member guarded: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert guarded: %w", err)
	}
	return nil
}

// UpdateStatusGuarded atomically updates the status with a last-owner
// guard using a transaction that locks all active owner rows.
func (s SQLStore) UpdateStatusGuarded(ctx context.Context, id, status string, updatedUnix int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update status guarded tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.lockActiveOwners(ctx, tx); err != nil {
		return fmt.Errorf("lock active owners: %w", err)
	}

	existing, err := s.getByIDInTx(ctx, tx, id)
	if err != nil {
		return err
	}
	// If the row is an active owner being disabled, check the invariant.
	if existing.RoleLabel == "owner" && existing.Status == "active" && status == "disabled" {
		count, err := s.countActiveOwnersInTx(ctx, tx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrGuardedBlocked
		}
	}

	query := database.Bind(s.dialect, `UPDATE staff_members SET status = ?, updated_unix = ? WHERE id = ?`)
	res, err := tx.ExecContext(ctx, query, status, updatedUnix, id)
	if err != nil {
		return fmt.Errorf("update staff status guarded: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update status guarded: %w", err)
	}
	return nil
}

// DeleteGuarded atomically deletes with a last-owner guard using a
// transaction that locks all active owner rows.
func (s SQLStore) DeleteGuarded(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete guarded tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.lockActiveOwners(ctx, tx); err != nil {
		return fmt.Errorf("lock active owners: %w", err)
	}

	existing, err := s.getByIDInTx(ctx, tx, id)
	if err != nil {
		return err
	}
	// If the row is an active owner, check the last-owner invariant.
	if existing.RoleLabel == "owner" && existing.Status == "active" {
		count, err := s.countActiveOwnersInTx(ctx, tx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrGuardedBlocked
		}
	}

	query := database.Bind(s.dialect, `DELETE FROM staff_members WHERE id = ?`)
	if _, err := tx.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("delete staff member guarded: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete guarded: %w", err)
	}
	return nil
}

func (s SQLStore) CountActiveByRole(ctx context.Context, roleLabel string) (int, error) {
	query := database.Bind(s.dialect, `SELECT COUNT(*) FROM staff_members WHERE status = 'active' AND role_label = ?`)
	var n int
	err := s.db.QueryRowContext(ctx, query, roleLabel).Scan(&n)
	return n, err
}

func (s SQLStore) Delete(ctx context.Context, id string) error {
	query := database.Bind(s.dialect, `DELETE FROM staff_members WHERE id = ?`)
	if _, err := s.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("delete staff member: %w", err)
	}
	return nil
}
