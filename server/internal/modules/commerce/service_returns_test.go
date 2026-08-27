package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/example/ai-site-starter/server/internal/auth"
	"strings"
	"sync"
	"testing"
)

func TestIdempotencySameKeySamePayloadReturnsSameOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	in := OrderInput{
		CustomerName:    "Carol",
		Email:           "carol@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-1",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	first, err := svc.CreateOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.CreateOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent reuse returned different order: %q vs %q", first.ID, second.ID)
	}
}

// TestReturnReceivedToRefundedIsGated proves that the admin return endpoint
// cannot transition "received" to "refunded" — the refund transition requires
// a verified provider webhook/refund adapter or governed offline-payment
// confirmation, a dedicated orders.refund capability, idempotency, and AAL2,
// none of which are implemented yet (B7 gated remainder, AC-010 pending).
// Exposing "refunded" here would re-introduce the fake-refund path that
// Phase E removed. The transition must return ErrInvalidTransition with no
// side effects (order unchanged, version unchanged, no new event).
func TestReturnReceivedToRefundedIsGated(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-G", Name: "Gated", Slug: "gated", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Dave",
		Email:           "dave@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-G", Quantity: 2}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	for _, step := range []string{"processing", "shipped", "delivered"} {
		order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step)
		if err != nil {
			t.Fatalf("transition -> %s: %v", step, err)
		}
	}
	for _, step := range []string{"requested", "approved", "received"} {
		order, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step)
		if err != nil {
			t.Fatalf("return -> %s: %v", step, err)
		}
	}

	versionBefore := order.Version
	statusBefore := order.ReturnRequestStatus

	// The refund transition must be rejected.
	_, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "refunded")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("received -> refunded error = %v, want ErrInvalidTransition", err)
	}

	// No side effects: order unchanged, version unchanged.
	updated, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.ReturnRequestStatus != statusBefore {
		t.Fatalf("return_request_status after gated refund = %q, want %q", updated.ReturnRequestStatus, statusBefore)
	}
	if updated.Version != versionBefore {
		t.Fatalf("version after gated refund = %d, want %d (must not increment)", updated.Version, versionBefore)
	}

	// No new event was appended.
	events, err := store.ListOrderEvents(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	// 4 creation/fulfillment events + 3 return events = 7.
	if len(events) != 7 {
		t.Fatalf("event count after gated refund = %d, want 7 (no new event)", len(events))
	}

	// Stock unchanged.
	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Fatalf("stock after gated refund = %d, want 8", p.Stock)
	}
}

// TestConcurrentSameKeySamePayloadReturnsSameOrder proves that two
// concurrent goroutines using the same idempotency_key with the same
// payload both get the same order back. Stock is decremented exactly once.
func TestConcurrentSameKeySamePayloadReturnsSameOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	in := OrderInput{
		CustomerName:    "Concurrent Carol",
		Email:           "carol@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 2}},
		IdempotencyKey:  "idem-concurrent-same",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}

	var (
		wg      sync.WaitGroup
		results [2]Order
		errs    [2]error
	)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = svc.CreateOrder(context.Background(), in)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d error: %v", i, err)
		}
	}
	if results[0].ID != results[1].ID {
		t.Errorf("concurrent same-key same-payload should return same order: %q vs %q", results[0].ID, results[1].ID)
	}
	// Stock should be decremented exactly once (10 - 2 = 8).
	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Errorf("stock after concurrent same-key orders = %d, want 8 (decremented once)", p.Stock)
	}
}

