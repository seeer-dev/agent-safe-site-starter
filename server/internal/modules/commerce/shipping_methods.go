package commerce

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/example/ai-site-starter/server/internal/auth"
)

const (
	shippingMethodKeyMax   = 64
	shippingMethodLabelMax = 80
	shippingMethodDescMax  = 500
)

func validateShippingMethodKey(method string) error {
	if method != strings.TrimSpace(method) {
		return fmt.Errorf("%w: method contains leading or trailing whitespace", ErrInvalidAdminInput)
	}
	if method == "" {
		return fmt.Errorf("%w: method is required", ErrInvalidAdminInput)
	}
	if len(method) > shippingMethodKeyMax {
		return fmt.Errorf("%w: method must be at most %d characters", ErrInvalidAdminInput, shippingMethodKeyMax)
	}
	if strings.ContainsAny(method, " \t\n\r/\\.") {
		return fmt.Errorf("%w: method contains whitespace or path characters", ErrInvalidAdminInput)
	}
	for _, r := range method {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("%w: method %q contains invalid character %q", ErrInvalidAdminInput, method, string(r))
	}
	return nil
}

func validateShippingMethodInput(in ShippingMethodInput) error {
	if err := validateShippingMethodKey(in.Method); err != nil {
		return err
	}
	label := strings.TrimSpace(in.Label)
	if label == "" {
		return fmt.Errorf("%w: label is required", ErrInvalidAdminInput)
	}
	if utf8.RuneCountInString(label) > shippingMethodLabelMax {
		return fmt.Errorf("%w: label must be at most %d characters", ErrInvalidAdminInput, shippingMethodLabelMax)
	}
	if utf8.RuneCountInString(in.Description) > shippingMethodDescMax {
		return fmt.Errorf("%w: description must be at most %d characters", ErrInvalidAdminInput, shippingMethodDescMax)
	}
	if in.Fee < 0 {
		return fmt.Errorf("%w: fee must be non-negative", ErrInvalidAdminInput)
	}
	if in.FreeThreshold != nil && *in.FreeThreshold <= 0 {
		return fmt.Errorf("%w: free_threshold must be null or greater than 0", ErrInvalidAdminInput)
	}
	if in.SortOrder < 0 {
		return fmt.Errorf("%w: sort_order must be non-negative", ErrInvalidAdminInput)
	}
	return nil
}

func (s Service) ListShippingMethods(ctx context.Context, principal auth.Principal) ([]ShippingMethod, error) {
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		return nil, ErrForbidden
	}
	methods, err := s.store.ListShippingMethods(ctx)
	if err != nil {
		return nil, err
	}
	if methods == nil {
		return []ShippingMethod{}, nil
	}
	return methods, nil
}

func (s Service) CreateShippingMethod(ctx context.Context, principal auth.Principal, in ShippingMethodInput) (ShippingMethod, error) {
	if !auth.Can(principal, "twcommerce.create") {
		return ShippingMethod{}, ErrForbidden
	}
	in.Label = strings.TrimSpace(in.Label)
	if err := validateShippingMethodInput(in); err != nil {
		return ShippingMethod{}, err
	}
	id, err := randomID()
	if err != nil {
		return ShippingMethod{}, err
	}
	m := ShippingMethod{
		ID:            id,
		Method:        in.Method,
		Label:         in.Label,
		Description:   in.Description,
		Fee:           in.Fee,
		FreeThreshold: in.FreeThreshold,
		Enabled:       in.Enabled,
		SortOrder:     in.SortOrder,
		Version:       1,
		UpdatedUnix:   time.Now().Unix(),
	}
	if err := s.store.CreateShippingMethod(ctx, m); err != nil {
		return ShippingMethod{}, err
	}
	return m, nil
}

func (s Service) UpdateShippingMethod(ctx context.Context, principal auth.Principal, id string, in ShippingMethodUpdateInput) (ShippingMethod, error) {
	if !auth.Can(principal, "twcommerce.update") {
		return ShippingMethod{}, ErrForbidden
	}
	if strings.TrimSpace(id) == "" {
		return ShippingMethod{}, ErrNotFound
	}
	if in.ExpectedVersion < 1 {
		return ShippingMethod{}, fmt.Errorf("%w: expected_version is required", ErrInvalidAdminInput)
	}
	existing, err := s.store.GetShippingMethod(ctx, id)
	if err != nil {
		return ShippingMethod{}, err
	}
	// Method is immutable. Validate against the stored key so a tampered
	// payload cannot rename the row, and so label/fee checks still run.
	validated := ShippingMethodInput{
		Method:        existing.Method,
		Label:         strings.TrimSpace(in.Label),
		Description:   in.Description,
		Fee:           in.Fee,
		FreeThreshold: in.FreeThreshold,
		Enabled:       in.Enabled,
		SortOrder:     in.SortOrder,
	}
	if err := validateShippingMethodInput(validated); err != nil {
		return ShippingMethod{}, err
	}
	m := ShippingMethod{
		ID:            existing.ID,
		Method:        existing.Method,
		Label:         validated.Label,
		Description:   validated.Description,
		Fee:           validated.Fee,
		FreeThreshold: validated.FreeThreshold,
		Enabled:       validated.Enabled,
		SortOrder:     validated.SortOrder,
		UpdatedUnix:   time.Now().Unix(),
	}
	if err := s.store.UpdateShippingMethod(ctx, m, in.ExpectedVersion); err != nil {
		return ShippingMethod{}, err
	}
	return s.store.GetShippingMethod(ctx, id)
}
