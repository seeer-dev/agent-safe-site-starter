package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/platform/database"
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

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root containing go.mod")
		}
		dir = parent
	}
}

func TestPostgresLiveMigrateApplyAndTimestamp(t *testing.T) {
	dsn := getTestPostgresDSN(t)
	root := findRepoRoot(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, dialect, err := database.Open(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("Open(postgres) failed: %v", err)
	}
	defer db.Close()

	if dialect != database.Postgres {
		t.Fatalf("expected dialect %q, got %q", database.Postgres, dialect)
	}

	// Apply migrations via migrate.Apply (executing Branch 3: PostgreSQL timestamp insertion)
	if err := Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("Apply(postgres) failed: %v", err)
	}

	// Verify schema_migrations table exists and holds applied records with valid timestamps
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("query schema_migrations count: %v", err)
	}
	if count < 16 {
		t.Fatalf("expected at least 16 migrations applied in PostgreSQL, got %d", count)
	}

	var sampleVersion string
	var appliedUnix int64
	err = db.QueryRowContext(ctx, `SELECT version, applied_unix FROM schema_migrations ORDER BY version LIMIT 1`).Scan(&sampleVersion, &appliedUnix)
	if err != nil {
		t.Fatalf("query sample migration: %v", err)
	}
	if sampleVersion != "001_init.sql" {
		t.Fatalf("expected first version 001_init.sql, got %q", sampleVersion)
	}
	if appliedUnix <= 0 {
		t.Fatalf("expected positive applied_unix timestamp from EXTRACT(EPOCH FROM NOW()), got %d", appliedUnix)
	}
}