// TestConcurrentSameKeyDifferentPayloadReturnsConflict proves that two
// concurrent goroutines using the same idempotency_key with different
// payloads result in one success and one ErrIdempotencyConflict.
//
// NOTE: SQLite's SetMaxOpenConns(1) serializes these goroutines, so this
// test proves the idempotency LOGIC (check-in-tx, unique index, rollback
// + read-back) but not true concurrent access. PostgreSQL validation
// requires a live PG instance (AC-022 remains pending without it).
func TestConcurrentSameKeyDifferentPayloadReturnsConflict(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p2", SKU: "SKU-B", Name: "Beta", Slug: "beta", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	in1 := OrderInput{
		CustomerName:    "Concurrent Dave",
		Email:           "dave@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-concurrent-diff",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	in2 := OrderInput{
		CustomerName:    "Concurrent Dave",
		Email:           "dave@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-B", Quantity: 1}}, // different SKU
		IdempotencyKey:  "idem-concurrent-diff",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}

	var (
		wg      sync.WaitGroup
		results [2]Order
		errs    [2]error
	)
	for i, in := range []OrderInput{in1, in2} {
		wg.Add(1)
		go func(idx int, input OrderInput) {
			defer wg.Done()
			results[idx], errs[idx] = svc.CreateOrder(context.Background(), input)
		}(i, in)
	}
	wg.Wait()

	// Exactly one should succeed, one should get ErrIdempotencyConflict.
	successCount := 0
	conflictCount := 0
	for i, err := range errs {
		if err == nil {
			successCount++
		} else if errors.Is(err, ErrIdempotencyConflict) {
			conflictCount++
		} else {
			t.Errorf("goroutine %d unexpected error: %v", i, err)
		}
	}
	if successCount != 1 {
		t.Errorf("expected exactly 1 success, got %d", successCount)
	}
	if conflictCount != 1 {
		t.Errorf("expected exactly 1 conflict, got %d", conflictCount)
	}
}

// TestGetOrderForGuestWrongTokenReturnsNotFound proves that a wrong
// token returns ErrNotFound (not Forbidden) to avoid leaking order existence.
func TestGetOrderForGuestWrongTokenReturnsNotFound(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Grace",
		Email:           "grace@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Wrong token should return ErrNotFound.
	_, err = svc.GetOrderForGuest(context.Background(), order.ID, "wrong-token-abc123")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound (not forbidden, to avoid leaking existence)", err)
	}
}

// TestGetOrderForGuestEmptyTokenReturnsNotFound proves that an empty
// token is rejected.
func TestGetOrderForGuestEmptyTokenReturnsNotFound(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Heidi",
		Email:           "heidi@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Empty token should return ErrNotFound.
	_, err = svc.GetOrderForGuest(context.Background(), order.ID, "")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound for empty token", err)
	}
}

// TestGetOrderForMemberCrossUserReturnsNotFound proves that an
// authenticated member cannot access another member's order. Cross-user
// access returns 404 (not 403) to avoid leaking existence.
func TestGetOrderForMemberCrossUserReturnsNotFound(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	ivanPrincipal := auth.Principal{UserID: "member-ivan", Role: "member"}
	order, err := svc.CreateOrderForMember(context.Background(), ivanPrincipal, OrderInput{
		CustomerName:    "Ivan",
		Email:           "ivan@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order for member: %v", err)
	}
	if order.MemberID != "member-ivan" {
		t.Errorf("MemberID = %q, want member-ivan (must come from principal, not client)", order.MemberID)
	}

	// A different member should get ErrNotFound.
	otherPrincipal := auth.Principal{UserID: "member-karl", Role: "member"}
	_, err = svc.GetOrderForMember(context.Background(), otherPrincipal, order.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-user access should return ErrNotFound, got %v", err)
	}
}

