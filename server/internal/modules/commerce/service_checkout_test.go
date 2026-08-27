package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/example/ai-site-starter/server/internal/auth"
	"math"
	"strings"
	"testing"
)

func TestCreateOrderServerCalculatesTotals(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	// Catalog product: authoritative price is 250, stock 10.
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 250})
	seedReadyPaymentMethod(t, store)

	// Client sends a wrong price (999), a fabricated shipping (999), and a
	// fabricated total; the service must ignore all three and recompute from
	// the catalog and approved shipping rules.
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Alice",
		Email:           "alice@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items: []OrderItem{
			{SKU: "SKU-A", Name: "Alpha", Price: 999, Quantity: 2},
		},
		ShippingMethod: "store_711",
		PaymentMethod:  "atm",
		Shipping:       999, // must be ignored
		Discount:       999, // must be ignored
		IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.Subtotal != 500 {
		t.Errorf("subtotal = %d, want 500 (250*2)", order.Subtotal)
	}
	if order.Shipping != 60 {
		t.Errorf("shipping = %d, want 60 (store_711 flat rate)", order.Shipping)
	}
	if order.Discount != 0 {
		t.Errorf("discount = %d, want 0 (no promo code)", order.Discount)
	}
	if order.Total != 560 {
		t.Errorf("total = %d, want 560 (500+60)", order.Total)
	}
	if order.Items[0].Price != 250 {
		t.Errorf("item price = %d, want 250 (server-calculated)", order.Items[0].Price)
	}

	// Stock must be decremented.
	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Errorf("stock after order = %d, want 8", p.Stock)
	}
}

// TestCreateOrderClientDiscountTampering is the P0 regression test: a
// client cannot discount a 1000 TWD order to 0 by sending Discount=1000.
// The server must ignore the client Discount field and apply only
// server-approved promo codes.
func TestCreateOrderClientDiscountTampering(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-B", Name: "Bravo", Slug: "bravo", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	// Client attempts to discount a 1000 TWD order to 0.
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Mallory",
		Email:           "mallory@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items: []OrderItem{
			{SKU: "SKU-B", Name: "Bravo", Price: 500, Quantity: 2},
		},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		Discount:       1000, // tampering attempt — must be ignored
		PromoCode:      "",   // no approved promo
		IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.Discount != 0 {
		t.Errorf("discount = %d, want 0 (client Discount must be ignored)", order.Discount)
	}
	if order.Total != 1000+120 {
		t.Errorf("total = %d, want 1120 (1000 subtotal + 120 shipping, no discount)", order.Total)
	}
}

func TestCreateOrderInsufficientStock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 1, Price: 100})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Alice",
		Items:        []OrderItem{{SKU: "SKU-A", Quantity: 5}},

		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		IdempotencyKey: testKey(),
	})
	if err != ErrInsufficientStock {
		t.Fatalf("error = %v, want ErrInsufficientStock", err)
	}
}

func TestIdempotencySameKeyDifferentPayloadErrors(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p2", SKU: "SKU-B", Name: "Beta", Slug: "beta", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	first, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Carol",
		Email:           "carol@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-2",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err = svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Carol",
		Email:           "carol@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-B", Quantity: 1}}, // different SKU
		IdempotencyKey:  "idem-2",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	})
	if err != ErrIdempotencyConflict {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
	_ = first
}

// TestCreateOrderDoesNotAcceptMemberID proves that the guest CreateOrder
// path does not accept or store a MemberID. Member identity is only set
// via the authenticated CreateOrderForMember path.
func TestCreateOrderDoesNotAcceptMemberID(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	// OrderInput has no MemberID field — this is a compile-time guarantee.
	// At runtime, a guest order should have MemberID="".
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Guest Only",
		Email:           "guest@test.local",
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
	if order.MemberID != "" {
		t.Errorf("guest order MemberID = %q, want empty (guest checkout must not set member id)", order.MemberID)
	}
}

// TestCreateOrderForMemberAtomicNoOrphanOrder proves that CreateOrderForMember
// writes the MemberID in the SAME transaction as the order insert and stock
// decrement. If the transaction fails, no order is left behind and no stock
// is decremented. This test uses a fault-injection store that fails the
// CreateOrderTxWithIdempotency call to prove rollback.
func TestCreateOrderForMemberAtomicNoOrphanOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-ATOMIC", Name: "Atomic", Slug: "atomic", Status: "active", Stock: 5, Price: 100})
	seedReadyPaymentMethod(t, store)

	// Wrap the store with a fault-injection wrapper that fails
	// CreateOrderTxWithIdempotency.
	faulty := &faultInjectStore{Store: store, failOnCreateTx: true}
	faultySvc := func() Service { seedDefaultShippingMethods(t, store); return NewService(faulty) }()

	memberPrincipal := auth.Principal{UserID: "member-atomic", Role: "member"}
	_, err := faultySvc.CreateOrderForMember(context.Background(), memberPrincipal, OrderInput{
		CustomerName:    "Atomic Test",
		Email:           "atomic@test.local",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-ATOMIC", Quantity: 2}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err == nil {
		t.Fatal("expected error from fault-injection, got nil")
	}

	// Verify NO order was left behind — the transaction rolled back.
	orders, err := svc.ListOrders(context.Background(), OrderFilter{})
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("expected 0 orders after failed CreateOrderForMember, got %d (orphan order left behind)", len(orders))
	}

	// Verify stock was NOT decremented — the transaction rolled back.
	product, err := svc.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if product.Stock != 5 {
		t.Errorf("stock = %d, want 5 (stock should not be decremented on rollback)", product.Stock)
	}

	// Now verify the happy path: a successful CreateOrderForMember writes
	// the MemberID atomically in the same transaction.
	order, err := svc.CreateOrderForMember(context.Background(), memberPrincipal, OrderInput{
		CustomerName:    "Atomic Happy",
		Email:           "atomic-happy@test.local",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-ATOMIC", Quantity: 2}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order for member: %v", err)
	}
	if order.MemberID != "member-atomic" {
		t.Errorf("MemberID = %q, want member-atomic (must be set in same tx, not post-commit patch)", order.MemberID)
	}

	// Verify the order in the DB has the MemberID (not just the response).
	stored, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get stored order: %v", err)
	}
	if stored.MemberID != "member-atomic" {
		t.Errorf("stored MemberID = %q, want member-atomic (must be persisted in same tx)", stored.MemberID)
	}

	// Verify stock was decremented.
	product, err = svc.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product after success: %v", err)
	}
	if product.Stock != 3 {
		t.Errorf("stock = %d, want 3 (5 - 2 = 3)", product.Stock)
	}
}

// TestCreateOrderForMemberIDOnlyFromPrincipal proves that the MemberID
// in the created order comes exclusively from the principal, not from
// any client-supplied field. OrderInput has no MemberID field (compile-time
// guarantee), and the runtime check confirms the stored MemberID matches
// the principal.
func TestCreateOrderForMemberIDOnlyFromPrincipal(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-MID", Name: "Member ID", Slug: "member-id", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	principal := auth.Principal{UserID: "member-principal-id", Role: "member"}
	order, err := svc.CreateOrderForMember(context.Background(), principal, OrderInput{
		CustomerName:    "Member ID Test",
		Email:           "mid@test.local",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-MID", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order for member: %v", err)
	}
	if order.MemberID != principal.UserID {
		t.Errorf("MemberID = %q, want %q (must come from principal, not client)", order.MemberID, principal.UserID)
	}

	// Verify in DB.
	stored, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get stored order: %v", err)
	}
	if stored.MemberID != principal.UserID {
		t.Errorf("stored MemberID = %q, want %q", stored.MemberID, principal.UserID)
	}
}

