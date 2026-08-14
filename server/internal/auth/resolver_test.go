package auth

import (
	"context"
	"errors"
	"testing"
)

type stubLookup struct {
	bySupabaseID  StaffRow
	bySupabaseErr error
}

func (s *stubLookup) GetBySupabaseUserID(ctx context.Context, id string) (StaffRow, error) {
	return s.bySupabaseID, s.bySupabaseErr
}

func TestStaffCapabilityResolverActiveOwner(t *testing.T) {
	t.Parallel()
	lookup := &stubLookup{
		bySupabaseID: StaffRow{
			ID:        "stf-1",
			RoleLabel: "owner",
			Status:    "active",
		},
	}
	r := NewStaffCapabilityResolver(lookup)
	p := Principal{UserID: "sup-1", Email: "owner@example.com"}
	resolved, err := r.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.StaffID != "stf-1" {
		t.Errorf("StaffID = %q, want stf-1", resolved.StaffID)
	}
	if resolved.Role != "owner" {
		t.Errorf("Role = %q, want owner", resolved.Role)
	}
	want := capabilitiesForRole("owner")
	if len(resolved.Capabilities) != len(want) {
		t.Fatalf("capabilities: got %d, want %d", len(resolved.Capabilities), len(want))
	}
	got := map[string]bool{}
	for _, c := range resolved.Capabilities {
		got[c] = true
	}
	for _, c := range want {
		if !got[c] {
			t.Errorf("missing capability %q", c)
		}
	}
	// Explicitly assert content.approve is granted to owner (B5 approval gate).
	if !got["content.approve"] {
		t.Error("owner must have content.approve capability (B5 approval gate)")
	}
}

func TestStaffCapabilityResolverActiveManager(t *testing.T) {
	t.Parallel()
	lookup := &stubLookup{
		bySupabaseID: StaffRow{
			ID:        "stf-mgr",
			RoleLabel: "manager",
			Status:    "active",
		},
	}
	r := NewStaffCapabilityResolver(lookup)
	p := Principal{UserID: "sup-mgr", Email: "manager@example.com"}
	resolved, err := r.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Role != "manager" {
		t.Errorf("Role = %q, want manager", resolved.Role)
	}
	got := map[string]bool{}
	for _, c := range resolved.Capabilities {
		got[c] = true
	}
	// Manager must have content.approve (B5 approval gate).
	if !got["content.approve"] {
		t.Error("manager must have content.approve capability (B5 approval gate)")
	}
}

func TestStaffCapabilityResolverDisabledStaff(t *testing.T) {
	t.Parallel()
	lookup := &stubLookup{
		bySupabaseID: StaffRow{
			ID:        "stf-2",
			RoleLabel: "owner",
			Status:    "disabled",
		},
	}
	r := NewStaffCapabilityResolver(lookup)
	p := Principal{UserID: "sup-2", Email: "owner@example.com"}
	resolved, err := r.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved.Capabilities) != 0 {
		t.Errorf("disabled staff should have 0 capabilities, got %v", resolved.Capabilities)
	}
	if resolved.Role != "disabled" {
		t.Errorf("Role = %q, want disabled", resolved.Role)
	}
}

func TestStaffCapabilityResolverNoStaffRow(t *testing.T) {
	t.Parallel()
	lookup := &stubLookup{
		bySupabaseErr: errors.New("not found"),
	}
	r := NewStaffCapabilityResolver(lookup)
	p := Principal{UserID: "sup-3", Email: "nobody@example.com"}
	resolved, err := r.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved.Capabilities) != 0 {
		t.Errorf("user without staff row should have 0 capabilities, got %v", resolved.Capabilities)
	}
	if resolved.Role != "user" {
		t.Errorf("Role = %q, want user", resolved.Role)
	}
}

func TestStaffCapabilityResolverReadonlyRole(t *testing.T) {
	t.Parallel()
	lookup := &stubLookup{
		bySupabaseID: StaffRow{
			ID:        "stf-3",
			RoleLabel: "readonly",
			Status:    "active",
		},
	}
	r := NewStaffCapabilityResolver(lookup)
	p := Principal{UserID: "sup-4", Email: "staff@example.com"}
	resolved, err := r.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want := capabilitiesForRole("readonly")
	if len(resolved.Capabilities) != len(want) {
		t.Fatalf("capabilities: got %d, want %d", len(resolved.Capabilities), len(want))
	}
	for _, c := range resolved.Capabilities {
		if c == "content.publish" || c == "content.approve" || c == "staff.update" || c == "twcommerce.update" {
			t.Errorf("readonly should not have %q", c)
		}
	}
}

func TestStaffCapabilityResolverPreservesExistingCapabilities(t *testing.T) {
	t.Parallel()
	lookup := &stubLookup{}
	r := NewStaffCapabilityResolver(lookup)
	p := Principal{
		UserID:       "dev",
		Role:         "admin",
		Capabilities: []string{"twcommerce.admin", "content.publish"},
	}
	resolved, err := r.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved.Capabilities) != 2 {
		t.Errorf("should preserve existing capabilities, got %v", resolved.Capabilities)
	}
}

// TestStaffCapabilityResolverEmailImpersonation proves that a Supabase user
// whose email matches a staff row but whose supabase_user_id is NOT linked
// gets no capabilities. Email matching is not a valid linking mechanism.
func TestStaffCapabilityResolverEmailImpersonation(t *testing.T) {
	t.Parallel()
	// The lookup returns "not found" for the supabase_user_id, even though
	// the staff row exists with the same email. The resolver must NOT
	// fall back to email lookup.
	lookup := &stubLookup{
		bySupabaseErr: errors.New("not found"),
	}
	r := NewStaffCapabilityResolver(lookup)
	p := Principal{
		UserID: "sup-attacker",
		Email:  "owner@example.com", // matches staff email but not linked
	}
	resolved, err := r.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved.Capabilities) != 0 {
		t.Errorf("email impersonation: should have 0 capabilities, got %v", resolved.Capabilities)
	}
	if resolved.Role != "user" {
		t.Errorf("Role = %q, want user", resolved.Role)
	}
}

// TestStaffCapabilityResolverUnlinkedUser proves that a valid Supabase user
// who simply hasn't been linked to a staff row yet gets no capabilities,
// even if their email is known to the system.
func TestStaffCapabilityResolverUnlinkedUser(t *testing.T) {
	t.Parallel()
	lookup := &stubLookup{
		bySupabaseErr: errors.New("not found"),
	}
	r := NewStaffCapabilityResolver(lookup)
	p := Principal{UserID: "sup-new-user", Email: "new@example.com"}
	resolved, err := r.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved.Capabilities) != 0 {
		t.Errorf("unlinked user should have 0 capabilities, got %v", resolved.Capabilities)
	}
	if resolved.StaffID != "" {
		t.Errorf("unlinked user should have empty StaffID, got %q", resolved.StaffID)
	}
}