// TestListOrdersForMemberReturnsMaskedPII proves that member-scoped order
// listing masks PII fields including shipping address.
func TestListOrdersForMemberReturnsMaskedPII(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	ivanPrincipal := auth.Principal{UserID: "member-ivan", Role: "member"}
	_, err := svc.CreateOrderForMember(context.Background(), ivanPrincipal, OrderInput{
		CustomerName:    "Ivan",
		Email:           "ivan@example.com",
		Phone:           "0987654321",
		ShippingAddress: "台北市大安區仁愛路一段200號",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},

		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order for member: %v", err)
	}

	orders, err := svc.ListOrdersForMember(context.Background(), "member-ivan")
	if err != nil {
		t.Fatalf("list orders for member: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].Phone == "0987654321" {
		t.Errorf("phone should be masked, got %q", orders[0].Phone)
	}
	if orders[0].Email == "ivan@example.com" {
		t.Errorf("email should be masked, got %q", orders[0].Email)
	}
	if orders[0].ShippingAddress == "台北市大安區仁愛路一段200號" {
		t.Errorf("shipping address should be masked, got %q", orders[0].ShippingAddress)
	}
	if orders[0].AccessToken != "" {
		t.Errorf("access token should never be in response")
	}
}

// TestIdempotencySameKeySamePayloadAllFieldsReturnsSameOrder verifies
// that the expanded samePayload comparison still returns the original
// order when ALL fields match exactly (including phone, shipping_address,
// shipping_method, payment_method, and memberID).
func TestIdempotencySameKeySamePayloadAllFieldsReturnsSameOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	base := OrderInput{
		CustomerName:    "Carol",
		Email:           "carol@example.com",
		Phone:           "02-1111-1111",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-full",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	first, err := svc.CreateOrder(context.Background(), base)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	// Exact same payload → same order.
	second, err := svc.CreateOrder(context.Background(), base)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent reuse returned different order: %q vs %q", first.ID, second.ID)
	}
}

// TestIdempotencyReplayAfterStockDepletedReturnsOriginalOrder is the
// critical regression test for revision 5: when the first order buys
// the last unit (stock reaches 0), an exact same-key same-payload
// retry must return the original order — NOT ErrInsufficientStock.
// Before the fix, createOrder checked stock BEFORE the idempotency
// key lookup, so the retry failed with ErrInsufficientStock instead
// of returning the original order. The fix moves the idempotency
// check before mutable-state validation and uses a persisted request
// fingerprint for comparison.
func TestIdempotencyReplayAfterStockDepletedReturnsOriginalOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	// Stock exactly equal to ordered quantity — first order buys the last unit.
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-LAST", Name: "LastUnit", Slug: "lastunit", Status: "active", Stock: 1, Price: 100})
	seedReadyPaymentMethod(t, store)

	base := OrderInput{
		CustomerName:    "Carol",
		Email:           "carol@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-LAST", Quantity: 1}},
		IdempotencyKey:  "idem-stock-zero",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	first, err := svc.CreateOrder(context.Background(), base)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Verify stock is now 0.
	p, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if p.Stock != 0 {
		t.Fatalf("expected stock=0 after first order, got %d", p.Stock)
	}

	// Replay with same key + same payload. Must return the original
	// order, NOT ErrInsufficientStock.
	second, err := svc.CreateOrder(context.Background(), base)
	if err != nil {
		t.Fatalf("replay after stock depleted: error = %v, want nil (same order)", err)
	}
	if first.ID != second.ID {
		t.Fatalf("replay returned different order: %q vs %q", first.ID, second.ID)
	}

	// Verify stock is still 0 — no second decrement.
	p2, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product after replay: %v", err)
	}
	if p2.Stock != 0 {
		t.Fatalf("stock changed after replay: expected 0, got %d", p2.Stock)
	}
}

// TestIdempotencyReplayAfterPaymentMethodDisabledReturnsOriginalOrder
// is a regression test for revision 5: when the payment method used by
// the original order is disabled after creation, an exact same-key
// same-payload retry must return the original order — NOT
// ErrInvalidPaymentMethod. This proves the early idempotency check
// bypasses mutable payment-method config validation on replay.
func TestIdempotencyReplayAfterPaymentMethodDisabledReturnsOriginalOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	base := OrderInput{
		CustomerName:    "Carol",
		Email:           "carol@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-pay-disabled",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	first, err := svc.CreateOrder(context.Background(), base)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Disable the payment method after the original order succeeded.
	if err := store.UpsertPaymentMethod(context.Background(), PaymentMethod{
		ID:              "atm",
		Method:          "atm",
		ProviderLabel:   "ATM 轉帳",
		Environment:     "sandbox",
		ReadinessStatus: "ready",
		Enabled:         false, // disabled after original order
		UpdatedUnix:     2,
	}); err != nil {
		t.Fatalf("disable payment method: %v", err)
	}

	// Replay with same key + same payload. Must return the original
	// order, NOT ErrInvalidPaymentMethod.
	second, err := svc.CreateOrder(context.Background(), base)
	if err != nil {
		t.Fatalf("replay after payment disabled: error = %v, want nil (same order)", err)
	}
	if first.ID != second.ID {
		t.Fatalf("replay returned different order: %q vs %q", first.ID, second.ID)
	}
}