// TestQuoteRejectsEmptyShippingMethod is a P0 regression test: Quote must
// reject an empty shipping method with ErrInvalidShippingMethod, not
// silently default to home delivery pricing.
func TestQuoteRejectsEmptyShippingMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	// Production service — no fee schedule (nil). But even with a fee
	// schedule, empty method must be rejected before the fee check.
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "",
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("error = %v, want ErrInvalidShippingMethod", err)
	}
}

// TestQuoteRejectsUnknownShippingMethod is a P0 regression test: Quote must
// reject an unknown shipping method (e.g. "totally_invalid") with
// ErrInvalidShippingMethod, not silently default to home delivery pricing
// and echo the invalid method back.
func TestQuoteRejectsUnknownShippingMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "totally_invalid",
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("error = %v, want ErrInvalidShippingMethod", err)
	}
}

func TestQuoteRejectsDisabledAndMissingShippingMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	if err := store.CreateShippingMethod(context.Background(), ShippingMethod{
		ID: "sm-disabled", Method: "home_delivery", Label: "宅配", Fee: 120, Enabled: false, Version: 1, UpdatedUnix: 1,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("disabled error = %v, want ErrInvalidShippingMethod", err)
	}

	_, err = svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "not_configured",
		PaymentMethod:  "atm",
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("missing error = %v, want ErrInvalidShippingMethod", err)
	}
}

func TestQuoteRejectsUnknownPromoCode(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-PROMO", Name: "Promo Product", Slug: "promo-product", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-PROMO", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		PromoCode:      "NOT-A-REAL-CODE",
	})
	if !errors.Is(err, ErrInvalidPromoCode) {
		t.Fatalf("error = %v, want ErrInvalidPromoCode", err)
	}
}

func TestQuoteRejectsExpiredPromoCode(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-PROMO", Name: "Promo Product", Slug: "promo-product", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	if err := store.UpsertPromo(context.Background(), Promo{
		ID: "expired", Code: "EXPIRED", Type: "fixed", Value: 100,
		Enabled: true, StartsUnix: 0, ExpiresUnix: 1, UpdatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed expired promo: %v", err)
	}

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-PROMO", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		PromoCode:      "EXPIRED",
	})
	if !errors.Is(err, ErrInvalidPromoCode) {
		t.Fatalf("error = %v, want ErrInvalidPromoCode", err)
	}
}

func TestQuoteAcceptsActivePromoCode(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-PROMO", Name: "Promo Product", Slug: "promo-product", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	if err := store.UpsertPromo(context.Background(), Promo{
		ID: "active", Code: "SAVE100", Type: "fixed", Value: 100,
		Enabled: true, StartsUnix: 0, ExpiresUnix: 0, UpdatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed active promo: %v", err)
	}

	quote, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-PROMO", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		PromoCode:      "SAVE100",
	})
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	if quote.Discount != 100 || quote.PromoCode != "SAVE100" {
		t.Fatalf("quote discount=%d promo_code=%q, want 100 and SAVE100", quote.Discount, quote.PromoCode)
	}
}

func TestQuoteFailsClosedWhenPromoValidationUnavailable(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	faulty := &faultInjectStore{Store: store, failGetActivePromo: true}
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(faulty) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-PROMO", Name: "Promo Product", Slug: "promo-product", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-PROMO", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		PromoCode:      "SAVE100",
	})
	if !errors.Is(err, ErrPromoValidationUnavailable) {
		t.Fatalf("error = %v, want ErrPromoValidationUnavailable", err)
	}
}

func TestCreateOrderRejectsUnknownPromoWithoutSideEffects(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-PROMO", Name: "Promo Product", Slug: "promo-product", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Alice", Email: "alice@example.com", Phone: "0912345678",
		ShippingAddress: "Taipei", Items: []OrderItem{{SKU: "SKU-PROMO", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm",
		PromoCode: "NOT-A-REAL-CODE", IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrInvalidPromoCode) {
		t.Fatalf("error = %v, want ErrInvalidPromoCode", err)
	}
	orders, err := svc.ListOrders(context.Background(), OrderFilter{})
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("orders created after invalid promo = %d, want 0", len(orders))
	}
	product, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if product.Stock != 10 {
		t.Fatalf("stock after invalid promo = %d, want 10", product.Stock)
	}
}

// TestCreateOrderRejectsEmptyShippingMethod is a P0 regression test:
// CreateOrder must reject an empty shipping method with
// ErrInvalidShippingMethod, not silently default to home delivery.
func TestCreateOrderRejectsEmptyShippingMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:   "Alice",
		Email:          "alice@example.com",
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "",
		IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("error = %v, want ErrInvalidShippingMethod", err)
	}
}

// TestCreateOrderRejectsUnknownShippingMethod is a P0 regression test:
// CreateOrder must reject an unknown shipping method (e.g.
// "totally_invalid") with ErrInvalidShippingMethod, not silently default
// to home delivery pricing and create the order with shipping 0.
func TestCreateOrderRejectsUnknownShippingMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:   "Alice",
		Email:          "alice@example.com",
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "totally_invalid",
		IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("error = %v, want ErrInvalidShippingMethod", err)
	}
}

