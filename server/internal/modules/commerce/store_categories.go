package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Categories -----------------------------------------------------------

func scanCategories(rows *sql.Rows) ([]Category, error) {
	var out []Category
	for rows.Next() {
		var c Category
		var active int
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.Description, &c.Image, &c.SortOrder, &active, &c.UpdatedUnix); err != nil {
			return nil, err
		}
		c.IsActive = active == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListCategories returns categories ordered by sort_order then name.
// activeOnly restricts to is_active rows for public storefront use.
func (s SQLStore) ListCategories(ctx context.Context, activeOnly bool) ([]Category, error) {
	query := `SELECT id, slug, name, description, image, sort_order, is_active, updated_unix FROM categories`
	if activeOnly {
		query += ` WHERE is_active = ?`
	}
	query += ` ORDER BY sort_order, name`
	query = database.Bind(s.dialect, query)
	var rows *sql.Rows
	var err error
	if activeOnly {
		rows, err = s.db.QueryContext(ctx, query, true)
	} else {
		rows, err = s.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCategories(rows)
}

func (s SQLStore) GetCategory(ctx context.Context, id string) (Category, error) {
	query := database.Bind(s.dialect, `SELECT id, slug, name, description, image, sort_order, is_active, updated_unix
		FROM categories WHERE id = ? LIMIT 1`)
	var c Category
	var active int
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Slug, &c.Name, &c.Description, &c.Image, &c.SortOrder, &active, &c.UpdatedUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, ErrNotFound
		}
		return Category{}, err
	}
	c.IsActive = active == 1
	return c, nil
}

func (s SQLStore) UpsertCategory(ctx context.Context, c Category) error {
	query := database.Bind(s.dialect, `INSERT INTO categories
		(id, slug, name, description, image, sort_order, is_active, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			slug = excluded.slug,
			name = excluded.name,
			description = excluded.description,
			image = excluded.image,
			sort_order = excluded.sort_order,
			is_active = excluded.is_active,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		c.ID, c.Slug, c.Name, c.Description, c.Image, c.SortOrder, c.IsActive, c.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert category: %w", err)
	}
	return nil
}

func (s SQLStore) DeleteCategory(ctx context.Context, id string) error {
	query := database.Bind(s.dialect, `DELETE FROM categories WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return requireAffected(res)
}
