package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckParityMatchingSets(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	sqliteDir := filepath.Join(tmp, "sqlite")
	postgresDir := filepath.Join(tmp, "postgres")
	if err := os.MkdirAll(sqliteDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(postgresDir, 0o755); err != nil {
		t.Fatal(err)
	}

	files := []string{"001_init.sql", "002_commerce.sql", "003_staff.sql"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(sqliteDir, f), []byte("-- sqlite"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(postgresDir, f), []byte("-- pg"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	violations, count, err := checkParity(sqliteDir, postgresDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %v", violations)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}
}

func TestCheckParityAsymmetricSets(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	sqliteDir := filepath.Join(tmp, "sqlite")
	postgresDir := filepath.Join(tmp, "postgres")
	if err := os.MkdirAll(sqliteDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(postgresDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 001 exists in both
	_ = os.WriteFile(filepath.Join(sqliteDir, "001_init.sql"), []byte("-- sqlite"), 0o644)
	_ = os.WriteFile(filepath.Join(postgresDir, "001_init.sql"), []byte("-- pg"), 0o644)

	// 002 exists only in sqlite
	_ = os.WriteFile(filepath.Join(sqliteDir, "002_sqlite_only.sql"), []byte("-- sqlite"), 0o644)

	// 003 exists only in postgres
	_ = os.WriteFile(filepath.Join(postgresDir, "003_postgres_only.sql"), []byte("-- pg"), 0o644)

	violations, count, err := checkParity(sqliteDir, postgresDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count 0 on violation, got %d", count)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d: %v", len(violations), violations)
	}
}

func TestCheckParityActualRepo(t *testing.T) {
	t.Parallel()

	root := findRepoRoot(t)
	sqliteDir := filepath.Join(root, "db", "migrations", "sqlite")
	postgresDir := filepath.Join(root, "db", "migrations", "postgres")

	violations, count, err := checkParity(sqliteDir, postgresDir)
	if err != nil {
		t.Fatalf("check repo parity error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("repo has migration parity violations: %v", violations)
	}
	if count < 16 {
		t.Fatalf("expected at least 16 migrations matching, got %d", count)
	}
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