func TestCreateOrderRejectsDisabledShippingMethodWithoutSideEffects(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	if err := store.CreateShippingMethod(context.Background(), ShippingMethod{
		ID: "sm-disabled", Method: "home_delivery", Label: "宅配", Fee: 120, Enabled: false, Version: 1, UpdatedUnix: 1,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Alice", Email: "alice@example.com", Phone: "0912345678",
		ShippingAddress: "Taipei", Items: []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("error = %v, want ErrInvalidShippingMethod", err)
	}
	orders, err := svc.ListOrders(context.Background(), OrderFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 0 {
		t.Fatalf("orders created after disabled shipping = %d, want 0", len(orders))
	}
	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 10 {
		t.Fatalf("stock = %d, want 10", p.Stock)
	}
}

// TestCreateOrderForMemberRejectsUnknownShippingMethod proves that the
// member checkout path also rejects unknown shipping methods.
func TestCreateOrderForMemberRejectsUnknownShippingMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	principal := auth.Principal{UserID: "member-1", Role: "member"}
	_, err := svc.CreateOrderForMember(context.Background(), principal, OrderInput{
		CustomerName:   "Alice",
		Email:          "alice@example.com",
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "totally_invalid",
		IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("error = %v, want ErrInvalidShippingMethod", err)
	}
}

func TestCreateOrderUsesCurrentShippingAfterQuoteChange(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-SHIP", Name: "Ship", Slug: "ship", Status: "active", Stock: 5, Price: 500})
	seedReadyPaymentMethod(t, store)
	seedOneShipping(t, store, ShippingMethod{Method: "flat", Label: "固定", Fee: 80, Enabled: true})

	q, o := quoteAndOrderInputs("SKU-SHIP", "flat", testKey(), 1)
	quote, err := svc.Quote(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if quote.Shipping != 80 {
		t.Fatalf("quote shipping = %d, want 80", quote.Shipping)
	}

	row, err := store.GetShippingMethod(context.Background(), "sm-flat")
	if err != nil {
		t.Fatal(err)
	}
	row.Fee = 200
	if err := store.UpdateShippingMethod(context.Background(), row, row.Version); err != nil {
		t.Fatal(err)
	}

	order, err := svc.CreateOrder(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if order.Shipping != 200 {
		t.Fatalf("order shipping = %d, want current fee 200 (not quoted 80)", order.Shipping)
	}
}

func TestCreateOrderReplayAfterDisableUsesOriginalSnapshot(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-SHIP", Name: "Ship", Slug: "ship", Status: "active", Stock: 5, Price: 500})
	seedReadyPaymentMethod(t, store)
	seedOneShipping(t, store, ShippingMethod{Method: "flat", Label: "固定", Fee: 80, Enabled: true})

	key := testKey()
	_, o := quoteAndOrderInputs("SKU-SHIP", "flat", key, 1)
	first, err := svc.CreateOrder(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if first.Shipping != 80 {
		t.Fatalf("first shipping = %d, want 80", first.Shipping)
	}

	row, err := store.GetShippingMethod(context.Background(), "sm-flat")
	if err != nil {
		t.Fatal(err)
	}
	row.Enabled = false
	if err := store.UpdateShippingMethod(context.Background(), row, row.Version); err != nil {
		t.Fatal(err)
	}

	replay, err := svc.CreateOrder(context.Background(), o)
	if err != nil {
		t.Fatalf("same-key replay after disable: %v", err)
	}
	if replay.ID != first.ID || replay.Shipping != first.Shipping || replay.Total != first.Total {
		t.Fatalf("replay diverged: first=%s/%d replay=%s/%d", first.ID, first.Shipping, replay.ID, replay.Shipping)
	}
	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 4 {
		t.Fatalf("stock = %d, want 4 (replay must not decrement again)", p.Stock)
	}
}

func TestQuoteAndCreateOrderFailClosedWhenShippingStoreUnavailable(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedDefaultShippingMethods(t, store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-SHIP", Name: "Ship", Slug: "ship", Status: "active", Stock: 5, Price: 500})
	seedReadyPaymentMethod(t, store)
	faulty := &faultInjectStore{Store: store, failListShipping: true}
	svc := NewService(faulty)

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items: []OrderItem{{SKU: "SKU-SHIP", Quantity: 1}}, ShippingMethod: "home_delivery", PaymentMethod: "atm",
	})
	if !errors.Is(err, ErrShippingConfigUnavailable) {
		t.Fatalf("quote error = %v, want ErrShippingConfigUnavailable", err)
	}
	if strings.Contains(fmt.Sprint(err), "injected fault") {
		t.Fatalf("quote error leaked store cause: %v", err)
	}

	_, err = svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Alice", Email: "alice@example.com", Phone: "0912345678",
		ShippingAddress: "Taipei", Items: []OrderItem{{SKU: "SKU-SHIP", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrShippingConfigUnavailable) {
		t.Fatalf("order error = %v, want ErrShippingConfigUnavailable", err)
	}
	_, err = svc.CreateOrderForMember(context.Background(), auth.Principal{UserID: "m1", Role: "member"}, OrderInput{
		CustomerName: "Alice", Email: "alice@example.com", Phone: "0912345678",
		ShippingAddress: "Taipei", Items: []OrderItem{{SKU: "SKU-SHIP", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrShippingConfigUnavailable) {
		t.Fatalf("member order error = %v, want ErrShippingConfigUnavailable", err)
	}
	orders, err := NewService(store).ListOrders(context.Background(), OrderFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 0 {
		t.Fatalf("orders created after store failure = %d, want 0", len(orders))
	}
	p, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Stock != 5 {
		t.Fatalf("stock after guest/member store failure = %d, want 5", p.Stock)
	}
}

func TestQuoteAndCreateOrderRejectPersistedShippingFeeOverflow(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-FEE-OVF", Name: "Fee Overflow", Slug: "fee-ovf", Status: "active", Stock: 5, Price: 1})
	seedReadyPaymentMethod(t, store)
	seedOneShipping(t, store, ShippingMethod{
		Method: "huge_fee", Label: "溢位運費", Fee: math.MaxInt, Enabled: true,
	})

	quote, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-FEE-OVF", Quantity: 1}},
		ShippingMethod: "huge_fee",
		PaymentMethod:  "atm",
	})
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("quote error = %v, want ErrOverflow from subtotal+fee", err)
	}
	if quote.Total != 0 || quote.Shipping != 0 {
		t.Fatalf("quote wrapped overflow into totals: %#v", quote)
	}

	_, err = svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Alice", Email: "alice@example.com", Phone: "0912345678",
		ShippingAddress: "Taipei", Items: []OrderItem{{SKU: "SKU-FEE-OVF", Quantity: 1}},
		ShippingMethod: "huge_fee", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("order error = %v, want ErrOverflow from subtotal+fee", err)
	}
	orders, err := svc.ListOrders(context.Background(), OrderFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 0 {
		t.Fatalf("orders created after shipping-fee overflow = %d, want 0", len(orders))
	}
	p, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Stock != 5 {
		t.Fatalf("stock after shipping-fee overflow = %d, want 5", p.Stock)
	}
}

// TestQuoteRejectsEmptyPaymentMethod is a regression test: Quote must
// reject an empty payment method with ErrInvalidPaymentMethod. The server
// is the authority for payment method availability — the browser must not
// decide which payment methods are available (REQ-004/AC-007).
func TestQuoteRejectsEmptyPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "",
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("error = %v, want ErrInvalidPaymentMethod", err)
	}
}

// TestQuoteRejectsUnknownPaymentMethod is a regression test: Quote must
// reject an unknown payment method with ErrInvalidPaymentMethod.
func TestQuoteRejectsUnknownPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "totally_invalid",
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("error = %v, want ErrInvalidPaymentMethod", err)
	}
}

// TestQuoteRejectsDisabledPaymentMethod proves that Quote rejects a
// payment method that exists but is disabled (enabled=false).
func TestQuoteRejectsDisabledPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	// Seed a disabled payment method (enabled=false, but readiness=ready).
	if err := store.UpsertPaymentMethod(context.Background(), PaymentMethod{
		ID:              "cod",
		Method:          "cod",
		ProviderLabel:   "貨到付款",
		Environment:     "sandbox",
		ReadinessStatus: "ready",
		Enabled:         false,
		UpdatedUnix:     1,
	}); err != nil {
		t.Fatalf("seed disabled payment method: %v", err)
	}

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "cod",
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("error = %v, want ErrInvalidPaymentMethod", err)
	}
}

