package commerce

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// testDBCounter gives each test a uniquely-named in-memory SQLite database so
// parallel tests do not collide on schema_migrations or shared catalog state.
var testDBCounter uint64

// testKeyCounter generates unique idempotency keys for test OrderInputs.
// Each test that creates an order must set a non-empty IdempotencyKey
// (the service rejects empty keys with ErrMissingIdempotencyKey).
var testKeyCounter uint64

// seedDefaultShippingMethods inserts the persisted shipping rows used by
// quote/order tests. Fees live in shipping_methods, not a test seam.
func seedDefaultShippingMethods(t *testing.T, s Store) {
	t.Helper()
	threshold := 1500
	rows := []ShippingMethod{
		{ID: "sm-home", Method: "home_delivery", Label: "宅配", Fee: 120, FreeThreshold: &threshold, Enabled: true, SortOrder: 1, Version: 1, UpdatedUnix: 1},
		{ID: "sm-711", Method: "store_711", Label: "7-11", Fee: 60, Enabled: true, SortOrder: 2, Version: 1, UpdatedUnix: 1},
		{ID: "sm-fm", Method: "store_familymart", Label: "全家", Fee: 60, Enabled: true, SortOrder: 3, Version: 1, UpdatedUnix: 1},
	}
	for _, m := range rows {
		if err := s.CreateShippingMethod(context.Background(), m); err != nil && !errors.Is(err, ErrDuplicateShippingMethod) {
			t.Fatalf("seed shipping method %s: %v", m.Method, err)
		}
	}
}

// testKey returns a unique idempotency key for tests. Uses an atomic
// counter so parallel tests don't collide.
func testKey() string {
	n := atomic.AddUint64(&testKeyCounter, 1)
	return fmt.Sprintf("test-key-%d", n)
}

// newTestStore opens an in-memory SQLite database, applies all migrations, and
// returns a SQLStore ready for commerce tests.
func newTestStore(t *testing.T) Store {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:commerce_test_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return NewSQLStore(db, dialect)
}

// adminPrincipal is a principal that holds every commerce capability.
func adminPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "admin",
		Role:   "admin",
		Capabilities: []string{
			"twcommerce.admin",
			"twcommerce.create",
			"twcommerce.update",
			"twcommerce.delete",
			"twcommerce.read",
			"orders.returns",
			"inventory.adjust",
		},
	}
}

// emptyPrincipal has no capabilities and is used to assert forbidden paths.
func emptyPrincipal() auth.Principal {
	return auth.Principal{UserID: "guest", Role: "viewer"}
}

// seedProduct inserts a product directly via the store for test setup.
func seedProduct(t *testing.T, s Store, p Product) {
	t.Helper()
	p.UpdatedUnix = 1
	if err := s.UpsertProduct(context.Background(), p); err != nil {
		t.Fatalf("seed product: %v", err)
	}
}

// seedReadyPaymentMethod inserts an enabled, ready payment method for
// test setup. Tests that successfully create orders or quotes must seed
// a ready payment method and include PaymentMethod in the input, since
// the server validates payment methods against the payment_methods table.
func seedReadyPaymentMethod(t *testing.T, s Store) {
	t.Helper()
	if err := s.UpsertPaymentMethod(context.Background(), PaymentMethod{
		ID:              "atm",
		Method:          "atm",
		ProviderLabel:   "ATM 轉帳",
		Environment:     "sandbox",
		ReadinessStatus: "ready",
		Enabled:         true,
		UpdatedUnix:     1,
	}); err != nil {
		t.Fatalf("seed payment method: %v", err)
	}
}

func TestProductStatusAutoDerivation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		stock int
		want  string
	}{
		{stock: 0, want: "out_of_stock"},
		{stock: 3, want: "low_stock"},
		{stock: 5, want: "active"},
		{stock: 100, want: "active"},
	}
	for _, tc := range cases {
		got := deriveProductStatus("active", tc.stock)
		if got != tc.want {
			t.Errorf("deriveProductStatus(active, %d) = %q, want %q", tc.stock, got, tc.want)
		}
	}
	// draft is never auto-derived; it stays operator-set.
	if got := deriveProductStatus("draft", 0); got != "draft" {
		t.Errorf("deriveProductStatus(draft, 0) = %q, want draft", got)
	}
}

func TestCreateProductDefaultsToDraft(t *testing.T) {
	t.Parallel()
	svc := NewService(newTestStore(t))

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-1", Name: "Widget", Slug: "widget", Price: 100, Stock: 10,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if p.Status != "draft" {
		t.Fatalf("status = %q, want draft", p.Status)
	}
}

func TestCreateProductRejectsUnverifiedProductImageKeys(t *testing.T) {
	t.Parallel()
	// Without a media verifier wired, any product_images payload
	// is rejected with ErrUnverifiedMedia. This proves the service
	// does not accept arbitrary keys without verification.
	store := newTestStore(t)
	svc := NewService(store) // no media verifier wired
	_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-MEDIA", Name: "Media", Slug: "media", Price: 100,
		ProductImages: []ProductImageInput{
			{Key: "verified/product-images/someone-else/forged.jpg"},
		},
	})
	if !errors.Is(err, ErrUnverifiedMedia) {
		t.Fatalf("error = %v, want ErrUnverifiedMedia", err)
	}
	if _, err := store.GetProductBySlug(context.Background(), "media"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("product persisted after rejected media: %v", err)
	}
}

func TestCreateProductRejectsArbitraryURLInImageField(t *testing.T) {
	t.Parallel()
	// ProductInput no longer has Image/Images fields. The handler
	// uses httpx.DecodeJSON with DisallowUnknownFields, so any
	// payload containing image or images is rejected with 400.
	// This test proves the rejection at the JSON decoding level.
	dec := json.NewDecoder(strings.NewReader(`{"image":"https://attacker.example/forged.jpg","sku":"S","name":"N","slug":"s"}`))
	dec.DisallowUnknownFields()
	var in ProductInput
	if err := dec.Decode(&in); err == nil {
		t.Fatal("payload with image field accepted; want rejection")
	}
}

func TestUpdateProductClearsLegacyImageColumnsWithoutProductImages(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	existing := Product{
		ID: "p-media", SKU: "SKU-MEDIA", Name: "Media", Slug: "media", Status: "draft",
		Price: 100, Stock: 2, Category: "apparel", Image: "/legacy/main.jpg",
		Images: `["/legacy/main.jpg","/legacy/side.jpg"]`,
	}
	seedProduct(t, store, existing)

	// ProductInput no longer has Image/Images fields. The legacy
	// flat columns are always written empty — no legacy backfill.
	// Update should succeed, clear legacy image/images, and not
	// preserve the old unverified URLs.
	updated, err := svc.UpdateProduct(context.Background(), adminPrincipal(), existing.ID, ProductInput{
		SKU: existing.SKU, Name: "Renamed", Slug: existing.Slug, Status: existing.Status,
		Price: existing.Price, Stock: existing.Stock, Category: existing.Category,
	})
	if err != nil {
		t.Fatalf("update without image fields: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Fatalf("updated name = %q, want Renamed", updated.Name)
	}
	if updated.Image != "" {
		t.Fatalf("legacy image should be cleared, got %q", updated.Image)
	}
	if updated.Images != "[]" {
		t.Fatalf("legacy images should be cleared to [], got %q", updated.Images)
	}
}

func TestCreateProductForbiddenWithoutCapability(t *testing.T) {
	t.Parallel()
	svc := NewService(newTestStore(t))

	_, err := svc.CreateProduct(context.Background(), emptyPrincipal(), ProductInput{
		SKU: "SKU-1", Name: "Widget", Slug: "widget",
	})
	if err != ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestUpdateProductStatusDerivesFromStock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-1", Name: "Widget", Slug: "widget", Status: "draft", Stock: 0, Price: 100})
	seedReadyPaymentMethod(t, store)

	p, err := svc.UpdateProductStatus(context.Background(), adminPrincipal(), "p1", "active")
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if p.Status != "out_of_stock" {
		t.Fatalf("status = %q, want out_of_stock (stock=0)", p.Status)
	}
}

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

func TestOrderStateMachineValidTransitions(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
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

	for _, step := range []struct{ next string }{
		{"processing"},
		{"shipped"},
		{"delivered"},
	} {
		order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step.next)
		if err != nil {
			t.Fatalf("transition -> %s: %v", step.next, err)
		}
		if order.Status != step.next {
			t.Fatalf("status = %q, want %q", order.Status, step.next)
		}
	}
}

func TestOrderStateMachineInvalidTransition(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
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

	_, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "delivered")
	if err != ErrInvalidTransition {
		t.Fatalf("error = %v, want ErrInvalidTransition", err)
	}
}

// TestStaleVersionRejectsTransitionWithoutSideEffect proves AC-009: a stale
// expected_version on a LEGAL transition returns ErrStaleVersion (409) without
// changing the order status, incrementing the version, or appending a success
// event. This is distinct from ErrInvalidTransition (illegal target state).
func TestStaleVersionRejectsTransitionWithoutSideEffect(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Bob", Email: "bob@example.com", Phone: "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei", Items: []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.Version != 1 {
		t.Fatalf("initial version = %d, want 1", order.Version)
	}

	// Advance to processing so version becomes 2.
	order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "processing")
	if err != nil {
		t.Fatalf("advance to processing: %v", err)
	}
	if order.Version != 2 {
		t.Fatalf("version after transition = %d, want 2", order.Version)
	}

	// Stale expected_version (1) on a LEGAL transition (processing -> shipped).
	_, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, 1, "shipped")
	if !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("stale version error = %v, want ErrStaleVersion", err)
	}

	// The order must be unchanged: status still processing, version still 2.
	persisted, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if persisted.Status != "processing" {
		t.Fatalf("status after stale attempt = %q, want processing", persisted.Status)
	}
	if persisted.Version != 2 {
		t.Fatalf("version after stale attempt = %d, want 2", persisted.Version)
	}

	// No success event appended for the rejected stale attempt.
	events, err := store.ListOrderEvents(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("event count after stale attempt = %d, want 2 (created + processing)", len(events))
	}
}

// TestVersionIncrementsOnEachTransition proves the aggregate version increments
// on every successful fulfillment and return transition, so a subsequent
// mutation with the old version is rejected as stale.
func TestVersionIncrementsOnEachTransition(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Carol", Email: "carol@example.com", Phone: "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei", Items: []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.Version != 1 {
		t.Fatalf("initial version = %d, want 1", order.Version)
	}

	// Each fulfillment transition increments version by 1.
	order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "processing")
	if err != nil {
		t.Fatalf("transition to processing: %v", err)
	}
	if order.Version != 2 {
		t.Fatalf("version after processing = %d, want 2", order.Version)
	}
	order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "shipped")
	if err != nil {
		t.Fatalf("transition to shipped: %v", err)
	}
	if order.Version != 3 {
		t.Fatalf("version after shipped = %d, want 3", order.Version)
	}
	order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "delivered")
	if err != nil {
		t.Fatalf("transition to delivered: %v", err)
	}
	if order.Version != 4 {
		t.Fatalf("version after delivered = %d, want 4", order.Version)
	}

	// Return transitions also increment the same aggregate version.
	order, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "requested")
	if err != nil {
		t.Fatalf("request return: %v", err)
	}
	if order.Version != 5 {
		t.Fatalf("version after return requested = %d, want 5", order.Version)
	}
	order, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "approved")
	if err != nil {
		t.Fatalf("approve return: %v", err)
	}
	if order.Version != 6 {
		t.Fatalf("version after return approved = %d, want 6", order.Version)
	}

	// Using the pre-return version (4) on a legal return transition must fail.
	_, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, 4, "received")
	if !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("stale return version error = %v, want ErrStaleVersion", err)
	}
}

