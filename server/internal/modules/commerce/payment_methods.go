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
	Fee             int    `json:"fee"`
	UpdatedUnix     int64  `json:"updated_unix"`
}

// PaymentMethodInput is the browser-supplied payload for payment method
// updates. On update of an existing row, absent fields preserve the
// stored values (presentFields).
type PaymentMethodInput struct {
	Method          string `json:"method"`
	ProviderLabel   string `json:"provider_label"`
	Environment     string `json:"environment"`
	ReadinessStatus string `json:"readiness_status"`
	Enabled         bool   `json:"enabled"`
	Fee             int    `json:"fee"`

	presentFields
}

// UnmarshalJSON decodes with unknown-field rejection and records
// top-level key presence for partial-update merge.
func (in *PaymentMethodInput) UnmarshalJSON(data []byte) error {
	type plain PaymentMethodInput
	present, err := decodeStrict(data, (*plain)(in))
	if err != nil {
		return err
	}
	in.present = present
	return nil
}

// PublicPaymentMethod is the public-facing payment method descriptor.
// It is derived from the admin-managed payment_methods table. Only enabled
// methods with readiness_status="ready" are available to customers.
type PublicPaymentMethod struct {
	ID        string `json:"id"`
	Method    string `json:"method"`
	Label     string `json:"label"`
	Available bool   `json:"available"`
	Fee       int    `json:"fee"`
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
	// manual_test is a test-only method: an enabled+ready row is necessary
	// but never sufficient — the server-only activation window must also be
	// open. This is checked here so public listing, quote, and both order
	// paths share one decision and no endpoint can bypass it.
	if strings.EqualFold(strings.TrimSpace(m.Method), "manual_test") {
		return s.manualTestAvailable()
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
			Fee:       m.Fee,
		})
	}
	return out, nil
}

// resolvePaymentMethod validates the payment method against the
// admin-managed payment_methods table and the runtime provider state. It
// rejects empty, unknown, disabled, not-ready, or runtime-unavailable
// methods with ErrInvalidPaymentMethod. The server is the authority for
// payment method availability — the browser must not decide which payment
// methods are usable. On success it returns the method row so callers can
// derive the server-side fee.
func (s Service) resolvePaymentMethod(ctx context.Context, methodID string) (PaymentMethod, error) {
	methodID = strings.TrimSpace(methodID)
	if methodID == "" {
		return PaymentMethod{}, ErrInvalidPaymentMethod
	}
	methods, err := s.store.ListPaymentMethods(ctx)
	if err != nil {
		return PaymentMethod{}, fmt.Errorf("resolve payment method: %w", err)
	}
	for _, m := range methods {
		if m.ID == methodID || m.Method == methodID {
			if !s.paymentMethodRuntimeAvailable(m) {
				return PaymentMethod{}, ErrInvalidPaymentMethod
			}
			return m, nil
		}
	}
	return PaymentMethod{}, ErrInvalidPaymentMethod
}

// validatePaymentMethod keeps the original signature for existing callers
// that only need availability validation.
func (s Service) validatePaymentMethod(ctx context.Context, methodID string) error {
	_, err := s.resolvePaymentMethod(ctx, methodID)
	return err
}

// ----- Payment methods ------------------------------------------------------

func (s Service) ListPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	return s.store.ListPaymentMethods(ctx)
}

func (s Service) UpdatePaymentMethod(ctx context.Context, principal auth.Principal, id string, in PaymentMethodInput) (PaymentMethod, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return PaymentMethod{}, ErrForbidden
	}
	// Merge semantics on an existing row: absent fields preserve stored
	// values — a partial body must not silently blank the provider label,
	// disable the method, or zero its fee. An unknown id keeps the
	// historical PUT-upsert behavior: the body applies as-is.
	merged := in
	methods, err := s.store.ListPaymentMethods(ctx)
	if err != nil {
		return PaymentMethod{}, err
	}
	for i := range methods {
		if methods[i].ID != id {
			continue
		}
		ex := methods[i]
		if !in.has("method") {
			merged.Method = ex.Method
		}
		if !in.has("provider_label") {
			merged.ProviderLabel = ex.ProviderLabel
		}
		if !in.has("environment") {
			merged.Environment = ex.Environment
		}
		if !in.has("readiness_status") {
			merged.ReadinessStatus = ex.ReadinessStatus
		}
		if !in.has("enabled") {
			merged.Enabled = ex.Enabled
		}
		if !in.has("fee") {
			merged.Fee = ex.Fee
		}
		break
	}
	if merged.Environment != "production" && merged.Environment != "sandbox" {
		return PaymentMethod{}, fmt.Errorf("%w: environment must be production or sandbox", ErrInvalidAdminInput)
	}
	if merged.ReadinessStatus != "ready" && merged.ReadinessStatus != "pending_setup" {
		return PaymentMethod{}, fmt.Errorf("%w: readiness_status must be ready or pending_setup", ErrInvalidAdminInput)
	}
	if merged.Fee < 0 {
		return PaymentMethod{}, fmt.Errorf("%w: fee must be non-negative", ErrInvalidAdminInput)
	}
	pm := PaymentMethod{
		ID:              id,
		Method:          strings.TrimSpace(merged.Method),
		ProviderLabel:   merged.ProviderLabel,
		Environment:     merged.Environment,
		ReadinessStatus: merged.ReadinessStatus,
		Enabled:         merged.Enabled,
		Fee:             merged.Fee,
		UpdatedUnix:     time.Now().Unix(),
	}
	if err := s.store.UpsertPaymentMethod(ctx, pm); err != nil {
		return PaymentMethod{}, err
	}
	return pm, nil
}
