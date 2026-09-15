package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

// Promo is a discount code applicable at checkout. Type is percent,
// fixed, or freeshipping. MinSubtotal gates eligibility; UsageLimit caps
// total redemptions (nil = unlimited); UsedCount is incremented when an
// order is created with the code.
type Promo struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Label       string `json:"label"`
	Type        string `json:"type"` // percent|fixed|freeshipping
	Value       int    `json:"value"`
	Enabled     bool   `json:"enabled"`
	MinSubtotal int    `json:"min_subtotal"`
	UsageLimit  *int   `json:"usage_limit"`
	UsedCount   int    `json:"used_count"`
	StartsUnix  int64  `json:"starts_unix"`
	ExpiresUnix int64  `json:"expires_unix"`
	UpdatedUnix int64  `json:"updated_unix"`
}

// PromoInput is the browser-supplied payload for promo create/update.
// On update, absent fields preserve the existing row; an explicit null
// on usage_limit clears the cap (presentFields.set).
type PromoInput struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Value       int    `json:"value"`
	Enabled     bool   `json:"enabled"`
	MinSubtotal int    `json:"min_subtotal"`
	UsageLimit  *int   `json:"usage_limit"`
	StartsUnix  int64  `json:"starts_unix"`
	ExpiresUnix int64  `json:"expires_unix"`

	presentFields
}

// UnmarshalJSON decodes with unknown-field rejection and records
// top-level key presence for partial-update merge.
func (in *PromoInput) UnmarshalJSON(data []byte) error {
	type plain PromoInput
	present, err := decodeStrict(data, (*plain)(in))
	if err != nil {
		return err
	}
	in.present = present
	return nil
}

// NotificationLogFilter narrows notification log listings for the admin
// view. OrderID/Code/Status empty means unfiltered.
type NotificationLogFilter struct {
	OrderID string
	Code    string
	Status  string
}

// ----- Promos ---------------------------------------------------------------

func (s Service) ListPromos(ctx context.Context) ([]Promo, error) {
	return s.store.ListPromos(ctx)
}

func (s Service) CreatePromo(ctx context.Context, principal auth.Principal, in PromoInput) (Promo, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return Promo{}, ErrForbidden
	}
	if strings.TrimSpace(in.Code) == "" {
		return Promo{}, fmt.Errorf("%w: promo code is required", ErrInvalidAdminInput)
	}
	if err := validatePromoInput(in); err != nil {
		return Promo{}, err
	}
	id, err := randomID()
	if err != nil {
		return Promo{}, err
	}
	p := Promo{
		ID:          id,
		Code:        strings.TrimSpace(in.Code),
		Label:       in.Label,
		Type:        in.Type,
		Value:       in.Value,
		Enabled:     in.Enabled,
		MinSubtotal: in.MinSubtotal,
		UsageLimit:  in.UsageLimit,
		StartsUnix:  in.StartsUnix,
		ExpiresUnix: in.ExpiresUnix,
		UpdatedUnix: time.Now().Unix(),
	}
	if err := s.store.UpsertPromo(ctx, p); err != nil {
		return Promo{}, err
	}
	return p, nil
}

func (s Service) UpdatePromo(ctx context.Context, principal auth.Principal, id string, in PromoInput) (Promo, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return Promo{}, ErrForbidden
	}
	existing, err := s.store.ListPromos(ctx)
	if err != nil {
		return Promo{}, err
	}
	var found *Promo
	for i := range existing {
		if existing[i].ID == id {
			found = &existing[i]
			break
		}
	}
	if found == nil {
		return Promo{}, ErrNotFound
	}
	// Merge semantics: fields absent from the JSON body preserve the
	// existing row — a partial update must not silently zero value,
	// disable the promo, or drop its usage limit. usage_limit uses set()
	// so an explicit null still clears the cap. Validation runs on the
	// merged result so an omitted type keeps the existing one.
	merged := PromoInput{
		Code:        defaultString(strings.TrimSpace(in.Code), found.Code),
		Label:       found.Label,
		Type:        found.Type,
		Value:       found.Value,
		Enabled:     found.Enabled,
		MinSubtotal: found.MinSubtotal,
		UsageLimit:  found.UsageLimit,
		StartsUnix:  found.StartsUnix,
		ExpiresUnix: found.ExpiresUnix,
	}
	if in.has("label") {
		merged.Label = in.Label
	}
	if in.has("type") {
		merged.Type = in.Type
	}
	if in.has("value") {
		merged.Value = in.Value
	}
	if in.has("enabled") {
		merged.Enabled = in.Enabled
	}
	if in.has("min_subtotal") {
		merged.MinSubtotal = in.MinSubtotal
	}
	if in.set("usage_limit") {
		merged.UsageLimit = in.UsageLimit
	}
	if in.has("starts_unix") {
		merged.StartsUnix = in.StartsUnix
	}
	if in.has("expires_unix") {
		merged.ExpiresUnix = in.ExpiresUnix
	}
	if err := validatePromoInput(merged); err != nil {
		return Promo{}, err
	}
	p := Promo{
		ID:          id,
		Code:        merged.Code,
		Label:       merged.Label,
		Type:        merged.Type,
		Value:       merged.Value,
		Enabled:     merged.Enabled,
		MinSubtotal: merged.MinSubtotal,
		UsageLimit:  merged.UsageLimit,
		UsedCount:   found.UsedCount,
		StartsUnix:  merged.StartsUnix,
		ExpiresUnix: merged.ExpiresUnix,
		UpdatedUnix: time.Now().Unix(),
	}
	if err := s.store.UpsertPromo(ctx, p); err != nil {
		return Promo{}, err
	}
	return p, nil
}

