package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	// Collect the exact list of migration filenames from disk
	migDir := filepath.Join(root, "db", "migrations", "postgres")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("read postgres migrations dir: %v", err)
	}
	var expectedVersions []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			expectedVersions = append(expectedVersions, e.Name())
		}
	}
	sort.Strings(expectedVersions)
	if len(expectedVersions) < 16 {
		t.Fatalf("expected at least 16 postgres migration files on disk, found %d", len(expectedVersions))
	}

	testStartUnix := time.Now().Unix()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Open admin database connection to create and cleanup isolated schema namespace
	adminDB, dialect, err := database.Open(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("Open adminDB(postgres) failed: %v", err)
	}
	defer adminDB.Close()

	if dialect != database.Postgres {
		t.Fatalf("expected dialect %q, got %q", database.Postgres, dialect)
	}

	schemaName := fmt.Sprintf("fresh_mig_%d", time.Now().UnixNano())
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, schemaName)); err != nil {
		t.Fatalf("create fresh schema %s: %v", schemaName, err)
	}
	defer func() {
		ctxClean, cancelClean := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelClean()
		_, _ = adminDB.ExecContext(ctxClean, fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schemaName))
	}()

	// 2. Open dedicated database handle targeting the fresh schema.
	// To guarantee that all operations in Apply and assertions use this exact schema:
	// (a) Configure search_path in the connection DSN.
	// (b) Enforce SetMaxOpenConns(1) on the test *sql.DB so connection-scoped state is never split.
	// (c) Explicitly SET search_path on the handle and assert SHOW search_path.
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	schemaDSN := fmt.Sprintf("%s%ssearch_path=%s", dsn, sep, schemaName)

	db, _, err := database.Open(ctx, "postgres", schemaDSN)
	if err != nil {
		t.Fatalf("Open schemaDB(postgres) failed: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`SET search_path TO %s`, schemaName)); err != nil {
		t.Fatalf("set search_path on schemaDB: %v", err)
	}

	var currentSearchPath string
	if err := db.QueryRowContext(ctx, `SHOW search_path`).Scan(&currentSearchPath); err != nil {
		t.Fatalf("query current search_path: %v", err)
	}
	if !strings.Contains(currentSearchPath, schemaName) {
		t.Fatalf("expected search_path to contain %q, got %q", schemaName, currentSearchPath)
	}

	// Assert that before Apply, schema_migrations does not exist in the fresh schema
	var preCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name = 'schema_migrations'`, schemaName).Scan(&preCount); err != nil {
		t.Fatalf("query pre-existing table: %v", err)
	}
	if preCount != 0 {
		t.Fatalf("expected 0 schema_migrations tables in fresh schema, found %d", preCount)
	}

	// Apply migrations via migrate.Apply (executing fresh PostgreSQL migration sequence)
	if err := Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("Apply(postgres) failed in fresh schema: %v", err)
	}

	// Verify exact migration filename set and current timestamps in schema_migrations
	rows, err := db.QueryContext(ctx, `SELECT version, applied_unix FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	defer rows.Close()

	type migRecord struct {
		version     string
		appliedUnix int64
	}
	var observed []migRecord
	for rows.Next() {
		var rec migRecord
		if err := rows.Scan(&rec.version, &rec.appliedUnix); err != nil {
			t.Fatalf("scan schema_migrations row: %v", err)
		}
		observed = append(observed, rec)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	if len(observed) != len(expectedVersions) {
		t.Fatalf("expected exactly %d applied migrations matching disk set, observed %d", len(expectedVersions), len(observed))
	}

	for i, exp := range expectedVersions {
		obs := observed[i]
		if obs.version != exp {
			t.Fatalf("migration index %d mismatch: expected version %q, got %q", i, exp, obs.version)
		}
		// Assert that the timestamp was generated during this exact test run
		if obs.appliedUnix < testStartUnix {
			t.Fatalf("migration %q applied_unix (%d) is older than test start time (%d); must be newly generated", exp, obs.appliedUnix, testStartUnix)
		}
	}
}
