package staff

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

var (
	ErrForbidden     = errors.New("forbidden")
	ErrLastOwner     = errors.New("cannot remove the last active owner")
	ErrSelfElevation = errors.New("cannot change your own role or status")
)

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

func (s Service) List(ctx context.Context) ([]StaffMember, error) {
	return s.store.List(ctx)
}

func (s Service) ListActive(ctx context.Context) ([]StaffMember, error) {
	return s.store.ListActive(ctx)
}

func (s Service) GetByID(ctx context.Context, id string) (StaffMember, error) {
	return s.store.GetByID(ctx, id)
}

func (s Service) Create(ctx context.Context, principal auth.Principal, in StaffMemberInput) (StaffMember, error) {
	if !auth.Can(principal, "staff.update") {
		return StaffMember{}, ErrForbidden
	}
	in = normalizeInput(in)
	if in.DisplayName == "" {
		return StaffMember{}, errors.New("display name is required")
	}
	if in.Email == "" {
		return StaffMember{}, errors.New("email is required")
	}
	if in.RoleLabel == "" {
		in.RoleLabel = "readonly"
	}
	if !validRoleLabel(in.RoleLabel) {
		return StaffMember{}, fmt.Errorf("invalid role_label %q", in.RoleLabel)
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "disabled" {
		return StaffMember{}, fmt.Errorf("invalid status %q", status)
	}

	id, err := randomID()
	if err != nil {
		return StaffMember{}, err
	}
	sm := StaffMember{
		ID:             id,
		DisplayName:    in.DisplayName,
		Email:          in.Email,
		RoleLabel:      in.RoleLabel,
		SupabaseUserID: in.SupabaseUserID,
		Status:         status,
		UpdatedUnix:    time.Now().Unix(),
	}
	if err := s.store.Upsert(ctx, sm); err != nil {
		return StaffMember{}, err
	}
	return sm, nil
}

func (s Service) Update(ctx context.Context, principal auth.Principal, id string, in StaffMemberInput) (StaffMember, error) {
	if !auth.Can(principal, "staff.update") {
		return StaffMember{}, ErrForbidden
	}
	// Anti-self-elevation: a principal cannot change their own role or status.
	if principal.UserID != "" && id == principal.StaffID {
		return StaffMember{}, ErrSelfElevation
	}
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		return StaffMember{}, err
	}
	in = normalizeInput(in)
	if in.DisplayName == "" {
		in.DisplayName = existing.DisplayName
	}
	if in.Email == "" {
		in.Email = existing.Email
	}
	if in.RoleLabel == "" {
		in.RoleLabel = existing.RoleLabel
	}
	if !validRoleLabel(in.RoleLabel) {
		return StaffMember{}, fmt.Errorf("invalid role_label %q", in.RoleLabel)
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = existing.Status
	}
	if status != "active" && status != "disabled" {
		return StaffMember{}, fmt.Errorf("invalid status %q", status)
	}

	// Last-owner protection: cannot demote or disable the last active
	// owner. This is enforced atomically via UpsertGuarded, which uses a
	// transaction that locks all active owner rows (FOR UPDATE on
	// PostgreSQL, serialized on SQLite) and checks the count inside the
	// transaction. This prevents the cross-row race where two concurrent
	// demotes on different owner rows both see count=2 and both proceed.
	sm := StaffMember{
		ID:             existing.ID,
		DisplayName:    in.DisplayName,
		Email:          in.Email,
		RoleLabel:      in.RoleLabel,
		SupabaseUserID: defaultString(in.SupabaseUserID, existing.SupabaseUserID),
		Status:         status,
		UpdatedUnix:    time.Now().Unix(),
	}
	if err := s.store.UpsertGuarded(ctx, sm); err != nil {
		if errors.Is(err, ErrGuardedBlocked) {
			return StaffMember{}, ErrLastOwner
		}
		return StaffMember{}, err
	}
	return sm, nil
}

func (s Service) UpdateStatus(ctx context.Context, principal auth.Principal, id, status string) (StaffMember, error) {
	if !auth.Can(principal, "staff.update") {
		return StaffMember{}, ErrForbidden
	}
	if principal.UserID != "" && id == principal.StaffID {
		return StaffMember{}, ErrSelfElevation
	}
	if status != "active" && status != "disabled" {
		return StaffMember{}, fmt.Errorf("invalid status %q", status)
	}
	// Verify the row exists first (for a clean ErrNotFound).
	if _, err := s.store.GetByID(ctx, id); err != nil {
		return StaffMember{}, err
	}
	// Last-owner protection: cannot disable the last active owner.
	// Enforced atomically via UpdateStatusGuarded, which uses a
	// transaction that locks all active owner rows and checks the count
	// inside the transaction.
	if err := s.store.UpdateStatusGuarded(ctx, id, status, time.Now().Unix()); err != nil {
		if errors.Is(err, ErrGuardedBlocked) {
			return StaffMember{}, ErrLastOwner
		}
		return StaffMember{}, err
	}
	return s.store.GetByID(ctx, id)
}

func (s Service) Delete(ctx context.Context, principal auth.Principal, id string) error {
	if !auth.Can(principal, "staff.update") {
		return ErrForbidden
	}
	if principal.UserID != "" && id == principal.StaffID {
		return ErrSelfElevation
	}
	// Verify the row exists first (for a clean ErrNotFound).
	if _, err := s.store.GetByID(ctx, id); err != nil {
		return err
	}
	// Last-owner protection: cannot delete the last active owner.
	// Enforced atomically via DeleteGuarded, which uses a transaction
	// that locks all active owner rows and checks the count inside the
	// transaction.
	if err := s.store.DeleteGuarded(ctx, id); err != nil {
		if errors.Is(err, ErrGuardedBlocked) {
			return ErrLastOwner
		}
		return err
	}
	return nil
}

func validRoleLabel(label string) bool {
	return label == "owner" || label == "manager" || label == "readonly"
}

func normalizeInput(in StaffMemberInput) StaffMemberInput {
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Email = strings.TrimSpace(in.Email)
	in.RoleLabel = strings.TrimSpace(in.RoleLabel)
	in.SupabaseUserID = strings.TrimSpace(in.SupabaseUserID)
	in.Status = strings.TrimSpace(in.Status)
	return in
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
