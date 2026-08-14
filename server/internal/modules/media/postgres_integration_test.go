package media

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

func TestPostgresLiveMediaLifecycleAndLocks(t *testing.T) {
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

	store := NewSQLRegistryStore(db, dialect)

	now := time.Now().Unix()
	obj := MediaObject{
		ID:               "pg-media-obj-1",
		ObjectKey:        "uploads/pg-test-obj-1.webp",
		SourceUploadKey:  "source-pg-1",
		ContentType:      "image/webp",
		Bytes:            2048,
		Width:            400,
		Height:           300,
		UploadedByUserID: "pg-user-1",
		VerifiedUnix:     now,
	}
	token := "pg-reservation-token-1"

	// 1. ReserveVerified (exercises Branch 5: SELECT ... FROM media_assets ... FOR UPDATE in ReserveVerified)
	reserved, err := store.ReserveVerified(ctx, obj, token)
	if err != nil {
		t.Fatalf("ReserveVerified failed: %v", err)
	}
	if !reserved {
		t.Fatal("expected ReserveVerified to return true for new object")
	}

	// 2. CompleteVerified (exercises Branch 6: SELECT ... FROM media_assets ... FOR UPDATE in CompleteVerified)
	if err := store.CompleteVerified(ctx, obj, token); err != nil {
		t.Fatalf("CompleteVerified failed: %v", err)
	}

	// Verify object is active and retrievable
	fetched, err := store.GetByObjectKey(ctx, obj.ObjectKey)
	if err != nil {
		t.Fatalf("GetByObjectKey failed: %v", err)
	}
	if fetched.ID != obj.ID {
		t.Fatalf("expected object ID %q, got %q", obj.ID, fetched.ID)
	}

	// 3. Mark unassociated and exercise ClaimEligible (exercises Branch 7: SELECT ... FOR UPDATE SKIP LOCKED in listEligible)
	staleTime := now - int64(verifiedMediaRetention.Seconds()) - 100
	_, err = db.ExecContext(ctx, `UPDATE media_assets SET unassociated_since_unix = $1 WHERE object_key = $2`, staleTime, obj.ObjectKey)
	if err != nil {
		t.Fatalf("update unassociated_since_unix: %v", err)
	}

	jobs, err := store.ClaimEligible(ctx, now, 10)
	if err != nil {
		t.Fatalf("ClaimEligible failed: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ObjectKey != obj.ObjectKey {
		t.Fatalf("expected 1 GC job for %q, got: %v", obj.ObjectKey, jobs)
	}
}
