package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type Dialect string

const (
	SQLite   Dialect = "sqlite"
	Postgres Dialect = "postgres"
)

// Open uses the default pool bounds. Callers that carry configuration should
// use OpenWithPool.

// PoolConfig bounds a PostgreSQL connection pool. database/sql defaults to an
// unlimited number of open connections and two idle ones, which lets a traffic
// spike exhaust a provider connection allowance and makes steady-state traffic
// churn connections. These are a conservative floor for one small instance,
// not a sizing recommendation: the right numbers depend on whether a session
// pooler or a transaction pooler sits in front of PostgreSQL, and this package
// does not make that decision.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Default pool bounds. Each is applied when the corresponding field is not a
// positive value, so an unset, zero, negative, or unparseable setting can never
// produce an unbounded pool.
const (
	DefaultMaxOpenConns    = 10
	DefaultMaxIdleConns    = 10
	DefaultConnMaxLifetime = 30 * time.Minute
	DefaultConnMaxIdleTime = 5 * time.Minute
)

// withDefaults replaces any non-positive field with its default.
func (p PoolConfig) withDefaults() PoolConfig {
	if p.MaxOpenConns <= 0 {
		p.MaxOpenConns = DefaultMaxOpenConns
	}
	if p.MaxIdleConns <= 0 {
		p.MaxIdleConns = DefaultMaxIdleConns
	}
	if p.ConnMaxLifetime <= 0 {
		p.ConnMaxLifetime = DefaultConnMaxLifetime
	}
	if p.ConnMaxIdleTime <= 0 {
		p.ConnMaxIdleTime = DefaultConnMaxIdleTime
	}
	// Idle above open is meaningless to database/sql, which silently reduces
	// it; clamping here keeps the effective configuration inspectable.
	if p.MaxIdleConns > p.MaxOpenConns {
		p.MaxIdleConns = p.MaxOpenConns
	}
	return p
}

// applyPool sets the bounds for PostgreSQL. SQLite is deliberately excluded:
// its single connection is load-bearing, because several lock guards are
// documented as SQLite no-ops that rely on that serialization.
func applyPool(db *sql.DB, dialect Dialect, pool PoolConfig) {
	if dialect == SQLite {
		db.SetMaxOpenConns(1)
		return
	}
	p := pool.withDefaults()
	db.SetMaxOpenConns(p.MaxOpenConns)
	db.SetMaxIdleConns(p.MaxIdleConns)
	db.SetConnMaxLifetime(p.ConnMaxLifetime)
	db.SetConnMaxIdleTime(p.ConnMaxIdleTime)
}

func Open(ctx context.Context, driver, dsn string) (*sql.DB, Dialect, error) {
	return OpenWithPool(ctx, driver, dsn, PoolConfig{})
}

// OpenWithPool opens a database and applies the given pool bounds.
func OpenWithPool(ctx context.Context, driver, dsn string, pool PoolConfig) (*sql.DB, Dialect, error) {
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
	applyPool(db, dialect, pool)
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