func TestOrderCancellationRestocks(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 3}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 7 {
		t.Fatalf("stock after order = %d, want 7", p.Stock)
	}

	if _, err := svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "cancelled"); err != nil {
		t.Fatalf("cancel order: %v", err)
	}

	p, _ = store.GetProduct(context.Background(), "p1")
	if p.Stock != 10 {
		t.Fatalf("stock after cancel = %d, want 10 (restocked)", p.Stock)
	}
}

func TestConcurrentOrderCancellationRestocksOnce(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Bob", Email: "bob@example.com", Phone: "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei", Items: []OrderItem{{SKU: "SKU-A", Quantity: 3}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			_, transitionErr := svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "cancelled")
			errs <- transitionErr
		}()
	}
	close(start)

	var successes, stale int
	for i := 0; i < 2; i++ {
		switch err := <-errs; {
		case err == nil:
			successes++
		case errors.Is(err, ErrStaleVersion):
			stale++
		default:
			t.Fatalf("cancel error = %v", err)
		}
	}
	if successes != 1 || stale != 1 {
		t.Fatalf("cancel results: successes=%d stale=%d, want 1/1", successes, stale)
	}
	p, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if p.Stock != 10 {
		t.Fatalf("stock after concurrent cancel = %d, want 10", p.Stock)
	}
}

func TestOrderTransitionRollsBackWhenAuditEventFails(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Bob", Email: "bob@example.com", Phone: "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei", Items: []OrderItem{{SKU: "SKU-A", Quantity: 3}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Reusing the creation-event ID forces the final event insert to fail. The
	// status update and restock must roll back with it.
	err = store.TransitionOrderStatus(context.Background(), order.ID, order.Version, "cancelled", order.TimelineJSON, order.UpdatedUnix+1,
		[]OrderItem{{SKU: "SKU-A", Quantity: 3}}, OrderEvent{
			ID: order.ID + ":created", OrderID: order.ID, EventType: "order_status",
			FromStatus: "pending", ToStatus: "cancelled", CreatedUnix: order.UpdatedUnix + 1,
		})
	if err == nil {
		t.Fatal("transition error = nil, want duplicate audit event failure")
	}
	persisted, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if persisted.Status != "pending" {
		t.Fatalf("status after rollback = %q, want pending", persisted.Status)
	}
	p, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if p.Stock != 7 {
		t.Fatalf("stock after rollback = %d, want 7", p.Stock)
	}
	events, err := store.ListOrderEvents(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Sequence != 1 || events[0].ToStatus != "pending" {
		t.Fatalf("events after rollback = %#v, want only creation event", events)
	}
}

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

func TestUpdateOrderStatusForbiddenWithoutCapability(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
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
	_, err = svc.UpdateOrderStatus(context.Background(), emptyPrincipal(), order.ID, order.Version, "processing")
	if err != ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestReturnStateMachineDoesNotRestockOnReceipt(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Dave",
		Email:           "dave@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 2}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	// Advance to delivered so a return can be received.
	for _, step := range []struct{ next string }{
		{"processing"},
		{"shipped"},
		{"delivered"},
	} {
		order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step.next)
		if err != nil {
			t.Fatalf("transition -> %s: %v", step.next, err)
		}
	}

	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Fatalf("stock before return = %d, want 8", p.Stock)
	}

	if order, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "requested"); err != nil {
		t.Fatalf("request return: %v", err)
	}
	if order, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "approved"); err != nil {
		t.Fatalf("approve return: %v", err)
	}

	p, _ = store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Fatalf("stock after return approval = %d, want 8 (not yet received)", p.Stock)
	}
	if _, err := svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "received"); err != nil {
		t.Fatalf("receive return: %v", err)
	}

	p, _ = store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Fatalf("stock after return receipt = %d, want 8 (inspection/restock is separate)", p.Stock)
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

func TestOrderEventsPreserveTransitionSequenceAndReason(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Eve", Email: "eve@example.com", Phone: "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei", Items: []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order, err = svc.UpdateOrderStatusWithNote(context.Background(), adminPrincipal(), order.ID, order.Version, "processing", "packing started"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if order, err = svc.UpdateOrderStatusWithNote(context.Background(), adminPrincipal(), order.ID, order.Version, "shipped", "carrier accepted parcel"); err != nil {
		t.Fatalf("ship order: %v", err)
	}
	// Stale version: legal transition (shipped -> delivered) but wrong
	// expected_version must return ErrStaleVersion without appending an event.
	if _, err := svc.UpdateOrderStatusWithNote(context.Background(), adminPrincipal(), order.ID, order.Version-1, "delivered", "stale retry"); !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("stale version error = %v, want ErrStaleVersion", err)
	}

	events, err := store.ListOrderEvents(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("event count = %d, want 3", len(events))
	}
	want := []struct {
		eventType, from, to, reason string
	}{
		{"order_status", "", "pending", "order placed"},
		{"order_status", "pending", "processing", "packing started"},
		{"order_status", "processing", "shipped", "carrier accepted parcel"},
	}
	for i, event := range events {
		if event.Sequence != i+1 {
			t.Fatalf("event %d sequence = %d, want %d", i, event.Sequence, i+1)
		}
		if event.EventType != want[i].eventType || event.FromStatus != want[i].from || event.ToStatus != want[i].to || event.Reason != want[i].reason {
			t.Fatalf("event %d = %#v, want %#v", i, event, want[i])
		}
		if event.ActorUserID == "" {
			t.Fatalf("event %d has empty actor_user_id", i)
		}
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

// TestStockDecrementRollbackOnInsufficientStock proves that if stock
// decrement fails for any item, the entire transaction rolls back —
// no order is persisted and no stock is decremented.
func TestStockDecrementRollbackOnInsufficientStock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	// Product with stock=1 but we try to order 2.
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 1, Price: 100})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:   "Eve",
		Items:          []OrderItem{{SKU: "SKU-A", Quantity: 2}},
		IdempotencyKey: "idem-rollback",

		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("error = %v, want ErrInsufficientStock", err)
	}

	// Stock should be unchanged (no partial decrement).
	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 1 {
		t.Errorf("stock after failed order = %d, want 1 (no side effect)", p.Stock)
	}

	// No order should exist with this idempotency key.
	_, err = store.FindOrderByIdempotencyKey(context.Background(), "idem-rollback")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("order should not exist after rollback, got err: %v", err)
	}
}

// TestGetOrderForGuestValidToken proves that a guest can access their
// own order with the correct opaque access token, and all PII is masked.
func TestGetOrderForGuestValidToken(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Frank",
		Email:           "frank@example.com",
		Phone:           "0912345678",
		ShippingAddress: "台北市大安區忠孝東路一段100號",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},

		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.AccessToken == "" {
		t.Fatal("access token should be generated at order creation")
	}

	// Guest with valid token can access the order.
	got, err := svc.GetOrderForGuest(context.Background(), order.ID, order.AccessToken)
	if err != nil {
		t.Fatalf("get order for guest: %v", err)
	}
	// PII should be masked.
	if got.Phone == "0912345678" {
		t.Errorf("phone should be masked, got %q", got.Phone)
	}
	if got.Email == "frank@example.com" {
		t.Errorf("email should be masked, got %q", got.Email)
	}
	if got.ShippingAddress == "台北市大安區忠孝東路一段100號" {
		t.Errorf("shipping address should be masked, got %q", got.ShippingAddress)
	}
	if got.ShippingAddress == "" {
		t.Errorf("shipping address should not be empty (city/district kept)")
	}
	// Access token should never be in the response.
	if got.AccessToken != "" {
		t.Errorf("access token should never be in response, got %q", got.AccessToken)
	}
	// Order ID and status should be intact.
	if got.ID != order.ID {
		t.Errorf("order ID = %q, want %q", got.ID, order.ID)
	}
	if got.Status != "pending" {
		t.Errorf("status = %q, want pending", got.Status)
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

// TestGetOrderForMemberOwnerSucceeds proves that the owning member can
// access their own order with PII masked.
func TestGetOrderForMemberOwnerSucceeds(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	judyPrincipal := auth.Principal{UserID: "member-judy", Role: "member"}
	order, err := svc.CreateOrderForMember(context.Background(), judyPrincipal, OrderInput{
		CustomerName:    "Judy",
		Email:           "judy@example.com",
		Phone:           "0987654321",
		ShippingAddress: "台北市信義區松仁路50號",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},

		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order for member: %v", err)
	}
	if order.MemberID != "member-judy" {
		t.Errorf("MemberID = %q, want member-judy", order.MemberID)
	}

	got, err := svc.GetOrderForMember(context.Background(), judyPrincipal, order.ID)
	if err != nil {
		t.Fatalf("owner access should succeed: %v", err)
	}
	if got.Phone == "0987654321" {
		t.Errorf("phone should be masked, got %q", got.Phone)
	}
	if got.Email == "judy@example.com" {
		t.Errorf("email should be masked, got %q", got.Email)
	}
	if got.ShippingAddress == "台北市信義區松仁路50號" {
		t.Errorf("shipping address should be masked, got %q", got.ShippingAddress)
	}
	if got.AccessToken != "" {
		t.Errorf("access token should never be in response")
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

// TestListOrdersForMemberExcludesOtherMembers proves that member-scoped
// listing does not return orders from other members.
func TestListOrdersForMemberExcludesOtherMembers(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	judyPrincipal := auth.Principal{UserID: "member-judy", Role: "member"}
	_, err := svc.CreateOrderForMember(context.Background(), judyPrincipal, OrderInput{
		CustomerName:    "Judy",
		Email:           "judy@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order for judy: %v", err)
	}

	karlPrincipal := auth.Principal{UserID: "member-karl", Role: "member"}
	_, err = svc.CreateOrderForMember(context.Background(), karlPrincipal, OrderInput{
		CustomerName:    "Karl",
		Email:           "karl@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order for karl: %v", err)
	}

	// member-judy should only see their own order.
	orders, err := svc.ListOrdersForMember(context.Background(), "member-judy")
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order for member-judy, got %d", len(orders))
	}
	if orders[0].CustomerName != "Judy" {
		t.Errorf("customer name = %q, want Judy", orders[0].CustomerName)
	}
}

// TestOrderIDIsRandomAndUnguessable proves that order IDs are not
// sequential (the old COUNT+1 scheme was enumerable).
func TestOrderIDIsRandomAndUnguessable(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 100, Price: 100})
	seedReadyPaymentMethod(t, store)

	ids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		order, err := svc.CreateOrder(context.Background(), OrderInput{
			CustomerName:    fmt.Sprintf("Customer %d", i),
			Email:           fmt.Sprintf("c%d@example.com", i),
			Phone:           "02-1234-5678",
			ShippingAddress: "123 Main St, Taipei",
			Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
			ShippingMethod:  "home_delivery",
			PaymentMethod:   "atm",
			IdempotencyKey:  testKey(),
		})
		if err != nil {
			t.Fatalf("create order %d: %v", i, err)
		}
		// Order ID should start with "TW-" and have a long hex suffix.
		if len(order.ID) < 20 {
			t.Errorf("order ID %q is too short (should be random hex, not sequential)", order.ID)
		}
		if ids[order.ID] {
			t.Errorf("duplicate order ID %q — random generation should be unique", order.ID)
		}
		ids[order.ID] = true
		// Access token should also be generated and unguessable.
		if len(order.AccessToken) < 32 {
			t.Errorf("access token %q is too short (should be 64 hex chars)", order.AccessToken)
		}
	}
}

