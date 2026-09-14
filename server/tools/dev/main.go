package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/bootstrap"
	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/modules/sitecontent"
	"github.com/example/ai-site-starter/server/internal/modules/staff"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	siterender "github.com/example/ai-site-starter/server/internal/render"
	"github.com/example/ai-site-starter/server/tools/internal/rendercompose"
)

// devPrincipal is the synthetic admin principal used to seed first-run data.
// Capabilities cover commerce, site content, and staff admin operations.
var devPrincipal = auth.Principal{
	UserID: "dev",
	Role:   "admin",
	Capabilities: []string{
		"twcommerce.read",
		"twcommerce.create",
		"twcommerce.update",
		"twcommerce.delete",
		"twcommerce.admin",
		"content.read",
		"content.create",
		"content.update",
		"content.approve",
		"content.publish",
		"staff.read",
		"staff.update",
	},
}

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, dialect, err := database.Open(ctx, cfg.DBDriver, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := migrate.Apply(ctx, db, dialect, "."); err != nil {
		log.Fatal(err)
	}

	siteTheme := cfg.SiteTheme
	if siteTheme == "" {
		siteTheme = "minimal-cart"
	}

	// ----- Article seed -----------------------------------------------------
	store := content.NewSQLStore(db, dialect)
	count, err := store.Count(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 && shouldSeedSampleArticle(siteTheme) {
		service := content.NewService(store)
		_, err := service.Publish(ctx, auth.Principal{UserID: "dev", Role: "admin"}, content.UpsertInput{
			Slug:      "hello-starter",
			Title:     "Hello from the starter",
			Excerpt:   "SQLite -> Go renderer -> static HTML. No frontend framework required.",
			BodyHTML:  "<p>This page was rendered from the local SQLite database.</p>",
			Published: true,
		})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("seed: 1 sample article")
	}

	// ----- Commerce seed ----------------------------------------------------
	commerceStore := commerce.NewSQLStore(db, dialect)
	commerceService := commerce.NewService(commerceStore)
	existingProducts, err := commerceService.ListProducts(ctx, commerce.ProductFilter{})
	if err != nil {
		log.Fatal(err)
	}
	if len(existingProducts) == 0 {
		products := seedProductsForTheme(siteTheme)
		for _, p := range products {
			if _, err := commerceService.CreateProduct(ctx, devPrincipal, p); err != nil {
				log.Fatalf("seed product %s: %v", p.SKU, err)
			}
		}
		log.Printf("seed: %d commerce products", len(products))
	}

	// Payment methods: there is no create service method, so seed via the
	// store directly using PaymentMethod structs with generated IDs.
	existingPMs, err := commerceStore.ListPaymentMethods(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if len(existingPMs) == 0 {
		paymentMethods := seedPaymentMethodsForTheme(siteTheme)
		now := time.Now().Unix()
		for i := range paymentMethods {
			id, err := randomID()
			if err != nil {
				log.Fatalf("seed payment method: %v", err)
			}
			paymentMethods[i].ID = id
			paymentMethods[i].UpdatedUnix = now
			if err := commerceStore.UpsertPaymentMethod(ctx, paymentMethods[i]); err != nil {
				log.Fatalf("seed payment method %s: %v", paymentMethods[i].Method, err)
			}
		}
		log.Printf("seed: %d payment methods", len(paymentMethods))
	}

	// ----- Site content seed ------------------------------------------------
	sitecontentStore := sitecontent.NewSQLStore(db, dialect)
	sitecontentService := sitecontent.NewService(sitecontentStore)
	existingContent, err := sitecontentStore.ListAll(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if len(existingContent) == 0 {
		siteContents := []sitecontent.SiteContentInput{
			{Key: "home.hero", Placement: "hero", Title: "質物選物", Body: "為生活選一件好物。", Status: "published"},
			{Key: "home.popup", Placement: "popup", Title: "歡迎光臨", Body: "感謝您的光臨。", Status: "published"},
			{Key: "footer.about", Placement: "footer", Title: "關於質物", Body: "質物選物的品牌故事將於正式上線前公告。", Status: "published"},
		}
		for _, sc := range siteContents {
			created, err := sitecontentService.Create(ctx, devPrincipal, sc)
			if err != nil {
				log.Fatalf("seed site content %s: %v", sc.Key, err)
			}
			// Create saves as draft; approve then publish so the renderer includes it.
			if _, err := sitecontentService.Approve(ctx, devPrincipal, created.ID, sitecontent.ApprovalInput{
				ExpiryUnix:           time.Now().Add(24 * time.Hour).Unix(),
				ExpectedDraftVersion: created.DraftVersion,
			}); err != nil {
				log.Fatalf("approve site content %s: %v", sc.Key, err)
			}
			if _, err := sitecontentService.Publish(ctx, devPrincipal, created.ID, sitecontent.PublishInput{
				ExpectedDraftVersion: created.DraftVersion,
			}); err != nil {
				log.Fatalf("publish site content %s: %v", sc.Key, err)
			}
		}
		log.Printf("seed: %d site content blocks", len(siteContents))
	}

	// ----- Staff seed -------------------------------------------------------
	staffStore := staff.NewSQLStore(db, dialect)
	staffService := staff.NewService(staffStore)
	existingStaff, err := staffStore.List(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if len(existingStaff) == 0 {
		staffMembers := []staff.StaffMemberInput{
			{DisplayName: "質物 Owner", Email: "owner@tw-minimal-cart.local", RoleLabel: "owner"},
			{DisplayName: "質物 Manager", Email: "manager@tw-minimal-cart.local", RoleLabel: "manager"},
			{DisplayName: "質物 Staff", Email: "staff@tw-minimal-cart.local", RoleLabel: "readonly"},
		}
		for _, sm := range staffMembers {
			if _, err := staffService.Create(ctx, devPrincipal, sm); err != nil {
				log.Fatalf("seed staff %s: %v", sm.Email, err)
			}
		}
		log.Printf("seed: %d staff members", len(staffMembers))
	}

	if siteTheme == "curatory" {
		seedCuratory(ctx, commerceService, sitecontentService, content.NewService(store))
	}

	// ----- Render -----------------------------------------------------------
	// Compose the full minimal-cart public input (articles, products,
	// categories, site content) using the same shared composition as
	// server/tools/render. This ensures dev renders the integrated
	// product/category/content site, not legacy article-only output.
	input, err := rendercompose.Compose(ctx, db, dialect, cfg.R2PublicBaseURL, siteTheme)
	if err != nil {
		log.Fatalf("compose render input: %v", err)
	}
	// In dev, the site server proxies /api/* to the API server, so the
	// browser uses same-origin relative paths. This avoids cross-origin
	// fetch failures without weakening production CORS. We render with
	// an empty PublicAPIBase so data-api-base is empty and the browser
	// fetches /api/products from the same origin as the page.
	devCfg := cfg
	devCfg.PublicAPIBase = ""
	renderer := siterender.New(rendererConfig(devCfg, siteTheme))
	if err := renderer.RenderSite(input); err != nil {
		log.Fatalf("render failed (dist preserved): %v", err)
	}
	log.Printf("rendered %d article(s), %d product(s), %d categor(y/ies), %d content page(s), %d static page(s) into dist/",
		len(input.Articles), len(input.Products), len(input.Categories), len(input.ContentBlocks), len(input.Pages))

	app, err := bootstrap.NewWithDB(ctx, cfg, db, dialect)
	if err != nil {
		log.Fatal(err)
	}
	apiServer := &http.Server{Addr: cfg.HTTPAddr, Handler: app.Handler, ReadHeaderTimeout: 5 * time.Second}
	siteServer := &http.Server{Addr: cfg.SiteAddr, Handler: devSiteHandler(cfg.HTTPAddr, "dist"), ReadHeaderTimeout: 5 * time.Second}

	errCh := make(chan error, 2)
	go func() {
		log.Printf("api:  http://localhost%s", cfg.HTTPAddr)
		errCh <- apiServer.ListenAndServe()
	}()
	go func() {
		log.Printf("site: http://localhost%s", cfg.SiteAddr)
		errCh <- siteServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server stopped: %v", err)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = apiServer.Shutdown(shutdownCtx)
	_ = siteServer.Shutdown(shutdownCtx)
}

// seedProductsForTheme returns the first-run product catalogue. The
// curatory theme gets a fuller catalogue matching the reference
// storefront (featured flags, long descriptions, variants).
func seedProductsForTheme(siteTheme string) []commerce.ProductInput {
	if siteTheme == "curatory" {
		return []commerce.ProductInput{
			{SKU: "CUR-TEA-01", Name: "手作粗陶馬克杯", Slug: "stoneware-mug", Category: "tea-ware", Status: "active", IsFeatured: true, Price: 880, OriginalPrice: 1080, Stock: 32, Tag: "職人手作", Material: "粗陶", Origin: "鶯歌",
				Description:     "手拉坯粗陶馬克杯，霧面釉色，每一只的釉痕都不相同。",
				LongDescription: "以鶯歌在地陶土手拉坯成形，1300°C 高溫燒製。杯緣修得內斂，握感厚實；霧面釉在光線下呈現深淺不一的土色層次。容量約 280ml，適合手沖咖啡與台灣茶。",
				Variants: []commerce.ProductVariantInput{
					{Name: "米白", SKU: "CUR-TEA-01-A", PriceDelta: 0, Stock: 18},
					{Name: "墨黑", SKU: "CUR-TEA-01-B", PriceDelta: 50, Stock: 14},
				}},
			{SKU: "CUR-TEA-02", Name: "柴燒小茶壺", Slug: "woodfired-teapot", Category: "tea-ware", Status: "active", IsFeatured: true, Price: 2680, Stock: 6, Tag: "限量", Material: "陶", Origin: "台中",
				Description:     "柴燒落灰自然成釉，壺身火痕流動。",
				LongDescription: "五天四夜柴窯燒成，落灰於壺肩形成天然灰釉。容量約 180ml，適合一至二人沖泡。出水流暢、斷水利落。"},
			{SKU: "CUR-APP-01", Name: "植物染亞麻襯衫", Slug: "plant-dyed-linen-shirt", Category: "apparel", Status: "active", IsFeatured: true, Price: 2280, OriginalPrice: 2680, Stock: 15, Tag: "植物染", Material: "亞麻", Origin: "台灣",
				Description:     "以薯榔與藍染染製的亞麻襯衫，色澤會隨時間變化。",
				LongDescription: "選用歐洲亞麻布，由台灣染整職人以植物染工序製成。植物染的特性是色落不均、越穿越柔。寬鬆剪裁，男女皆可。",
				Variants: []commerce.ProductVariantInput{
					{Name: "薯榔褐 M", SKU: "CUR-APP-01-M", PriceDelta: 0, Stock: 8},
					{Name: "薯榔褐 L", SKU: "CUR-APP-01-L", PriceDelta: 0, Stock: 7},
				}},
			{SKU: "CUR-STA-01", Name: "手工線裝筆記本", Slug: "thread-bound-notebook", Category: "stationery", Status: "active", Price: 320, Stock: 64, Material: "紙", Origin: "台灣",
				Description:     "傳統四目線裝，內頁為台灣產手工抄紙。",
				LongDescription: "封面採用再生紙漿壓紋，內頁 80 頁空白手抄紙。線裝可 180 度攤平，適合鋼筆書寫。"},
			{SKU: "CUR-TEA-03", Name: "杉林溪高山烏龍", Slug: "high-mountain-oolong", Category: "pantry", Status: "active", Price: 680, Stock: 40, Tag: "茶", Material: "茶葉", Origin: "南投",
				Description:     "海拔 1600 公尺茶園手採，輕發酵清香型。",
				LongDescription: "杉林溪茶區春茶，輕發酵輕焙火。湯色蜜綠、帶天然花香與冷礦韻。每罐 150g 散茶。"},
			{SKU: "CUR-ACC-01", Name: "植鞣革長夾", Slug: "vegetable-tanned-wallet", Category: "apparel", Status: "active", Price: 1480, Stock: 0, Tag: "手工", Material: "植鞣牛皮", Origin: "台南",
				Description:     "台南皮件工坊手縫長夾，養成後呈深琥珀色。",
				LongDescription: "義大利植鞣協會認證牛皮，黃銅五金。八卡層、雙鈔票夾層與拉鍊零錢袋。植鞣革會隨使用產生獨特色澤。"},
			{SKU: "CUR-HOME-01", Name: "竹編托盤", Slug: "bamboo-tray", Category: "home-goods", Status: "active", Price: 540, Stock: 21, Material: "孟宗竹", Origin: "南投",
				Description:     "南投竹藝職人編製，收邊紮實。",
				LongDescription: "孟宗竹剖篾後手工編製，表面無漆僅上天然護木油。直徑 28cm，可作茶盤或早餐托盤。"},
			{SKU: "CUR-STA-02", Name: "黃銅書鎮", Slug: "brass-paperweight", Category: "stationery", Status: "draft", Price: 760, Stock: 9, Material: "黃銅", Origin: "台灣",
				Description: "CNC 車製實心黃銅，底部貼軟木。"},
		}
	}
	return []commerce.ProductInput{
		{SKU: "SKU-APP-01", Name: "素面亞麻襯衫", Slug: "linen-shirt", Description: "透氣亞麻襯衫，適合台灣夏季。", Category: "apparel", Status: "active", Material: "亞麻", Origin: "台灣", Price: 1680, OriginalPrice: 1980, Stock: 24, Tag: "經典"},
		{SKU: "SKU-HOME-02", Name: "陶製小碟", Slug: "ceramic-dish", Description: "手作陶碟，每一件獨一無二。", Category: "home", Status: "active", Material: "陶", Origin: "鶯歌", Price: 420, Stock: 2, Tag: ""},
		{SKU: "SKU-ACC-03", Name: "植鞣皮名片夾", Slug: "leather-card-holder", Description: "植鞣牛皮名片夾，使用越久越有味道。", Category: "accessories", Status: "active", Material: "牛皮", Origin: "台南", Price: 980, Stock: 0, Tag: "手工"},
		{SKU: "SKU-STA-04", Name: "線裝筆記本", Slug: "thread-bound-notebook", Description: "傳統線裝筆記本，書寫流暢。", Category: "stationery", Status: "active", Material: "紙", Origin: "台灣", Price: 260, Stock: 88, Tag: ""},
		{SKU: "SKU-APP-05", Name: "寬版工作褲", Slug: "wide-work-pants", Description: "寬版剪裁工作褲，舒適耐穿。", Category: "apparel", Status: "draft", Material: "棉", Origin: "台灣", Price: 2280, Stock: 11, Tag: "新品"},
		{SKU: "SKU-HOME-06", Name: "手抄紙燈罩", Slug: "paper-lampshade", Description: "埔里手抄紙燈罩，溫暖柔光。", Category: "home", Status: "active", Material: "紙", Origin: "埔里", Price: 1540, Stock: 5, Tag: ""},
	}
}

// seedPaymentMethodsForTheme returns the first-run payment methods.
// Curatory adds LINE Pay and per-method handling fees; minimal-cart
// keeps its original three-method baseline.
func seedPaymentMethodsForTheme(siteTheme string) []commerce.PaymentMethod {
	if siteTheme == "curatory" {
		return []commerce.PaymentMethod{
			{Method: "credit_card", ProviderLabel: "信用卡（ECPay 沙箱）", Environment: "sandbox", ReadinessStatus: "ready", Enabled: true, Fee: 0},
			{Method: "line_pay", ProviderLabel: "LINE Pay", Environment: "sandbox", ReadinessStatus: "ready", Enabled: true, Fee: 0},
			{Method: "cod", ProviderLabel: "貨到付款", Environment: "production", ReadinessStatus: "ready", Enabled: true, Fee: 15},
			{Method: "atm", ProviderLabel: "銀行ATM轉帳", Environment: "production", ReadinessStatus: "ready", Enabled: true, Fee: 15},
		}
	}
	return []commerce.PaymentMethod{
		{Method: "atm", ProviderLabel: "銀行ATM轉帳", Environment: "production", ReadinessStatus: "ready", Enabled: true},
		{Method: "cod", ProviderLabel: "貨到付款", Environment: "production", ReadinessStatus: "ready", Enabled: true},
		{Method: "credit_card", ProviderLabel: "信用卡（尚未啟用）", Environment: "sandbox", ReadinessStatus: "pending_setup", Enabled: false},
	}
}

// seedCuratory populates the curatory storefront's first-run data:
// categories, shipping methods, promos, news articles, product comments,
// notification templates, and published store settings. Each block is
// independently idempotent — it only seeds when its collection is empty.
func seedCuratory(ctx context.Context, commerceService commerce.Service, sitecontentService sitecontent.Service, contentService content.Service) {
	// Categories -----------------------------------------------------------
	if cats, err := commerceService.ListCategories(ctx); err == nil && len(cats) == 0 {
		categories := []commerce.CategoryInput{
			{Slug: "tea-ware", Name: "茶器物", Description: "陶、瓷、漆器——讓一杯茶變得更好的器物。", IsActive: true},
			{Slug: "apparel", Name: "職人衣著", Description: "植物染、亞麻、手縫皮件。", IsActive: true},
			{Slug: "stationery", Name: "生活文具", Description: "紙、筆與桌上小物。", IsActive: true},
			{Slug: "pantry", Name: "日常飲食", Description: "台灣茶與發酵風味。", IsActive: true},
			{Slug: "home-goods", Name: "居家器物", Description: "竹編、木作與生活道具。", IsActive: true},
		}
		for _, c := range categories {
			if _, err := commerceService.CreateCategory(ctx, devPrincipal, c); err != nil {
				log.Fatalf("seed category %s: %v", c.Slug, err)
			}
		}
		log.Printf("seed: %d categories", len(categories))
	}

	// Shipping methods -----------------------------------------------------
	if methods, err := commerceService.ListShippingMethods(ctx, devPrincipal); err == nil && len(methods) == 0 {
		freeThreshold := 1500
		shipping := []commerce.ShippingMethodInput{
			{Method: "home_delivery", Label: "宅配到府", Description: "黑貓宅急便，1-2 個工作天到貨。", Fee: 80, FreeThreshold: &freeThreshold, Enabled: true},
			{Method: "cvs_711", Label: "7-ELEVEN 取貨", Description: "全台門市取貨，到店簡訊通知。", Fee: 60, Enabled: true},
			{Method: "cvs_family", Label: "全家取貨", Description: "全台門市取貨，到店簡訊通知。", Fee: 60, Enabled: true},
		}
		for _, sm := range shipping {
			if _, err := commerceService.CreateShippingMethod(ctx, devPrincipal, sm); err != nil {
				log.Fatalf("seed shipping method %s: %v", sm.Method, err)
			}
		}
		log.Printf("seed: %d shipping methods", len(shipping))
	}

	// Promos ---------------------------------------------------------------
	if promos, err := commerceService.ListPromos(ctx); err == nil && len(promos) == 0 {
		now := time.Now().Unix()
		promoInputs := []commerce.PromoInput{
			{Code: "WELCOME10", Label: "新朋友九折", Type: "percent", Value: 10, Enabled: true, StartsUnix: now},
			{Code: "FREESHIP", Label: "滿額免運", Type: "freeshipping", Value: 0, Enabled: true, MinSubtotal: 800, StartsUnix: now},
		}
		for _, p := range promoInputs {
			if _, err := commerceService.CreatePromo(ctx, devPrincipal, p); err != nil {
				log.Fatalf("seed promo %s: %v", p.Code, err)
			}
		}
		log.Printf("seed: %d promos", len(promoInputs))
	}

	// News articles --------------------------------------------------------
	seedCuratoryArticles(ctx, contentService)

	// Product comments -----------------------------------------------------
	seedCuratoryComments(ctx, commerceService)

	// Notification templates ------------------------------------------------
	if tpls, err := commerceService.ListNotificationTemplates(ctx, devPrincipal); err == nil && len(tpls) == 0 {
		templates := []commerce.NotificationTemplateInput{
			{Code: "order_placed", Name: "訂單成立", Subject: "【質選所】訂單成立通知 {{order_id}}", Body: "{{customer_name}} 您好，您的訂單 {{order_id}} 已成立，金額 NT$ {{total}}。我們將盡快為您出貨。", IsEnabled: true},
			{Code: "order_paid", Name: "付款完成", Subject: "【質選所】付款完成 {{order_id}}", Body: "{{customer_name}} 您好，訂單 {{order_id}} 已確認收款，感謝您的購買。", IsEnabled: true},
			{Code: "order_shipped", Name: "訂單出貨", Subject: "【質選所】訂單已出貨 {{order_id}}", Body: "{{customer_name}} 您好，訂單 {{order_id}} 已交由物流配送，請留意到貨通知。", IsEnabled: true},
			{Code: "order_completed", Name: "訂單完成", Subject: "【質選所】訂單完成 {{order_id}}", Body: "{{customer_name}} 您好，訂單 {{order_id}} 已完成。期待您再次光臨質選所。", IsEnabled: true},
			{Code: "order_cancelled", Name: "訂單取消", Subject: "【質選所】訂單取消 {{order_id}}", Body: "{{customer_name}} 您好，訂單 {{order_id}} 已取消。如有疑問請與我們聯繫。", IsEnabled: true},
		}
		for _, t := range templates {
			if _, err := commerceService.UpsertNotificationTemplate(ctx, devPrincipal, "", t); err != nil {
				log.Fatalf("seed notification template %s: %v", t.Code, err)
			}
		}
		log.Printf("seed: %d notification templates", len(templates))
	}

	// Store settings (draft -> publish) -------------------------------------
	settings, err := sitecontentService.GetStoreSettings(ctx, devPrincipal)
	if err != nil {
		log.Fatalf("load store settings: %v", err)
	}
	if string(settings.Published) == "{}" || len(settings.Published) == 0 {
		draft := []byte(`{
			"storeName": "質選所",
			"storeNameEn": "CURATORY",
			"tagline": "為生活選一件好物",
			"promoBanner": "全館滿 NT$1,500 免運 · 新朋友輸入 WELCOME10 享九折",
			"promoBannerEnabled": true,
			"freeShippingThreshold": 1500,
			"lowStockThreshold": 5,
			"notificationMaster": true,
			"contactEmail": "hello@curatory.example",
			"contactPhone": "02-2345-6789"
		}`)
		updated, err := sitecontentService.UpdateStoreSettingsDraft(ctx, devPrincipal, sitecontent.StoreSettingsInput{
			Draft:           draft,
			ExpectedVersion: settings.Version,
		})
		if err != nil {
			log.Fatalf("seed store settings draft: %v", err)
		}
		if _, err := sitecontentService.PublishStoreSettings(ctx, devPrincipal, sitecontent.PublishInput{
			ExpectedDraftVersion: updated.Version,
		}); err != nil {
			log.Fatalf("seed store settings publish: %v", err)
		}
		log.Printf("seed: store settings published")
	}
}

// seedCuratoryArticles publishes three news posts when none exist —
// one pinned announcement plus two regular entries so the pinned-first
// ordering and the news list layout are both exercised.
func seedCuratoryArticles(ctx context.Context, contentService content.Service) {
	articles, err := contentService.ListAll(ctx, devPrincipal)
	if err != nil {
		log.Fatalf("list articles: %v", err)
	}
	if len(articles) > 0 {
		return
	}
	now := time.Now().Unix()
	entries := []content.UpsertInput{
		{Slug: "opening-notice", Title: "質選所開幕：選品理念與運送服務說明", Pinned: true, Published: true, PublishedAt: now - 86400*7,
			Excerpt:  "質選所正式開幕。說明我們的選品標準、出貨時間與退換貨方式。",
			BodyHTML: "<p>質選所正式開幕了。我們只選用得來不易、製作者可被追溯的物件：陶、紙、皮、竹與茶。</p><p>全館滿 NT$1,500 免運；訂單於工作日 48 小時內出貨。收到商品七日內若有瑕疵，請來信附上照片，我們會直接換新或退款。</p>"},
		{Slug: "woodfired-kiln-visit", Title: "柴窯見學：一窯五天四夜的等待", Published: true, PublishedAt: now - 86400*3,
			Excerpt:  "本月走訪台中柴窯，記錄新一窯茶器的燒成過程。",
			BodyHTML: "<p>柴燒的魅力在於不可控。投柴的節奏、窯內氣流與落灰的位置，共同決定每一件作品的表情。</p><p>本批柴燒小茶壺共出窯十二只，已全數上架，售完後需待下一窯。</p>"},
		{Slug: "linen-care-guide", Title: "植物染衣物的日常照護", Published: true, PublishedAt: now - 86400,
			Excerpt:  "植物染會隨穿著與洗滌緩慢變化——這正是它的迷人之處。",
			BodyHTML: "<p>植物染衣物請以冷水單獨手洗，使用中性洗劑，避免長時間曝曬。初期色落屬正常現象，數次洗滌後即趨穩定。</p>"},
	}
	for _, a := range entries {
		if _, err := contentService.Publish(ctx, devPrincipal, a); err != nil {
			log.Fatalf("seed article %s: %v", a.Slug, err)
		}
	}
	log.Printf("seed: %d articles", len(entries))
}

// seedCuratoryComments leaves a few customer comments on seeded
// products: two approved (one with an admin reply) and one pending, so
// both the storefront display and the moderation queue are populated.
func seedCuratoryComments(ctx context.Context, commerceService commerce.Service) {
	if comments, err := commerceService.ListComments(ctx, devPrincipal, ""); err != nil || len(comments) > 0 {
		return
	}
	products, err := commerceService.ListProducts(ctx, commerce.ProductFilter{})
	if err != nil || len(products) == 0 {
		return
	}
	bySlug := map[string]string{}
	for _, p := range products {
		bySlug[p.Slug] = p.ID
	}
	five, four := 5, 4
	type commentSeed struct {
		slug    string
		in      commerce.CommentInput
		approve bool
		reply   string
	}
	seeds := []commentSeed{
		{slug: "stoneware-mug", in: commerce.CommentInput{Nickname: "陳小姐", Content: "杯子比照片還好看，釉色層次很細。已經變成每天早上的咖啡杯。", Rating: &five}, approve: true, reply: "謝謝喜歡！粗陶杯會越用越有光澤，期待它陪您很久。"},
		{slug: "stoneware-mug", in: commerce.CommentInput{Nickname: "Wei", Content: "包裝很用心，送禮很體面。", Rating: &four}, approve: true},
		{slug: "plant-dyed-linen-shirt", in: commerce.CommentInput{Nickname: "阿翰", Content: "布料很舒服，第一次洗有點掉色但之後就穩定了。", Rating: &five}, approve: true},
		{slug: "high-mountain-oolong", in: commerce.CommentInput{Nickname: "老茶客", Content: "冷泡也很好喝，會回購。", Rating: &five}, approve: false},
	}
	for _, s := range seeds {
		productID, ok := bySlug[s.slug]
		if !ok {
			continue
		}
		c, err := commerceService.SubmitComment(ctx, productID, s.in)
		if err != nil {
			log.Fatalf("seed comment on %s: %v", s.slug, err)
		}
		if s.approve {
			if _, err := commerceService.ModerateComment(ctx, devPrincipal, c.ID, "approved", s.reply); err != nil {
				log.Fatalf("approve comment %s: %v", c.ID, err)
			}
		}
	}
	log.Printf("seed: %d product comments", len(seeds))
}

// randomID generates a 32-char hex id, matching the pattern used by the
// commerce, sitecontent, and staff modules.
func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func siteNameForTheme(siteTheme string) string {
	switch siteTheme {
	case "minimal-cart":
		return "質物選物"
	case "curatory":
		return "質選所"
	}
	return "AI Site Starter"
}

// articleDirForTheme selects the article detail output directory.
// The curatory theme brands articles as news and links to /news/<slug>/.
func articleDirForTheme(siteTheme string) string {
	if siteTheme == "curatory" {
		return "news"
	}
	return ""
}

func shouldSeedSampleArticle(siteTheme string) bool {
	return siteTheme != "minimal-cart" && siteTheme != "curatory"
}

func rendererConfig(cfg config.Config, siteTheme string) siterender.Config {
	return siterender.Config{
		SiteName:        siteNameForTheme(siteTheme),
		PublicSiteURL:   cfg.PublicSiteURL,
		PublicAPIBase:   cfg.PublicAPIBase,
		TemplateDir:     "site/templates",
		AssetDir:        "site/assets",
		OutputDir:       "dist",
		SiteTheme:       siteTheme,
		R2PublicBaseURL: cfg.R2PublicBaseURL,
		SupabaseURL:     cfg.SupabaseURL,
		ArticleDir:      articleDirForTheme(siteTheme),
	}
}

// devSiteHandler returns an http.Handler that serves static files from
// staticDir and proxies /api/* requests to the API server at apiAddr.
// This makes the dev site same-origin for the browser — no CORS
// configuration is needed. The proxy is dev-only; production uses
// Cloudflare Pages with the configured PublicAPIBase.
//
// apiAddr is the HTTPAddr of the API server. It may be ":8080" (leading
// colon, implicit localhost) or "host:port" (explicit host). The target
// is resolved to a validated host:port string. Reverse proxy errors
// return 502 and do not fall through to static content.
func devSiteHandler(apiAddr, staticDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(staticDir))
	target := resolveDevAPIAddr(apiAddr)
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = target
			req.Host = target
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, fmt.Sprintf("dev proxy: %v", err), http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			proxy.ServeHTTP(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// resolveDevAPIAddr converts an API listen address into a host:port
// string suitable for use as a reverse proxy target. A leading-colon
// address (e.g. ":8080") is resolved to "localhost:8080"; an explicit
// host:port (e.g. "127.0.0.1:8080") is passed through unchanged.
func resolveDevAPIAddr(apiAddr string) string {
	if strings.HasPrefix(apiAddr, ":") {
		return "localhost" + apiAddr
	}
	return apiAddr
}
