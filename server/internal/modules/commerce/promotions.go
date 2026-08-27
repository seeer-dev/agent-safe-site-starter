package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

// Promo is a discount code applicable at checkout.
type Promo struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Label       string `json:"label"`
	Type        string `json:"type"` // percent|fixed
	Value       int    `json:"value"`
	Enabled     bool   `json:"enabled"`
	StartsUnix  int64  `json:"starts_unix"`
	ExpiresUnix int64  `json:"expires_unix"`
	UpdatedUnix int64  `json:"updated_unix"`
}

// PromoInput is the browser-supplied payload for promo create/update.
type PromoInput struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Value       int    `json:"value"`
	Enabled     bool   `json:"enabled"`
	StartsUnix  int64  `json:"starts_unix"`
	ExpiresUnix int64  `json:"expires_unix"`
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
	if in.Type != "percent" && in.Type != "fixed" {
		return Promo{}, fmt.Errorf("%w: promo type must be percent or fixed", ErrInvalidAdminInput)
	}
	if in.Value < 0 {
		return Promo{}, fmt.Errorf("%w: promo value must be non-negative", ErrInvalidAdminInput)
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
	if in.Type != "percent" && in.Type != "fixed" {
		return Promo{}, fmt.Errorf("%w: promo type must be percent or fixed", ErrInvalidAdminInput)
	}
	if in.Value < 0 {
		return Promo{}, fmt.Errorf("%w: promo value must be non-negative", ErrInvalidAdminInput)
	}
	p := Promo{
		ID:          id,
		Code:        defaultString(strings.TrimSpace(in.Code), found.Code),
		Label:       in.Label,
		Type:        in.Type,
		Value:       in.Value,
		Enabled:     in.Enabled,
		StartsUnix:  in.StartsUnix,
		ExpiresUnix: in.ExpiresUnix,
		UpdatedUnix: time.Now().Unix(),
	}
	if err := s.store.UpsertPromo(ctx, p); err != nil {
		return Promo{}, err
	}
	return p, nil
}

func (s Service) DeletePromo(ctx context.Context, principal auth.Principal, id string) error {
	if !auth.Can(principal, "twcommerce.admin") {
		return ErrForbidden
	}
	return s.store.DeletePromo(ctx, id)
}

// calculateDiscount applies an active promo code to the subtotal. Empty means
// no promo. Any other code that is not currently active fails closed with
// ErrInvalidPromoCode. Store errors are propagated rather than treated as a
// zero discount. Returns ErrOverflow on arithmetic overflow.
func (s Service) calculateDiscount(ctx context.Context, subtotal int, promoCode string) (int, error) {
	promoCode = strings.TrimSpace(promoCode)
	if promoCode == "" {
		return 0, nil
	}
	p, err := s.store.GetActivePromoByCode(ctx, promoCode, time.Now().Unix())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, ErrInvalidPromoCode
		}
		return 0, fmt.Errorf("%w: %v", ErrPromoValidationUnavailable, err)
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