// TestQuoteRejectsNotReadyPaymentMethod proves that Quote rejects a
// payment method that exists and is enabled but not ready
// (readiness_status != "ready").
func TestQuoteRejectsNotReadyPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	// Seed an enabled but not-ready payment method.
	if err := store.UpsertPaymentMethod(context.Background(), PaymentMethod{
		ID:              "credit_card",
		Method:          "credit_card",
		ProviderLabel:   "信用卡",
		Environment:     "sandbox",
		ReadinessStatus: "pending_setup",
		Enabled:         true,
		UpdatedUnix:     1,
	}); err != nil {
		t.Fatalf("seed not-ready payment method: %v", err)
	}

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "credit_card",
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("error = %v, want ErrInvalidPaymentMethod", err)
	}
}

// TestQuoteAcceptsReadyPaymentMethod proves that Quote accepts a
// payment method that is both enabled and ready.
func TestQuoteAcceptsReadyPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	result, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
	})
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	if result.PaymentMethod != "atm" {
		t.Errorf("payment_method = %q, want atm", result.PaymentMethod)
	}
}

// TestCreateOrderRejectsEmptyPaymentMethod is a regression test:
// CreateOrder must reject an empty payment method with
// ErrInvalidPaymentMethod.
func TestCreateOrderRejectsEmptyPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:   "Alice",
		Email:          "alice@example.com",
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "",
		IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("error = %v, want ErrInvalidPaymentMethod", err)
	}
}

// TestCreateOrderRejectsUnknownPaymentMethod is a regression test:
// CreateOrder must reject an unknown payment method with
// ErrInvalidPaymentMethod.
func TestCreateOrderRejectsUnknownPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:   "Alice",
		Email:          "alice@example.com",
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "totally_invalid",
		IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("error = %v, want ErrInvalidPaymentMethod", err)
	}
}

// TestCreateOrderPersistsPaymentMethod proves that the payment method is
// persisted on the order and echoed back in the response.
func TestCreateOrderPersistsPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Alice",
		Email:           "alice@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.PaymentMethod != "atm" {
		t.Errorf("payment_method = %q, want atm", order.PaymentMethod)
	}

	// Verify it was persisted — re-read from store.
	persisted, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if persisted.PaymentMethod != "atm" {
		t.Errorf("persisted payment_method = %q, want atm", persisted.PaymentMethod)
	}
}

// TestCreateOrderForMemberRejectsUnknownPaymentMethod proves that the
// member checkout path also rejects unknown payment methods.
func TestCreateOrderForMemberRejectsUnknownPaymentMethod(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-APP-01", Name: "Apparel", Slug: "apparel", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)

	principal := auth.Principal{UserID: "member-1", Role: "member"}
	_, err := svc.CreateOrderForMember(context.Background(), principal, OrderInput{
		CustomerName:   "Alice",
		Email:          "alice@example.com",
		Items:          []OrderItem{{SKU: "SKU-APP-01", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "totally_invalid",
		IdempotencyKey: testKey(),
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("error = %v, want ErrInvalidPaymentMethod", err)
	}
}

// TestIdempotencySameKeyDifferentPaymentMethodErrors is a regression test
// for revision 5: samePayload previously ignored payment_method, so
// reusing an idempotency key with a different ready payment method would
// return the original order instead of ErrIdempotencyConflict. The
// payment method is a material order-defining field — it determines
// fulfillment and must not be silently replayed.
func TestIdempotencySameKeyDifferentPaymentMethodErrors(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)
	seedSecondReadyPaymentMethod(t, store)

	base := OrderInput{
		CustomerName:    "Carol",
		Email:           "carol@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-pay",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	if _, err := svc.CreateOrder(context.Background(), base); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Same key, different payment method → must conflict.
	different := base
	different.PaymentMethod = "credit_card"
	_, err := svc.CreateOrder(context.Background(), different)
	if err != ErrIdempotencyConflict {
		t.Fatalf("error = %v, want ErrIdempotencyConflict for different payment_method", err)
	}
}

// TestIdempotencySameKeyDifferentShippingMethodErrors is a regression test
// for revision 5: samePayload previously ignored shipping_method. A
// different shipping method means a different shipping fee and delivery
// channel — a material order difference.
func TestIdempotencySameKeyDifferentShippingMethodErrors(t *testing.T) {
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
		IdempotencyKey:  "idem-ship",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	if _, err := svc.CreateOrder(context.Background(), base); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Same key, different shipping method → must conflict.
	different := base
	different.ShippingMethod = "store_711"
	_, err := svc.CreateOrder(context.Background(), different)
	if err != ErrIdempotencyConflict {
		t.Fatalf("error = %v, want ErrIdempotencyConflict for different shipping_method", err)
	}
}

// TestIdempotencySameKeyDifferentPhoneErrors is a regression test for
// revision 5: samePayload previously ignored phone. Phone is a PII field
// used for fulfillment contact — a material difference.
func TestIdempotencySameKeyDifferentPhoneErrors(t *testing.T) {
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
		IdempotencyKey:  "idem-phone",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	if _, err := svc.CreateOrder(context.Background(), base); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Same key, different phone → must conflict.
	different := base
	different.Phone = "02-2222-2222"
	_, err := svc.CreateOrder(context.Background(), different)
	if err != ErrIdempotencyConflict {
		t.Fatalf("error = %v, want ErrIdempotencyConflict for different phone", err)
	}
}

// TestIdempotencySameKeyDifferentShippingAddressErrors is a regression
// test for revision 5: samePayload previously ignored shipping_address.
// The shipping address determines where the order is delivered — a
// material difference.
func TestIdempotencySameKeyDifferentShippingAddressErrors(t *testing.T) {
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
		IdempotencyKey:  "idem-addr",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	if _, err := svc.CreateOrder(context.Background(), base); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Same key, different shipping address → must conflict.
	different := base
	different.ShippingAddress = "456 Oak Ave, Kaohsiung"
	_, err := svc.CreateOrder(context.Background(), different)
	if err != ErrIdempotencyConflict {
		t.Fatalf("error = %v, want ErrIdempotencyConflict for different shipping_address", err)
	}
}

// TestIdempotencySameKeyMemberVsGuestErrors is a regression test for
// revision 5: samePayload previously ignored memberID. A guest replay
// of a member order (or vice versa) is a material identity difference
// — the order ownership differs.
func TestIdempotencySameKeyMemberVsGuestErrors(t *testing.T) {
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
		IdempotencyKey:  "idem-member",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}

	// Create as a member.
	principal := auth.Principal{UserID: "member-1", Role: "member"}
	if _, err := svc.CreateOrderForMember(context.Background(), principal, base); err != nil {
		t.Fatalf("first create (member): %v", err)
	}

	// Replay as guest with same key and same payload → must conflict
	// because memberID differs (member-1 vs "").
	_, err := svc.CreateOrder(context.Background(), base)
	if err != ErrIdempotencyConflict {
		t.Fatalf("error = %v, want ErrIdempotencyConflict for member-vs-guest identity mismatch", err)
	}
}

// TestIdempotencyReplayDifferentPayloadAfterPaymentDisabledStillConflicts
// verifies that disabling the payment method does not bypass
// different-payload conflict detection. Even though the early
// idempotency check skips payment validation, a different-payload
// replay must still return ErrIdempotencyConflict.
func TestIdempotencyReplayDifferentPayloadAfterPaymentDisabledStillConflicts(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)
	seedSecondReadyPaymentMethod(t, store)

	base := OrderInput{
		CustomerName:    "Carol",
		Email:           "carol@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-pay-disabled-conflict",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	if _, err := svc.CreateOrder(context.Background(), base); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Disable the original payment method.
	if err := store.UpsertPaymentMethod(context.Background(), PaymentMethod{
		ID:              "atm",
		Method:          "atm",
		ProviderLabel:   "ATM 轉帳",
		Environment:     "sandbox",
		ReadinessStatus: "ready",
		Enabled:         false,
		UpdatedUnix:     2,
	}); err != nil {
		t.Fatalf("disable payment method: %v", err)
	}

	// Replay with same key but DIFFERENT payment method. Even though
	// atm is now disabled, the early idempotency check must detect the
	// fingerprint mismatch and return ErrIdempotencyConflict — NOT
	// ErrInvalidPaymentMethod (which would mean it tried to validate
	// the new payment method and failed, leaking that the original
	// order exists but returning the wrong error).
	different := base
	different.PaymentMethod = "credit_card"
	_, err := svc.CreateOrder(context.Background(), different)
	if err != ErrIdempotencyConflict {
		t.Fatalf("error = %v, want ErrIdempotencyConflict for different payload after payment disabled", err)
	}
}

// TestIdempotencyLegacyEmptyFingerprintDifferentPayloadConflicts
// verifies that the samePayloadLegacy fallback still detects
// different-payload conflicts for legacy rows. A different payment
// method with the same key must return ErrIdempotencyConflict even
// when the stored fingerprint is empty.
func TestIdempotencyLegacyEmptyFingerprintDifferentPayloadConflicts(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)
	seedSecondReadyPaymentMethod(t, store)

	// Insert a legacy order with empty fingerprint.
	items := []OrderItem{{SKU: "SKU-A", Quantity: 1}}
	itemsJSON, _ := json.Marshal(items)
	timeline := []TimelineEvent{{Status: "pending", At: 1, Note: "order placed"}}
	timelineJSON, _ := json.Marshal(timeline)
	legacyOrder := Order{
		ID:                  "LEGACY-002",
		MemberID:            "",
		CustomerName:        "Legacy Dave",
		Email:               "legacy-dave@example.com",
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
		IdempotencyKey:      "idem-legacy-conflict",
		RequestFingerprint:  "", // empty — simulates pre-migration row
		AccessTokenHash:     hashAccessToken("dummy-token-2"),
		Timeline:            timeline,
		TimelineJSON:        string(timelineJSON),
		UpdatedUnix:         1,
	}
	if err := store.CreateOrderTx(context.Background(), legacyOrder, items); err != nil {
		t.Fatalf("insert legacy order: %v", err)
	}

	// Replay with same key but DIFFERENT payment method. The
	// samePayloadLegacy fallback must detect the mismatch and return
	// ErrIdempotencyConflict.
	different := OrderInput{
		CustomerName:   "Legacy Dave",
		Email:          "legacy-dave@example.com",
		Phone:          "02-1111-1111",
		Items:          []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey: "idem-legacy-conflict",
		ShippingMethod: "home_delivery",
		PaymentMethod:  "credit_card", // different
	}
	_, err := svc.CreateOrder(context.Background(), different)
	if err != ErrIdempotencyConflict {
		t.Fatalf("legacy different-payload replay: error = %v, want ErrIdempotencyConflict", err)
	}
}

// TestCreateOrderRejectsEmptyIdempotencyKey is a regression test for
// revision 5: createOrder must reject empty/whitespace idempotency keys
// with ErrMissingIdempotencyKey. An empty key bypasses both the early
// lookup and the in-transaction idempotency check, creating duplicate-
// order risk on transport retry.
func TestCreateOrderRejectsEmptyIdempotencyKey(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	base := OrderInput{
		CustomerName:   "Carol",
		Email:          "carol@example.com",
		Items:          []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
	}

	// Empty key → ErrMissingIdempotencyKey
	emptyKey := base
	emptyKey.IdempotencyKey = ""
	_, err := svc.CreateOrder(context.Background(), emptyKey)
	if !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("empty key: error = %v, want ErrMissingIdempotencyKey", err)
	}

	// Whitespace-only key → ErrMissingIdempotencyKey
	wsKey := base
	wsKey.IdempotencyKey = "   "
	_, err = svc.CreateOrder(context.Background(), wsKey)
	if !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("whitespace key: error = %v, want ErrMissingIdempotencyKey", err)
	}
}

