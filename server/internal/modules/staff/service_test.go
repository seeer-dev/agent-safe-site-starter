package staff

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

var staffTestDBCounter uint64

func newStaffTestStore(t *testing.T) Store {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&staffTestDBCounter, 1)
	dsn := fmt.Sprintf("file:staff_test_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return NewSQLStore(db, dialect)
}

func ownerPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "dev",
		Role:   "admin",
		Capabilities: []string{
			"staff.read",
			"staff.update",
		},
	}
}

func TestStaffCreateDefaultsToActive(t *testing.T) {
	t.Parallel()
	svc := NewService(newStaffTestStore(t))
	sm, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Test Owner",
		Email:       "owner@test.local",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sm.Status != "active" {
		t.Errorf("status = %q, want active", sm.Status)
	}
}

func TestStaffSelfElevationBlocked(t *testing.T) {
	t.Parallel()
	svc := NewService(newStaffTestStore(t))
	sm, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Self Owner",
		Email:       "self@test.local",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Principal with StaffID matching the target cannot update themselves.
	principal := ownerPrincipal()
	principal.StaffID = sm.ID
	_, err = svc.Update(context.Background(), principal, sm.ID, StaffMemberInput{
		RoleLabel: "readonly",
	})
	if !errors.Is(err, ErrSelfElevation) {
		t.Errorf("self-elevation should be blocked, got %v", err)
	}
}

func TestStaffLastOwnerCannotBeDisabled(t *testing.T) {
	t.Parallel()
	svc := NewService(newStaffTestStore(t))
	sm, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Only Owner",
		Email:       "only@test.local",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.UpdateStatus(context.Background(), ownerPrincipal(), sm.ID, "disabled")
	if !errors.Is(err, ErrLastOwner) {
		t.Errorf("disabling last owner should be blocked, got %v", err)
	}
}

func TestStaffLastOwnerCannotBeDeleted(t *testing.T) {
	t.Parallel()
	svc := NewService(newStaffTestStore(t))
	sm, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Only Owner",
		Email:       "only2@test.local",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	err = svc.Delete(context.Background(), ownerPrincipal(), sm.ID)
	if !errors.Is(err, ErrLastOwner) {
		t.Errorf("deleting last owner should be blocked, got %v", err)
	}
}

func TestStaffLastOwnerCannotBeDemoted(t *testing.T) {
	t.Parallel()
	svc := NewService(newStaffTestStore(t))
	sm, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Only Owner",
		Email:       "only3@test.local",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.Update(context.Background(), ownerPrincipal(), sm.ID, StaffMemberInput{
		RoleLabel: "manager",
	})
	if !errors.Is(err, ErrLastOwner) {
		t.Errorf("demoting last owner should be blocked, got %v", err)
	}
}

func TestStaffMultipleOwnersOneCanBeDisabled(t *testing.T) {
	t.Parallel()
	svc := NewService(newStaffTestStore(t))
	sm1, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner 1",
		Email:       "o1@test.local",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create 1: %v", err)
	}
	_, err = svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner 2",
		Email:       "o2@test.local",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create 2: %v", err)
	}
	_, err = svc.UpdateStatus(context.Background(), ownerPrincipal(), sm1.ID, "disabled")
	if err != nil {
		t.Errorf("disabling one of two owners should succeed, got %v", err)
	}
}

func TestStaffGetBySupabaseUserID(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	svc := NewService(store)
	sm, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName:    "Linked Owner",
		Email:          "linked@test.local",
		RoleLabel:      "owner",
		SupabaseUserID: "sup-abc-123",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	found, err := store.GetBySupabaseUserID(context.Background(), "sup-abc-123")
	if err != nil {
		t.Fatalf("get by supabase id: %v", err)
	}
	if found.ID != sm.ID {
		t.Errorf("found ID = %q, want %q", found.ID, sm.ID)
	}
	// Empty ID should return ErrNotFound
	_, err = store.GetBySupabaseUserID(context.Background(), "")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("empty supabase id should return ErrNotFound, got %v", err)
	}
}

// TestStaffDuplicateSupabaseUserID proves the unique index on
// supabase_user_id (migration 004) prevents two staff rows from being
// linked to the same Supabase identity.
func TestStaffDuplicateSupabaseUserID(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	svc := NewService(store)
	_, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName:    "First Owner",
		Email:          "first@test.local",
		RoleLabel:      "owner",
		SupabaseUserID: "sup-dup-001",
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	_, err = svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName:    "Second Owner",
		Email:          "second@test.local",
		RoleLabel:      "owner",
		SupabaseUserID: "sup-dup-001", // same supabase_user_id
	})
	if err == nil {
		t.Fatal("duplicate supabase_user_id should fail, got nil error")
	}
}

// TestStaffDuplicateSupabaseUserIDEmptyAllowed proves that multiple staff
// rows can have empty supabase_user_id (the unique index is partial).
func TestStaffDuplicateSupabaseUserIDEmptyAllowed(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	svc := NewService(store)
	_, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner A",
		Email:       "a@test.local",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	_, err = svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Manager B",
		Email:       "b@test.local",
		RoleLabel:   "manager",
	})
	if err != nil {
		t.Fatalf("create B with empty supabase_user_id should succeed: %v", err)
	}
}

