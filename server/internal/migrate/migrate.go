package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

func Apply(ctx context.Context, db *sql.DB, dialect database.Dialect, root string) error {
	dir := filepath.Join(root, "db", "migrations", string(dialect))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations %s: %w", dir, err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_unix BIGINT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, name := range names {
		var exists int
		query := database.Bind(dialect, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`)
		if err := db.QueryRowContext(ctx, query, name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}

		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if err := execStatements(ctx, tx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		insert := database.Bind(dialect, `INSERT INTO schema_migrations (version, applied_unix) VALUES (?, CAST(strftime('%s','now') AS INTEGER))`)
		if dialect == database.Postgres {
			insert = database.Bind(dialect, `INSERT INTO schema_migrations (version, applied_unix) VALUES (?, EXTRACT(EPOCH FROM NOW())::BIGINT)`)
		}
		if _, err := tx.ExecContext(ctx, insert, name); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func execStatements(ctx context.Context, tx *sql.Tx, script string) error {
	// Migrations in this starter deliberately stay simple and portable.
	// Keep semicolons out of quoted literals; when migrations outgrow this
	// profile, adopt a dedicated migration library instead of a custom DSL.
	for _, statement := range strings.Split(script, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
