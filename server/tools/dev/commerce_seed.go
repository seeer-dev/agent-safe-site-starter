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
	"github.com/example/ai-site-starter/server/internal/modules/media"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

func seedCommerce(ctx context.Context, db *sql.DB, dialect database.Dialect, cfg config.Config) error {
	siteTheme := cfg.SiteTheme
	if siteTheme == "" {
		siteTheme = "minimal-cart"
	}
	store := commerce.NewSQLStore(db, dialect)
	service := commerce.NewService(store)
	if siteTheme == "curatory" {
		// 註冊種子圖為已驗證 media objects，讓商品圖走與 API 相同的
		// verified-key 邊界（產品圖不接受裸 URL）。
		seedCuratoryMedia(ctx, db)
		service = service.WithMediaVerifier(devMediaVerifier{
			registry: media.NewSQLRegistryStore(db, dialect),
		})
	}

	existingProducts, err := service.ListProducts(ctx, commerce.ProductFilter{})
	if err != nil {
		return fmt.Errorf("list products for seed: %w", err)
	}
	if len(existingProducts) == 0 {
		products := seedProductsForTheme(siteTheme)
		for _, p := range products {
			if _, err := service.CreateProduct(ctx, devPrincipal, p); err != nil {
				return fmt.Errorf("seed product %s: %w", p.SKU, err)
			}
		}
		if siteTheme == "curatory" {
			// sold_count 由訂單狀態機衍生、不接受輸入；種子用 SQL 直接補，
			// 讓「熱銷」排序與已售數顯示有真實資料可看。
			for sku, sold := range curatorySeedSoldCount {
				if _, err := db.ExecContext(ctx, `UPDATE products SET sold_count = ? WHERE sku = ?`, sold, sku); err != nil {
					return fmt.Errorf("seed sold_count %s: %w", sku, err)
				}
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
		baseMethods := seedPaymentMethodsForTheme(siteTheme)
		for _, method := range baseMethods {
			if err := seedPaymentMethod(ctx, store, method); err != nil {
				return err
			}
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

func seedPaymentMethod(ctx context.Context, store commerce.SQLStore, method commerce.PaymentMethod) error {
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

	runtimeCfg, err := commerce.NewECPayConfig(
		cfg.ECPayEnvironment,
		cfg.PublicAPIBase,
		cfg.PublicSiteURL,
		cfg.ECPayMerchantID,
		cfg.ECPayHashKey,
		cfg.ECPayHashIV,
	)
	if err != nil {
		return commerce.PaymentMethod{}, fmt.Errorf("seed ECPay payment method: runtime config invalid: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(runtimeCfg.Environment)) {
	case "stage":
		method.Environment = "sandbox"
	case "production":
		method.Environment = "production"
	default:
		return commerce.PaymentMethod{}, fmt.Errorf("seed ECPay payment method: unsupported runtime environment %q", runtimeCfg.Environment)
	}
	method.ReadinessStatus = "ready"
	method.Enabled = true
	return method, nil
}
