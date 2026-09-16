package main

import (
	"context"
	"testing"

	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/modules/commerce"
)

func TestFreshDBStageECPayReachesHostedPaymentHandoff(t *testing.T) {
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

	products, err := service.ListProducts(ctx, commerce.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	product, ok := findProductBySKU(products, "SKU-APP-01")
	if !ok {
		t.Fatal("seeded product SKU-APP-01 not found")
	}
	shipping, err := service.ListPublicShippingMethods(ctx)
	if err != nil || len(shipping) == 0 {
		t.Fatalf("ListPublicShippingMethods: methods=%#v err=%v", shipping, err)
	}
	payments, err := service.ListPublicPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("ListPublicPaymentMethods: %v", err)
	}
	ecpayPublic, ok := findPublicPayment(payments, "ecpay")
	if !ok {
		t.Fatalf("public payments = %#v, want ECPay", payments)
	}

	quote, err := service.Quote(ctx, commerce.QuoteInput{
		Items:          []commerce.OrderItem{{SKU: product.SKU, Quantity: 1}},
		ShippingMethod: shipping[0].ID,
		PaymentMethod:  ecpayPublic.ID,
	})
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	order, err := service.CreateOrder(ctx, commerce.OrderInput{
		CustomerName:    "ECPay Fresh DB Guest",
		Email:           "ecpay-guest@example.com",
		Phone:           "0912345678",
		Items:           []commerce.OrderItem{{SKU: product.SKU, Quantity: 1}},
		ShippingAddress: "台北市測試路 2 號",
		ShippingMethod:  shipping[0].ID,
		PaymentMethod:   ecpayPublic.ID,
		IdempotencyKey:  "fresh-db-ecpay-order-1",
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if order.AccessToken == "" {
		t.Fatal("CreateOrder did not return one-time access token")
	}
	if order.Total != quote.Total || order.PaymentStatus != "unpaid" {
		t.Fatalf("order total/payment state = %#v, quote=%#v", order, quote)
	}

	launch, err := service.PrepareECPayPayment(ctx, order.ID, order.AccessToken)
	if err != nil {
		t.Fatalf("PrepareECPayPayment: %v", err)
	}
	if launch.Action != "https://payment-stage.ecpay.com.tw/Cashier/AioCheckOut/V5" {
		t.Fatalf("launch action = %q", launch.Action)
	}
	if launch.Fields["TotalAmount"] != "1780" || launch.Fields["ChoosePayment"] != "Credit" {
		t.Fatalf("launch fields = %#v", launch.Fields)
	}
	if launch.Fields["ReturnURL"] != "https://api.example.com/api/payments/ecpay/return" {
		t.Fatalf("ReturnURL = %q", launch.Fields["ReturnURL"])
	}
	if launch.Fields["OrderResultURL"] != "https://api.example.com/api/payments/ecpay/browser-return" {
		t.Fatalf("OrderResultURL = %q", launch.Fields["OrderResultURL"])
	}
	if launch.Fields["CheckMacValue"] == "" {
		t.Fatal("hosted launch missing CheckMacValue")
	}
}
