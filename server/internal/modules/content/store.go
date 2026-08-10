package content

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

var ErrNotFound = errors.New("article not found")

type Store interface {
	ListPublished(ctx context.Context) ([]Article, error)
	GetBySlug(ctx context.Context, slug string) (Article, error)
	Upsert(ctx context.Context, article Article) error
	Count(ctx context.Context) (int, error)
}

type SQLStore struct {
	db      *sql.DB
	dialect database.Dialect
}

func NewSQLStore(db *sql.DB, dialect database.Dialect) SQLStore {
	return SQLStore{db: db, dialect: dialect}
}

func (s SQLStore) ListPublished(ctx context.Context) ([]Article, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, slug, title, excerpt, body_html, published, updated_unix
		FROM articles WHERE published = 1 ORDER BY updated_unix DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var a Article
		var published int
		if err := rows.Scan(&a.ID, &a.Slug, &a.Title, &a.Excerpt, &a.BodyHTML, &published, &a.UpdatedUnix); err != nil {
			return nil, err
		}
		a.Published = published == 1
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s SQLStore) GetBySlug(ctx context.Context, slug string) (Article, error) {
	query := database.Bind(s.dialect, `SELECT id, slug, title, excerpt, body_html, published, updated_unix
		FROM articles WHERE slug = ? LIMIT 1`)
	var a Article
	var published int
	if err := s.db.QueryRowContext(ctx, query, slug).Scan(&a.ID, &a.Slug, &a.Title, &a.Excerpt, &a.BodyHTML, &published, &a.UpdatedUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Article{}, ErrNotFound
		}
		return Article{}, err
	}
	a.Published = published == 1
	return a, nil
}

func (s SQLStore) Upsert(ctx context.Context, a Article) error {
	query := database.Bind(s.dialect, `INSERT INTO articles
		(id, slug, title, excerpt, body_html, published, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			excerpt = excluded.excerpt,
			body_html = excluded.body_html,
			published = excluded.published,
			updated_unix = excluded.updated_unix`)
	published := 0
	if a.Published {
		published = 1
	}
	if _, err := s.db.ExecContext(ctx, query, a.ID, a.Slug, a.Title, a.Excerpt, a.BodyHTML, published, a.UpdatedUnix); err != nil {
		return fmt.Errorf("upsert article: %w", err)
	}
	return nil
}

func (s SQLStore) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles`).Scan(&n)
	return n, err
}
