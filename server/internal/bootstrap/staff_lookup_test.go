package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/modules/staff"
)

type stubStaffStore struct {
	staff.Store
	getBySupabaseUserFunc func(ctx context.Context, supabaseUserID string) (staff.StaffMember, error)
}

func (s stubStaffStore) GetBySupabaseUserID(ctx context.Context, supabaseUserID string) (staff.StaffMember, error) {
	if s.getBySupabaseUserFunc != nil {
		return s.getBySupabaseUserFunc(ctx, supabaseUserID)
	}
	return staff.StaffMember{}, errors.New("not implemented")
}

func TestStaffLookupAdapterTranslatesNotFound(t *testing.T) {
	t.Parallel()
	adapter := newStaffLookupAdapter(stubStaffStore{
		getBySupabaseUserFunc: func(ctx context.Context, supabaseUserID string) (staff.StaffMember, error) {
			return staff.StaffMember{}, staff.ErrNotFound
		},
	})

	_, err := adapter.GetBySupabaseUserID(context.Background(), "missing-user")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, auth.ErrStaffNotFound) {
		t.Fatalf("expected auth.ErrStaffNotFound, got %v", err)
	}
}

func TestStaffLookupAdapterPreservesInfrastructureError(t *testing.T) {
	t.Parallel()
	dbErr := errors.New("database connection refused")
	adapter := newStaffLookupAdapter(stubStaffStore{
		getBySupabaseUserFunc: func(ctx context.Context, supabaseUserID string) (staff.StaffMember, error) {
			return staff.StaffMember{}, dbErr
		},
	})

	_, err := adapter.GetBySupabaseUserID(context.Background(), "user-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, auth.ErrStaffNotFound) {
		t.Fatal("infrastructure error must NOT be translated to auth.ErrStaffNotFound")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected original cause to be preserved, got %v", err)
	}
}

func TestStaffLookupAdapterSuccess(t *testing.T) {
	t.Parallel()
	adapter := newStaffLookupAdapter(stubStaffStore{
		getBySupabaseUserFunc: func(ctx context.Context, supabaseUserID string) (staff.StaffMember, error) {
			return staff.StaffMember{
				ID:             "stf-1",
				DisplayName:    "Alice",
				Email:          "alice@example.com",
				RoleLabel:      "manager",
				SupabaseUserID: supabaseUserID,
				Status:         "active",
			}, nil
		},
	})

	row, err := adapter.GetBySupabaseUserID(context.Background(), "sup-alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.ID != "stf-1" || row.RoleLabel != "manager" || row.Status != "active" {
		t.Fatalf("unexpected row mapped: %+v", row)
	}
}