// TestAccessTokenHashedNotPlaintextInDB proves that the DB stores a
// SHA-256 hash of the access token, not the plaintext. The plaintext is
// only returned once at creation and is never persisted.
func TestAccessTokenHashedNotPlaintextInDB(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Token Test",
		Email:           "token@test.local",
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
	if order.AccessToken == "" {
		t.Fatal("plaintext access token should be returned at creation")
	}

	// Read the order directly from the store (bypassing the service's
	// masking). The AccessTokenHash field should be a SHA-256 hex digest
	// (64 chars), NOT the plaintext token.
	stored, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order from store: %v", err)
	}
	if stored.AccessTokenHash == "" {
		t.Fatal("AccessTokenHash should be populated in the DB")
	}
	if len(stored.AccessTokenHash) != 64 {
		t.Errorf("AccessTokenHash length = %d, want 64 (SHA-256 hex)", len(stored.AccessTokenHash))
	}
	if stored.AccessTokenHash == order.AccessToken {
		t.Error("AccessTokenHash should NOT equal the plaintext token (must be hashed)")
	}
	// The plaintext token should NOT be stored in any field.
	if stored.AccessToken != "" {
		t.Errorf("stored AccessToken should be empty (plaintext not persisted), got %q", stored.AccessToken)
	}
	// The hash should match what hashAccessToken produces.
	expectedHash := hashAccessToken(order.AccessToken)
	if stored.AccessTokenHash != expectedHash {
		t.Errorf("AccessTokenHash = %q, want %q", stored.AccessTokenHash, expectedHash)
	}
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

// TestCreateProductRejectsReservedSlug proves that reserved route segments
// cannot be used as product slugs.
func TestCreateProductRejectsReservedSlug(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	reserved := []string{"api", "admin", "assets", "articles", "content", "products", "category", "orders", "checkout", "account"}
	for _, slug := range reserved {
		_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
			SKU: "SKU-R-" + slug, Name: "Reserved", Slug: slug, Price: 100, Stock: 1,
		})
		if err == nil {
			t.Errorf("expected error for reserved slug %q, got nil", slug)
		}
	}
}

// TestCreateProductRejectsTraversalSlug proves that path traversal attempts
// in slugs are rejected.
func TestCreateProductRejectsTraversalSlug(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	traversal := []string{"../etc", "..\\windows", "foo/bar", "foo/../../bar"}
	for _, slug := range traversal {
		_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
			SKU: "SKU-T-" + slug, Name: "Traversal", Slug: slug, Price: 100, Stock: 1,
		})
		if err == nil {
			t.Errorf("expected error for traversal slug %q, got nil", slug)
		}
	}
}

// TestCreateProductRejectsInvalidSlugCharacters proves that slugs with
// invalid characters (uppercase, spaces, special chars) are rejected.
func TestCreateProductRejectsInvalidSlugCharacters(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	invalid := []string{"Upper-Case", "with space", "with.dot", "with@at", "-leading", "trailing-"}
	for _, slug := range invalid {
		_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
			SKU: "SKU-I-" + slug, Name: "Invalid", Slug: slug, Price: 100, Stock: 1,
		})
		if err == nil {
			t.Errorf("expected error for invalid slug %q, got nil", slug)
		}
	}
}

// TestCreateProductAcceptsValidSlug proves that valid slugs are accepted.
func TestCreateProductAcceptsValidSlug(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	valid := []string{"linen-shirt", "oak-desk-01", "brass_lamp", "cotton-tee-v2"}
	for _, slug := range valid {
		_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
			SKU: "SKU-V-" + slug, Name: "Valid", Slug: slug, Price: 100, Stock: 1,
		})
		if err != nil {
			t.Errorf("unexpected error for valid slug %q: %v", slug, err)
		}
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

// faultInjectStore wraps a Store and injects failures into specific methods
// for testing rollback/atomicity guarantees.
type faultInjectStore struct {
	Store
	failOnCreateTx        bool
	failGetActivePromo    bool
	failListProductImages bool
	failBulkUpdate        bool
	failUpsertMember      bool
	failUpdateMemberStat  bool
	failUpsertPromo       bool
	failUpsertPayment     bool
	failListShipping      bool
	failCreateShipping    bool
	failUpdateShipping    bool
}

func (f *faultInjectStore) CreateOrderTxWithIdempotency(ctx context.Context, o Order, items []OrderItem) (Order, bool, error) {
	if f.failOnCreateTx {
		return Order{}, false, fmt.Errorf("injected fault: simulated transaction failure")
	}
	return f.Store.CreateOrderTxWithIdempotency(ctx, o, items)
}

func (f *faultInjectStore) GetActivePromoByCode(ctx context.Context, code string, now int64) (Promo, error) {
	if f.failGetActivePromo {
		return Promo{}, fmt.Errorf("injected promo lookup failure")
	}
	return f.Store.GetActivePromoByCode(ctx, code, now)
}

func (f *faultInjectStore) ListProductImages(ctx context.Context, productID string) ([]ProductImage, error) {
	if f.failListProductImages {
		return nil, fmt.Errorf("injected fault: simulated ListProductImages DB failure")
	}
	return f.Store.ListProductImages(ctx, productID)
}

func (f *faultInjectStore) BulkUpdateProductStatus(ctx context.Context, ids []string, status string, updatedUnix int64) error {
	if f.failBulkUpdate {
		return fmt.Errorf("injected fault: simulated bulk update DB failure")
	}
	return f.Store.BulkUpdateProductStatus(ctx, ids, status, updatedUnix)
}

func (f *faultInjectStore) UpsertMember(ctx context.Context, m Member) error {
	if f.failUpsertMember {
		return fmt.Errorf("injected fault: simulated member upsert DB failure")
	}
	return f.Store.UpsertMember(ctx, m)
}

func (f *faultInjectStore) UpdateMemberStatus(ctx context.Context, id, status string, updatedUnix int64) error {
	if f.failUpdateMemberStat {
		return fmt.Errorf("injected fault: simulated member status update DB failure")
	}
	return f.Store.UpdateMemberStatus(ctx, id, status, updatedUnix)
}

func (f *faultInjectStore) UpsertPromo(ctx context.Context, p Promo) error {
	if f.failUpsertPromo {
		return fmt.Errorf("injected fault: simulated promo upsert DB failure")
	}
	return f.Store.UpsertPromo(ctx, p)
}

func (f *faultInjectStore) UpsertPaymentMethod(ctx context.Context, pm PaymentMethod) error {
	if f.failUpsertPayment {
		return fmt.Errorf("injected fault: simulated payment method upsert DB failure")
	}
	return f.Store.UpsertPaymentMethod(ctx, pm)
}

func (f *faultInjectStore) ListShippingMethods(ctx context.Context) ([]ShippingMethod, error) {
	if f.failListShipping {
		return nil, fmt.Errorf("injected fault: simulated shipping list DB failure")
	}
	return f.Store.ListShippingMethods(ctx)
}

func (f *faultInjectStore) CreateShippingMethod(ctx context.Context, m ShippingMethod) error {
	if f.failCreateShipping {
		return fmt.Errorf("injected fault: simulated shipping create DB failure")
	}
	return f.Store.CreateShippingMethod(ctx, m)
}

