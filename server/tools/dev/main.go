package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/bootstrap"
	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/modules/media"
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
	if siteTheme == "curatory" {
		// 註冊種子圖為已驗證 media objects，讓商品圖走與 API 相同的
		// verified-key 邊界（產品圖不接受裸 URL）。
		seedCuratoryMedia(ctx, db)
		commerceService = commerceService.WithMediaVerifier(devMediaVerifier{
			registry: media.NewSQLRegistryStore(db, dialect),
		})
	}
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
		if siteTheme == "curatory" {
			// sold_count 由訂單狀態機衍生、不接受輸入；種子用 SQL 直接補，
			// 讓「熱銷」排序與已售數顯示有真實資料可看。
			for sku, sold := range curatorySeedSoldCount {
				if _, err := db.ExecContext(ctx, `UPDATE products SET sold_count = ? WHERE sku = ?`, sold, sku); err != nil {
					log.Fatalf("seed sold_count %s: %v", sku, err)
				}
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
		// 鏡像 reference storefront 的 12 件商品：名稱、價格、特價
		// （salePrice→price/original_price）、精選旗標、規格與種子圖。
		// 圖片以 verified media key 關聯（seed/images/pXX.png），
		// sold_count 於建立後由 curatorySeedSoldCount 補上。
		img := func(name string) []commerce.ProductImageInput {
			return []commerce.ProductImageInput{{Key: "images/" + name}}
		}
		return []commerce.ProductInput{
			{SKU: "CUR-TAB-01", Name: "手作釉彩馬克杯・霧灰", Slug: "glazed-mug-grey", Category: "tableware", Status: "active", IsFeatured: true, Price: 480, Stock: 37, Tag: "手工釉燒", Material: "陶瓷", Origin: "苗栗",
				Description:     "匠人以還原燒製成的霧灰釉色，每只的流釉紋理皆不相同，是獨一無二的一只杯。",
				LongDescription: "來自苗栗陶窯的手作器皿，採用天然礦物釉料，以 1230°C 高溫還原燒製。霧灰釉面帶有細膩的結晶斑點，握於掌心溫潤厚實。容量約 350ml，適合早晨的第一杯咖啡。",
				ProductImages:   img("p01.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "霧灰", SKU: "CUR-TAB-01-A", Stock: 16},
					{Name: "陶白", SKU: "CUR-TAB-01-B", Stock: 14},
					{Name: "赭紅", SKU: "CUR-TAB-01-C", PriceDelta: 30, Stock: 7},
				}},
			{SKU: "CUR-HOM-01", Name: "亞麻工作圍裙・自然米", Slug: "linen-work-apron", Category: "home-living", Status: "active", IsFeatured: true, Price: 890, Stock: 26, Tag: "天然亞麻", Material: "亞麻", Origin: "台灣",
				Description:     "未染色的天然亞麻布料，越洗越柔軟。大口袋設計，料理、植栽、手作皆宜。",
				LongDescription: "選用歐洲進口一級亞麻，平紋織造保留布面肌理。自然米色未經染色化學處理，隨使用時間產生獨特皺褶與柔軟度。胸前雙口袋可置放工具與手機。",
				ProductImages:   img("p02.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "S / 自然米", SKU: "CUR-HOM-01-S", Stock: 9},
					{Name: "M / 自然米", SKU: "CUR-HOM-01-M", Stock: 11},
					{Name: "L / 燕麥褐", SKU: "CUR-HOM-01-L", Stock: 6},
				}},
			{SKU: "CUR-TAB-02", Name: "胡桃木端盤托盤", Slug: "walnut-serving-tray", Category: "tableware", Status: "active", IsFeatured: true, Price: 1280, Stock: 15, Tag: "北美胡桃木", Material: "胡桃木", Origin: "北美",
				Description:     "一體成型的北美黑胡桃木，邊緣手工導圓，木紋如山水流轉，端上桌就是風景。",
				LongDescription: "精選 FSC 認證北美黑胡桃木實木，由木工職人以傳統榫接工法製成。表面塗裝食品級天然木蠟油。尺寸約 35 × 22 cm。",
				ProductImages:   img("p03.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "原木色（單件）", SKU: "CUR-TAB-02-A", Stock: 15},
				}},
			{SKU: "CUR-FRA-01", Name: "雪松森林大豆蠟燭", Slug: "cedar-forest-candle", Category: "fragrance", Status: "active", IsFeatured: true, Price: 520, OriginalPrice: 650, Stock: 59, Tag: "大豆蠟", Material: "大豆蠟", Origin: "台灣",
				Description:     "前調雪松、中調岩蘭草、後調琥珀。純淨大豆蠟搭配棉芯，燃燒時光像走進一場森林晨霧。",
				LongDescription: "100% 美國進口大豆蠟，不含石蠟與人工添加，搭配無鉛棉芯，燃燒乾淨不黑壁。180g 可燃燒約 35-40 小時。",
				ProductImages:   img("p04.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "雪松森林 180g", SKU: "CUR-FRA-01-180", Stock: 38},
					{Name: "雪松森林 90g（旅行款）", SKU: "CUR-FRA-01-90", PriceDelta: -180, Stock: 21},
				}},
			{SKU: "CUR-TAB-03", Name: "手工吹製玻璃水瓶組", Slug: "glass-carafe-set", Category: "tableware", Status: "active", Price: 1480, Stock: 11, Tag: "手工吹製", Material: "玻璃", Origin: "新竹",
				Description:     "師傅以口吹製成的水瓶與兩只杯，氣泡與微不規則是手作的指紋，注水入杯，光線都變得溫柔。",
				LongDescription: "由新竹玻璃工坊師傅手工吹製，每件都帶有自然氣泡與流動紋理。壺身 1000ml，杯 280ml，一組三件。耐熱玻璃材質。",
				ProductImages:   img("p05.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "一壺二杯組", SKU: "CUR-TAB-03-A", Stock: 11},
				}},
			{SKU: "CUR-TEX-01", Name: "羊毛編織圍巾・焦糖", Slug: "wool-scarf-caramel", Category: "textiles", Status: "active", IsFeatured: true, Price: 1980, Stock: 17, Tag: "純新羊毛", Material: "羊毛", Origin: "台灣",
				Description:     "以粗紡純新羊毛手工框織，焦糖般的暖色與厚實手感，是冬天裡最可靠的擁抱。",
				LongDescription: "採用 100% 純新羊毛（Merino blend），以古董織框一條條手工編織而成，鬚邊收尾。尺寸 180 × 35cm（不含鬚）。",
				ProductImages:   img("p06.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "焦糖棕", SKU: "CUR-TEX-01-A", Stock: 9},
					{Name: "燕麥米", SKU: "CUR-TEX-01-B", Stock: 5},
					{Name: "墨黑", SKU: "CUR-TEX-01-C", Stock: 0},
				}},
			{SKU: "CUR-STA-01", Name: "植鞣皮革筆記本・深棕", Slug: "leather-journal-brown", Category: "stationery", Status: "active", Price: 1180, Stock: 23, Tag: "植鞣革", Material: "植鞣牛皮", Origin: "義大利",
				Description:     "義大利植鞣牛皮隨年月養出光澤，內頁用罄可替換，一本可以用很多年的筆記本。",
				LongDescription: "選用義大利 Tuscan 植鞣革，未經塗飾，會隨使用產生獨特蜜色包漿。A5 尺寸，六孔活頁設計，內頁用畢可單獨添購替換。",
				ProductImages:   img("p07.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "深棕 / 橫線", SKU: "CUR-STA-01-A", Stock: 11},
					{Name: "深棕 / 空白", SKU: "CUR-STA-01-B", Stock: 8},
					{Name: "原色 / 橫線", SKU: "CUR-STA-01-C", Stock: 4},
				}},
			{SKU: "CUR-TAB-04", Name: "抹茶茶碗與茶筅組", Slug: "matcha-bowl-set", Category: "tableware", Status: "active", Price: 980, Stock: 20, Tag: "日本職人", Material: "陶瓷", Origin: "日本",
				Description:     "日本職人手拉胚的苔綠色茶碗，搭配百年竹細工老舖的茶筅，在家也能好好點一碗抹茶。",
				LongDescription: "茶碗為日本有田燒職人手拉胚製作，釉色苔綠帶窯變流紋；茶筅出自京都百年竹細工舖，白竹 100 本立規格。含茶碗、茶筅、竹茶杓。",
				ProductImages:   img("p08.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "標準組合", SKU: "CUR-TAB-04-A", Stock: 14},
					{Name: "加購茶罐（+ 茶筅收納）", SKU: "CUR-TAB-04-B", PriceDelta: 350, Stock: 6},
				}},
			{SKU: "CUR-STA-02", Name: "黃銅鋼筆與筆架組", Slug: "brass-fountain-pen", Category: "stationery", Status: "active", Price: 1560, Stock: 10, Tag: "實心黃銅", Material: "黃銅", Origin: "台灣",
				Description:     "實心黃銅車製的筆身，德國 SCHMIDT 鍍銠筆尖。書寫的手感與重量，都是剛剛好的專注。",
				LongDescription: "筆身以實心黃銅 CNC 車製，重量 34g，重心前移適合長時間書寫。德國 SCHMIDT 鍍銠不鏽鋼 F 尖，隨附黃銅筆架與墨水管。",
				ProductImages:   img("p09.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "黃銅原色", SKU: "CUR-STA-02-A", Stock: 7},
					{Name: "墨黑鍍層", SKU: "CUR-STA-02-B", Stock: 3},
				}},
			{SKU: "CUR-HOM-02", Name: "有機棉麻抱枕套", Slug: "cotton-linen-cushion", Category: "home-living", Status: "active", Price: 590, OriginalPrice: 720, Stock: 32, Tag: "GOTS 有機棉", Material: "棉麻", Origin: "台灣",
				Description:     "GOTS 認證有機棉與亞麻混紡，燕麥色織紋低調溫柔，讓沙發立刻有家的樣子。",
				LongDescription: "70% GOTS 有機棉、30% 亞麻混紡，雙面同布織造，隱形拉鏈設計。尺寸 45 × 45cm（不含枕心）。",
				ProductImages:   img("p10.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "燕麥色", SKU: "CUR-HOM-02-A", Stock: 18},
					{Name: "陶灰色", SKU: "CUR-HOM-02-B", Stock: 14},
				}},
			{SKU: "CUR-TAB-05", Name: "手沖陶瓷濾杯組", Slug: "pour-over-dripper-set", Category: "tableware", Status: "active", Price: 1080, Stock: 15, Tag: "經典 V60 型", Material: "陶瓷", Origin: "台灣",
				Description:     "暖褐釉色的陶瓷濾杯與耐熱玻璃下壺，加上胡桃木托環，晨間手沖的完整儀式。",
				LongDescription: "濾杯為 V60 型螺旋肋槽陶瓷（1-2 人份，搭配 01 濾紙），釉色暖褐手浸釉。下壺為耐熱玻璃 400ml，附胡桃木隔熱托環。",
				ProductImages:   img("p11.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "濾杯 + 玻璃壺 + 木環", SKU: "CUR-TAB-05-A", Stock: 15},
				}},
			{SKU: "CUR-FRA-02", Name: "白瓷擴香瓶・木質調", Slug: "porcelain-diffuser-wood", Category: "fragrance", Status: "active", Price: 860, Stock: 26, Tag: "天然藤枝", Material: "白瓷", Origin: "台灣",
				Description:     "白瓷瓶身配 8 支天然藤枝，無火安全的持續香氣。木質調基底，安靜地替空間上色。",
				LongDescription: "霧面白瓷瓶身（100ml），含 8 支義大利天然藤枝與精油補充瓶。香調：雪松 + 檀香 + 一絲柑橘。持香約 2-3 個月。",
				ProductImages:   img("p12.png"),
				Variants: []commerce.ProductVariantInput{
					{Name: "木質調 100ml", SKU: "CUR-FRA-02-A", Stock: 14},
					{Name: "柑苔調 100ml", SKU: "CUR-FRA-02-B", Stock: 12},
				}},
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

// curatorySeedSoldCount mirrors the reference storefront's sold counts.
// sold_count is derived by the order state machine and not accepted as
// input, so the seed back-fills it with SQL after product creation.
var curatorySeedSoldCount = map[string]int{
	"CUR-TAB-01": 133,
	"CUR-HOM-01": 76,
	"CUR-TAB-02": 54,
	"CUR-FRA-01": 211,
	"CUR-TAB-03": 44,
	"CUR-TEX-01": 68,
	"CUR-STA-01": 90,
	"CUR-TAB-04": 45,
	"CUR-STA-02": 31,
	"CUR-HOM-02": 120,
	"CUR-TAB-05": 53,
	"CUR-FRA-02": 98,
}

// devMediaVerifier mirrors bootstrap's mediaVerifierAdapter: it checks
// the media registry so seed products associate images through the same
// verified-key boundary the API enforces (product images never take raw
// URLs).
type devMediaVerifier struct {
	registry media.RegistryStore
}

func (v devMediaVerifier) VerifyKey(ctx context.Context, userID, objectKey string) error {
	obj, err := v.registry.GetByObjectKey(ctx, objectKey)
	if err != nil {
		if errors.Is(err, media.ErrObjectNotFound) {
			return commerce.ErrUnverifiedMedia
		}
		return fmt.Errorf("media verifier: registry lookup: %w", err)
	}
	if obj.UploadedByUserID != userID {
		return commerce.ErrUnverifiedMedia
	}
	return nil
}

// seedCuratoryMedia registers the bundled product photos as verified media
// objects owned by the dev principal. Files live in site/assets/images and
// are copied to dist/assets/images; product image keys are
// "images/<file>" so URLs resolve as {R2_PUBLIC_BASE_URL}/images/<file>.
// Idempotent: skips when media_objects already has rows.
func seedCuratoryMedia(ctx context.Context, db *sql.DB) {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_assets`).Scan(&count); err != nil {
		log.Printf("seed media: count failed: %v", err)
		return
	}
	if count > 0 {
		return
	}
	entries, err := os.ReadDir(filepath.Join("site", "assets", "images"))
	if err != nil {
		log.Printf("seed media: %v (run from repo root)", err)
		return
	}
	seeded := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".png") {
			continue
		}
		data, err := os.ReadFile(filepath.Join("site", "assets", "images", name))
		if err != nil {
			continue
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			log.Printf("seed media %s: decode: %v", name, err)
			continue
		}
		now := time.Now().Unix()
		id := "seed-" + strings.TrimSuffix(name, ".png")
		// media_assets 是 active 狀態的資產表；media_objects 是 source 列。
		// GetByObjectKey 需要兩表 JOIN 同時命中才算已驗證。
		// unassociated_since_unix=0：已與商品圖關聯，不進 GC 佇列。
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			continue
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO media_assets
			 (object_key, state, content_type, bytes, width, height, uploaded_by_user_id, verified_unix, reservation_token, reserved_unix, unassociated_since_unix)
			 VALUES (?, 'active', ?, ?, ?, ?, ?, ?, '', 0, 0)`,
			"images/"+name, "image/"+format, len(data), cfg.Width, cfg.Height, devPrincipal.UserID, now)
		if err == nil {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO media_objects
				 (id, object_key, source_upload_key, content_type, bytes, width, height, uploaded_by_user_id, verified_unix)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				id, "images/"+name, "seed-upload/"+name, "image/"+format, len(data), cfg.Width, cfg.Height, devPrincipal.UserID, now)
		}
		if err != nil {
			_ = tx.Rollback()
			log.Printf("seed media %s: %v", name, err)
			continue
		}
		if err := tx.Commit(); err != nil {
			log.Printf("seed media %s: commit: %v", name, err)
			continue
		}
		seeded++
	}
	if seeded > 0 {
		log.Printf("seed: %d media objects", seeded)
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
			{Slug: "tableware", Name: "餐廚食器", Description: "陶、瓷、木與玻璃——讓餐桌成為風景的器物。", Image: "/assets/images/cat-table.png", SortOrder: 1, IsActive: true},
			{Slug: "fragrance", Name: "香氛療癒", Description: "蠟燭、擴香與空間氣味。", Image: "/assets/images/cat-fragrance.png", SortOrder: 2, IsActive: true},
			{Slug: "stationery", Name: "文具書房", Description: "筆、紙與桌上的好工作夥伴。", Image: "/assets/images/cat-stationery.png", SortOrder: 3, IsActive: true},
			{Slug: "textiles", Name: "服飾織品", Description: "羊毛、亞麻與職人織作。", Image: "/assets/images/cat-textile.png", SortOrder: 4, IsActive: true},
			{Slug: "home-living", Name: "家居生活", Description: "讓家更像我們的樣子。", Image: "/assets/images/cat-home.png", SortOrder: 5, IsActive: true},
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
			"tagline": "為日常，嚴選美好",
			"promoBanner": "開幕慶限定・全站滿 NT$1,500 即享免運",
			"promoBannerEnabled": true,
			"freeShippingThreshold": 1500,
			"lowStockThreshold": 5,
			"notificationMaster": true,
			"contactEmail": "service@curatory.tw",
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