// validatePromoInput enforces the promo domain rules shared by create and
// update. freeshipping promos carry no discount value; percent values are
// bounded at 100.
func validatePromoInput(in PromoInput) error {
	switch in.Type {
	case "percent":
		if in.Value < 0 || in.Value > 100 {
			return fmt.Errorf("%w: percent promo value must be 0-100", ErrInvalidAdminInput)
		}
	case "fixed", "freeshipping":
		if in.Value < 0 {
			return fmt.Errorf("%w: promo value must be non-negative", ErrInvalidAdminInput)
		}
	default:
		return fmt.Errorf("%w: promo type must be percent, fixed, or freeshipping", ErrInvalidAdminInput)
	}
	if in.MinSubtotal < 0 {
		return fmt.Errorf("%w: promo min_subtotal must be non-negative", ErrInvalidAdminInput)
	}
	if in.UsageLimit != nil && *in.UsageLimit <= 0 {
		return fmt.Errorf("%w: promo usage_limit must be positive when set", ErrInvalidAdminInput)
	}
	return nil
}

func (s Service) DeletePromo(ctx context.Context, principal auth.Principal, id string) error {
	if !auth.Can(principal, "twcommerce.admin") {
		return ErrForbidden
	}
	return s.store.DeletePromo(ctx, id)
}

// resolvePromo loads an active promo by code, enforcing the eligibility
// rules beyond the time window: minimum subtotal and usage limit. A promo
// that is unknown, disabled, outside its window, below min_subtotal, or
// exhausted fails closed with ErrInvalidPromoCode — the failure reason is
// never enumerated to the public browser.
func (s Service) resolvePromo(ctx context.Context, subtotal int, promoCode string) (Promo, error) {
	promoCode = strings.TrimSpace(promoCode)
	if promoCode == "" {
		return Promo{}, ErrInvalidPromoCode
	}
	p, err := s.store.GetActivePromoByCode(ctx, promoCode, time.Now().Unix())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Promo{}, ErrInvalidPromoCode
		}
		return Promo{}, fmt.Errorf("%w: %v", ErrPromoValidationUnavailable, err)
	}
	if p.MinSubtotal > 0 && subtotal < p.MinSubtotal {
		return Promo{}, ErrInvalidPromoCode
	}
	if p.UsageLimit != nil && p.UsedCount >= *p.UsageLimit {
		return Promo{}, ErrInvalidPromoCode
	}
	return p, nil
}

// calculateDiscount applies an active promo code to the subtotal. Empty means
// no promo. Any other code that is not currently eligible fails closed with
// ErrInvalidPromoCode. Store errors are propagated rather than treated as a
// zero discount. Returns ErrOverflow on arithmetic overflow. A freeshipping
// promo yields zero discount — the shipping effect is applied separately.
func (s Service) calculateDiscount(ctx context.Context, subtotal int, promoCode string) (int, error) {
	promoCode = strings.TrimSpace(promoCode)
	if promoCode == "" {
		return 0, nil
	}
	p, err := s.resolvePromo(ctx, subtotal, promoCode)
	if err != nil {
		return 0, err
	}
	var discount int
	switch p.Type {
	case "percent":
		product, err := mulOverflow(subtotal, p.Value)
		if err != nil {
			return 0, fmt.Errorf("percent discount overflow: %w", err)
		}
		discount = product / 100
	case "fixed":
		discount = p.Value
	case "freeshipping":
		discount = 0
	default:
		return 0, ErrInvalidPromoCode
	}
	if discount > subtotal {
		discount = subtotal
	}
	if discount < 0 {
		discount = 0
	}
	return discount, nil
}

// PreviewDiscount exposes the same discount calculation used at checkout
// for display-only previews (coupon validation on the cart page).
func (s Service) PreviewDiscount(ctx context.Context, subtotal int, promoCode string) (int, error) {
	return s.calculateDiscount(ctx, subtotal, promoCode)
}

// promoFreeShipping reports whether the given code resolves to a
// freeshipping promo at the given subtotal. Unknown/ineligible codes
// return false (the error was already surfaced by calculateDiscount).
func (s Service) promoFreeShipping(ctx context.Context, subtotal int, promoCode string) bool {
	promoCode = strings.TrimSpace(promoCode)
	if promoCode == "" {
		return false
	}
	p, err := s.resolvePromo(ctx, subtotal, promoCode)
	return err == nil && p.Type == "freeshipping"
}

// ValidateCoupon is the public coupon check used by the storefront cart
// drawer. It returns the promo's discount semantics for display when the
// code is currently eligible against the supplied subtotal. Empty codes
// and ineligible codes return ErrInvalidPromoCode.
func (s Service) ValidateCoupon(ctx context.Context, code string, subtotal int) (Promo, error) {
	if subtotal < 0 {
		return Promo{}, ErrInvalidPromoCode
	}
	return s.resolvePromo(ctx, subtotal, code)
}
