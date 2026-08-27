package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Members --------------------------------------------------------------

func (s SQLStore) ListMembers(ctx context.Context, filter MemberFilter) ([]Member, error) {
	var (
		clauses []string
		args    []any
	)
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Tier != "" {
		clauses = append(clauses, "tier = ?")
		args = append(args, filter.Tier)
	}
	query := `SELECT id, email, name, status, tier, tags, notes, total_orders, total_spent, updated_unix FROM members`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY updated_unix DESC"
	query = database.Bind(s.dialect, query)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.Name, &m.Status, &m.Tier, &m.Tags, &m.Notes, &m.TotalOrders, &m.TotalSpent, &m.UpdatedUnix); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s SQLStore) GetMember(ctx context.Context, id string) (Member, error) {
	query := database.Bind(s.dialect, `SELECT id, email, name, status, tier, tags, notes, total_orders, total_spent, updated_unix
		FROM members WHERE id = ? LIMIT 1`)
	var m Member
	err := s.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.Email, &m.Name, &m.Status, &m.Tier, &m.Tags, &m.Notes, &m.TotalOrders, &m.TotalSpent, &m.UpdatedUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Member{}, ErrNotFound
		}
		return Member{}, err
	}
	return m, nil
}

func (s SQLStore) UpsertMember(ctx context.Context, m Member) error {
	query := database.Bind(s.dialect, `INSERT INTO members
		(id, email, name, status, tier, tags, notes, total_orders, total_spent, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email = excluded.email,
			name = excluded.name,
			status = excluded.status,
			tier = excluded.tier,
			tags = excluded.tags,
			notes = excluded.notes,
			total_orders = excluded.total_orders,
			total_spent = excluded.total_spent,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		m.ID, m.Email, m.Name, m.Status, m.Tier, m.Tags, m.Notes, m.TotalOrders, m.TotalSpent, m.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert member: %w", err)
	}
	return nil
}

func (s SQLStore) UpdateMemberStatus(ctx context.Context, id, status string, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE members SET status = ?, updated_unix = ? WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, status, updatedUnix, id)
	if err != nil {
		return fmt.Errorf("update member status: %w", err)
	}
	return requireAffected(res)
}
