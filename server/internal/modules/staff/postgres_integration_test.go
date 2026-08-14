package staff

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/migrate"
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

func TestPostgresLiveStaffLockActiveOwners(t *testing.T) {
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

	// Ensure migrations applied
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store := NewSQLStore(db, dialect)

	// Seed two active owners
	owner1 := StaffMember{
		ID:             "pg-owner-1",
		DisplayName:    "PG Owner One",
		Email:          "pg-owner-1@example.com",
		RoleLabel:      "owner",
		SupabaseUserID: "pg-sub-1",
		Status:         "active",
		UpdatedUnix:    time.Now().Unix(),
	}
	owner2 := StaffMember{
		ID:             "pg-owner-2",
		DisplayName:    "PG Owner Two",
		Email:          "pg-owner-2@example.com",
		RoleLabel:      "owner",
		SupabaseUserID: "pg-sub-2",
		Status:         "active",
		UpdatedUnix:    time.Now().Unix(),
	}

	if err := store.Upsert(ctx, owner1); err != nil {
		t.Fatalf("upsert owner 1: %v", err)
	}
	if err := store.Upsert(ctx, owner2); err != nil {
		t.Fatalf("upsert owner 2: %v", err)
	}

	// Test UpsertGuarded (demoting owner1 while owner2 exists, which exercises lockActiveOwners FOR UPDATE in PostgreSQL)
	demoted := owner1
	demoted.RoleLabel = "admin"
	if err := store.UpsertGuarded(ctx, demoted); err != nil {
		t.Fatalf("UpsertGuarded demotion failed: %v", err)
	}

	// Verify demotion succeeded
	fetched, err := store.GetByID(ctx, "pg-owner-1")
	if err != nil {
		t.Fatalf("get demoted owner: %v", err)
	}
	if fetched.RoleLabel != "admin" {
		t.Fatalf("expected role_label admin, got %q", fetched.RoleLabel)
	}

	// Test last-owner protection (demoting owner2 when no other owner exists)
	lastOwnerDemote := owner2
	lastOwnerDemote.RoleLabel = "admin"
	if err := store.UpsertGuarded(ctx, lastOwnerDemote); err != ErrLastOwner {
		t.Fatalf("expected ErrLastOwner, got %v", err)
	}
}
