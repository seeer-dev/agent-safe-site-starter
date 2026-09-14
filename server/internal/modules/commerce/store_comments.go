package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Product comments -----------------------------------------------------

const commentColumns = "id, product_id, nickname, content, rating, status, admin_reply, replied_unix, created_unix, updated_unix"

func scanComment(row *sql.Row) (ProductComment, error) {
	var c ProductComment
	var rating sql.NullInt64
	err := row.Scan(&c.ID, &c.ProductID, &c.Nickname, &c.Content, &rating, &c.Status,
		&c.AdminReply, &c.RepliedUnix, &c.CreatedUnix, &c.UpdatedUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProductComment{}, ErrNotFound
		}
		return ProductComment{}, err
	}
	if rating.Valid {
		r := int(rating.Int64)
		c.Rating = &r
	}
	return c, nil
}

func scanComments(rows *sql.Rows) ([]ProductComment, error) {
	var out []ProductComment
	for rows.Next() {
		var c ProductComment
		var rating sql.NullInt64
		if err := rows.Scan(&c.ID, &c.ProductID, &c.Nickname, &c.Content, &rating, &c.Status,
			&c.AdminReply, &c.RepliedUnix, &c.CreatedUnix, &c.UpdatedUnix); err != nil {
			return nil, err
		}
		if rating.Valid {
			r := int(rating.Int64)
			c.Rating = &r
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListProductComments returns comments for one product. status empty
// returns all; "approved" is the public storefront view.
func (s SQLStore) ListProductComments(ctx context.Context, productID, status string) ([]ProductComment, error) {
	query := `SELECT ` + commentColumns + ` FROM product_comments WHERE product_id = ?`
	args := []any{productID}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_unix DESC`
	rows, err := s.db.QueryContext(ctx, database.Bind(s.dialect, query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComments(rows)
}

// ListComments returns comments across products for admin moderation.
// status empty returns all.
func (s SQLStore) ListComments(ctx context.Context, status string) ([]ProductComment, error) {
	query := `SELECT ` + commentColumns + ` FROM product_comments`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_unix DESC`
	rows, err := s.db.QueryContext(ctx, database.Bind(s.dialect, query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComments(rows)
}

func (s SQLStore) GetComment(ctx context.Context, id string) (ProductComment, error) {
	query := database.Bind(s.dialect, `SELECT `+commentColumns+` FROM product_comments WHERE id = ? LIMIT 1`)
	return scanComment(s.db.QueryRowContext(ctx, query, id))
}

func (s SQLStore) InsertComment(ctx context.Context, c ProductComment) error {
	query := database.Bind(s.dialect, `INSERT INTO product_comments
		(` + commentColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	var rating any
	if c.Rating != nil {
		rating = *c.Rating
	}
	_, err := s.db.ExecContext(ctx, query,
		c.ID, c.ProductID, c.Nickname, c.Content, rating, c.Status,
		c.AdminReply, c.RepliedUnix, c.CreatedUnix, c.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

// UpdateCommentModeration sets status and the optional admin reply. The
// reply timestamp is set only when adminReply is non-empty.
func (s SQLStore) UpdateCommentModeration(ctx context.Context, id, status, adminReply string, repliedUnix, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE product_comments
		SET status = ?, admin_reply = ?, replied_unix = ?, updated_unix = ? WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, status, adminReply, repliedUnix, updatedUnix, id)
	if err != nil {
		return fmt.Errorf("update comment moderation: %w", err)
	}
	return requireAffected(res)
}

func (s SQLStore) CountPendingComments(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM product_comments WHERE status = 'pending'`).Scan(&n)
	return n, err
}
