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

// IsUniqueViolation returns true if err is a unique constraint violation
// from either SQLite or PostgreSQL. This is used to detect concurrent
// duplicate-key inserts and convert them to application-level conflicts.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// SQLite: "constraint failed: UNIQUE constraint failed: ..."
	// PostgreSQL (pgx): "unique violation" or SQLSTATE 23505
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "constraint failed: unique") ||
		strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "23505")
}

// IsForeignKeyViolation recognizes foreign-key constraint errors from both
// supported drivers. It is used where a concurrent parent-row deletion turns
// an otherwise verified media association into a safe validation failure.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "foreign key constraint failed") ||
		strings.Contains(msg, "violates foreign key constraint") ||
		strings.Contains(msg, "sqlstate 23503") ||
		strings.Contains(msg, "23503")
}