// TestCreateOrderForMemberRejectsEmptyIdempotencyKey verifies that the
// member checkout path also rejects empty idempotency keys.
func TestCreateOrderForMemberRejectsEmptyIdempotencyKey(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	principal := auth.Principal{UserID: "member-1", Role: "member"}
	_, err := svc.CreateOrderForMember(context.Background(), principal, OrderInput{
		CustomerName:   "Alice",
		Email:          "alice@example.com",
		Items:          []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		IdempotencyKey: "", // empty
	})
	if !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("error = %v, want ErrMissingIdempotencyKey", err)
	}
}

// TestRequestFingerprintNoDelimiterCollision is a regression test for
// revision 5 blocker 3: the old computeRequestFingerprint used
// \n-separated "field=value" encoding. A field value containing \n
// could shift the boundary between fields, making two materially
// different requests encode to the same bytes and produce the same
// SHA-256 digest. The new JSON-based encoding escapes newlines and
// other special characters, so this collision is impossible.
//
// This test constructs two materially different OrderInputs that DO
// collide under the old scheme (proven by legacyRequestFingerprint),
// then asserts the new JSON-based fingerprints differ and that a
// same-key replay with the second input returns ErrIdempotencyConflict.
//
// The colliding pair:
//
//	A: customer_name = "Alice\nemail=attacker@example.com"
//	   email          = "real@example.com"
//	B: customer_name = "Alice"
//	   email          = "attacker@example.com\nemail=real@example.com"
//
// Under the old \n scheme, both produce identical bytes:
//
//	customer_name=Alice\nemail=attacker@example.com\nemail=real@example.com\n...
//
// because the \n inside A's customer_name merges with the next field
// line, and the \n inside B's email merges with the next field line.
func TestRequestFingerprintNoDelimiterCollision(t *testing.T) {
	t.Parallel()

	// Two materially different inputs that collide under the old scheme.
	inputA := OrderInput{
		CustomerName:    "Alice\nemail=attacker@example.com",
		Email:           "real@example.com",
		Phone:           "02-1111-1111",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-collision-test",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	inputB := OrderInput{
		CustomerName:    "Alice",
		Email:           "attacker@example.com\nemail=real@example.com",
		Phone:           "02-1111-1111",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-collision-test",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}

	// Prove the old encoding collides: both inputs produce the same
	// legacy fingerprint. This demonstrates the vulnerability.
	legacyFpA := legacyRequestFingerprint(inputA, "")
	legacyFpB := legacyRequestFingerprint(inputB, "")
	if legacyFpA != legacyFpB {
		t.Fatalf("legacy encoding sanity check: expected collision but got different legacy fingerprints A=%q B=%q", legacyFpA, legacyFpB)
	}
	t.Logf("legacy encoder confirmed collision: both produce %q", legacyFpA)

	// Assert the new JSON-based encoding does NOT collide.
	fpA := computeRequestFingerprint(inputA, "")
	fpB := computeRequestFingerprint(inputB, "")
	if fpA == fpB {
		t.Fatalf("new encoding collision: two materially different inputs produced the same fingerprint %q", fpA)
	}
	t.Logf("new encoding distinguishes: A=%q B=%q", fpA, fpB)

	// End-to-end: create order with input A, then replay with input B
	// using the same key. Must get ErrIdempotencyConflict, NOT the
	// original order (which would mean the fingerprints collided).
	//
	// inputB's email contains a newline (malformed). With the corrected
	// createOrder ordering (normalize → idempotency check → validate),
	// the idempotency check runs BEFORE email validation. Since the key
	// already exists and the payloads differ, the server returns
	// ErrIdempotencyConflict — NOT ErrInvalidEmail. This is the correct
	// invariant: the client must know the key was already used for a
	// different request, regardless of the new payload's validity.
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	orderA, err := svc.CreateOrder(context.Background(), inputA)
	if err != nil {
		t.Fatalf("create order A: %v", err)
	}

	_, err = svc.CreateOrder(context.Background(), inputB)
	if err != ErrIdempotencyConflict {
		t.Fatalf("replay with colliding input: error = %v, want ErrIdempotencyConflict (got original order %s)", err, orderA.ID)
	}
}

// TestRequestFingerprintNewlineInShippingAddress is a second collision
// regression test using shipping_address + shipping_method. The old
// scheme separated fields with \n, so a shipping_address containing
// \n followed by "shipping_method=" could collide with a different
// address + method combination.
//
// The colliding pair:
//
//	A: shipping_address = "123 Main St\nshipping_method=store_711"
//	   shipping_method  = "home_delivery"
//	B: shipping_address = "123 Main St"
//	   shipping_method  = "store_711\nshipping_method=home_delivery"
//
// Under the old \n scheme, both produce identical bytes:
//
//	...shipping_address=123 Main St\nshipping_method=store_711\nshipping_method=home_delivery\n...
func TestRequestFingerprintNewlineInShippingAddress(t *testing.T) {
	t.Parallel()

	inputA := OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
		Phone:           "02-2222-2222",
		ShippingAddress: "123 Main St\nshipping_method=store_711",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-addr-collision",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	inputB := OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
		Phone:           "02-2222-2222",
		ShippingAddress: "123 Main St",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-addr-collision",
		ShippingMethod:  "store_711\nshipping_method=home_delivery",
		PaymentMethod:   "atm",
	}

	// Prove the old encoding collides.
	legacyFpA := legacyRequestFingerprint(inputA, "")
	legacyFpB := legacyRequestFingerprint(inputB, "")
	if legacyFpA != legacyFpB {
		t.Fatalf("legacy encoding sanity check: expected collision but got different legacy fingerprints A=%q B=%q", legacyFpA, legacyFpB)
	}
	t.Logf("legacy encoder confirmed collision: both produce %q", legacyFpA)

	// Assert the new encoding does NOT collide.
	fpA := computeRequestFingerprint(inputA, "")
	fpB := computeRequestFingerprint(inputB, "")
	if fpA == fpB {
		t.Fatalf("new encoding collision in shipping_address: two materially different inputs produced the same fingerprint %q", fpA)
	}

	// End-to-end verification.
	//
	// inputB's shipping_method contains a newline (would be rejected by
	// computeShipping as ErrInvalidShippingMethod). With the corrected
	// createOrder ordering (normalize → idempotency check → validate),
	// the idempotency check runs BEFORE shipping validation. Since the
	// key already exists and the payloads differ, the server returns
	// ErrIdempotencyConflict — NOT ErrInvalidShippingMethod.
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), inputA)
	if err != nil {
		t.Fatalf("create order A: %v", err)
	}
	_, err = svc.CreateOrder(context.Background(), inputB)
	if err != ErrIdempotencyConflict {
		t.Fatalf("replay with colliding shipping_address: error = %v, want ErrIdempotencyConflict", err)
	}
}