func (f *faultInjectStore) UpdateShippingMethod(ctx context.Context, m ShippingMethod, expectedVersion int) error {
	if f.failUpdateShipping {
		return fmt.Errorf("injected fault: simulated shipping update DB failure")
	}
	return f.Store.UpdateShippingMethod(ctx, m, expectedVersion)
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

func seedOneShipping(t *testing.T, s Store, m ShippingMethod) {
	t.Helper()
	if m.ID == "" {
		m.ID = "sm-" + m.Method
	}
	if m.Version == 0 {
		m.Version = 1
	}
	if m.UpdatedUnix == 0 {
		m.UpdatedUnix = 1
	}
	if err := s.CreateShippingMethod(context.Background(), m); err != nil {
		t.Fatalf("seed shipping %s: %v", m.Method, err)
	}
}

func quoteAndOrderInputs(sku, method, key string, qty int) (QuoteInput, OrderInput) {
	q := QuoteInput{
		Items:          []OrderItem{{SKU: sku, Quantity: qty}},
		ShippingMethod: method,
		PaymentMethod:  "atm",
	}
	o := OrderInput{
		CustomerName: "Alice", Email: "alice@example.com", Phone: "0912345678",
		ShippingAddress: "Taipei", Items: []OrderItem{{SKU: sku, Quantity: qty}},
		ShippingMethod: method, PaymentMethod: "atm", IdempotencyKey: key,
	}
	return q, o
}

func TestShippingFeeZeroNoThresholdAndThresholdBoundary(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-SHIP", Name: "Ship", Slug: "ship", Status: "active", Stock: 50, Price: 500})
	seedReadyPaymentMethod(t, store)
	threshold := 1500
	seedOneShipping(t, store, ShippingMethod{Method: "free_always", Label: "免運", Fee: 0, Enabled: true})
	seedOneShipping(t, store, ShippingMethod{Method: "flat", Label: "固定", Fee: 80, Enabled: true})
	seedOneShipping(t, store, ShippingMethod{Method: "home_delivery", Label: "宅配", Fee: 120, FreeThreshold: &threshold, Enabled: true})

	cases := []struct {
		method string
		qty    int
		want   int
	}{
		{method: "free_always", qty: 1, want: 0},
		{method: "flat", qty: 20, want: 80},
		{method: "home_delivery", qty: 2, want: 120}, // 1000 < 1500
		{method: "home_delivery", qty: 3, want: 0},   // 1500 == threshold
	}
	seedProduct(t, store, Product{ID: "p2", SKU: "SKU-1499", Name: "Near", Slug: "near", Status: "active", Stock: 5, Price: 1499})
	q1499, o1499 := quoteAndOrderInputs("SKU-1499", "home_delivery", testKey(), 1)
	quote, err := svc.Quote(context.Background(), q1499)
	if err != nil {
		t.Fatalf("threshold-1 quote: %v", err)
	}
	if quote.Shipping != 120 {
		t.Fatalf("threshold-1 shipping = %d, want 120", quote.Shipping)
	}
	order, err := svc.CreateOrder(context.Background(), o1499)
	if err != nil {
		t.Fatalf("threshold-1 order: %v", err)
	}
	if order.Shipping != 120 || order.Shipping != quote.Shipping {
		t.Fatalf("threshold-1 order shipping = %d quote = %d", order.Shipping, quote.Shipping)
	}

	for _, tc := range cases {
		qk, ok := quoteAndOrderInputs("SKU-SHIP", tc.method, testKey(), tc.qty)
		quote, err := svc.Quote(context.Background(), qk)
		if err != nil {
			t.Fatalf("%s qty=%d quote: %v", tc.method, tc.qty, err)
		}
		if quote.Shipping != tc.want {
			t.Fatalf("%s qty=%d shipping = %d, want %d", tc.method, tc.qty, quote.Shipping, tc.want)
		}
		order, err := svc.CreateOrder(context.Background(), ok)
		if err != nil {
			t.Fatalf("%s qty=%d order: %v", tc.method, tc.qty, err)
		}
		if order.Shipping != quote.Shipping || order.Total != quote.Total {
			t.Fatalf("%s qty=%d quote/order mismatch ship %d/%d total %d/%d",
				tc.method, tc.qty, quote.Shipping, order.Shipping, quote.Total, order.Total)
		}
	}

	member := auth.Principal{UserID: "member-ship", Role: "member"}
	qMember, oMember := quoteAndOrderInputs("SKU-SHIP", "flat", testKey(), 1)
	quote, err = svc.Quote(context.Background(), qMember)
	if err != nil {
		t.Fatal(err)
	}
	memberOrder, err := svc.CreateOrderForMember(context.Background(), member, oMember)
	if err != nil {
		t.Fatalf("member order: %v", err)
	}
	if memberOrder.Shipping != quote.Shipping || memberOrder.Shipping != 80 {
		t.Fatalf("member shipping = %d quote = %d, want 80", memberOrder.Shipping, quote.Shipping)
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

// seedSecondReadyPaymentMethod seeds a second enabled+ready payment method
// (credit_card) so tests can exercise different-payment-method idempotency
// conflicts.
func seedSecondReadyPaymentMethod(t *testing.T, s Store) {
	t.Helper()
	if err := s.UpsertPaymentMethod(context.Background(), PaymentMethod{
		ID:              "credit_card",
		Method:          "credit_card",
		ProviderLabel:   "信用卡",
		Environment:     "sandbox",
		ReadinessStatus: "ready",
		Enabled:         true,
		UpdatedUnix:     1,
	}); err != nil {
		t.Fatalf("seed second payment method: %v", err)
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

// legacyRequestFingerprint reproduces the pre-revision-5-blocker-3
// \n-separated "field=value" encoding. It is used ONLY in collision
// regression tests to prove that the old scheme was vulnerable.
// DO NOT use this in production code.
func legacyRequestFingerprint(in OrderInput, memberID string) string {
	type itemKey struct {
		SKU      string
		Quantity int
	}
	items := make([]itemKey, len(in.Items))
	for i, item := range in.Items {
		items[i] = itemKey{
			SKU:      strings.TrimSpace(item.SKU),
			Quantity: item.Quantity,
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].SKU < items[j].SKU
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "customer_name=%s\n", strings.TrimSpace(in.CustomerName))
	fmt.Fprintf(&sb, "email=%s\n", in.Email)
	fmt.Fprintf(&sb, "phone=%s\n", in.Phone)
	fmt.Fprintf(&sb, "shipping_address=%s\n", in.ShippingAddress)
	fmt.Fprintf(&sb, "shipping_method=%s\n", strings.TrimSpace(in.ShippingMethod))
	fmt.Fprintf(&sb, "payment_method=%s\n", strings.TrimSpace(in.PaymentMethod))
	fmt.Fprintf(&sb, "promo_code=%s\n", strings.TrimSpace(in.PromoCode))
	fmt.Fprintf(&sb, "member_id=%s\n", memberID)
	for _, item := range items {
		fmt.Fprintf(&sb, "item=%s:%d\n", item.SKU, item.Quantity)
	}

	h := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(h[:])
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

// ----- Phase A: catalog publishable filter + category validation ----------

// TestListPublishedProductsIncludesLowAndOutOfStock proves that the public
// catalog listing includes active, low_stock, and out_of_stock products, and
// excludes draft products.
func TestListPublishedProductsIncludesLowAndOutOfStock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p-active", SKU: "SKU-A", Name: "Active", Slug: "active", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-low", SKU: "SKU-L", Name: "Low", Slug: "low", Status: "low_stock", Stock: 3, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-out", SKU: "SKU-O", Name: "Out", Slug: "out", Status: "out_of_stock", Stock: 0, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-draft", SKU: "SKU-D", Name: "Draft", Slug: "draft", Status: "draft", Stock: 10, Price: 100, Category: "apparel"})

	products, err := svc.ListPublishedProducts(context.Background())
	if err != nil {
		t.Fatalf("ListPublishedProducts: %v", err)
	}

	skus := map[string]bool{}
	for _, p := range products {
		skus[p.SKU] = true
	}
	if !skus["SKU-A"] {
		t.Errorf("active product missing from published list")
	}
	if !skus["SKU-L"] {
		t.Errorf("low_stock product missing from published list")
	}
	if !skus["SKU-O"] {
		t.Errorf("out_of_stock product missing from published list")
	}
	if skus["SKU-D"] {
		t.Errorf("draft product should NOT appear in published list")
	}
}

// TestGetProductBySlugExcludesDraft proves that the public product detail
// lookup applies the same publishable status filter — draft products return
// ErrNotFound (404), while active/low_stock/out_of_stock are found.
func TestGetProductBySlugExcludesDraft(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p-active", SKU: "SKU-A", Name: "Active", Slug: "active-prod", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-low", SKU: "SKU-L", Name: "Low", Slug: "low-prod", Status: "low_stock", Stock: 3, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-out", SKU: "SKU-O", Name: "Out", Slug: "out-prod", Status: "out_of_stock", Stock: 0, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-draft", SKU: "SKU-D", Name: "Draft", Slug: "draft-prod", Status: "draft", Stock: 10, Price: 100, Category: "apparel"})

	for _, tc := range []struct {
		slug    string
		wantErr bool
	}{
		{"active-prod", false},
		{"low-prod", false},
		{"out-prod", false},
		{"draft-prod", true},
	} {
		_, err := svc.GetProductBySlug(context.Background(), tc.slug)
		if tc.wantErr {
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("GetProductBySlug(%q) err = %v, want ErrNotFound", tc.slug, err)
			}
		} else {
			if err != nil {
				t.Errorf("GetProductBySlug(%q) err = %v, want nil", tc.slug, err)
			}
		}
	}
}

// TestCreateProductRejectsTraversalCategory proves that category values
// containing path traversal or invalid route segments are rejected at
// create time, after the default is applied. Each case uses a fixed valid
// slug/SKU so only the category gate is exercised — slug validation must
// not be the reason for rejection.
func TestCreateProductRejectsTraversalCategory(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	traversal := []string{"../etc", "..\\windows", "foo/bar", "foo/../../bar"}
	for i, cat := range traversal {
		_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
			SKU: fmt.Sprintf("SKU-TC-%d", i), Name: "Traversal Cat", Slug: fmt.Sprintf("trav-cat-%d", i), Category: cat, Price: 100, Stock: 1,
		})
		if err == nil {
			t.Errorf("expected error for traversal category %q, got nil", cat)
		}
	}
}

// TestCreateProductRejectsReservedCategory proves that reserved route
// segments cannot be used as category values.
func TestCreateProductRejectsReservedCategory(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	reserved := []string{"api", "admin", "assets", "products", "category", "categories", "orders", "checkout"}
	for i, cat := range reserved {
		_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
			SKU: fmt.Sprintf("SKU-RC-%d", i), Name: "Reserved Cat", Slug: fmt.Sprintf("res-cat-%d", i), Category: cat, Price: 100, Stock: 1,
		})
		if err == nil {
			t.Errorf("expected error for reserved category %q, got nil", cat)
		}
	}
}

// TestCreateProductRejectsInvalidCategoryCharacters proves that category
// values with invalid characters (uppercase, spaces, dots, special chars,
// leading/trailing hyphens) are rejected.
func TestCreateProductRejectsInvalidCategoryCharacters(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	invalid := []string{"Apparel", "with space", "with.dot", "with@at", "-leading", "trailing-"}
	for i, cat := range invalid {
		_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
			SKU: fmt.Sprintf("SKU-IC-%d", i), Name: "Invalid Cat", Slug: fmt.Sprintf("inv-cat-%d", i), Category: cat, Price: 100, Stock: 1,
		})
		if err == nil {
			t.Errorf("expected error for invalid category %q, got nil", cat)
		}
	}
}

// TestCreateProductAcceptsValidCategory proves that valid category values
// are accepted, including the default when category is empty.
func TestCreateProductAcceptsValidCategory(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	valid := []string{"apparel", "accessories", "home-living", "stationery-01"}
	for i, cat := range valid {
		_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
			SKU: fmt.Sprintf("SKU-VC-%d", i), Name: "Valid Cat", Slug: fmt.Sprintf("val-cat-%d", i), Category: cat, Price: 100, Stock: 1,
		})
		if err != nil {
			t.Errorf("unexpected error for valid category %q: %v", cat, err)
		}
	}
}

// TestCreateProductDefaultsEmptyCategory proves that an empty category
// defaults to "apparel" and is stored trimmed.
func TestCreateProductDefaultsEmptyCategory(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-DEF", Name: "Default Cat", Slug: "default-cat", Category: "", Price: 100, Stock: 1,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if p.Category != "apparel" {
		t.Errorf("default Category = %q, want %q", p.Category, "apparel")
	}
}

// TestCreateProductTrimsCategoryWhitespace proves that a category with
// surrounding whitespace is trimmed before validation and storage.
func TestCreateProductTrimsCategoryWhitespace(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-TRIM", Name: "Trim Cat", Slug: "trim-cat", Category: "  apparel  ", Price: 100, Stock: 1,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if p.Category != "apparel" {
		t.Errorf("trimmed Category = %q, want %q", p.Category, "apparel")
	}
}

// TestUpdateProductRejectsTraversalCategory proves that category validation
// also applies on update, after the existing-category fallback.
func TestUpdateProductRejectsTraversalCategory(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-UP", Name: "Up", Slug: "up", Status: "active", Stock: 10, Price: 100, Category: "apparel"})

	_, err := svc.UpdateProduct(context.Background(), adminPrincipal(), "p1", ProductInput{
		Category: "../etc", Price: 100, Stock: 10,
	})
	if err == nil {
		t.Errorf("expected error for traversal category on update, got nil")
	}
}

