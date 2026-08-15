package media

import (
	"context"
	"errors"
	"fmt"
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

	// Open two independent database connection pools to live PostgreSQL to guarantee
	// distinct backend sessions for competing locking and SKIP LOCKED transactions.
	db1, dialect, err := database.Open(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("Open db1(postgres) failed: %v", err)
	}
	defer db1.Close()

	db2, _, err := database.Open(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("Open db2(postgres) failed: %v", err)
	}
	defer db2.Close()

	if dialect != database.Postgres {
		t.Fatalf("expected dialect %q, got %q", database.Postgres, dialect)
	}

	// Ensure migrations applied
	if err := migrate.Apply(ctx, db1, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store1 := NewSQLRegistryStore(db1, dialect)
	store2 := NewSQLRegistryStore(db2, dialect)

	now := time.Now().Unix()
	nano := time.Now().UnixNano()

	// -------------------------------------------------------------------------
	// 1. Basic verified lifecycle with unique keys:
	// ReserveVerified -> CompleteVerified -> GetByObjectKey
	// -------------------------------------------------------------------------
	obj1 := MediaObject{
		ID:               fmt.Sprintf("pg-media-lifecycle-%d", nano),
		ObjectKey:        fmt.Sprintf("uploads/pg-test-lifecycle-%d.webp", nano),
		SourceUploadKey:  fmt.Sprintf("source-pg-%d", nano),
		ContentType:      "image/webp",
		Bytes:            2048,
		Width:            400,
		Height:           300,
		UploadedByUserID: "pg-user-1",
		VerifiedUnix:     now,
	}
	token1 := fmt.Sprintf("pg-reservation-token-%d", nano)

	reserved, err := store1.ReserveVerified(ctx, obj1, token1)
	if err != nil {
		t.Fatalf("ReserveVerified failed: %v", err)
	}
	if !reserved {
		t.Fatal("expected ReserveVerified to return true for new object")
	}

	if err := store1.CompleteVerified(ctx, obj1, token1); err != nil {
		t.Fatalf("CompleteVerified failed: %v", err)
	}

	fetched, err := store1.GetByObjectKey(ctx, obj1.ObjectKey)
	if err != nil {
		t.Fatalf("GetByObjectKey failed: %v", err)
	}
	if fetched.ID != obj1.ID {
		t.Fatalf("expected object ID %q, got %q", obj1.ID, fetched.ID)
	}

	// -------------------------------------------------------------------------
	// 2. FOR UPDATE Clause Mutation-Sensitive Bounded Blocking:
	// An asset in state='active' (objActive) is held by Tx1 via SELECT ... FOR UPDATE.
	// Tx2 calls ReserveVerified with objMismatch (same ObjectKey, but mismatched Bytes).
	// In store_sql.go:52-54, metadata mismatch returns an error immediately after
	// the SELECT query without executing any UPDATE statement (lines 55-61).
	// Therefore:
	// - With FOR UPDATE on SELECT (store_sql.go:41): Tx2 blocks at line 46 and times out with DeadlineExceeded.
	// - Without FOR UPDATE on SELECT (mutation): Tx2 performs a non-locking SELECT, encounters the metadata
	//   mismatch check at line 52, and returns a non-deadline error before executing the UPDATE statement,
	//   which triggers the !errors.Is(DeadlineExceeded) fatal assertion.
	// -------------------------------------------------------------------------
	objActive := MediaObject{
		ID:               fmt.Sprintf("pg-media-active-%d", nano),
		ObjectKey:        fmt.Sprintf("uploads/pg-test-active-%d.webp", nano),
		SourceUploadKey:  fmt.Sprintf("source-active-%d", nano),
		ContentType:      "image/webp",
		Bytes:            1024,
		Width:            100,
		Height:           100,
		UploadedByUserID: "pg-user-lock",
		VerifiedUnix:     now,
	}
	activeToken := fmt.Sprintf("token-active-%d", nano)

	if _, err := store1.ReserveVerified(ctx, objActive, activeToken); err != nil {
		t.Fatalf("reserve objActive: %v", err)
	}
	if err := store1.CompleteVerified(ctx, objActive, activeToken); err != nil {
		t.Fatalf("complete objActive: %v", err)
	}

	// Tx1 holds row lock on objActive
	tx1, err := db1.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}

	var dummy string
	if err := tx1.QueryRowContext(ctx, `SELECT state FROM media_assets WHERE object_key = $1 FOR UPDATE`, objActive.ObjectKey).Scan(&dummy); err != nil {
		_ = tx1.Rollback()
		t.Fatalf("lock row in tx1: %v", err)
	}

	// Tx2 calls ReserveVerified on objMismatch with short bounded timeout (200ms).
	// With FOR UPDATE, Tx2 blocks on SELECT ... FOR UPDATE (store_sql.go:46).
	// If FOR UPDATE is removed, Tx2 returns metadata mismatch at store_sql.go:53 before UPDATE (store_sql.go:55).
	objMismatch := objActive
	objMismatch.Bytes = 999999

	shortCtx, shortCancel := context.WithTimeout(ctx, 200*time.Millisecond)
	reserving, errBlocked := store2.ReserveVerified(shortCtx, objMismatch, "competing-token")
	shortCancel()

	if errBlocked == nil {
		_ = tx1.Rollback()
		t.Fatalf("expected ReserveVerified on locked active row to block on SELECT ... FOR UPDATE, but returned (reserving=%v) immediately", reserving)
	}
	if !errors.Is(errBlocked, context.DeadlineExceeded) && !errors.Is(errBlocked, context.Canceled) {
		_ = tx1.Rollback()
		t.Fatalf("expected deadline exceeded while waiting for SELECT ... FOR UPDATE row lock, got: %v", errBlocked)
	}

	// Release Tx1 lock
	_ = tx1.Rollback()

	// Once Tx1 releases the lock, ReserveVerified with matching metadata executes renewal and returns (false, nil)
	resAfterRelease, err := store2.ReserveVerified(ctx, objActive, activeToken)
	if err != nil {
		t.Fatalf("ReserveVerified after lock release failed: %v", err)
	}
	if resAfterRelease {
		t.Fatal("expected ReserveVerified renewal on existing active object to return false, got true")
	}

	// -------------------------------------------------------------------------
	// 3. SKIP LOCKED Non-Blocking Selection and Non-Duplication:
	// Two eligible GC rows with unique keys (objA and objB).
	// Tx1 on db1 locks objA with FOR UPDATE.
	// Tx2 on db2 calls ClaimEligible.
	// It must skip held objA, claim objB without blocking, and never duplicate claims.
	// -------------------------------------------------------------------------
	staleTime := now - int64(verifiedMediaRetention.Seconds()) - 100

	objA := MediaObject{
		ID:               fmt.Sprintf("pg-media-skip-a-%d", nano),
		ObjectKey:        fmt.Sprintf("uploads/pg-test-skip-a-%d.webp", nano),
		SourceUploadKey:  fmt.Sprintf("source-pg-a-%d", nano),
		ContentType:      "image/webp",
		Bytes:            500,
		Width:            50,
		Height:           50,
		UploadedByUserID: "pg-user-skip",
		VerifiedUnix:     staleTime,
	}
	objB := MediaObject{
		ID:               fmt.Sprintf("pg-media-skip-b-%d", nano),
		ObjectKey:        fmt.Sprintf("uploads/pg-test-skip-b-%d.webp", nano),
		SourceUploadKey:  fmt.Sprintf("source-pg-b-%d", nano),
		ContentType:      "image/webp",
		Bytes:            500,
		Width:            50,
		Height:           50,
		UploadedByUserID: "pg-user-skip",
		VerifiedUnix:     staleTime,
	}

	if _, err := store1.ReserveVerified(ctx, objA, "token-a"); err != nil {
		t.Fatalf("reserve A: %v", err)
	}
	if err := store1.CompleteVerified(ctx, objA, "token-a"); err != nil {
		t.Fatalf("complete A: %v", err)
	}
	if _, err := store1.ReserveVerified(ctx, objB, "token-b"); err != nil {
		t.Fatalf("reserve B: %v", err)
	}
	if err := store1.CompleteVerified(ctx, objB, "token-b"); err != nil {
		t.Fatalf("complete B: %v", err)
	}

	// Mark both as unassociated and eligible for GC
	if _, err := db1.ExecContext(ctx, `UPDATE media_assets SET unassociated_since_unix = $1 WHERE object_key IN ($2, $3)`, staleTime, objA.ObjectKey, objB.ObjectKey); err != nil {
		t.Fatalf("mark unassociated: %v", err)
	}

	// Hold lock on objA in Tx1 on db1
	txHoldA, err := db1.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin txHoldA: %v", err)
	}

	var stateA string
	if err := txHoldA.QueryRowContext(ctx, `SELECT state FROM media_assets WHERE object_key = $1 FOR UPDATE`, objA.ObjectKey).Scan(&stateA); err != nil {
		_ = txHoldA.Rollback()
		t.Fatalf("lock objA in txHoldA: %v", err)
	}

	// Concurrently on db2, ClaimEligible executes listEligible (which uses FOR UPDATE SKIP LOCKED)
	claimStart := time.Now()
	jobs1, err := store2.ClaimEligible(ctx, now, 10)
	claimDuration := time.Since(claimStart)

	if err != nil {
		_ = txHoldA.Rollback()
		t.Fatalf("ClaimEligible with locked row failed: %v", err)
	}

	// Assert non-blocking execution (SKIP LOCKED does not wait on held lock)
	if claimDuration > 2*time.Second {
		_ = txHoldA.Rollback()
		t.Fatalf("ClaimEligible took too long (%v), SKIP LOCKED should not block", claimDuration)
	}

	// Assert objB was claimed while held objA was skipped
	if len(jobs1) != 1 || jobs1[0].ObjectKey != objB.ObjectKey {
		_ = txHoldA.Rollback()
		t.Fatalf("ClaimEligible batch 1 expected exactly [objB (%s)], got: %v", objB.ObjectKey, jobs1)
	}

	// Release objA lock
	_ = txHoldA.Rollback()

	// Batch 2: ClaimEligible must now claim released objA AND MUST NOT duplicate objB
	jobs2, err := store2.ClaimEligible(ctx, now, 10)
	if err != nil {
		t.Fatalf("ClaimEligible batch 2 failed: %v", err)
	}
	if len(jobs2) != 1 || jobs2[0].ObjectKey != objA.ObjectKey {
		t.Fatalf("ClaimEligible batch 2 expected exactly [objA (%s)], got: %v (must not duplicate objB)", objA.ObjectKey, jobs2)
	}

	// Batch 3: Neither objA nor objB may be claimed again
	jobs3, err := store2.ClaimEligible(ctx, now, 10)
	if err != nil {
		t.Fatalf("ClaimEligible batch 3 failed: %v", err)
	}
	if len(jobs3) != 0 {
		t.Fatalf("ClaimEligible batch 3 expected 0 jobs, got %v; duplicate claim detected", jobs3)
	}
}