// TestRequestFingerprintItemOrderInvariant verifies that reordering
// items in the cart does not produce a different fingerprint (items
// are sorted by SKU then quantity).
func TestRequestFingerprintItemOrderInvariant(t *testing.T) {
	t.Parallel()

	inputA := OrderInput{
		CustomerName: "Carol",
		Email:        "carol@example.com",
		Phone:        "02-3333-3333",
		Items: []OrderItem{
			{SKU: "SKU-A", Quantity: 2},
			{SKU: "SKU-B", Quantity: 1},
		},
		IdempotencyKey: "idem-order-invariant",
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
	}
	inputB := OrderInput{
		CustomerName: "Carol",
		Email:        "carol@example.com",
		Phone:        "02-3333-3333",
		Items: []OrderItem{
			{SKU: "SKU-B", Quantity: 1},
			{SKU: "SKU-A", Quantity: 2},
		},
		IdempotencyKey: "idem-order-invariant",
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
	}

	fpA := computeRequestFingerprint(inputA, "")
	fpB := computeRequestFingerprint(inputB, "")
	if fpA != fpB {
		t.Fatalf("item reorder changed fingerprint: A=%q, B=%q (should be equal)", fpA, fpB)
	}
}

// TestRequestFingerprintWhitespaceDifferentSKUConflicts verifies that
// the fingerprint does NOT trim SKU, matching creation semantics.
// resolveItems/findProductBySKU uses ci.SKU as-is (no TrimSpace), so
// " SKU-A " would fail product lookup while "SKU-A" succeeds. These
// are materially different requests. If the fingerprint trimmed SKU,
// both would hash the same and a whitespace-different replay would
// return the original order instead of ErrIdempotencyConflict.
//
// This test creates an order with SKU "SKU-A", then replays with the
// same key but SKU " SKU-A " (surrounding whitespace). The replay must
// return ErrIdempotencyConflict (different fingerprint), NOT the
// original order.
func TestRequestFingerprintWhitespaceDifferentSKUConflicts(t *testing.T) {
	t.Parallel()

	inputA := OrderInput{
		CustomerName:    "Dave",
		Email:           "dave@example.com",
		Phone:           "02-4444-4444",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		IdempotencyKey:  "idem-ws-sku",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}
	inputB := OrderInput{
		CustomerName:    "Dave",
		Email:           "dave@example.com",
		Phone:           "02-4444-4444",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: " SKU-A ", Quantity: 1}}, // whitespace-padded
		IdempotencyKey:  "idem-ws-sku",
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
	}

	// Fingerprints must differ — SKU is not trimmed.
	fpA := computeRequestFingerprint(inputA, "")
	fpB := computeRequestFingerprint(inputB, "")
	if fpA == fpB {
		t.Fatalf("whitespace SKU collision: \"SKU-A\" and \" SKU-A \" produced the same fingerprint %q", fpA)
	}

	// End-to-end: create with A, replay with B (same key, whitespace-
	// different SKU). Must get ErrIdempotencyConflict.
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	orderA, err := svc.CreateOrder(context.Background(), inputA)
	if err != nil {
		t.Fatalf("create order A: %v", err)
	}

	_, err = svc.CreateOrder(context.Background(), inputB)
	if err != ErrIdempotencyConflict {
		t.Fatalf("replay with whitespace-different SKU: error = %v, want ErrIdempotencyConflict (got original order %s)", err, orderA.ID)
	}
}

