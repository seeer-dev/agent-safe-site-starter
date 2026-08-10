package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type Dialect string

const (
	SQLite   Dialect = "sqlite"
	Postgres Dialect = "postgres"
)

func Open(ctx context.Context, driver, dsn string) (*sql.DB, Dialect, error) {
	var sqlDriver string
	var dialect Dialect

	switch strings.ToLower(driver) {
	case "sqlite":
		sqlDriver = "sqlite"
		dialect = SQLite
		if err := ensureSQLiteDir(dsn); err != nil {
			return nil, "", err
		}
	case "postgres":
		sqlDriver = "pgx"
		dialect = Postgres
	default:
		return nil, "", fmt.Errorf("unsupported database driver %q", driver)
	}

	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, "", err
	}
	if dialect == SQLite {
		db.SetMaxOpenConns(1)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("database ping: %w", err)
	}
	return db, dialect, nil
}

// Bind converts the starter's deliberately simple '?' placeholders to
// PostgreSQL $1, $2... placeholders. Query strings in this repository must not
// use '?' as a literal. If queries grow more complex, replace this helper with
// a real query builder rather than making this function clever.
func Bind(d Dialect, query string) string {
	if d != Postgres {
		return query
	}
	var b strings.Builder
	arg := 1
	for _, r := range query {
		if r == '?' {
			fmt.Fprintf(&b, "$%d", arg)
			arg++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func ensureSQLiteDir(dsn string) error {
	if !strings.HasPrefix(dsn, "file:") {
		return nil
	}
	path := strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	if path == "" || path == ":memory:" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
