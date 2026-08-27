package commerce

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
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
