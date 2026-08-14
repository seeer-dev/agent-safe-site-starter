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
		products := []commerce.ProductInput{
			{SKU: "SKU-APP-01", Name: "素面亞麻襯衫", Slug: "linen-shirt", Description: "透氣亞麻襯衫，適合台灣夏季。", Category: "apparel", Status: "active", Material: "亞麻", Origin: "台灣", Price: 1680, OriginalPrice: 1980, Stock: 24, Tag: "經典"},
			{SKU: "SKU-HOME-02", Name: "陶製小碟", Slug: "ceramic-dish", Description: "手作陶碟，每一件獨一無二。", Category: "home", Status: "active", Material: "陶", Origin: "鶯歌", Price: 420, Stock: 2, Tag: ""},
			{SKU: "SKU-ACC-03", Name: "植鞣皮名片夾", Slug: "leather-card-holder", Description: "植鞣牛皮名片夾，使用越久越有味道。", Category: "accessories", Status: "active", Material: "牛皮", Origin: "台南", Price: 980, Stock: 0, Tag: "手工"},
			{SKU: "SKU-STA-04", Name: "線裝筆記本", Slug: "thread-bound-notebook", Description: "傳統線裝筆記本，書寫流暢。", Category: "stationery", Status: "active", Material: "紙", Origin: "台灣", Price: 260, Stock: 88, Tag: ""},
			{SKU: "SKU-APP-05", Name: "寬版工作褲", Slug: "wide-work-pants", Description: "寬版剪裁工作褲，舒適耐穿。", Category: "apparel", Status: "draft", Material: "棉", Origin: "台灣", Price: 2280, Stock: 11, Tag: "新品"},
			{SKU: "SKU-HOME-06", Name: "手抄紙燈罩", Slug: "paper-lampshade", Description: "埔里手抄紙燈罩，溫暖柔光。", Category: "home", Status: "active", Material: "紙", Origin: "埔里", Price: 1540, Stock: 5, Tag: ""},
		}
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
		paymentMethods := []commerce.PaymentMethod{
			{Method: "atm", ProviderLabel: "銀行ATM轉帳", Environment: "production", ReadinessStatus: "ready", Enabled: true},
			{Method: "cod", ProviderLabel: "貨到付款", Environment: "production", ReadinessStatus: "ready", Enabled: true},
			{Method: "credit_card", ProviderLabel: "信用卡（尚未啟用）", Environment: "sandbox", ReadinessStatus: "pending_setup", Enabled: false},
		}
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
			{Key: "home.hero", Placement: "hero", Title: "質物選物", Body: "為生活選一件好物。", Status: "published", SortOrder: 0},
			{Key: "home.popup", Placement: "popup", Title: "歡迎光臨", Body: "感謝您的光臨。", Status: "published", SortOrder: 0},
			{Key: "footer.about", Placement: "footer", Title: "關於質物", Body: "質物選物的品牌故事將於正式上線前公告。", Status: "published", SortOrder: 0},
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

	// ----- Render -----------------------------------------------------------
	// Compose the full minimal-cart public input (articles, products,
	// categories, site content) using the same shared composition as
	// server/tools/render. This ensures dev renders the integrated
	// product/category/content site, not legacy article-only output.
	input, err := rendercompose.Compose(ctx, db, dialect, cfg.R2PublicBaseURL)
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
	if err := renderer.RenderAllFull(input.Articles, input.Products, input.Categories, input.CategoryLabels, input.ProductsByCategory, input.ContentBlocks); err != nil {
		log.Fatalf("render failed (dist preserved): %v", err)
	}
	log.Printf("rendered %d article(s), %d product(s), %d categor(y/ies), %d content page(s) into dist/",
		len(input.Articles), len(input.Products), len(input.Categories), len(input.ContentBlocks))

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
	if siteTheme == "minimal-cart" {
		return "質物選物"
	}
	return "AI Site Starter"
}

func shouldSeedSampleArticle(siteTheme string) bool {
	return siteTheme != "minimal-cart"
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
