package bootstrap

import (
	"context"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/modules/staff"
)

// staffLookupAdapter wraps the staff store so the auth resolver can look up
// canonical staff rows without the auth package importing the staff module.
type staffLookupAdapter struct {
	store staff.Store
}

func newStaffLookupAdapter(store staff.Store) staffLookupAdapter {
	return staffLookupAdapter{store: store}
}

func (a staffLookupAdapter) GetBySupabaseUserID(ctx context.Context, supabaseUserID string) (auth.StaffRow, error) {
	sm, err := a.store.GetBySupabaseUserID(ctx, supabaseUserID)
	if err != nil {
		return auth.StaffRow{}, err
	}
	return staffToRow(sm), nil
}

func staffToRow(sm staff.StaffMember) auth.StaffRow {
	return auth.StaffRow{
		ID:             sm.ID,
		DisplayName:    sm.DisplayName,
		Email:          sm.Email,
		RoleLabel:      sm.RoleLabel,
		SupabaseUserID: sm.SupabaseUserID,
		Status:         sm.Status,
	}
}
