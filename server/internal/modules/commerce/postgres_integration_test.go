package commerce

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

func getTestPostgresDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI; live PostgreSQL coverage must not be skipped")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping live PostgreSQL integration test")
	}
	return dsn
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root containing go.mod")
		}
		dir = parent
	}
}

// TestPostgresLiveCommerceBooleanAdaptersAndCheckout verifies that the promo,
// payment-method, and shipping-method boolean adapters work correctly on live
// PostgreSQL and that a full quote + create-order round-trip succeeds from a
// fresh migration in an isolated schema.
//
// This test is mutation-sensitive: if any bool bind is reverted to int 0/1,
// the PostgreSQL BOOLEAN column semantics cause the test to fail.
func TestPostgresLiveCommerceBooleanAdaptersAndCheckout(t *testing.T) {
	dsn := getTestPostgresDSN(t)
	root := findRepoRoot(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Isolate this test in a unique PostgreSQL schema so parallel package tests
	// do not race on CREATE TABLE/type catalog entries during migrate.Apply.
	adminDB, _, err := database.Open(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("Open adminDB(postgres) failed: %v", err)
	}
	schemaName := fmt.Sprintf("commerce_test_%d", time.Now().UnixNano())
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, schemaName)); err != nil {
		t.Fatalf("create schema %s: %v", schemaName, err)
	}
	defer func() {
		cleanCtx, cancelClean := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelClean()
		_, _ = adminDB.ExecContext(cleanCtx, fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schemaName))
		adminDB.Close()
	}()

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	schemaDSN := fmt.Sprintf("%s%ssearch_path=%s", dsn, sep, schemaName)

	db, dialect, err := database.Open(ctx, "postgres", schemaDSN)
	if err != nil {
		t.Fatalf("Open db(postgres) failed: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if dialect != database.Postgres {
		t.Fatalf("expected dialect %q, got %q", database.Postgres, dialect)
	}

	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store := NewSQLStore(db, dialect)
	svc := NewService(store)
	now := time.Now().Unix()

	// -------------------------------------------------------------------------
	// 1. Promo boolean adapter: enabled/disabled/expired
	// -------------------------------------------------------------------------
	activePromo := Promo{
		ID:          fmt.Sprintf("pg-promo-active-%d", now),
		Code:        fmt.Sprintf("PGSAVE%d", now),
		Label:       "PG Save",
		Type:        "fixed",
		Value:       100,
		Enabled:     true,
		StartsUnix:  0,
		ExpiresUnix: 0,
		UpdatedUnix: now,
	}
	disabledPromo := Promo{
		ID:          fmt.Sprintf("pg-promo-disabled-%d", now),
		Code:        fmt.Sprintf("PGOFF%d", now),
		Label:       "PG Off",
		Type:        "fixed",
		Value:       50,
		Enabled:     false,
		StartsUnix:  0,
		ExpiresUnix: 0,
		UpdatedUnix: now,
	}
	expiredPromo := Promo{
		ID:          fmt.Sprintf("pg-promo-expired-%d", now),
		Code:        fmt.Sprintf("PGEXP%d", now),
		Label:       "PG Expired",
		Type:        "fixed",
		Value:       200,
		Enabled:     true,
		StartsUnix:  0,
		ExpiresUnix: 1,
		UpdatedUnix: now,
	}

	if err := store.UpsertPromo(ctx, activePromo); err != nil {
		t.Fatalf("upsert active promo: %v", err)
	}
	if err := store.UpsertPromo(ctx, disabledPromo); err != nil {
		t.Fatalf("upsert disabled promo: %v", err)
	}
	if err := store.UpsertPromo(ctx, expiredPromo); err != nil {
		t.Fatalf("upsert expired promo: %v", err)
	}

	// Active promo must be found by GetActivePromoByCode
	found, err := store.GetActivePromoByCode(ctx, activePromo.Code, now)
	if err != nil {
		t.Fatalf("GetActivePromoByCode(active): %v", err)
	}
	if !found.Enabled {
		t.Fatalf("active promo Enabled = false, want true (bool adapter scan)")
	}
	if found.Code != activePromo.Code {
		t.Fatalf("active promo Code = %q, want %q", found.Code, activePromo.Code)
	}

	// Disabled promo must NOT be found
	if _, err := store.GetActivePromoByCode(ctx, disabledPromo.Code, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("disabled promo lookup: err = %v, want ErrNotFound", err)
	}

	// Expired promo must NOT be found
	if _, err := store.GetActivePromoByCode(ctx, expiredPromo.Code, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired promo lookup: err = %v, want ErrNotFound", err)
	}

	// -------------------------------------------------------------------------
	// 2. Payment method boolean adapter: enabled/disabled
	// -------------------------------------------------------------------------
	enabledPayment := PaymentMethod{
		ID:              fmt.Sprintf("pg-pm-enabled-%d", now),
		Method:          "atm",
		ProviderLabel:   "ATM 轉帳",
		Environment:     "sandbox",
		ReadinessStatus: "ready",
		Enabled:         true,
		UpdatedUnix:     now,
	}
	disabledPayment := PaymentMethod{
		ID:              fmt.Sprintf("pg-pm-disabled-%d", now),
		Method:          "credit_card",
		ProviderLabel:   "Credit Card",
		Environment:     "sandbox",
		ReadinessStatus: "ready",
		Enabled:         false,
		UpdatedUnix:     now,
	}

	if err := store.UpsertPaymentMethod(ctx, enabledPayment); err != nil {
		t.Fatalf("upsert enabled payment: %v", err)
	}
	if err := store.UpsertPaymentMethod(ctx, disabledPayment); err != nil {
		t.Fatalf("upsert disabled payment: %v", err)
	}

	methods, err := store.ListPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("ListPaymentMethods: %v", err)
	}
	var enabledPM, disabledPM *PaymentMethod
	for i := range methods {
		if methods[i].ID == enabledPayment.ID {
			enabledPM = &methods[i]
		}
		if methods[i].ID == disabledPayment.ID {
			disabledPM = &methods[i]
		}
	}
	if enabledPM == nil {
		t.Fatal("enabled payment method not found in list")
	}
	if !enabledPM.Enabled {
		t.Fatalf("enabled payment Enabled = false, want true (bool adapter scan)")
	}
	if disabledPM == nil {
		t.Fatal("disabled payment method not found in list")
	}
	if disabledPM.Enabled {
		t.Fatalf("disabled payment Enabled = true, want false (bool adapter scan)")
	}

	// validatePaymentMethod accepts enabled, rejects disabled
	if err := svc.validatePaymentMethod(ctx, enabledPayment.ID); err != nil {
		t.Fatalf("validate enabled payment: %v", err)
	}
	if err := svc.validatePaymentMethod(ctx, disabledPayment.ID); !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("validate disabled payment: err = %v, want ErrInvalidPaymentMethod", err)
	}

	// -------------------------------------------------------------------------
	// 3. Shipping method boolean adapter: enabled/disabled + public filter
	// -------------------------------------------------------------------------
	enabledShipping := ShippingMethod{
		ID:          fmt.Sprintf("pg-sm-enabled-%d", now),
		Method:      "home_delivery",
		Label:       "宅配",
		Description: "本島配送",
		Fee:         120,
		Enabled:     true,
		SortOrder:   1,
		Version:     1,
		UpdatedUnix: now,
	}
	disabledShipping := ShippingMethod{
		ID:          fmt.Sprintf("pg-sm-disabled-%d", now),
		Method:      "store_711",
		Label:       "7-11",
		Description: "超商取貨",
		Fee:         60,
		Enabled:     false,
		SortOrder:   2,
		Version:     1,
		UpdatedUnix: now,
	}

	if err := store.CreateShippingMethod(ctx, enabledShipping); err != nil {
		t.Fatalf("create enabled shipping: %v", err)
	}
	if err := store.CreateShippingMethod(ctx, disabledShipping); err != nil {
		t.Fatalf("create disabled shipping: %v", err)
	}

	allShipping, err := store.ListShippingMethods(ctx)
	if err != nil {
		t.Fatalf("ListShippingMethods: %v", err)
	}
	if len(allShipping) != 2 {
		t.Fatalf("shipping method count = %d, want 2", len(allShipping))
	}
	var enabledSM *ShippingMethod
	for i := range allShipping {
		if allShipping[i].ID == enabledShipping.ID {
			enabledSM = &allShipping[i]
		}
	}
	if enabledSM == nil {
		t.Fatal("enabled shipping method not found")
	}
	if !enabledSM.Enabled {
		t.Fatalf("enabled shipping Enabled = false, want true (bool adapter scan)")
	}

	// Public list must only show enabled methods
	publicShipping, err := svc.ListPublicShippingMethods(ctx)
	if err != nil {
		t.Fatalf("ListPublicShippingMethods: %v", err)
	}
	if len(publicShipping) != 1 {
		t.Fatalf("public shipping count = %d, want 1 (only enabled)", len(publicShipping))
	}
	if publicShipping[0].ID != enabledShipping.Method {
		t.Fatalf("public shipping ID = %q, want %q (method name)", publicShipping[0].ID, enabledShipping.Method)
	}

	// -------------------------------------------------------------------------
	// 4. Quote with enabled settings + active promo
	// -------------------------------------------------------------------------
	seedProduct(t, store, Product{
		ID:     fmt.Sprintf("pg-product-%d", now),
		SKU:    fmt.Sprintf("PG-SKU-%d", now),
		Name:   "PG Product",
		Slug:   fmt.Sprintf("pg-product-%d", now),
		Status: "active",
		Stock:  10,
		Price:  500,
	})

	quote, err := svc.Quote(ctx, QuoteInput{
		Items:          []OrderItem{{SKU: fmt.Sprintf("PG-SKU-%d", now), Quantity: 2}},
		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		PromoCode:      activePromo.Code,
	})
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	if quote.Subtotal != 1000 {
		t.Fatalf("quote subtotal = %d, want 1000 (500*2)", quote.Subtotal)
	}
	if quote.Discount != 100 {
		t.Fatalf("quote discount = %d, want 100 (fixed promo)", quote.Discount)
	}
	if quote.Shipping != 120 {
		t.Fatalf("quote shipping = %d, want 120 (home_delivery fee)", quote.Shipping)
	}
	if quote.Total != 1020 {
		t.Fatalf("quote total = %d, want 1020 (1000-100+120)", quote.Total)
	}

	// -------------------------------------------------------------------------
	// 5. Create order with enabled settings + active promo
	// -------------------------------------------------------------------------
	order, err := svc.CreateOrder(ctx, OrderInput{
		CustomerName:    "PG Customer",
		Email:           "pg-customer@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 PG St, Taipei",
		Items:           []OrderItem{{SKU: fmt.Sprintf("PG-SKU-%d", now), Name: "PG Product", Price: 999, Quantity: 2}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		PromoCode:       activePromo.Code,
		IdempotencyKey:  fmt.Sprintf("pg-order-key-%d", now),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.Subtotal != 1000 {
		t.Fatalf("order subtotal = %d, want 1000", order.Subtotal)
	}
	if order.Discount != 100 {
		t.Fatalf("order discount = %d, want 100", order.Discount)
	}
	if order.Shipping != 120 {
		t.Fatalf("order shipping = %d, want 120", order.Shipping)
	}
	if order.Total != 1020 {
		t.Fatalf("order total = %d, want 1020", order.Total)
	}
	if order.Status != "pending" {
		t.Fatalf("order status = %q, want pending", order.Status)
	}

	// Verify stock was decremented
	product, err := store.GetProduct(ctx, fmt.Sprintf("pg-product-%d", now))
	if err != nil {
		t.Fatalf("get product after order: %v", err)
	}
	if product.Stock != 8 {
		t.Fatalf("stock after order = %d, want 8 (10-2)", product.Stock)
	}

	// Verify order persisted
	persisted, err := store.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get persisted order: %v", err)
	}
	if persisted.Total != 1020 {
		t.Fatalf("persisted order total = %d, want 1020", persisted.Total)
	}

	// -------------------------------------------------------------------------
	// 6. Disabled configuration produces no order side effects
	// -------------------------------------------------------------------------
	stockBeforeFail := product.Stock

	// Disabled payment method must reject order
	_, err = svc.CreateOrder(ctx, OrderInput{
		CustomerName:    "PG Customer 2",
		Email:           "pg-customer-2@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 PG St, Taipei",
		Items:           []OrderItem{{SKU: fmt.Sprintf("PG-SKU-%d", now), Name: "PG Product", Price: 500, Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   disabledPayment.ID,
		IdempotencyKey:  fmt.Sprintf("pg-order-fail-payment-%d", now),
	})
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("disabled payment order: err = %v, want ErrInvalidPaymentMethod", err)
	}

	// Disabled shipping method must reject order
	_, err = svc.CreateOrder(ctx, OrderInput{
		CustomerName:    "PG Customer 3",
		Email:           "pg-customer-3@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 PG St, Taipei",
		Items:           []OrderItem{{SKU: fmt.Sprintf("PG-SKU-%d", now), Name: "PG Product", Price: 500, Quantity: 1}},
		ShippingMethod:  "store_711",
		PaymentMethod:   "atm",
		IdempotencyKey:  fmt.Sprintf("pg-order-fail-shipping-%d", now),
	})
	if !errors.Is(err, ErrInvalidShippingMethod) {
		t.Fatalf("disabled shipping order: err = %v, want ErrInvalidShippingMethod", err)
	}

	// Disabled promo must reject order
	_, err = svc.CreateOrder(ctx, OrderInput{
		CustomerName:    "PG Customer 4",
		Email:           "pg-customer-4@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 PG St, Taipei",
		Items:           []OrderItem{{SKU: fmt.Sprintf("PG-SKU-%d", now), Name: "PG Product", Price: 500, Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		PromoCode:       disabledPromo.Code,
		IdempotencyKey:  fmt.Sprintf("pg-order-fail-promo-%d", now),
	})
	if !errors.Is(err, ErrInvalidPromoCode) {
		t.Fatalf("disabled promo order: err = %v, want ErrInvalidPromoCode", err)
	}

	// No side effects: stock unchanged, no extra orders
	productAfterFail, err := store.GetProduct(ctx, fmt.Sprintf("pg-product-%d", now))
	if err != nil {
		t.Fatalf("get product after failed orders: %v", err)
	}
	if productAfterFail.Stock != stockBeforeFail {
		t.Fatalf("stock after failed orders = %d, want %d (no side effects)", productAfterFail.Stock, stockBeforeFail)
	}

	orderCount, err := store.CountOrders(ctx)
	if err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 1 {
		t.Fatalf("order count = %d, want 1 (only the successful order)", orderCount)
	}
}
