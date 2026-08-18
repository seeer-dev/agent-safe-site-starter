package staff

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
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

	// Isolate this test in a unique PostgreSQL schema so parallel package tests
	// do not race on CREATE TABLE/type catalog entries during migrate.Apply.
	adminDB, _, err := database.Open(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("Open adminDB(postgres) failed: %v", err)
	}
	schemaName := fmt.Sprintf("staff_test_%d", time.Now().UnixNano())
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, schemaName)); err != nil {
		t.Fatalf("create schema %s: %v", schemaName, err)
	}
	defer func() {
		cleanCtx, cancelClean := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelClean()
		_, _ = adminDB.ExecContext(cleanCtx, fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schemaName))
		adminDB.Close()
	}()

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	schemaDSN := fmt.Sprintf("%s%ssearch_path=%s", dsn, sep, schemaName)

	// Open two independent connection pools to live PostgreSQL to guarantee
	// separate backend connections and distinct concurrent transactions.
	// SetMaxOpenConns(1) preserves the connection-scoped search_path.
	db1, dialect, err := database.Open(ctx, "postgres", schemaDSN)
	if err != nil {
		t.Fatalf("Open db1(postgres) failed: %v", err)
	}
	defer db1.Close()
	db1.SetMaxOpenConns(1)

	db2, _, err := database.Open(ctx, "postgres", schemaDSN)
	if err != nil {
		t.Fatalf("Open db2(postgres) failed: %v", err)
	}
	defer db2.Close()
	db2.SetMaxOpenConns(1)

	if dialect != database.Postgres {
		t.Fatalf("expected dialect %q, got %q", database.Postgres, dialect)
	}

	// Ensure migrations applied to the isolated schema
	if err := migrate.Apply(ctx, db1, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store1 := NewSQLStore(db1, dialect)
	store2 := NewSQLStore(db2, dialect)

	svc1 := NewService(store1)
	svc2 := NewService(store2)

	// Seed two active owners (CI uses an ephemeral database; reuse may contaminate owner count)
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

	if err := store1.Upsert(ctx, owner1); err != nil {
		t.Fatalf("upsert owner 1: %v", err)
	}
	if err := store1.Upsert(ctx, owner2); err != nil {
		t.Fatalf("upsert owner 2: %v", err)
	}

	// Setup deterministic synchronization barrier on store1 and store2 instances.
	// When Tx1 executes lockActiveOwners in UpsertGuarded, it acquires FOR UPDATE on all active owner rows.
	// Tx2 on store2 then enters lockActiveOwners, fires testLockAttemptHook, and blocks in PostgreSQL waiting for Tx1.
	tx1Locked := make(chan struct{})
	proceedTx1 := make(chan struct{})
	tx2LockAttempted := make(chan struct{})

	store1.testAfterLockHook = func(_ context.Context, _ *sql.Tx) {
		select {
		case tx1Locked <- struct{}{}:
		default:
		}
		select {
		case <-proceedTx1:
		case <-ctx.Done():
		}
	}
	svc1 = NewService(store1)

	store2.testLockAttemptHook = func(_ context.Context, _ *sql.Tx) {
		select {
		case tx2LockAttempted <- struct{}{}:
		default:
		}
	}
	svc2 = NewService(store2)

	operatorPrincipal := auth.Principal{
		UserID:       "operator-admin",
		Capabilities: []string{"staff.update"},
	}

	// Start Op 1 in a background goroutine: demote owner 1 via svc1 on db1
	var wg1 sync.WaitGroup
	var err1 error

	wg1.Add(1)
	go func() {
		defer wg1.Done()
		_, err1 = svc1.Update(ctx, operatorPrincipal, owner1.ID, StaffMemberInput{
			DisplayName: owner1.DisplayName,
			RoleLabel:   "readonly",
		})
	}()

	// Wait until Tx1 holds the FOR UPDATE lock on all active owner rows
	select {
	case <-tx1Locked:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Tx1 to acquire active owner locks")
	}

	// -------------------------------------------------------------------------
	// Competing Op 2 Overlap & Serialization:
	// Start Op 2 on db2 attempting to demote owner 2 concurrently while Tx1 holds the lock.
	// Op 2 MUST block on the PostgreSQL FOR UPDATE row lock until Tx1 releases.
	// -------------------------------------------------------------------------
	op2Done := make(chan struct{})
	var err2 error

	go func() {
		defer close(op2Done)
		_, err2 = svc2.Update(ctx, operatorPrincipal, owner2.ID, StaffMemberInput{
			DisplayName: owner2.DisplayName,
			RoleLabel:   "readonly",
		})
	}()

	// Wait until Op 2 enters lockActiveOwners before checking the 150ms wait
	select {
	case <-tx2LockAttempted:
	case <-time.After(5 * time.Second):
		close(proceedTx1)
		t.Fatal("timed out waiting for Op 2 to attempt lockActiveOwners")
	}

	// Verify that Op 2 does not complete prematurely while Tx1 is holding the lock (blocking observation)
	select {
	case <-op2Done:
		close(proceedTx1)
		t.Fatal("Op 2 completed prematurely before Tx1 released lock; active owner row lock failed to block competing operation")
	case <-time.After(150 * time.Millisecond):
		// Op 2 reached lockActiveOwners and did not complete within 150ms, confirming it is blocked on the PostgreSQL FOR UPDATE row lock
	}

	// Release Tx1 to proceed and commit the demotion of owner 1
	close(proceedTx1)

	// Wait for both operations to conclude
	wg1.Wait()

	select {
	case <-op2Done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Op 2 to complete after Tx1 released lock")
	}

	// Assert outcomes: Op 1 succeeded, Op 2 was unblocked and rejected with canonical ErrLastOwner
	if err1 != nil {
		t.Fatalf("expected Op 1 to succeed, got error: %v", err1)
	}
	if !errors.Is(err2, ErrLastOwner) {
		t.Fatalf("expected Op 2 to return canonical ErrLastOwner after serialization, got: %v", err2)
	}

	// Assert invariant: exactly one active owner remains in the database
	activeOwners, err := store1.ListActive(ctx)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	ownerCount := 0
	for _, m := range activeOwners {
		if m.RoleLabel == "owner" && m.Status == "active" {
			ownerCount++
		}
	}
	if ownerCount != 1 {
		t.Fatalf("expected exactly 1 active owner remaining in DB, found %d", ownerCount)
	}

	// Sequential demotion of the final remaining owner must also return canonical ErrLastOwner
	_, err = svc1.Update(ctx, operatorPrincipal, owner2.ID, StaffMemberInput{
		DisplayName: owner2.DisplayName,
		RoleLabel:   "readonly",
	})
	if !errors.Is(err, ErrLastOwner) {
		t.Fatalf("expected ErrLastOwner when demoting the sole remaining owner, got: %v", err)
	}
}