// TestUpdateProductRejectsInvalidCategoryCharacters proves that invalid
// category characters (uppercase, dots, spaces, reserved, leading/trailing
// hyphens) are rejected on update, not just traversal.
func TestUpdateProductRejectsInvalidCategoryCharacters(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-UI", Name: "UI", Slug: "ui", Status: "active", Stock: 10, Price: 100, Category: "apparel"})

	invalid := []string{"Apparel", "with.dot", "with space", "api", "-leading", "trailing-"}
	for _, cat := range invalid {
		_, err := svc.UpdateProduct(context.Background(), adminPrincipal(), "p1", ProductInput{
			Category: cat, Price: 100, Stock: 10,
		})
		if err == nil {
			t.Errorf("expected error for invalid category %q on update, got nil", cat)
		}
	}
}

// TestUpdateProductFallsBackToExistingCategory proves that an empty category
// on update falls back to the existing category and validates it.
func TestUpdateProductFallsBackToExistingCategory(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-FB", Name: "FB", Slug: "fb", Status: "active", Stock: 10, Price: 100, Category: "accessories"})

	p, err := svc.UpdateProduct(context.Background(), adminPrincipal(), "p1", ProductInput{
		Category: "", Price: 200, Stock: 5,
	})
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}
	if p.Category != "accessories" {
		t.Errorf("fallback Category = %q, want %q", p.Category, "accessories")
	}
}

// TestUpdateProductTrimsCategoryWhitespace proves that a category with
// surrounding whitespace is trimmed before validation and storage on update.
func TestUpdateProductTrimsCategoryWhitespace(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-UT", Name: "UT", Slug: "ut", Status: "active", Stock: 10, Price: 100, Category: "apparel"})

	p, err := svc.UpdateProduct(context.Background(), adminPrincipal(), "p1", ProductInput{
		Category: "  accessories  ", Price: 100, Stock: 10,
	})
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}
	if p.Category != "accessories" {
		t.Errorf("trimmed Category = %q, want %q", p.Category, "accessories")
	}
}

// ----- B7: per-item restock tests -------------------------------------------

// seedReceivedOrder creates an order with 2 line items, advances it through
// the fulfillment state machine to "delivered", then through the return
// state machine to "received". Returns the order (with current Version) and
// the service. The product stock after order creation is initialStock - qty.
func seedReceivedOrder(t *testing.T, store Store, skuA, skuB string, qtyA, qtyB, initialStock, price int) (Order, Service) {
	t.Helper()
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p-a", SKU: skuA, Name: "Alpha", Slug: "alpha", Status: "active", Stock: initialStock, Price: price, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-b", SKU: skuB, Name: "Beta", Slug: "beta", Status: "active", Stock: initialStock, Price: price, Category: "apparel"})
	in := OrderInput{
		CustomerName: "Restock Tester", Email: "rt@example.com", Phone: "555-0100",
		ShippingAddress: "123 Main St", ShippingMethod: "home_delivery",
		PaymentMethod: "atm", IdempotencyKey: "seed-received-" + skuA,
		Items: []OrderItem{{SKU: skuA, Quantity: qtyA}, {SKU: skuB, Quantity: qtyB}},
	}
	order, err := svc.CreateOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	for _, step := range []string{"processing", "shipped", "delivered"} {
		order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step)
		if err != nil {
			t.Fatalf("transition to %s: %v", step, err)
		}
	}
	for _, step := range []string{"requested", "approved", "received"} {
		order, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step)
		if err != nil {
			t.Fatalf("return transition to %s: %v", step, err)
		}
	}
	return order, svc
}

func TestRestockOrderItemsHappyPath(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RH-A", "SKU-RH-B", 2, 3, 10, 100)

	// Restock all items: 2 returned + 2 restocked for A, 3 returned + 3 restocked for B.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-happy-1",
		Reason:          "goods inspected, resellable",
		Items: []RestockItemInput{
			{SKU: "SKU-RH-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-RH-B", ReturnedQuantity: 3, RestockedQuantity: 3},
		},
	}
	updated, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("RestockOrderItems: %v", err)
	}
	// Version must have incremented by 1.
	if updated.Version != order.Version+1 {
		t.Errorf("version = %d, want %d", updated.Version, order.Version+1)
	}
	// Item ledger must reflect returned + restocked.
	bySKU := map[string]OrderItem{}
	for _, it := range updated.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-RH-A"]; it.ReturnedQuantity != 2 || it.RestockedQuantity != 2 {
		t.Errorf("SKU-RH-A ledger = returned=%d restocked=%d, want 2/2", it.ReturnedQuantity, it.RestockedQuantity)
	}
	if it := bySKU["SKU-RH-B"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 3 {
		t.Errorf("SKU-RH-B ledger = returned=%d restocked=%d, want 3/3", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Product stock must have been incremented by the restocked amount.
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 10 { // 10 - 2 (order) + 2 (restock) = 10
		t.Errorf("SKU-RH-A stock = %d, want 10", pa.Stock)
	}
	pb, _ := store.GetProduct(context.Background(), "p-b")
	if pb.Stock != 10 { // 10 - 3 (order) + 3 (restock) = 10
		t.Errorf("SKU-RH-B stock = %d, want 10", pb.Stock)
	}
}

func TestRestockOrderItemsForbiddenWithoutCapabilities(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RF-A", "SKU-RF-B", 1, 1, 10, 100)

	// emptyPrincipal has no capabilities → must be forbidden.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-forbidden-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-RF-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), emptyPrincipal(), order.ID, in)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestRestockOrderItemsForbiddenWithOnlyOneCapability(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RF2-A", "SKU-RF2-B", 1, 1, 10, 100)

	// Principal with only orders.returns (missing inventory.adjust) → forbidden.
	onlyReturns := auth.Principal{UserID: "half", Role: "manager", Capabilities: []string{"orders.returns"}}
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-half-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-RF2-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), onlyReturns, order.ID, in)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden (missing inventory.adjust)", err)
	}

	// Principal with only inventory.adjust (missing orders.returns) → forbidden.
	onlyAdjust := auth.Principal{UserID: "half", Role: "manager", Capabilities: []string{"inventory.adjust"}}
	_, err = svc.RestockOrderItems(context.Background(), onlyAdjust, order.ID, in)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden (missing orders.returns)", err)
	}
}

func TestRestockOrderItemsRejectsReturnNotReceived(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	// Seed an order but only advance return to "approved" (not "received").
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p-na", SKU: "SKU-NA-A", Name: "NA", Slug: "na", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	in := OrderInput{
		CustomerName: "NA Tester", Email: "na@example.com", Phone: "555-0100",
		ShippingAddress: "123 Main St", ShippingMethod: "home_delivery",
		PaymentMethod: "atm", IdempotencyKey: "seed-na-1",
		Items: []OrderItem{{SKU: "SKU-NA-A", Quantity: 1}},
	}
	order, err := svc.CreateOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	for _, step := range []string{"processing", "shipped", "delivered"} {
		order, _ = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step)
	}
	for _, step := range []string{"requested", "approved"} {
		order, _ = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step)
	}
	// Now return_request_status = "approved", not "received".
	restockIn := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-na-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-NA-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, restockIn)
	if !errors.Is(err, ErrReturnNotReceived) {
		t.Fatalf("error = %v, want ErrReturnNotReceived", err)
	}
}

func TestRestockOrderItemsRejectsMissingReason(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-MR-A", "SKU-MR-B", 1, 1, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-mr-1",
		Reason:          "   ",
		Items:           []RestockItemInput{{SKU: "SKU-MR-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrMissingRestockReason) {
		t.Fatalf("error = %v, want ErrMissingRestockReason", err)
	}
}

func TestRestockOrderItemsRejectsMissingIdempotencyKey(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-MK-A", "SKU-MK-B", 1, 1, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-MK-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("error = %v, want ErrMissingIdempotencyKey", err)
	}
}

func TestRestockOrderItemsRejectsOverRestock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-OR-A", "SKU-OR-B", 2, 2, 10, 100)

	// restocked > returned → ErrOverRestock.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-over-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-OR-A", ReturnedQuantity: 1, RestockedQuantity: 2}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock", err)
	}
}

func TestRestockOrderItemsRejectsReturnedExceedsQuantity(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-OQ-A", "SKU-OQ-B", 2, 2, 10, 100)

	// returned > ordered quantity → ErrOverRestock.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-oq-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-OQ-A", ReturnedQuantity: 3, RestockedQuantity: 3}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock", err)
	}
}

func TestRestockOrderItemsRejectsUnknownSKU(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-UN-A", "SKU-UN-B", 1, 1, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-un-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-DOES-NOT-EXIST", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrRestockItemNotFound) {
		t.Fatalf("error = %v, want ErrRestockItemNotFound", err)
	}
}

func TestRestockOrderItemsRejectsStaleVersion(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-SV-A", "SKU-SV-B", 1, 1, 10, 100)

	// Use a stale expected_version (order.Version - 1).
	in := RestockInput{
		ExpectedVersion: order.Version - 1,
		IdempotencyKey:  "restock-stale-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-SV-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("error = %v, want ErrStaleVersion", err)
	}
}

func TestRestockOrderItemsIdempotentReplay(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-IR-A", "SKU-IR-B", 2, 2, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-replay-1",
		Reason:          "inspected",
		Items: []RestockItemInput{
			{SKU: "SKU-IR-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-IR-B", ReturnedQuantity: 2, RestockedQuantity: 1},
		},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}

	// Replay with the SAME key + SAME payload. The expected_version in the
	// replay payload is the ORIGINAL version (not first.Version), because
	// the fingerprint is computed from the original input. The service's
	// early idempotency check returns the current order state without
	// re-applying the mutation.
	second, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("replay RestockOrderItems: %v", err)
	}
	// The replay must return the same version as the first (no double-increment).
	if second.Version != first.Version {
		t.Errorf("replay version = %d, want %d (no double-increment)", second.Version, first.Version)
	}
	// Stock must not have been double-incremented.
	pb, _ := store.GetProduct(context.Background(), "p-b")
	if pb.Stock != 9 { // 10 - 2 (order) + 1 (restock) = 9
		t.Errorf("SKU-IR-B stock after replay = %d, want 9 (no double-increment)", pb.Stock)
	}
}

