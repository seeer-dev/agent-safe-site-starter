package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

func openFreshSeedDB(t *testing.T) (*sql.DB, database.Dialect) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "site.db"))
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	if err := migrate.Apply(ctx, db, dialect, repoRoot); err != nil {
		t.Fatalf("migrate.Apply: %v", err)
	}
	return db, dialect
}

func findProductBySKU(products []commerce.Product, sku string) (commerce.Product, bool) {
	for _, product := range products {
		if product.SKU == sku {
			return product, true
		}
	}
	return commerce.Product{}, false
}

func findPublicPayment(methods []commerce.PublicPaymentMethod, method string) (commerce.PublicPaymentMethod, bool) {
	for _, candidate := range methods {
		if candidate.Method == method {
			return candidate, true
		}
	}
	return commerce.PublicPaymentMethod{}, false
}

func findAdminPayment(methods []commerce.PaymentMethod, method string) (commerce.PaymentMethod, bool) {
	for _, candidate := range methods {
		if candidate.Method == method {
			return candidate, true
		}
	}
	return commerce.PaymentMethod{}, false
}

func TestFreshDBCommerceSeedReachesDurableGuestOrder(t *testing.T) {
	ctx := context.Background()
	db, dialect := openFreshSeedDB(t)
	cfg := config.Config{}
	if err := seedCommerce(ctx, db, dialect, cfg); err != nil {
		t.Fatalf("seedCommerce: %v", err)
	}

	store := commerce.NewSQLStore(db, dialect)
	service := commerce.NewService(store)

	products, err := service.ListProducts(ctx, commerce.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(products) != 6 {
		t.Fatalf("seeded products = %d, want 6", len(products))
	}
	product, ok := findProductBySKU(products, "SKU-APP-01")
	if !ok || product.Stock != 24 || product.Price != 1680 {
		t.Fatalf("seeded product = %#v", product)
	}

	shipping, err := service.ListPublicShippingMethods(ctx)
	if err != nil {
		t.Fatalf("ListPublicShippingMethods: %v", err)
	}
	if len(shipping) != 1 || shipping[0].Method != "home_delivery" {
		t.Fatalf("public shipping = %#v", shipping)
	}

	payments, err := service.ListPublicPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("ListPublicPaymentMethods: %v", err)
	}
	atm, ok := findPublicPayment(payments, "atm")
	if !ok {
		t.Fatalf("public payments = %#v, want seeded ATM", payments)
	}
	if _, ok := findPublicPayment(payments, "ecpay"); ok {
		t.Fatalf("unconfigured ECPay must not be public: %#v", payments)
	}

	adminPayments, err := store.ListPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("ListPaymentMethods: %v", err)
	}
	ecpay, ok := findAdminPayment(adminPayments, "ecpay")
	if !ok {
		t.Fatalf("admin payment rows = %#v, want ECPay row", adminPayments)
	}
	if ecpay.Enabled || ecpay.ReadinessStatus != "pending_setup" || ecpay.Environment != "sandbox" {
		t.Fatalf("unconfigured seeded ECPay = %#v", ecpay)
	}

	quote, err := service.Quote(ctx, commerce.QuoteInput{
		Items:          []commerce.OrderItem{{SKU: product.SKU, Quantity: 1, Price: 1}},
		ShippingMethod: shipping[0].ID,
		PaymentMethod:  atm.ID,
	})
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if quote.Subtotal != 1680 || quote.Shipping != 100 || quote.Discount != 0 || quote.Total != 1780 {
		t.Fatalf("quote = %#v", quote)
	}

	order, err := service.CreateOrder(ctx, commerce.OrderInput{
		CustomerName:    "Fresh DB Guest",
		Email:           "guest@example.com",
		Phone:           "0912345678",
		Items:           []commerce.OrderItem{{SKU: product.SKU, Quantity: 1, Price: 1}},
		ShippingAddress: "台北市測試路 1 號",
		ShippingMethod:  shipping[0].ID,
		PaymentMethod:   atm.ID,
		Shipping:        99999,
		Discount:        99999,
		IdempotencyKey:  "fresh-db-guest-order-1",
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if order.ID == "" || order.AccessToken == "" {
		t.Fatalf("created order missing identity/token: %#v", order)
	}
	if order.PaymentStatus != "unpaid" {
		t.Fatalf("payment_status = %q, want unpaid", order.PaymentStatus)
	}
	if order.Subtotal != quote.Subtotal || order.Shipping != quote.Shipping || order.Discount != quote.Discount || order.Total != quote.Total {
		t.Fatalf("order totals %#v do not match quote %#v", order, quote)
	}

	productsAfter, err := service.ListProducts(ctx, commerce.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProducts after order: %v", err)
	}
	productAfter, ok := findProductBySKU(productsAfter, product.SKU)
	if !ok || productAfter.Stock != 23 {
		t.Fatalf("stock after order = %#v, want 23", productAfter)
	}

	// The seed is idempotent: a second run must not duplicate sample rows or
	// overwrite the existing ECPay admin row.
	if err := seedCommerce(ctx, db, dialect, cfg); err != nil {
		t.Fatalf("seedCommerce second run: %v", err)
	}
	productsAgain, _ := service.ListProducts(ctx, commerce.ProductFilter{})
	shippingAgain, _ := service.ListPublicShippingMethods(ctx)
	adminPaymentsAgain, _ := store.ListPaymentMethods(ctx)
	if len(productsAgain) != 6 || len(shippingAgain) != 1 || len(adminPaymentsAgain) != 4 {
		t.Fatalf("idempotent seed counts products=%d shipping=%d payments=%d", len(productsAgain), len(shippingAgain), len(adminPaymentsAgain))
	}
}

func TestFreshDBStageECPaySeedIsReadyAndPublicWithMatchingRuntime(t *testing.T) {
	ctx := context.Background()
	db, dialect := openFreshSeedDB(t)
	cfg := config.Config{
		PublicAPIBase:    "https://api.example.com",
		PublicSiteURL:    "https://shop.example.com",
		ECPayEnvironment: "stage",
		ECPayMerchantID:  "3002607",
		ECPayHashKey:     "pwFHCqoQZGmho4w6",
		ECPayHashIV:      "EkRm7iFT261dpevs",
	}
	if err := seedCommerce(ctx, db, dialect, cfg); err != nil {
		t.Fatalf("seedCommerce: %v", err)
	}

	store := commerce.NewSQLStore(db, dialect)
	adminPayments, err := store.ListPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("ListPaymentMethods: %v", err)
	}
	ecpayRow, ok := findAdminPayment(adminPayments, "ecpay")
	if !ok {
		t.Fatalf("admin payment rows = %#v, want ECPay", adminPayments)
	}
	if !ecpayRow.Enabled || ecpayRow.ReadinessStatus != "ready" || ecpayRow.Environment != "sandbox" {
		t.Fatalf("stage seeded ECPay = %#v", ecpayRow)
	}

	ecpayCfg, err := commerce.NewECPayConfig(
		cfg.ECPayEnvironment,
		cfg.PublicAPIBase,
		cfg.PublicSiteURL,
		cfg.ECPayMerchantID,
		cfg.ECPayHashKey,
		cfg.ECPayHashIV,
	)
	if err != nil {
		t.Fatalf("NewECPayConfig: %v", err)
	}
	service := commerce.NewService(store).WithECPay(ecpayCfg)
	public, err := service.ListPublicPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("ListPublicPaymentMethods: %v", err)
	}
	if _, ok := findPublicPayment(public, "ecpay"); !ok {
		t.Fatalf("public payments = %#v, want ECPay", public)
	}
}
