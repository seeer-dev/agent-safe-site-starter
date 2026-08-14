package auth

import (
	"context"
)

// StaffLookup is the minimal store interface the resolver needs to map a
// verified identity to a canonical staff row. It is defined here to avoid
// importing the staff module (which would violate module isolation).
// Linking is exclusively via supabase_user_id — email-based lookup is
// intentionally excluded to prevent email impersonation attacks.
type StaffLookup interface {
	GetBySupabaseUserID(ctx context.Context, supabaseUserID string) (StaffRow, error)
}

// StaffRow is the canonical staff record used for capability resolution.
type StaffRow struct {
	ID             string
	DisplayName    string
	Email          string
	RoleLabel      string
	SupabaseUserID string
	Status         string // active|disabled
}

// StaffCapabilityResolver derives capabilities from the canonical active
// staff row. A verified Supabase user without an active staff row gets no
// capabilities. A disabled staff row gets no capabilities even if the
// identity-provider session is still valid.
type StaffCapabilityResolver struct {
	lookup StaffLookup
}

func NewStaffCapabilityResolver(lookup StaffLookup) StaffCapabilityResolver {
	return StaffCapabilityResolver{lookup: lookup}
}

func (r StaffCapabilityResolver) Resolve(ctx context.Context, p Principal) (Principal, error) {
	// If the verifier already provided capabilities (dev mode), keep them.
	if len(p.Capabilities) > 0 {
		return p, nil
	}

	// Link exclusively via supabase_user_id. Email matching is NOT a valid
	// linking mechanism — an attacker who controls a Supabase account with
	// the same email as a staff row would gain staff capabilities. Staff
	// rows must be explicitly linked via supabase_user_id by an owner.
	row, err := r.lookup.GetBySupabaseUserID(ctx, p.UserID)
	if err != nil {
		// No linked staff row: valid Supabase user, no admin capabilities.
		p.Role = "user"
		p.Capabilities = nil
		return p, nil
	}

	if row.Status != "active" {
		// Disabled staff: fail closed.
		p.Role = "disabled"
		p.StaffID = row.ID
		p.Capabilities = nil
		return p, nil
	}

	p.StaffID = row.ID
	p.Role = row.RoleLabel
	p.Capabilities = capabilitiesForRole(row.RoleLabel)
	return p, nil
}

// capabilitiesForRole returns the canonical capability set for a given
// role_label. These are the only capabilities the server grants; the admin
// UI must not add capabilities the server did not provide.
func capabilitiesForRole(roleLabel string) []string {
	switch roleLabel {
	case "owner":
		return []string{
			"twcommerce.read",
			"twcommerce.create",
			"twcommerce.update",
			"twcommerce.delete",
			"twcommerce.admin",
			"orders.returns",
			"inventory.adjust",
			"content.read",
			"content.create",
			"content.update",
			"content.approve",
			"content.publish",
			"staff.read",
			"staff.update",
			"media.upload",
		}
	case "manager":
		return []string{
			"twcommerce.read",
			"twcommerce.create",
			"twcommerce.update",
			"twcommerce.delete",
			"twcommerce.admin",
			"orders.returns",
			"inventory.adjust",
			"content.read",
			"content.create",
			"content.update",
			"content.approve",
			"content.publish",
			"staff.read",
			"media.upload",
		}
	case "readonly":
		return []string{
			"twcommerce.read",
			"content.read",
			"staff.read",
		}
	default:
		return nil
	}
}
