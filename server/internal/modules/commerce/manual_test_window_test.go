package commerce

import (
	"context"
	"errors"
	"testing"
	"time"
)

// seedManualTestMethod inserts an enabled+ready manual_test row — the
// database-side precondition that is necessary but never sufficient.
func seedManualTestMethod(t *testing.T, s Store) {
	t.Helper()
	if err := s.UpsertPaymentMethod(context.Background(), PaymentMethod{
		ID:              "mt",
		Method:          "manual_test",
		ProviderLabel:   "Manual test",
		Environment:     "sandbox",
		ReadinessStatus: "ready",
		Enabled:         true,
		UpdatedUnix:     1,
	}); err != nil {
		t.Fatalf("seed manual_test method: %v", err)
	}
}

func openWindow(svc Service, base time.Time) Service {
	return svc.WithManualTestWindow(base.Add(time.Hour), 24*time.Hour, func() time.Time { return base })
}

func manualTestListed(t *testing.T, svc Service) bool {
	t.Helper()
	methods, err := svc.ListPublicPaymentMethods(context.Background())
	if err != nil {
		t.Fatalf("list payment methods: %v", err)
	}
	for _, m := range methods {
		if m.Method == "manual_test" {
			return true
		}
	}
	return false
}

// The row is enabled and ready, yet a zero deadline (unset or malformed
// config) must hide manual_test everywhere.
func TestManualTestHiddenWithoutWindow(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedManualTestMethod(t, store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-MT", Name: "MT", Slug: "mt", Status: "active", Stock: 5, Price: 100})
	seedDefaultShippingMethods(t, store)
	svc := NewService(store)

	if manualTestListed(t, svc) {
		t.Fatal("manual_test listed with no activation window configured")
	}
	// The order path must fail at the payment gate, not somewhere earlier —
	// seed the product so item resolution succeeds first.
	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "A", Email: "a@b.c", Phone: "0912345678", ShippingAddress: "Taipei",
		Items:          []OrderItem{{SKU: "SKU-MT", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "manual_test", IdempotencyKey: "mt-none",
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("order without window: got %v, want ErrInvalidPaymentMethod", err)
	}
	// Quote shares the same predicate.
	_, err = svc.Quote(context.Background(), QuoteInput{
		Items: []OrderItem{{SKU: "SKU-MT", Quantity: 1}}, ShippingMethod: "home_delivery", PaymentMethod: "manual_test",
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("quote without window: got %v, want ErrInvalidPaymentMethod", err)
	}
}

func TestManualTestHiddenAfterDeadline(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedManualTestMethod(t, store)
	base := time.Unix(1_800_000_000, 0)
	// Deadline already passed at "base".
	svc := NewService(store).WithManualTestWindow(base.Add(-time.Minute), 24*time.Hour, func() time.Time { return base })

	if manualTestListed(t, svc) {
		t.Fatal("manual_test listed after its deadline")
	}
}

func TestManualTestHiddenWhenWindowTooLong(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedManualTestMethod(t, store)
	base := time.Unix(1_800_000_000, 0)
	// Deadline beyond the maximum window: overlong activation fails closed.
	svc := NewService(store).WithManualTestWindow(base.Add(48*time.Hour), 24*time.Hour, func() time.Time { return base })

	if manualTestListed(t, svc) {
		t.Fatal("manual_test listed for an overlong activation window")
	}
}

// Inside a bounded window the row works end to end: listed, quotable,
// orderable — proving the gate is the window, not the row.
func TestManualTestAvailableInsideWindow(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedManualTestMethod(t, store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-MT", Name: "MT", Slug: "mt", Status: "active", Stock: 5, Price: 100})
	seedDefaultShippingMethods(t, store)
	base := time.Unix(1_800_000_000, 0)
	svc := openWindow(NewService(store), base)

	if !manualTestListed(t, svc) {
		t.Fatal("manual_test hidden inside its activation window")
	}
	quote, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-MT", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "manual_test",
	})
	if err != nil {
		t.Fatalf("quote inside window: %v", err)
	}
	if quote.PaymentMethod != "manual_test" {
		t.Fatalf("quote payment method = %q", quote.PaymentMethod)
	}
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "A", Email: "a@b.c", Phone: "0912345678", ShippingAddress: "Taipei",
		Items:          []OrderItem{{SKU: "SKU-MT", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "manual_test", IdempotencyKey: "mt-order",
	})
	if err != nil {
		t.Fatalf("order inside window: %v", err)
	}
	if order.ID == "" {
		t.Fatal("order ID empty")
	}
}

// The boundary itself: one tick before the deadline is open, at the
// deadline it is closed.
func TestManualTestBoundaryTick(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedManualTestMethod(t, store)
	deadline := time.Unix(1_800_000_000, 0)
	justBefore := deadline.Add(-time.Second)
	svc := NewService(store).WithManualTestWindow(deadline, 24*time.Hour, func() time.Time { return justBefore })
	if !manualTestListed(t, svc) {
		t.Fatal("one tick before deadline should still be open")
	}
	atDeadline := deadline
	svc = svc.WithManualTestWindow(deadline, 24*time.Hour, func() time.Time { return atDeadline })
	if manualTestListed(t, svc) {
		t.Fatal("at the deadline the window must be closed")
	}
}