// TestCreateOrderValidationOrderingUnusedKeyMalformedEmail verifies
// that with an UNUSED idempotency key (no existing order), a malformed
// email is rejected with ErrInvalidEmail. This is the "new order" path
// where contact validation runs (STEP 6).
func TestCreateOrderValidationOrderingUnusedKeyMalformedEmail(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	// Unused key + malformed email → ErrInvalidEmail (new order path).
	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Test",
		Email:           "not-an-email",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != ErrInvalidEmail {
		t.Fatalf("unused key + malformed email: error = %v, want ErrInvalidEmail", err)
	}
}

// TestCreateOrderValidationOrderingExistingKeyMalformedPayloadConflicts
// verifies that with an EXISTING idempotency key (order already created),
// a replay with a DIFFERENT malformed payload returns ErrIdempotencyConflict
// — NOT ErrInvalidEmail. This is the critical invariant: the idempotency
// check (STEP 3) runs BEFORE contact validation (STEP 6), so the client
// learns the key was already used for a different request, not that the
// new payload is invalid.
func TestCreateOrderValidationOrderingExistingKeyMalformedPayloadConflicts(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	key := testKey()

	// Create a valid order with key K.
	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Alice",
		Email:           "alice@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  key,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Replay with same key K but DIFFERENT payload that has a malformed
	// email. Must return ErrIdempotencyConflict, NOT ErrInvalidEmail.
	_, err = svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Alice",
		Email:           "not-an-email", // malformed — would be ErrInvalidEmail on new order path
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  key, // same key, different payload
	})
	if err != ErrIdempotencyConflict {
		t.Fatalf("existing key + different malformed payload: error = %v, want ErrIdempotencyConflict", err)
	}
}

// TestCreateOrderValidationOrderingSamePayloadReplayNotBroken verifies
// that an exact same-key same-payload replay still returns the original
// order, even with the new contact validation in the pipeline. The
// replay bypasses all validation (STEP 3 returns the original order
// before STEP 6 contact validation runs).
func TestCreateOrderValidationOrderingSamePayloadReplayNotBroken(t *testing.T) {
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
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	}
	first, err := svc.CreateOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.CreateOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("replay: error = %v, want nil (same order)", err)
	}
	if first.ID != second.ID {
		t.Fatalf("replay returned different order: %q vs %q", first.ID, second.ID)
	}
}

// TestMulOverflow directly tests the mulOverflow helper with boundary
// values that would wrap silently on 64-bit int if the check used
// int64 intermediates.
func TestMulOverflow(t *testing.T) {
	t.Parallel()

	// MaxInt * 2 must overflow, not wrap to -2.
	_, err := mulOverflow(math.MaxInt, 2)
	if err != ErrOverflow {
		t.Fatalf("mulOverflow(MaxInt, 2): error = %v, want ErrOverflow", err)
	}

	// 2 * MaxInt must overflow (operand order independence).
	_, err = mulOverflow(2, math.MaxInt)
	if err != ErrOverflow {
		t.Fatalf("mulOverflow(2, MaxInt): error = %v, want ErrOverflow", err)
	}

	// MaxInt * 1 = MaxInt (no overflow).
	r, err := mulOverflow(math.MaxInt, 1)
	if err != nil || r != math.MaxInt {
		t.Fatalf("mulOverflow(MaxInt, 1): r=%d err=%v, want MaxInt nil", r, err)
	}

	// MaxInt * 0 = 0 (no overflow).
	r, err = mulOverflow(math.MaxInt, 0)
	if err != nil || r != 0 {
		t.Fatalf("mulOverflow(MaxInt, 0): r=%d err=%v, want 0 nil", r, err)
	}

	// Large but safe: 1<<31 * 1<<31 = 1<<62 (fits in int64 on 64-bit).
	r, err = mulOverflow(1<<31, 1<<31)
	if err != nil || r != 1<<62 {
		t.Fatalf("mulOverflow(1<<31, 1<<31): r=%d err=%v, want 1<<62 nil", r, err)
	}

	// MinInt * -1 would overflow (|MinInt| = MaxInt+1).
	_, err = mulOverflow(math.MinInt, -1)
	if err != ErrOverflow {
		t.Fatalf("mulOverflow(MinInt, -1): error = %v, want ErrOverflow", err)
	}
}

// TestAddOverflow directly tests the addOverflow helper with boundary
// values that would wrap silently on 64-bit int if the check used
// int64 intermediates.
func TestAddOverflow(t *testing.T) {
	t.Parallel()

	// MaxInt + 1 must overflow, not wrap to MinInt.
	_, err := addOverflow(math.MaxInt, 1)
	if err != ErrOverflow {
		t.Fatalf("addOverflow(MaxInt, 1): error = %v, want ErrOverflow", err)
	}

	// 1 + MaxInt must overflow (operand order independence).
	_, err = addOverflow(1, math.MaxInt)
	if err != ErrOverflow {
		t.Fatalf("addOverflow(1, MaxInt): error = %v, want ErrOverflow", err)
	}

	// MinInt + (-1) must overflow.
	_, err = addOverflow(math.MinInt, -1)
	if err != ErrOverflow {
		t.Fatalf("addOverflow(MinInt, -1): error = %v, want ErrOverflow", err)
	}

	// MaxInt + 0 = MaxInt (no overflow).
	r, err := addOverflow(math.MaxInt, 0)
	if err != nil || r != math.MaxInt {
		t.Fatalf("addOverflow(MaxInt, 0): r=%d err=%v, want MaxInt nil", r, err)
	}

	// MaxInt + (-1) = MaxInt-1 (different sign, no overflow).
	r, err = addOverflow(math.MaxInt, -1)
	if err != nil || r != math.MaxInt-1 {
		t.Fatalf("addOverflow(MaxInt, -1): r=%d err=%v, want MaxInt-1 nil", r, err)
	}
}

// TestCreateOrderOverflowInSubtotal verifies that an overflow in the
// subtotal accumulation (price * quantity sum) returns ErrOverflow,
// not a wrapped negative value. This uses a product with MaxInt price
// and quantity 2 to trigger mulOverflow in resolveItems.
func TestCreateOrderOverflowInSubtotal(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-OVF", Name: "Overflow", Slug: "overflow", Status: "active", Stock: 100, Price: math.MaxInt})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Test",
		Email:           "test@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-OVF", Quantity: 2}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflow in subtotal: error = %v, want ErrOverflow", err)
	}
}

// TestQuoteOverflowInSubtotal verifies that Quote also detects subtotal
// overflow and returns ErrOverflow (not a 500 or wrapped value).
func TestQuoteOverflowInSubtotal(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-OVF", Name: "Overflow", Slug: "overflow", Status: "active", Stock: 100, Price: math.MaxInt})
	seedReadyPaymentMethod(t, store)

	_, err := svc.Quote(context.Background(), QuoteInput{
		Items:          []OrderItem{{SKU: "SKU-OVF", Quantity: 2}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
	})
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Quote overflow in subtotal: error = %v, want ErrOverflow", err)
	}
}