// TestIdempotencyLegacyEmptyFingerprintReplayReturnsOriginalOrder is a
// regression test for migration 008 backward compatibility. Pre-
// migration orders have request_fingerprint=” (the column default).
// A same-key same-payload replay against such a legacy row must still
// return the original order via the samePayloadLegacy fallback, NOT
// ErrIdempotencyConflict. This test simulates a legacy row by
// inserting an order directly via the store with an empty fingerprint.
func TestIdempotencyLegacyEmptyFingerprintReplayReturnsOriginalOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	// Insert a legacy order directly via CreateOrderTx with an empty
	// RequestFingerprint, simulating a pre-migration-008 row.
	items := []OrderItem{{SKU: "SKU-A", Quantity: 1}}
	itemsJSON, _ := json.Marshal(items)
	timeline := []TimelineEvent{{Status: "pending", At: 1, Note: "order placed"}}
	timelineJSON, _ := json.Marshal(timeline)
	legacyOrder := Order{
		ID:                  "LEGACY-001",
		MemberID:            "",
		CustomerName:        "Legacy Carol",
		Email:               "legacy@example.com",
		Phone:               "02-1111-1111",
		Items:               items,
		ItemsJSON:           string(itemsJSON),
		ShippingAddress:     "123 Main St, Taipei",
		ShippingMethod:      "home_delivery",
		PaymentMethod:       "atm",
		Subtotal:            100,
		Discount:            0,
		Shipping:            120,
		Total:               220,
		Status:              "pending",
		PaymentStatus:       "unpaid",
		ReturnRequestStatus: "",
		IdempotencyKey:      "idem-legacy",
		RequestFingerprint:  "", // empty — simulates pre-migration row
		AccessTokenHash:     hashAccessToken("dummy-token"),
		Timeline:            timeline,
		TimelineJSON:        string(timelineJSON),
		UpdatedUnix:         1,
	}
	if err := store.CreateOrderTx(context.Background(), legacyOrder, items); err != nil {
		t.Fatalf("insert legacy order: %v", err)
	}

	// Replay with same key + same payload. The fingerprint is empty so
	// samePayload falls back to samePayloadLegacy. Must return the
	// original order, NOT ErrIdempotencyConflict.
	replay := OrderInput{
		CustomerName:    "Legacy Carol",
		Email:           "legacy@example.com",
		Phone:           "02-1111-1111",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-legacy",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	replayed, err := svc.CreateOrder(context.Background(), replay)
	if err != nil {
		t.Fatalf("legacy replay: error = %v, want nil (same order)", err)
	}
	if replayed.ID != "LEGACY-001" {
		t.Fatalf("legacy replay returned different order: %q, want LEGACY-001", replayed.ID)
	}
}

// TestCreateOrderUnusedKeyEmptyItemsReturnsValidationError verifies
// that with an UNUSED key (no existing order), empty items returns the
// original validation error ("order must contain at least one item").
// This is the "new order" path where structural validation runs.
func TestCreateOrderUnusedKeyEmptyItemsReturnsValidationError(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	// Unused key + empty items → validation error (new order path).
	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Test",
		Email:           "test@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           nil,
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err == nil || !strings.Contains(err.Error(), "at least one item") {
		t.Fatalf("unused key + empty items: error = %v, want validation error containing 'at least one item'", err)
	}
}

// TestCreateOrderUnusedKeyBlankCustomerNameReturnsValidationError
// verifies that with an UNUSED key, a blank customer_name returns the
// original validation error ("customer_name is required").
func TestCreateOrderUnusedKeyBlankCustomerNameReturnsValidationError(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	// Unused key + blank customer_name → validation error (new order path).
	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "  ",
		Email:           "test@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err == nil || !strings.Contains(err.Error(), "customer_name is required") {
		t.Fatalf("unused key + blank customer_name: error = %v, want validation error containing 'customer_name is required'", err)
	}
}