func TestRestockOrderItemsIdempotencyConflictOnDifferentPayload(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-IC-A", "SKU-IC-B", 2, 2, 10, 100)

	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-conflict-1",
		Reason:          "first",
		Items:           []RestockItemInput{{SKU: "SKU-IC-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	if _, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1); err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}

	// Same key, DIFFERENT payload (different reason) → ErrIdempotencyConflict.
	in2 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-conflict-1",
		Reason:          "different reason",
		Items:           []RestockItemInput{{SKU: "SKU-IC-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestRestockOrderItemsPartialRestock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-PR-A", "SKU-PR-B", 3, 3, 10, 100)

	// Restock only 2 of 3 returned for A, 0 restocked for B (damaged).
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-partial-1",
		Reason:          "2 resellable, 1 damaged",
		Items: []RestockItemInput{
			{SKU: "SKU-PR-A", ReturnedQuantity: 3, RestockedQuantity: 2},
			{SKU: "SKU-PR-B", ReturnedQuantity: 3, RestockedQuantity: 0},
		},
	}
	updated, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("RestockOrderItems: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range updated.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-PR-A"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 2 {
		t.Errorf("SKU-PR-A = returned=%d restocked=%d, want 3/2", it.ReturnedQuantity, it.RestockedQuantity)
	}
	if it := bySKU["SKU-PR-B"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 0 {
		t.Errorf("SKU-PR-B = returned=%d restocked=%d, want 3/0", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Stock: A = 10 - 3 + 2 = 9; B = 10 - 3 + 0 = 7.
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 9 {
		t.Errorf("SKU-PR-A stock = %d, want 9", pa.Stock)
	}
	pb, _ := store.GetProduct(context.Background(), "p-b")
	if pb.Stock != 7 {
		t.Errorf("SKU-PR-B stock = %d, want 7", pb.Stock)
	}
}

func TestRestockOrderItemsCumulativeRestockRespectsReturned(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-CR-A", "SKU-CR-B", 3, 1, 10, 100)

	// First restock: 3 returned, 1 restocked.
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-cum-1",
		Reason:          "first batch",
		Items:           []RestockItemInput{{SKU: "SKU-CR-A", ReturnedQuantity: 3, RestockedQuantity: 1}},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}

	// Second restock: try to restock 3 more (cumulative restocked would be 4 > returned 3).
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-cum-2",
		Reason:          "second batch",
		Items:           []RestockItemInput{{SKU: "SKU-CR-A", ReturnedQuantity: 0, RestockedQuantity: 3}},
	}
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock (cumulative restocked > returned)", err)
	}

	// Second restock valid: 0 returned, 2 restocked (cumulative 3 <= returned 3).
	in3 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-cum-3",
		Reason:          "second batch valid",
		Items:           []RestockItemInput{{SKU: "SKU-CR-A", ReturnedQuantity: 0, RestockedQuantity: 2}},
	}
	second, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in3)
	if err != nil {
		t.Fatalf("second RestockOrderItems: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range second.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-CR-A"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 3 {
		t.Errorf("SKU-CR-A = returned=%d restocked=%d, want 3/3", it.ReturnedQuantity, it.RestockedQuantity)
	}
}

func TestRestockOrderItemsRejectsDuplicateSKUInRequest(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-DUP-A", "SKU-DUP-B", 2, 2, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-dup-1",
		Reason:          "test",
		Items: []RestockItemInput{
			{SKU: "SKU-DUP-A", ReturnedQuantity: 1, RestockedQuantity: 1},
			{SKU: "SKU-DUP-A", ReturnedQuantity: 1, RestockedQuantity: 1},
		},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrInvalidAdminInput) {
		t.Fatalf("error = %v, want ErrInvalidAdminInput (duplicate SKU)", err)
	}
}

func TestRestockOrderItemsAppendsOrderEvent(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-EV-A", "SKU-EV-B", 1, 1, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-event-1",
		Reason:          "inspected and reshelved",
		Items:           []RestockItemInput{{SKU: "SKU-EV-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	if _, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in); err != nil {
		t.Fatalf("RestockOrderItems: %v", err)
	}
	events, err := store.ListOrderEvents(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("ListOrderEvents: %v", err)
	}
	var foundRestock bool
	for _, e := range events {
		if e.EventType == "restock" && e.Reason == "inspected and reshelved" && e.ActorUserID == "admin" {
			foundRestock = true
		}
	}
	if !foundRestock {
		t.Errorf("no restock event with reason/actor recorded; events = %+v", events)
	}
}

func TestRestockOrderItemsAtomicRollbackOnOverRestock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-AR-A", "SKU-AR-B", 2, 2, 10, 100)

	// First restock: 2 returned, 2 restocked for A (valid).
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-rollback-1",
		Reason:          "first",
		Items:           []RestockItemInput{{SKU: "SKU-AR-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}
	paAfter1, _ := store.GetProduct(context.Background(), "p-a")
	stockAfter1 := paAfter1.Stock

	// Second restock: try to restock 1 more of A (cumulative restocked = 3 > returned 2).
	// This must fail AND roll back — stock must not change, version must not increment.
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-rollback-2",
		Reason:          "second",
		Items:           []RestockItemInput{{SKU: "SKU-AR-A", ReturnedQuantity: 0, RestockedQuantity: 1}},
	}
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock", err)
	}
	paAfter2, _ := store.GetProduct(context.Background(), "p-a")
	if paAfter2.Stock != stockAfter1 {
		t.Errorf("stock after rolled-back restock = %d, want %d (atomic rollback)", paAfter2.Stock, stockAfter1)
	}
	// The order version must not have changed.
	unchanged, _ := svc.GetOrder(context.Background(), order.ID)
	if unchanged.Version != first.Version {
		t.Errorf("version after rolled-back restock = %d, want %d (atomic rollback)", unchanged.Version, first.Version)
	}
}

func TestRestockOrderItemsConcurrentSameKeyRace(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RACE-A", "SKU-RACE-B", 2, 2, 20, 100)

	// Two concurrent requests with the SAME idempotency key + SAME payload.
	// Only one should apply; the other must return the same result (replay)
	// or ErrIdempotencyConflict — never a double-increment.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-race-same-key",
		Reason:          "concurrent",
		Items:           []RestockItemInput{{SKU: "SKU-RACE-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var err1, err2 error
	go func() {
		defer wg.Done()
		_, err1 = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	}()
	go func() {
		defer wg.Done()
		_, err2 = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	}()
	wg.Wait()

	// At least one must succeed.
	if err1 != nil && err2 != nil {
		t.Fatalf("both concurrent restocks failed: err1=%v err2=%v", err1, err2)
	}
	// Stock must have been incremented exactly once (20 - 2 + 2 = 20).
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 20 {
		t.Errorf("stock after concurrent same-key restock = %d, want 20 (exactly one application)", pa.Stock)
	}
	// Version must have incremented exactly once.
	final, _ := svc.GetOrder(context.Background(), order.ID)
	if final.Version != order.Version+1 {
		t.Errorf("version after concurrent same-key restock = %d, want %d (exactly one increment)", final.Version, order.Version+1)
	}
}

// TestRestockOrderItemsRestockOnlyFollowUp verifies that a restock-only
// follow-up action (returned_delta=0, restocked_delta=1) succeeds when
// existingReturned > existingRestocked. This is the critical case that
// the old per-action restocked <= returned constraint would have blocked.
func TestRestockOrderItemsRestockOnlyFollowUp(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RO-A", "SKU-RO-B", 3, 1, 10, 100)

	// First action: receive 3, restock only 1 (2 damaged/un-restocked).
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-ro-1",
		Reason:          "received 3, 1 resellable",
		Items:           []RestockItemInput{{SKU: "SKU-RO-A", ReturnedQuantity: 3, RestockedQuantity: 1}},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range first.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-RO-A"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 1 {
		t.Fatalf("after first action: SKU-RO-A = returned=%d restocked=%d, want 3/1", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Stock after first: 10 - 3 (order) + 1 (restock) = 8.
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 8 {
		t.Fatalf("stock after first = %d, want 8", pa.Stock)
	}

	// Second action: returned_delta=0, restocked_delta=1 (restock one more
	// of the previously received but not-yet-restocked units). This MUST
	// succeed — the cumulative constraint is: restocked_delta - returned_delta
	// (= 1 - 0 = 1) <= currentReturned - currentRestocked (= 3 - 1 = 2). OK.
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-ro-2",
		Reason:          "second batch: 1 more resellable",
		Items:           []RestockItemInput{{SKU: "SKU-RO-A", ReturnedQuantity: 0, RestockedQuantity: 1}},
	}
	second, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if err != nil {
		t.Fatalf("second RestockOrderItems (restock-only follow-up): %v", err)
	}
	bySKU = map[string]OrderItem{}
	for _, it := range second.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-RO-A"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 2 {
		t.Errorf("after second action: SKU-RO-A = returned=%d restocked=%d, want 3/2", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Stock after second: 8 + 1 = 9.
	pa, _ = store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 9 {
		t.Errorf("stock after second = %d, want 9", pa.Stock)
	}
}

// TestRestockOrderItemsRestockOnlyExceedsAvailable verifies that a
// restock-only follow-up (returned_delta=0) is rejected when
// restocked_delta exceeds the un-restocked surplus
// (currentReturned - currentRestocked).
func TestRestockOrderItemsRestockOnlyExceedsAvailable(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-ROX-A", "SKU-ROX-B", 3, 1, 10, 100)

	// First: receive 3, restock 1 (surplus = 2).
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-rox-1",
		Reason:          "first",
		Items:           []RestockItemInput{{SKU: "SKU-ROX-A", ReturnedQuantity: 3, RestockedQuantity: 1}},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first: %v", err)
	}

	// Second: try to restock 3 (surplus is only 2). Must fail.
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-rox-2",
		Reason:          "second",
		Items:           []RestockItemInput{{SKU: "SKU-ROX-A", ReturnedQuantity: 0, RestockedQuantity: 3}},
	}
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock (restock-only delta exceeds available surplus)", err)
	}
}

// TestRestockOrderItemsRejectsAllZeroNoOp verifies that an all-zero
// delta (no positive returned or restocked) is rejected as a no-op.
func TestRestockOrderItemsRejectsAllZeroNoOp(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-NZ-A", "SKU-NZ-B", 2, 2, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-nz-1",
		Reason:          "no-op",
		Items: []RestockItemInput{
			{SKU: "SKU-NZ-A", ReturnedQuantity: 0, RestockedQuantity: 0},
			{SKU: "SKU-NZ-B", ReturnedQuantity: 0, RestockedQuantity: 0},
		},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrInvalidAdminInput) {
		t.Fatalf("error = %v, want ErrInvalidAdminInput (all-zero no-op)", err)
	}
}

// TestRestockOrderItemsTrimsSKUs verifies that whitespace-padded SKUs are
// trimmed and the trimmed value flows through to the SQL UPDATE. Without
// canonicalization, "  SKU-TR-A  " would pass validation (which trims for
// ledger lookup) but fail the SQL UPDATE (which uses the raw value).
func TestRestockOrderItemsTrimsSKUs(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-TR-A", "SKU-TR-B", 2, 2, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-tr-1",
		Reason:          "test with padded SKU",
		Items:           []RestockItemInput{{SKU: "  SKU-TR-A  ", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	updated, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("RestockOrderItems with padded SKU: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range updated.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-TR-A"]; it.ReturnedQuantity != 2 || it.RestockedQuantity != 2 {
		t.Errorf("SKU-TR-A ledger = returned=%d restocked=%d, want 2/2 (trimmed SKU must match)", it.ReturnedQuantity, it.RestockedQuantity)
	}
}

// TestRestockOrderItemsConcurrentSameKeyDifferentPayload verifies that
// two concurrent requests with the SAME idempotency key but DIFFERENT
// payloads result in exactly one success and one ErrIdempotencyConflict.
// This tests the store's in-transaction unique constraint detection and
// the service's fingerprint comparison using the STORED fingerprint
// (not a self-comparison).
func TestRestockOrderItemsConcurrentSameKeyDifferentPayload(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-CDP-A", "SKU-CDP-B", 2, 2, 20, 100)

	// Two concurrent requests with the SAME key but DIFFERENT reason
	// (different payload → different fingerprint).
	inA := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-cdp-same-key",
		Reason:          "payload A",
		Items:           []RestockItemInput{{SKU: "SKU-CDP-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	inB := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-cdp-same-key",
		Reason:          "payload B (different)",
		Items:           []RestockItemInput{{SKU: "SKU-CDP-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var errA, errB error
	go func() {
		defer wg.Done()
		_, errA = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, inA)
	}()
	go func() {
		defer wg.Done()
		_, errB = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, inB)
	}()
	wg.Wait()

	// Exactly one must succeed, the other must get ErrIdempotencyConflict.
	aSuccess := errA == nil
	bSuccess := errB == nil
	if aSuccess && bSuccess {
		t.Fatalf("both concurrent different-payload restocks succeeded — idempotency conflict not detected")
	}
	if !aSuccess && !bSuccess {
		t.Fatalf("both concurrent restocks failed: errA=%v errB=%v", errA, errB)
	}
	// The loser must be ErrIdempotencyConflict (not some other error).
	var loserErr error
	if !aSuccess {
		loserErr = errA
	} else {
		loserErr = errB
	}
	if !errors.Is(loserErr, ErrIdempotencyConflict) {
		t.Errorf("loser error = %v, want ErrIdempotencyConflict", loserErr)
	}
	// Stock must have been incremented exactly once (20 - 2 + 2 = 20).
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 20 {
		t.Errorf("stock after concurrent different-payload restock = %d, want 20 (exactly one application)", pa.Stock)
	}
	// Version must have incremented exactly once.
	final, _ := svc.GetOrder(context.Background(), order.ID)
	if final.Version != order.Version+1 {
		t.Errorf("version after concurrent different-payload restock = %d, want %d", final.Version, order.Version+1)
	}
}

// missFirstRestockLookupStore forces the service-layer early idempotency
// lookup to miss once for a given key, then delegates. This is the
// post-early-check race: both racers missed, the winner already committed
// the idempotency row plus ledger, and the loser's next Find must observe
// that committed record instead of returning the mutable-state error.
type missFirstRestockLookupStore struct {
	Store
	key        string
	missesLeft int
}

func (m *missFirstRestockLookupStore) FindRestockByIdempotencyKey(ctx context.Context, key string) (string, string, string, error) {
	if key == m.key && m.missesLeft > 0 {
		m.missesLeft--
		return "", "", "", ErrNotFound
	}
	return m.Store.FindRestockByIdempotencyKey(ctx, key)
}

// TestRestockOrderItemsPostEarlyCheckRaceResolvesIdempotency proves the
// path where the early lookup misses after a winner has already committed.
// Without the post-validation re-lookup, ledger validation would return
// ErrOverRestock and never reach unique-key arbitration.
func TestRestockOrderItemsPostEarlyCheckRaceResolvesIdempotency(t *testing.T) {
	t.Parallel()
	inner := newTestStore(t)
	order, seedSvc := seedReceivedOrder(t, inner, "SKU-PER-A", "SKU-PER-B", 2, 2, 20, 100)

	winnerIn := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-post-early-race",
		Reason:          "winner payload",
		Items:           []RestockItemInput{{SKU: "SKU-PER-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	winner, err := seedSvc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, winnerIn)
	if err != nil {
		t.Fatalf("winner RestockOrderItems: %v", err)
	}

	wrapped := &missFirstRestockLookupStore{Store: inner, key: winnerIn.IdempotencyKey, missesLeft: 1}
	svc := NewService(wrapped)

	loserIn := winnerIn
	loserIn.Reason = "loser different payload"
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, loserIn)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("different payload after forced early miss: error = %v, want ErrIdempotencyConflict", err)
	}
	if wrapped.missesLeft != 0 {
		t.Fatalf("expected the forced early miss to be consumed, missesLeft=%d", wrapped.missesLeft)
	}

	wrapped.missesLeft = 1
	replay, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, winnerIn)
	if err != nil {
		t.Fatalf("same payload after forced early miss: %v", err)
	}
	if replay.ID != winner.ID || replay.Version != winner.Version {
		t.Fatalf("replay snapshot = id=%s version=%d, want id=%s version=%d", replay.ID, replay.Version, winner.ID, winner.Version)
	}
	if len(replay.Items) != len(winner.Items) {
		t.Fatalf("replay items=%d, want %d", len(replay.Items), len(winner.Items))
	}

	genuine := winnerIn
	genuine.IdempotencyKey = "restock-post-early-unused"
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, genuine)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("unused key over-restock: error = %v, want ErrOverRestock", err)
	}
}

// ----- B7: migration upgrade + idempotency snapshot tests -------------------

// newPre014TestStore opens an in-memory SQLite database, applies all
// migrations EXCEPT 014_order_items, and returns the raw *sql.DB so the
// test can insert pre-014 fixture data (orders with items_json but no
// order_items table). The caller then applies 014 via migrate.Apply to
// test the backfill upgrade path.
func newPre014TestStore(t *testing.T) (*sql.DB, database.Dialect, string) {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:commerce_upgrade_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	// Apply migrations 001–013 by inserting a fake 014 record so
	// migrate.Apply skips it. Then the test removes the fake record
	// and calls migrate.Apply again to apply the real 014.
	migrationsDir := filepath.Join(root, "db", "migrations", "sqlite")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var pre014 []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") && e.Name() < "014_order_items.sql" {
			pre014 = append(pre014, e.Name())
		}
	}
	sort.Strings(pre014)

	// Create schema_migrations and apply each pre-014 migration manually.
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_unix BIGINT NOT NULL
	)`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	for _, name := range pre014 {
		body, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin tx for %s: %v", name, err)
		}
		for _, stmt := range strings.Split(string(body), ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				_ = tx.Rollback()
				t.Fatalf("exec migration %s: %v", name, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_unix) VALUES (?, 0)`, name); err != nil {
			_ = tx.Rollback()
			t.Fatalf("record migration %s: %v", name, err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit migration %s: %v", name, err)
		}
	}
	return db, dialect, root
}

