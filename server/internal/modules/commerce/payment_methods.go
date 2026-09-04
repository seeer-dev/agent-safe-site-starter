package commerce

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

// PaymentMethod describes a configured payment provider integration.
type PaymentMethod struct {
	ID              string `json:"id"`
	Method          string `json:"method"`
	ProviderLabel   string `json:"provider_label"`
	Environment     string `json:"environment"`      // production|sandbox
	ReadinessStatus string `json:"readiness_status"` // ready|pending_setup
	Enabled         bool   `json:"enabled"`
	UpdatedUnix     int64  `json:"updated_unix"`
}

// PaymentMethodInput is the browser-supplied payload for payment method updates.
type PaymentMethodInput struct {
	Method          string `json:"method"`
	ProviderLabel   string `json:"provider_label"`
	Environment     string `json:"environment"`
	ReadinessStatus string `json:"readiness_status"`
	Enabled         bool   `json:"enabled"`
}

// PublicPaymentMethod is the public-facing payment method descriptor.
// It is derived from the admin-managed payment_methods table. Only enabled
// methods with readiness_status="ready" are available to customers.
type PublicPaymentMethod struct {
	ID        string `json:"id"`
	Method    string `json:"method"`
	Label     string `json:"label"`
	Available bool   `json:"available"`
}

// paymentMethodRuntimeAvailable applies the runtime half of payment-method
// readiness. Most starter payment methods are configuration-only examples and
// need no additional runtime adapter. ECPay is different: a database row can
// be administratively marked ready while the server has no signing secrets or
// is configured for the other provider environment. In those cases the server
// must fail closed rather than advertise a checkout option that can only end in
// a 503 at payment launch.
func (s Service) paymentMethodRuntimeAvailable(m PaymentMethod) bool {
	if !m.Enabled || m.ReadinessStatus != "ready" {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(m.Method), "ecpay") {
		return true
	}
	if s.ecpay == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s.ecpay.Environment)) {
	case "stage":
		return strings.EqualFold(strings.TrimSpace(m.Environment), "sandbox")
	case "production":
		return strings.EqualFold(strings.TrimSpace(m.Environment), "production")
	default:
		return false
	}
}

// ListPublicPaymentMethods returns the admin-managed payment methods that
// are enabled and ready for customer use. Methods that are disabled, not yet
// set up, or missing their required runtime provider configuration are excluded.
func (s Service) ListPublicPaymentMethods(ctx context.Context) ([]PublicPaymentMethod, error) {
	methods, err := s.store.ListPaymentMethods(ctx)
	if err != nil {
		return nil, err
	}
	var out []PublicPaymentMethod
	for _, m := range methods {
		if !s.paymentMethodRuntimeAvailable(m) {
			continue
		}
		out = append(out, PublicPaymentMethod{
			ID:        m.ID,
			Method:    m.Method,
			Label:     m.ProviderLabel,
			Available: true,
		})
	}
	return out, nil
}

// validatePaymentMethod validates the payment method against the
// admin-managed payment_methods table and the runtime provider state. It
// rejects empty, unknown, disabled, not-ready, or runtime-unavailable methods
// with ErrInvalidPaymentMethod. The server is the authority for payment method
// availability — the browser must not decide which payment methods are usable.
func (s Service) validatePaymentMethod(ctx context.Context, methodID string) error {
	methodID = strings.TrimSpace(methodID)
	if methodID == "" {
		return ErrInvalidPaymentMethod
	}
	methods, err := s.store.ListPaymentMethods(ctx)
	if err != nil {
		return fmt.Errorf("validate payment method: %w", err)
	}
	for _, m := range methods {
		if m.ID == methodID || m.Method == methodID {
			if !s.paymentMethodRuntimeAvailable(m) {
				return ErrInvalidPaymentMethod
			}
			return nil
		}
	}
	return ErrInvalidPaymentMethod
}

// ----- Payment methods ------------------------------------------------------

func (s Service) ListPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	return s.store.ListPaymentMethods(ctx)
}

func (s Service) UpdatePaymentMethod(ctx context.Context, principal auth.Principal, id string, in PaymentMethodInput) (PaymentMethod, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return PaymentMethod{}, ErrForbidden
	}
	if in.Environment != "production" && in.Environment != "sandbox" {
		return PaymentMethod{}, fmt.Errorf("%w: environment must be production or sandbox", ErrInvalidAdminInput)
	}
	if in.ReadinessStatus != "ready" && in.ReadinessStatus != "pending_setup" {
		return PaymentMethod{}, fmt.Errorf("%w: readiness_status must be ready or pending_setup", ErrInvalidAdminInput)
	}
	pm := PaymentMethod{
		ID:              id,
		Method:          strings.TrimSpace(in.Method),
		ProviderLabel:   in.ProviderLabel,
		Environment:     in.Environment,
		ReadinessStatus: in.ReadinessStatus,
		Enabled:         in.Enabled,
		UpdatedUnix:     time.Now().Unix(),
	}
	if err := s.store.UpsertPaymentMethod(ctx, pm); err != nil {
		return PaymentMethod{}, err
	}
	return pm, nil
}
