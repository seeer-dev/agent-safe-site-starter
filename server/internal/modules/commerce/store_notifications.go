package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Notification templates and logs --------------------------------------

func (s SQLStore) ListNotificationTemplates(ctx context.Context) ([]NotificationTemplate, error) {
	query := `SELECT id, code, name, subject, body, is_enabled, updated_unix FROM notification_templates ORDER BY code`
	rows, err := s.db.QueryContext(ctx, database.Bind(s.dialect, query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NotificationTemplate
	for rows.Next() {
		var t NotificationTemplate
		var enabled int
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.Subject, &t.Body, &enabled, &t.UpdatedUnix); err != nil {
			return nil, err
		}
		t.IsEnabled = enabled == 1
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s SQLStore) GetNotificationTemplateByCode(ctx context.Context, code string) (NotificationTemplate, error) {
	query := database.Bind(s.dialect, `SELECT id, code, name, subject, body, is_enabled, updated_unix
		FROM notification_templates WHERE code = ? LIMIT 1`)
	var t NotificationTemplate
	var enabled int
	err := s.db.QueryRowContext(ctx, query, code).Scan(
		&t.ID, &t.Code, &t.Name, &t.Subject, &t.Body, &enabled, &t.UpdatedUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NotificationTemplate{}, ErrNotFound
		}
		return NotificationTemplate{}, err
	}
	t.IsEnabled = enabled == 1
	return t, nil
}

func (s SQLStore) UpsertNotificationTemplate(ctx context.Context, t NotificationTemplate) error {
	query := database.Bind(s.dialect, `INSERT INTO notification_templates
		(id, code, name, subject, body, is_enabled, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			code = excluded.code,
			name = excluded.name,
			subject = excluded.subject,
			body = excluded.body,
			is_enabled = excluded.is_enabled,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		t.ID, t.Code, t.Name, t.Subject, t.Body, t.IsEnabled, t.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert notification template: %w", err)
	}
	return nil
}

func (s SQLStore) DeleteNotificationTemplate(ctx context.Context, id string) error {
	query := database.Bind(s.dialect, `DELETE FROM notification_templates WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete notification template: %w", err)
	}
	return requireAffected(res)
}

func (s SQLStore) InsertNotificationLog(ctx context.Context, l NotificationLog) error {
	query := database.Bind(s.dialect, `INSERT INTO notification_logs
		(id, code, order_id, recipient, subject, body, status, provider, error, created_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.db.ExecContext(ctx, query,
		l.ID, l.Code, l.OrderID, l.Recipient, l.Subject, l.Body, l.Status, l.Provider, l.Error, l.CreatedUnix)
	if err != nil {
		return fmt.Errorf("insert notification log: %w", err)
	}
	return nil
}

func (s SQLStore) ListNotificationLogs(ctx context.Context, filter NotificationLogFilter) ([]NotificationLog, error) {
	query := `SELECT id, code, order_id, recipient, subject, body, status, provider, error, created_unix FROM notification_logs`
	var args []any
	clauses := []string{}
	if filter.OrderID != "" {
		clauses = append(clauses, "order_id = ?")
		args = append(args, filter.OrderID)
	}
	if filter.Code != "" {
		clauses = append(clauses, "code = ?")
		args = append(args, filter.Code)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_unix DESC"
	rows, err := s.db.QueryContext(ctx, database.Bind(s.dialect, query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NotificationLog
	for rows.Next() {
		var l NotificationLog
		if err := rows.Scan(&l.ID, &l.Code, &l.OrderID, &l.Recipient, &l.Subject, &l.Body, &l.Status, &l.Provider, &l.Error, &l.CreatedUnix); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
