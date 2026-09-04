package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

func seedCommerce(ctx context.Context, db *sql.DB, dialect database.Dialect, cfg config.Config) error {
	store := commerce.NewSQLStore(db, dialect)
	service := commerce.NewService(store)

	existingProducts, err := service.ListProducts(ctx, commerce.ProductFilter{})
	if err != nil {
		return fmt.Errorf("list products for seed: %w", err)
	}
	if len(existingProducts) == 0 {
		products := []commerce.ProductInput{
			{SKU: "SKU-APP-01", Name: "素面亞麻襯衫", Slug: "linen-shirt", Description: "透氣亞麻襯衫，適合台灣夏季。", Category: "apparel", Status: "active", Material: "亞麻", Origin: "台灣", Price: 1680, OriginalPrice: 1980, Stock: 24, Tag: "經典"},
			{SKU: "SKU-HOME-02", Name: "陶製小碟", Slug: "ceramic-dish", Description: "手作陶碟，每一件獨一無二。", Category: "home", Status: "active", Material: "陶", Origin: "鶯歌", Price: 420, Stock: 2, Tag: ""},
			{SKU: "SKU-ACC-03", Name: "植鞣皮名片夾", Slug: "leather-card-holder", Description: "植鞣牛皮名片夾，使用越久越有味道。", Category: "accessories", Status: "active", Material: "牛皮", Origin: "台南", Price: 980, Stock: 0, Tag: "手工"},
			{SKU: "SKU-STA-04", Name: "線裝筆記本", Slug: "thread-bound-notebook", Description: "傳統線裝筆記本，書寫流暢。", Category: "stationery", Status: "active", Material: "紙", Origin: "台灣", Price: 260, Stock: 88, Tag: ""},
			{SKU: "SKU-APP-05", Name: "寬版工作褲", Slug: "wide-work-pants", Description: "寬版剪裁工作褲，舒適耐穿。", Category: "apparel", Status: "draft", Material: "棉", Origin: "台灣", Price: 2280, Stock: 11, Tag: "新品"},
			{SKU: "SKU-HOME-06", Name: "手抄紙燈罩", Slug: "paper-lampshade", Description: "埔里手抄紙燈罩，溫暖柔光。", Category: "home", Status: "active", Material: "紙", Origin: "埔里", Price: 1540, Stock: 5, Tag: ""},
		}
		for _, p := range products {
			if _, err := service.CreateProduct(ctx, devPrincipal, p); err != nil {
				return fmt.Errorf("seed product %s: %w", p.SKU, err)
			}
		}
		log.Printf("seed: %d commerce products", len(products))
	}

	existingShipping, err := store.ListShippingMethods(ctx)
	if err != nil {
		return fmt.Errorf("list shipping methods for seed: %w", err)
	}
	if len(existingShipping) == 0 {
		freeThreshold := 2000
		if _, err := service.CreateShippingMethod(ctx, devPrincipal, commerce.ShippingMethodInput{
			Method:        "home_delivery",
			Label:         "宅配",
			Description:   "台灣本島宅配",
			Fee:           100,
			FreeThreshold: &freeThreshold,
			Enabled:       true,
			SortOrder:     10,
		}); err != nil {
			return fmt.Errorf("seed shipping method: %w", err)
		}
		log.Printf("seed: 1 shipping method")
	}

	existingPayments, err := store.ListPaymentMethods(ctx)
	if err != nil {
		return fmt.Errorf("list payment methods for seed: %w", err)
	}
	if len(existingPayments) == 0 {
		baseMethods := []commerce.PaymentMethod{
			{Method: "atm", ProviderLabel: "銀行ATM轉帳", Environment: "production", ReadinessStatus: "ready", Enabled: true},
			{Method: "cod", ProviderLabel: "貨到付款", Environment: "production", ReadinessStatus: "ready", Enabled: true},
			{Method: "credit_card", ProviderLabel: "信用卡（尚未啟用）", Environment: "sandbox", ReadinessStatus: "pending_setup", Enabled: false},
		}
		for _, method := range baseMethods {
			if err := seedPaymentMethod(ctx, store, method); err != nil {
				return err
			}
		log.Printf("seed: %d base payment methods", len(baseMethods))
	}

	if !hasPaymentMethod(existingPayments, "ecpay") {
		ecpayMethod, err := seededECPayPaymentMethod(cfg)
		if err != nil {
			return err
		}
		if err := seedPaymentMethod(ctx, store, ecpayMethod); err != nil {
			return err
		}
		log.Printf("seed: ECPay payment method (%s, enabled=%v, readiness=%s)", ecpayMethod.Environment, ecpayMethod.Enabled, ecpayMethod.ReadinessStatus)
	}

	return nil
}

func seedPaymentMethod(ctx context.Context, store *commerce.SQLStore, method commerce.PaymentMethod) error {
	id, err := randomID()
	if err != nil {
		return fmt.Errorf("seed payment method %s id: %w", method.Method, err)
	}
	method.ID = id
	method.UpdatedUnix = time.Now().Unix()
	if err := store.UpsertPaymentMethod(ctx, method); err != nil {
		return fmt.Errorf("seed payment method %s: %w", method.Method, err)
	}
	return nil
}

func hasPaymentMethod(methods []commerce.PaymentMethod, method string) bool {
	for _, candidate := range methods {
		if strings.EqualFold(strings.TrimSpace(candidate.Method), strings.TrimSpace(method)) {
			return true
		}
	}
	return false
}

func seededECPayPaymentMethod(cfg config.Config) (commerce.PaymentMethod, error) {
	method := commerce.PaymentMethod{
		Method:          "ecpay",
		ProviderLabel:   "綠界信用卡",
		Environment:     "sandbox",
		ReadinessStatus: "pending_setup",
		Enabled:         false,
	}
	if !cfg.ECPayEnabled() {
		return method, nil
	}

	switch strings.ToLower(strings.TrimSpace(cfg.ECPayEnvironment)) {
	case "stage":
		method.Environment = "sandbox"
	case "production":
		method.Environment = "production"
	default:
		return commerce.PaymentMethod{}, fmt.Errorf("seed ECPay payment method: unsupported runtime environment %q", cfg.ECPayEnvironment)
	}
	method.ReadinessStatus = "ready"
	method.Enabled = true
	return method, nil
}
