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
	ListAll(ctx context.Context) ([]Article, error)
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

const articleColumns = `id, slug, title, excerpt, body_html, published, pinned, publish_at_unix, updated_unix`

func scanArticle(row interface{ Scan(dest ...any) error }) (Article, error) {
	var a Article
	var published, pinned int
	err := row.Scan(&a.ID, &a.Slug, &a.Title, &a.Excerpt, &a.BodyHTML,
		&published, &pinned, &a.PublishedAt, &a.UpdatedUnix)
	a.Published = published == 1
	a.Pinned = pinned == 1
	return a, err
}

func (s SQLStore) ListPublished(ctx context.Context) ([]Article, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+articleColumns+`
		FROM articles WHERE published = 1
		ORDER BY pinned DESC, COALESCE(NULLIF(publish_at_unix, 0), updated_unix) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s SQLStore) ListAll(ctx context.Context) ([]Article, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+articleColumns+`
		FROM articles ORDER BY updated_unix DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s SQLStore) GetBySlug(ctx context.Context, slug string) (Article, error) {
	query := database.Bind(s.dialect, `SELECT `+articleColumns+`
		FROM articles WHERE slug = ? LIMIT 1`)
	a, err := scanArticle(s.db.QueryRowContext(ctx, query, slug))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Article{}, ErrNotFound
		}
		return Article{}, err
	}
	return a, nil
}

func (s SQLStore) Upsert(ctx context.Context, a Article) error {
	query := database.Bind(s.dialect, `INSERT INTO articles
		(id, slug, title, excerpt, body_html, published, pinned, publish_at_unix, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			excerpt = excluded.excerpt,
			body_html = excluded.body_html,
			published = excluded.published,
			pinned = excluded.pinned,
			publish_at_unix = excluded.publish_at_unix,
			updated_unix = excluded.updated_unix`)
	published := 0
	if a.Published {
		published = 1
	}
	pinned := 0
	if a.Pinned {
		pinned = 1
	}
	if _, err := s.db.ExecContext(ctx, query, a.ID, a.Slug, a.Title, a.Excerpt, a.BodyHTML, published, pinned, a.PublishedAt, a.UpdatedUnix); err != nil {
		return fmt.Errorf("upsert article: %w", err)
	}
	return nil
}

func (s SQLStore) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles`).Scan(&n)
	return n, err
}