// insertPre014Order inserts an order row with items_json directly into a
// pre-014 database (no order_items table). This simulates an order created
// before migration 014 was applied.
func insertPre014Order(t *testing.T, db *sql.DB, id, itemsJSON string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO orders
		(id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix)
		VALUES (?, '', 'Test', 'test@example.com', '555', ?, 'Addr', 'home_delivery', 'atm', '', 100, 0, 0, 100, 'delivered', 'paid', 'received', '', ?, '', '', '[]', '', 1, 1000)`,
		id, itemsJSON, id+"-key")
	if err != nil {
		t.Fatalf("insert pre-014 order %s: %v", id, err)
	}
}

// TestMigration014BackfillOrderItems tests that migration 014 backfills
// the order_items table from existing orders.items_json. This covers the
// upgrade path: orders created before 014 must have their items_json
// exploded into order_items rows so they can be restocked.
func TestMigration014BackfillOrderItems(t *testing.T) {
	t.Parallel()
	db, _, root := newPre014TestStore(t)
	ctx := context.Background()

	// Insert a normal order with 2 distinct SKUs.
	insertPre014Order(t, db, "ord-normal",
		`[{"sku":"SKU-N1","name":"Normal 1","price":100,"quantity":2},{"sku":"SKU-N2","name":"Normal 2","price":200,"quantity":3}]`)

	// Insert an order with a missing "name" field (should default to '').
	insertPre014Order(t, db, "ord-missing-name",
		`[{"sku":"SKU-MN","price":50,"quantity":1}]`)

	// Insert an order with duplicate SKUs (old contract did not reject).
	// The backfill must aggregate: quantity = 2+1 = 3, line_total = 100*2 + 100*1 = 300.
	insertPre014Order(t, db, "ord-dup-sku",
		`[{"sku":"SKU-DUP","name":"Dup","price":100,"quantity":2},{"sku":"SKU-DUP","name":"Dup","price":100,"quantity":1}]`)

	// Apply migration 014 (the remaining unapplied migration).
	if err := migrate.Apply(ctx, db, database.SQLite, root); err != nil {
		t.Fatalf("apply migration 014: %v", err)
	}

	// Verify normal order: 2 rows with correct values.
	rows, err := db.QueryContext(ctx,
		`SELECT sku, name, price, quantity, line_total, returned_quantity, restocked_quantity
		 FROM order_items WHERE order_id = ? ORDER BY sku`, "ord-normal")
	if err != nil {
		t.Fatalf("query order_items for ord-normal: %v", err)
	}
	defer rows.Close()
	type item struct {
		sku, name  string
		price, qty int
		lineTotal  int
		returned   int
		restocked  int
	}
	var normalItems []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.sku, &it.name, &it.price, &it.qty, &it.lineTotal, &it.returned, &it.restocked); err != nil {
			t.Fatalf("scan: %v", err)
		}
		normalItems = append(normalItems, it)
	}
	if len(normalItems) != 2 {
		t.Fatalf("ord-normal: got %d order_items rows, want 2", len(normalItems))
	}
	if it := normalItems[0]; it.sku != "SKU-N1" || it.name != "Normal 1" || it.price != 100 || it.qty != 2 || it.lineTotal != 200 || it.returned != 0 || it.restocked != 0 {
		t.Errorf("ord-normal SKU-N1: %+v", it)
	}
	if it := normalItems[1]; it.sku != "SKU-N2" || it.name != "Normal 2" || it.price != 200 || it.qty != 3 || it.lineTotal != 600 || it.returned != 0 || it.restocked != 0 {
		t.Errorf("ord-normal SKU-N2: %+v", it)
	}

	// Verify missing-name order: name defaults to ''.
	var mnName string
	err = db.QueryRowContext(ctx,
		`SELECT name FROM order_items WHERE order_id = ? AND sku = ?`,
		"ord-missing-name", "SKU-MN").Scan(&mnName)
	if err != nil {
		t.Fatalf("query missing-name order_item: %v", err)
	}
	if mnName != "" {
		t.Errorf("ord-missing-name: name = %q, want empty string", mnName)
	}

	// Verify duplicate-SKU order: aggregated to 1 row with summed quantity.
	var dupQty, dupLineTotal int
	err = db.QueryRowContext(ctx,
		`SELECT quantity, line_total FROM order_items WHERE order_id = ? AND sku = ?`,
		"ord-dup-sku", "SKU-DUP").Scan(&dupQty, &dupLineTotal)
	if err != nil {
		t.Fatalf("query dup-sku order_item: %v", err)
	}
	if dupQty != 3 {
		t.Errorf("ord-dup-sku: quantity = %d, want 3 (aggregated 2+1)", dupQty)
	}
	if dupLineTotal != 300 {
		t.Errorf("ord-dup-sku: line_total = %d, want 300 (100*2 + 100*1)", dupLineTotal)
	}
}

// TestMigration014BackfillRestockable tests that after migration 014
// backfills order_items, a pre-014 order can be loaded via GetOrder
// (with merged ledger columns) and restocked.
func TestMigration014BackfillRestockable(t *testing.T) {
	t.Parallel()
	db, dialect, root := newPre014TestStore(t)
	ctx := context.Background()

	// Seed a product so restock can increment stock.
	store := NewSQLStore(db, dialect)
	seedProduct(t, store, Product{ID: "p-up", SKU: "SKU-UP", Name: "Upgrade", Slug: "up", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	seedReadyPaymentMethod(t, store)

	// Insert a pre-014 order with one SKU, qty 2, return_request_status = "received".
	insertPre014Order(t, db, "ord-up",
		`[{"sku":"SKU-UP","name":"Upgrade Product","price":100,"quantity":2}]`)

	// Apply migration 014.
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migration 014: %v", err)
	}

	// Re-create the store (now with order_items table).
	store = NewSQLStore(db, dialect)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	// GetOrder must load the order and merge ledger columns from order_items.
	order, err := svc.GetOrder(ctx, "ord-up")
	if err != nil {
		t.Fatalf("GetOrder after upgrade: %v", err)
	}
	if len(order.Items) != 1 {
		t.Fatalf("order items = %d, want 1", len(order.Items))
	}
	if order.Items[0].SKU != "SKU-UP" || order.Items[0].Quantity != 2 {
		t.Errorf("order item: sku=%s qty=%d, want SKU-UP/2", order.Items[0].SKU, order.Items[0].Quantity)
	}
	if order.Items[0].ReturnedQuantity != 0 || order.Items[0].RestockedQuantity != 0 {
		t.Errorf("ledger: returned=%d restocked=%d, want 0/0", order.Items[0].ReturnedQuantity, order.Items[0].RestockedQuantity)
	}
	if order.ReturnRequestStatus != "received" {
		t.Fatalf("return_request_status = %q, want received", order.ReturnRequestStatus)
	}

	// Restock the upgraded order: return 2, restock 1.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-upgrade-1",
		Reason:          "post-upgrade restock",
		Items:           []RestockItemInput{{SKU: "SKU-UP", ReturnedQuantity: 2, RestockedQuantity: 1}},
	}
	updated, err := svc.RestockOrderItems(ctx, adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("RestockOrderItems after upgrade: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range updated.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-UP"]; it.ReturnedQuantity != 2 || it.RestockedQuantity != 1 {
		t.Errorf("post-restock ledger: returned=%d restocked=%d, want 2/1", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Stock: the pre-014 order was inserted via raw SQL (not CreateOrder),
	// so stock was never decremented at order time. After restocking 1 unit,
	// stock = 10 + 1 = 11. This tests that the backfilled order_items row
	// is functional for the restock stock-increment path.
	p, _ := store.GetProduct(ctx, "p-up")
	if p.Stock != 11 {
		t.Errorf("post-restock stock = %d, want 11 (10 initial + 1 restocked, no order-time decrement for pre-014 order)", p.Stock)
	}
}

// TestRestockIdempotencySnapshotSurvivesMutation tests the core idempotency
// contract: after a successful restock, a subsequent mutation (e.g. a second
// restock with a different key) must NOT change the replay response. The
// replay must return the exact same snapshot that was stored during the
// original restock tx.
func TestRestockIdempotencySnapshotSurvivesMutation(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-SNAP-A", "SKU-SNAP-B", 4, 4, 20, 100)

	// First restock: return 2 + restock 2 for A, return 2 + restock 1 for B.
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-snapshot-1",
		Reason:          "first restock",
		Items: []RestockItemInput{
			{SKU: "SKU-SNAP-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-SNAP-B", ReturnedQuantity: 2, RestockedQuantity: 1},
		},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first restock: %v", err)
	}

	// Perform a DIFFERENT mutation: a second restock with a different key
	// that increments version and changes ledger values. If the replay of
	// the first key called GetOrder, it would return the post-second-restock
	// state (version+2, different ledger) instead of the original snapshot.
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-snapshot-2",
		Reason:          "second restock (mutation)",
		Items: []RestockItemInput{
			{SKU: "SKU-SNAP-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-SNAP-B", ReturnedQuantity: 2, RestockedQuantity: 1},
		},
	}
	mutated, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if err != nil {
		t.Fatalf("second restock (mutation): %v", err)
	}
	if mutated.Version == first.Version {
		t.Fatalf("mutation did not increment version: both = %d", first.Version)
	}

	// Replay the FIRST restock with the SAME key + SAME payload.
	// The expected_version in the replay is the ORIGINAL version (from in1),
	// because the fingerprint is computed from the original input.
	replay, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	// The replay response must equal the FIRST response, not the current
	// (post-second-restock) order state.
	if replay.Version != first.Version {
		t.Errorf("replay version = %d, want %d (first snapshot, not current %d)",
			replay.Version, first.Version, mutated.Version)
	}

	// Items must match the first response exactly — NOT the mutated state.
	// After the second restock, the ledger has returned=4/restocked=4 for A
	// and returned=4/restocked=2 for B. The replay must show the FIRST
	// snapshot: returned=2/restocked=2 for A, returned=2/restocked=1 for B.
	if len(replay.Items) != len(first.Items) {
		t.Fatalf("replay items count = %d, want %d", len(replay.Items), len(first.Items))
	}
	for i := range first.Items {
		if replay.Items[i].ReturnedQuantity != first.Items[i].ReturnedQuantity {
			t.Errorf("replay item %s returned = %d, want %d (first snapshot)",
				replay.Items[i].SKU, replay.Items[i].ReturnedQuantity, first.Items[i].ReturnedQuantity)
		}
		if replay.Items[i].RestockedQuantity != first.Items[i].RestockedQuantity {
			t.Errorf("replay item %s restocked = %d, want %d (first snapshot)",
				replay.Items[i].SKU, replay.Items[i].RestockedQuantity, first.Items[i].RestockedQuantity)
		}
	}

	// Verify the current order state IS different from the replay (proves
	// the mutation actually happened and the snapshot is not just echoing
	// current state).
	current, _ := svc.GetOrder(context.Background(), order.ID)
	if current.Version == first.Version {
		t.Fatal("current order version matches first snapshot — mutation did not take effect")
	}
	currentBySKU := map[string]OrderItem{}
	for _, it := range current.Items {
		currentBySKU[it.SKU] = it
	}
	if it := currentBySKU["SKU-SNAP-A"]; it.ReturnedQuantity != 4 || it.RestockedQuantity != 4 {
		t.Errorf("current SKU-SNAP-A: returned=%d restocked=%d, want 4/4 (after second restock)", it.ReturnedQuantity, it.RestockedQuantity)
	}

	// Stock must not have been double-incremented by the replay.
	// After both restocks: 20 - 4 (order) + 2 (first) + 2 (second) = 20 for A.
	// The replay must NOT add another +2.
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 20 {
		t.Errorf("SKU-SNAP-A stock after replay = %d, want 20 (no double-increment)", pa.Stock)
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

// TestRestockIdempotencyCascadeOnOrderDelete proves that deleting an order
// cascades to delete its restock_idempotency rows (which contain
// response_json — an admin order snapshot with PII). This test opens a
// SQLite DB with _pragma=foreign_keys(1) so the REFERENCES orders(id) ON
// DELETE CASCADE clause in migration 014 is enforced. The default test
// store DSN does not enable foreign_keys, so we open a dedicated DB here.
func TestRestockIdempotencyCascadeOnOrderDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	seq := atomic.AddUint64(&testDBCounter, 1)
	// _pragma=foreign_keys(1) is required for SQLite to enforce FK cascades.
	dsn := fmt.Sprintf("file:commerce_cascade_%d?mode=memory&cache=private&_pragma=foreign_keys(1)", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite with foreign_keys: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	store := NewSQLStore(db, dialect)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p-casc", SKU: "SKU-CASC-A", Name: "Cascade", Slug: "casc", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-casc-b", SKU: "SKU-CASC-B", Name: "CascadeB", Slug: "cascb", Status: "active", Stock: 10, Price: 100, Category: "apparel"})

	// Create and advance an order to "received" return status.
	order, err := svc.CreateOrder(ctx, OrderInput{
		CustomerName: "Cascade Tester", Email: "casc@example.com", Phone: "555-0100",
		ShippingAddress: "123 Main St", ShippingMethod: "home_delivery",
		PaymentMethod: "atm", IdempotencyKey: "seed-casc-" + testKey(),
		Items: []OrderItem{{SKU: "SKU-CASC-A", Quantity: 2}, {SKU: "SKU-CASC-B", Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	for _, step := range []string{"processing", "shipped", "delivered"} {
		order, err = svc.UpdateOrderStatus(ctx, adminPrincipal(), order.ID, order.Version, step)
		if err != nil {
			t.Fatalf("transition to %s: %v", step, err)
		}
	}
	for _, step := range []string{"requested", "approved", "received"} {
		order, err = svc.UpdateOrderReturnStatus(ctx, adminPrincipal(), order.ID, order.Version, step)
		if err != nil {
			t.Fatalf("return transition to %s: %v", step, err)
		}
	}

	// Restock — this creates a restock_idempotency row with response_json
	// containing the admin order snapshot (PII: customer name, email, phone,
	// shipping address).
	restockKey := "restock-cascade-1"
	_, err = svc.RestockOrderItems(ctx, adminPrincipal(), order.ID, RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  restockKey,
		Reason:          "cascade test",
		Items: []RestockItemInput{
			{SKU: "SKU-CASC-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-CASC-B", ReturnedQuantity: 2, RestockedQuantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("RestockOrderItems: %v", err)
	}

	// Verify the restock_idempotency row exists before delete.
	_, _, respJSON, err := store.FindRestockByIdempotencyKey(ctx, restockKey)
	if err != nil {
		t.Fatalf("FindRestockByIdempotencyKey before delete: %v", err)
	}
	if respJSON == "" || respJSON == "{}" {
		t.Fatalf("restock_idempotency.response_json is empty/placeholder before delete")
	}

	// Delete the order via raw SQL. The ON DELETE CASCADE on
	// restock_idempotency.order_id must remove the idempotency row (and
	// order_items, order_events rows) because response_json contains the
	// admin order snapshot with PII — it must not outlive the order.
	if _, err := db.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, order.ID); err != nil {
		t.Fatalf("DELETE FROM orders: %v", err)
	}

	// The restock_idempotency row must be gone (cascade).
	_, _, _, err = store.FindRestockByIdempotencyKey(ctx, restockKey)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("FindRestockByIdempotencyKey after order delete: error = %v, want ErrNotFound (cascade failed — PII snapshot leaked)", err)
	}

	// order_items rows must also be gone (cascade).
	var itemCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM order_items WHERE order_id = ?`, order.ID).Scan(&itemCount); err != nil {
		t.Fatalf("count order_items after delete: %v", err)
	}
	if itemCount != 0 {
		t.Errorf("order_items after order delete = %d, want 0 (cascade failed)", itemCount)
	}

	// order_events rows must also be gone (cascade, migration 010).
	var eventCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM order_events WHERE order_id = ?`, order.ID).Scan(&eventCount); err != nil {
		t.Fatalf("count order_events after delete: %v", err)
	}
	if eventCount != 0 {
		t.Errorf("order_events after order delete = %d, want 0 (cascade failed)", eventCount)
	}
}