// TestCreateOrderExistingKeyEmptyItemsConflicts verifies that a replay
// with the same key but DIFFERENT payload that has empty items returns
// ErrIdempotencyConflict, NOT "order must contain at least one item".
// The idempotency check (STEP 3) runs BEFORE the items-non-empty check
// (STEP 4), so the client learns the key was already used.
func TestCreateOrderExistingKeyEmptyItemsConflicts(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	key := testKey()

	// Create a valid order with key K.
	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Alice",
		Email:           "alice@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  key,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Replay with same key K but DIFFERENT payload with empty items.
	// Must return ErrIdempotencyConflict, NOT "order must contain at
	// least one item".
	_, err = svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Alice",
		Email:           "alice@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           nil, // empty — would fail items check on new order path
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  key, // same key, different payload
	})
	if err != ErrIdempotencyConflict {
		t.Fatalf("existing key + empty items: error = %v, want ErrIdempotencyConflict", err)
	}
}

// TestCreateOrderExistingKeyBlankCustomerNameConflicts verifies that a
// replay with the same key but DIFFERENT payload that has a blank
// customer_name returns ErrIdempotencyConflict, NOT "customer_name is
// required". The idempotency check (STEP 3) runs BEFORE the
// customer_name-non-blank check (STEP 4).
func TestCreateOrderExistingKeyBlankCustomerNameConflicts(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	key := testKey()

	// Create a valid order with key K.
	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Alice",
		Email:           "alice@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  key,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Replay with same key K but DIFFERENT payload with blank customer_name.
	// Must return ErrIdempotencyConflict, NOT "customer_name is required".
	_, err = svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "  ", // blank after trim — would fail on new order path
		Email:           "alice@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  key, // same key, different payload
	})
	if err != ErrIdempotencyConflict {
		t.Fatalf("existing key + blank customer_name: error = %v, want ErrIdempotencyConflict", err)
	}
}

// TestCreateOrderBlankEmailRejected verifies that a blank email on the
// new-order path is rejected with ErrInvalidEmail.
func TestCreateOrderBlankEmailRejected(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Test",
		Email:           "",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != ErrInvalidEmail {
		t.Fatalf("blank email: error = %v, want ErrInvalidEmail", err)
	}
}

// TestCreateOrderBlankPhoneRejected verifies that a blank phone on the
// new-order path is rejected with ErrInvalidPhone.
func TestCreateOrderBlankPhoneRejected(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Test",
		Email:           "test@example.com",
		Phone:           "",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != ErrInvalidPhone {
		t.Fatalf("blank phone: error = %v, want ErrInvalidPhone", err)
	}
}

// TestCreateOrderBlankShippingAddressRejected verifies that a blank
// shipping_address on the new-order path is rejected with
// ErrInvalidShippingAddress.
func TestCreateOrderBlankShippingAddressRejected(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Test",
		Email:           "test@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != ErrInvalidShippingAddress {
		t.Fatalf("blank shipping_address: error = %v, want ErrInvalidShippingAddress", err)
	}
}

// TestCreateOrderTrimSpacePersisted verifies that surrounding whitespace
// on all string fields is trimmed before persistence. The persisted order
// should have trimmed values, not the raw input.
func TestCreateOrderTrimSpacePersisted(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "  Alice  ",
		Email:           "  alice@example.com  ",
		Phone:           "  02-1234-5678  ",
		ShippingAddress: "  123 Main St, Taipei  ",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "  home_delivery  ",
		PaymentMethod:   "  atm  ",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	if order.CustomerName != "Alice" {
		t.Errorf("CustomerName = %q, want %q (trimmed)", order.CustomerName, "Alice")
	}
	if order.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q (trimmed)", order.Email, "alice@example.com")
	}
	if order.Phone != "02-1234-5678" {
		t.Errorf("Phone = %q, want %q (trimmed)", order.Phone, "02-1234-5678")
	}
	if order.ShippingAddress != "123 Main St, Taipei" {
		t.Errorf("ShippingAddress = %q, want %q (trimmed)", order.ShippingAddress, "123 Main St, Taipei")
	}
	if order.ShippingMethod != "home_delivery" {
		t.Errorf("ShippingMethod = %q, want %q (trimmed)", order.ShippingMethod, "home_delivery")
	}
	if order.PaymentMethod != "atm" {
		t.Errorf("PaymentMethod = %q, want %q (trimmed)", order.PaymentMethod, "atm")
	}

	// Verify persisted in DB too.
	persisted, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if persisted.CustomerName != "Alice" {
		t.Errorf("persisted CustomerName = %q, want %q", persisted.CustomerName, "Alice")
	}
	if persisted.Email != "alice@example.com" {
		t.Errorf("persisted Email = %q, want %q", persisted.Email, "alice@example.com")
	}
	if persisted.Phone != "02-1234-5678" {
		t.Errorf("persisted Phone = %q, want %q", persisted.Phone, "02-1234-5678")
	}
	if persisted.ShippingAddress != "123 Main St, Taipei" {
		t.Errorf("persisted ShippingAddress = %q, want %q", persisted.ShippingAddress, "123 Main St, Taipei")
	}
}

// TestCreateOrderRejectsDuplicateSKU tests that a new order with duplicate
// SKUs in the input is rejected with ErrInvalidAdminInput (400), not a 500
// from the order_items unique index violation.
func TestCreateOrderRejectsDuplicateSKU(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p-dup", SKU: "SKU-DUP-ORDER", Name: "Dup", Slug: "dup", Status: "active", Stock: 100, Price: 100, Category: "apparel"})

	in := OrderInput{
		CustomerName: "Dup Tester", Email: "dup@example.com", Phone: "555-0100",
		ShippingAddress: "123 Main St", ShippingMethod: "home_delivery",
		PaymentMethod: "atm", IdempotencyKey: "dup-sku-" + testKey(),
		Items: []OrderItem{
			{SKU: "SKU-DUP-ORDER", Quantity: 1},
			{SKU: "SKU-DUP-ORDER", Quantity: 2},
		},
	}
	_, err := svc.CreateOrder(context.Background(), in)
	if !errors.Is(err, ErrInvalidAdminInput) {
		t.Fatalf("CreateOrder with duplicate SKU: error = %v, want ErrInvalidAdminInput", err)
	}
}

// TestCreateOrderRejectsDuplicateSKUAfterTrim tests that duplicate SKUs
// are detected after trimming (e.g. "  SKU-X  " and "SKU-X" are the same).
func TestCreateOrderRejectsDuplicateSKUAfterTrim(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p-trim", SKU: "SKU-TRIM", Name: "Trim", Slug: "trim", Status: "active", Stock: 100, Price: 100, Category: "apparel"})

	in := OrderInput{
		CustomerName: "Trim Tester", Email: "trim@example.com", Phone: "555-0100",
		ShippingAddress: "123 Main St", ShippingMethod: "home_delivery",
		PaymentMethod: "atm", IdempotencyKey: "trim-sku-" + testKey(),
		Items: []OrderItem{
			{SKU: "  SKU-TRIM  ", Quantity: 1},
			{SKU: "SKU-TRIM", Quantity: 2},
		},
	}
	_, err := svc.CreateOrder(context.Background(), in)
	if !errors.Is(err, ErrInvalidAdminInput) {
		t.Fatalf("CreateOrder with trimmed duplicate SKU: error = %v, want ErrInvalidAdminInput", err)
	}
}
