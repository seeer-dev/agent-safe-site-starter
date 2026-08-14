package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func getTestPostgresDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		// In CI these tests are the whole point of the change: a silent skip
		// would leave the PostgreSQL branches unexecuted while the workflow
		// still reports success. Fail loudly instead.
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI; live PostgreSQL coverage must not be skipped")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping live PostgreSQL integration test")
	}
	return dsn
}

func TestPostgresLiveOpenAndBind(t *testing.T) {
	dsn := getTestPostgresDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Branch 1: database.Open with driver "postgres"
	db, dialect, err := Open(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("Open(postgres) failed: %v", err)
	}
	defer db.Close()

	if dialect != Postgres {
		t.Fatalf("expected dialect %q, got %q", Postgres, dialect)
	}

	// Branch 2: database.Bind replacing '?' with '$1, $2...' on PostgreSQL
	query := Bind(dialect, `SELECT ?::int + ?::int AS total`)
	if query != `SELECT $1::int + $2::int AS total` {
		t.Fatalf("Bind output unexpected: %s", query)
	}

	var total int
	if err := db.QueryRowContext(ctx, query, 15, 27).Scan(&total); err != nil {
		t.Fatalf("live PostgreSQL query with Bind placeholders failed: %v", err)
	}
	if total != 42 {
		t.Fatalf("expected query result 42, got %d", total)
	}
}

func TestPostgresLiveConstraintViolations(t *testing.T) {
	dsn := getTestPostgresDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, _, err := Open(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("Open(postgres) failed: %v", err)
	}
	defer db.Close()

	// Create temporary table with unique and foreign key constraints
	_, err = db.ExecContext(ctx, `
		CREATE TEMP TABLE test_parent (
			id TEXT PRIMARY KEY
		);
		CREATE TEMP TABLE test_child (
			id TEXT PRIMARY KEY,
			parent_id TEXT REFERENCES test_parent(id),
			val TEXT UNIQUE
		);
		INSERT INTO test_parent (id) VALUES ('p1');
		INSERT INTO test_child (id, parent_id, val) VALUES ('c1', 'p1', 'unique_val');
	`)
	if err != nil {
		t.Fatalf("setup temp tables: %v", err)
	}

	// Test unique violation
	_, err = db.ExecContext(ctx, `INSERT INTO test_child (id, parent_id, val) VALUES ('c2', 'p1', 'unique_val')`)
	if err == nil {
		t.Fatal("expected unique violation error, got nil")
	}
	if !IsUniqueViolation(err) {
		t.Fatalf("expected IsUniqueViolation(err) == true, got false for: %v", err)
	}

	// Test foreign key violation
	_, err = db.ExecContext(ctx, `INSERT INTO test_child (id, parent_id, val) VALUES ('c3', 'nonexistent_parent', 'val3')`)
	if err == nil {
		t.Fatal("expected foreign key violation error, got nil")
	}
	if !IsForeignKeyViolation(err) {
		t.Fatalf("expected IsForeignKeyViolation(err) == true, got false for: %v", err)
	}
}