// TestStaffConcurrentDemoteCrossRowKeepsOneOwner proves the cross-row
// last-owner invariant: two active owners A and B, two concurrent
// goroutines demote A and B to manager simultaneously. At least one
// must be blocked so at least one active owner survives.
//
// NOTE: SQLite's SetMaxOpenConns(1) serializes goroutines, so this test
// proves the guard LOGIC but not true concurrent access. PostgreSQL
// validation requires a live PG instance (AC-004/AC-022 pending).
func TestStaffConcurrentDemoteCrossRowKeepsOneOwner(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	svc := NewService(store)
	smA, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner A",
		Email:       "owner-a@cross-row.test",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	smB, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner B",
		Email:       "owner-b@cross-row.test",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create B: %v", err)
	}

	var (
		wg      sync.WaitGroup
		errs    [2]error
		success int
		blocked int
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[0] = svc.Update(context.Background(), ownerPrincipal(), smA.ID, StaffMemberInput{
			RoleLabel: "manager",
		})
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[1] = svc.Update(context.Background(), ownerPrincipal(), smB.ID, StaffMemberInput{
			RoleLabel: "manager",
		})
	}()
	wg.Wait()

	for _, e := range errs {
		if e == nil {
			success++
		} else if errors.Is(e, ErrLastOwner) {
			blocked++
		} else {
			t.Errorf("unexpected error: %v", e)
		}
	}
	// At least one must be blocked — at least one active owner must survive.
	if blocked < 1 {
		t.Errorf("expected at least 1 blocked (cross-row last-owner guard), got success=%d blocked=%d", success, blocked)
	}
	if success == 0 && blocked == 2 {
		t.Errorf("expected at least 1 success (one owner can be demoted), got success=%d blocked=%d", success, blocked)
	}
	// Verify at least one active owner remains.
	active, err := store.ListActive(context.Background())
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	ownerCount := 0
	for _, sm := range active {
		if sm.RoleLabel == "owner" {
			ownerCount++
		}
	}
	if ownerCount < 1 {
		t.Errorf("expected at least 1 active owner to survive, got %d", ownerCount)
	}
}

// TestStaffConcurrentDisableCrossRowKeepsOneOwner proves the cross-row
// last-owner invariant for concurrent disable operations.
//
// NOTE: SQLite serializes goroutines (SetMaxOpenConns(1)). PostgreSQL
// validation requires a live PG instance (AC-004/AC-022 pending).
func TestStaffConcurrentDisableCrossRowKeepsOneOwner(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	svc := NewService(store)
	smA, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner A",
		Email:       "disable-a@cross-row.test",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	smB, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner B",
		Email:       "disable-b@cross-row.test",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create B: %v", err)
	}

	var (
		wg      sync.WaitGroup
		errs    [2]error
		success int
		blocked int
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[0] = svc.UpdateStatus(context.Background(), ownerPrincipal(), smA.ID, "disabled")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[1] = svc.UpdateStatus(context.Background(), ownerPrincipal(), smB.ID, "disabled")
	}()
	wg.Wait()

	for _, e := range errs {
		if e == nil {
			success++
		} else if errors.Is(e, ErrLastOwner) {
			blocked++
		} else {
			t.Errorf("unexpected error: %v", e)
		}
	}
	if blocked < 1 {
		t.Errorf("expected at least 1 blocked, got success=%d blocked=%d", success, blocked)
	}
	active, _ := store.ListActive(context.Background())
	ownerCount := 0
	for _, sm := range active {
		if sm.RoleLabel == "owner" {
			ownerCount++
		}
	}
	if ownerCount < 1 {
		t.Errorf("expected at least 1 active owner to survive, got %d", ownerCount)
	}
}

// TestStaffConcurrentDeleteCrossRowKeepsOneOwner proves the cross-row
// last-owner invariant for concurrent delete operations.
//
// NOTE: SQLite serializes goroutines (SetMaxOpenConns(1)). PostgreSQL
// validation requires a live PG instance (AC-004/AC-022 pending).
func TestStaffConcurrentDeleteCrossRowKeepsOneOwner(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	svc := NewService(store)
	smA, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner A",
		Email:       "delete-a@cross-row.test",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	smB, err := svc.Create(context.Background(), ownerPrincipal(), StaffMemberInput{
		DisplayName: "Owner B",
		Email:       "delete-b@cross-row.test",
		RoleLabel:   "owner",
	})
	if err != nil {
		t.Fatalf("create B: %v", err)
	}

	var (
		wg      sync.WaitGroup
		errs    [2]error
		success int
		blocked int
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		errs[0] = svc.Delete(context.Background(), ownerPrincipal(), smA.ID)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		errs[1] = svc.Delete(context.Background(), ownerPrincipal(), smB.ID)
	}()
	wg.Wait()

	for _, e := range errs {
		if e == nil {
			success++
		} else if errors.Is(e, ErrLastOwner) {
			blocked++
		} else {
			t.Errorf("unexpected error: %v", e)
		}
	}
	if blocked < 1 {
		t.Errorf("expected at least 1 blocked, got success=%d blocked=%d", success, blocked)
	}
	active, _ := store.ListActive(context.Background())
	ownerCount := 0
	for _, sm := range active {
		if sm.RoleLabel == "owner" {
			ownerCount++
		}
	}
	if ownerCount < 1 {
		t.Errorf("expected at least 1 active owner to survive, got %d", ownerCount)
	}
}
