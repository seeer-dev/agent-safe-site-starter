package rendercompose

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/modules/sitecontent"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	siterender "github.com/example/ai-site-starter/server/internal/render"
)

var testDBCounter uint64

// newTestDB opens an in-memory SQLite database, applies all migrations,
// and returns the db handle + dialect. The caller must close the db
// (via t.Cleanup). repoRoot is the absolute path to the repository root
// (used for migration file resolution).
func newTestDB(t *testing.T, repoRoot string) (*sql.DB, database.Dialect) {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:rendercompose_test_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := migrate.Apply(ctx, db, dialect, repoRoot); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db, dialect
}

// defaultRepoRoot resolves the repository root relative to this test
// file's location (server/tools/internal/rendercompose). Used by tests
// that do not change CWD.
func defaultRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

var seedPrincipal = auth.Principal{
	UserID: "test",
	Role:   "admin",
	Capabilities: []string{
		"twcommerce.create",
		"twcommerce.admin",
		"content.create",
		"content.approve",
		"content.publish",
	},
}

// seedTestDB inserts a published product, a published site content block,
// and a published article into the database so Compose has data to fetch.
func seedTestDB(t *testing.T, db *sql.DB, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()

	// Article
	articleStore := content.NewSQLStore(db, dialect)
	articleSvc := content.NewService(articleStore)
	_, err := articleSvc.Publish(ctx, seedPrincipal, content.UpsertInput{
		Slug:      "test-article",
		Title:     "Test Article",
		Excerpt:   "Test excerpt",
		BodyHTML:  "<p>Test body</p>",
		Published: true,
	})
	if err != nil {
		t.Fatalf("seed article: %v", err)
	}

	// Product
	commerceStore := commerce.NewSQLStore(db, dialect)
	commerceSvc := commerce.NewService(commerceStore)
	_, err = commerceSvc.CreateProduct(ctx, seedPrincipal, commerce.ProductInput{
		SKU:         "SKU-TEST-01",
		Name:        "Test Product",
		Slug:        "test-product",
		Description: "A test product",
		Category:    "apparel",
		Status:      "active",
		Price:       1000,
		Stock:       10,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}

	// Site content (footer block for standalone page)
	scStore := sitecontent.NewSQLStore(db, dialect)
	scSvc := sitecontent.NewService(scStore)
	created, err := scSvc.Create(ctx, seedPrincipal, sitecontent.SiteContentInput{
		Key:       "footer.about",
		Placement: "footer",
		Title:     "About",
		Body:      "About this store",
		Status:    "published",
		SortOrder: 0,
	})
	if err != nil {
		t.Fatalf("seed site content: %v", err)
	}
	_, err = scSvc.Approve(ctx, seedPrincipal, created.ID, sitecontent.ApprovalInput{
		ExpiryUnix:           time.Now().Add(24 * time.Hour).Unix(),
		ExpectedDraftVersion: created.DraftVersion,
	})
	if err != nil {
		t.Fatalf("approve site content: %v", err)
	}
	_, err = scSvc.Publish(ctx, seedPrincipal, created.ID, sitecontent.PublishInput{
		ExpectedDraftVersion: created.DraftVersion,
	})
	if err != nil {
		t.Fatalf("publish site content: %v", err)
	}
}

// TestComposeReturnsFullInput proves that Compose fetches articles,
// products, categories, and site content from the database and returns
// them in the Input struct with deterministic category ordering.
func TestComposeReturnsFullInput(t *testing.T) {
	t.Parallel()
	db, dialect := newTestDB(t, defaultRepoRoot(t))
	seedTestDB(t, db, dialect)

	ctx := context.Background()
	input, err := Compose(ctx, db, dialect, "")
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if len(input.Articles) != 1 {
		t.Errorf("Articles: got %d, want 1", len(input.Articles))
	}
	if len(input.Products) != 1 {
		t.Errorf("Products: got %d, want 1", len(input.Products))
	}
	if input.Products[0].Slug != "test-product" {
		t.Errorf("Product slug: got %q, want test-product", input.Products[0].Slug)
	}
	if len(input.Categories) != 1 {
		t.Errorf("Categories: got %d, want 1", len(input.Categories))
	}
	if input.Categories[0] != "apparel" {
		t.Errorf("Category[0]: got %q, want apparel", input.Categories[0])
	}
	if input.CategoryLabels["apparel"] != "服飾" {
		t.Errorf("CategoryLabel apparel: got %q, want 服飾", input.CategoryLabels["apparel"])
	}
	if len(input.ProductsByCategory["apparel"]) != 1 {
		t.Errorf("ProductsByCategory[apparel]: got %d, want 1", len(input.ProductsByCategory["apparel"]))
	}
	if len(input.ContentBlocks) != 1 {
		t.Errorf("ContentBlocks: got %d, want 1", len(input.ContentBlocks))
	}
	if input.ContentBlocks[0].Key != "footer.about" {
		t.Errorf("ContentBlock key: got %q, want footer.about", input.ContentBlocks[0].Key)
	}
}

// TestComposeCategoriesAreSorted proves that categories are returned in
// sorted order so identical DB input produces stable dist output.
func TestComposeCategoriesAreSorted(t *testing.T) {
	t.Parallel()
	db, dialect := newTestDB(t, defaultRepoRoot(t))
	ctx := context.Background()

	// Seed products in multiple categories (insertion order != sorted).
	commerceStore := commerce.NewSQLStore(db, dialect)
	commerceSvc := commerce.NewService(commerceStore)
	cats := []string{"home", "apparel", "stationery", "accessories"}
	for i, c := range cats {
		_, err := commerceSvc.CreateProduct(ctx, seedPrincipal, commerce.ProductInput{
			SKU:         fmt.Sprintf("SKU-SORT-%02d", i),
			Name:        "Product " + c,
			Slug:        "product-" + c,
			Description: "Test",
			Category:    c,
			Status:      "active",
			Price:       100,
			Stock:       5,
		})
		if err != nil {
			t.Fatalf("seed product %s: %v", c, err)
		}
	}

	input, err := Compose(ctx, db, dialect, "")
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	want := []string{"accessories", "apparel", "home", "stationery"}
	if len(input.Categories) != len(want) {
		t.Fatalf("Categories: got %d, want %d", len(input.Categories), len(want))
	}
	for i, c := range input.Categories {
		if c != want[i] {
			t.Errorf("Categories[%d]: got %q, want %q", i, c, want[i])
		}
	}
}

// TestComposeAndRenderProducesMinimalCartOutput is the end-to-end
// regression for the observed dev-render failure. It seeds a real
// SQLite DB, composes the full input, renders with the real minimal-cart
// theme, and asserts:
//   - dist/index.html contains minimal-cart semantic content ("質物選物")
//   - dist/index.html does NOT contain the legacy "Static Publish Starter"
//   - product/category/content route files exist
//
// This test must NOT run in parallel because the renderer resolves theme
// paths relative to CWD (filepath.Join("site", "themes", SiteTheme)).
func TestComposeAndRenderProducesMinimalCartOutput(t *testing.T) {
	// Do NOT call t.Parallel() — this test changes CWD to the repo root.

	root := defaultRepoRoot(t)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}

	db, dialect := newTestDB(t, root)
	seedTestDB(t, db, dialect)

	ctx := context.Background()
	input, err := Compose(ctx, db, dialect, "")
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	outputDir := filepath.Join(t.TempDir(), "dist")
	renderer := siterender.New(siterender.Config{
		SiteName:      "AI Site Starter",
		PublicSiteURL: "http://localhost:4173",
		PublicAPIBase: "http://localhost:8080",
		TemplateDir:   "site/templates",
		AssetDir:      "site/assets",
		OutputDir:     outputDir,
		SiteTheme:     "minimal-cart",
	})

	if err := renderer.RenderAllFull(input.Articles, input.Products, input.Categories, input.CategoryLabels, input.ProductsByCategory, input.ContentBlocks); err != nil {
		t.Fatalf("RenderAllFull: %v", err)
	}

	// 1. Home page must contain minimal-cart semantic content.
	home, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("read dist/index.html: %v", err)
	}
	homeStr := string(home)
	if !strings.Contains(homeStr, "質物選物") {
		t.Errorf("dist/index.html does not contain minimal-cart semantic content '質物選物'")
	}
	if strings.Contains(homeStr, "Static Publish Starter") {
		t.Errorf("dist/index.html contains legacy 'Static Publish Starter' -- dev/render fell back to legacy templates")
	}

	// 1b. Home page raw HTML must contain server-rendered product names
	// (no-JS baseline). The seeded product "Test Product" must be visible
	// without JavaScript.
	if !strings.Contains(homeStr, "Test Product") {
		t.Errorf("dist/index.html does not contain server-rendered product name 'Test Product' (no-JS baseline missing)")
	}
	// 1c. Home page must contain the static shop section.
	if !strings.Contains(homeStr, `id="shop-static"`) {
		t.Errorf("dist/index.html does not contain #shop-static section (no-JS baseline)")
	}
	// 1d. Home page must contain category navigation links.
	if !strings.Contains(homeStr, "/categories/apparel/") {
		t.Errorf("dist/index.html does not contain category navigation link /categories/apparel/")
	}

	// 2. Product route must exist.
	productPath := filepath.Join(outputDir, "products", "test-product", "index.html")
	if _, err := os.Stat(productPath); err != nil {
		t.Errorf("product route %s does not exist: %v", productPath, err)
	}

	// 3. Category route must exist (plural).
	categoryPath := filepath.Join(outputDir, "categories", "apparel", "index.html")
	if _, err := os.Stat(categoryPath); err != nil {
		t.Errorf("category route %s does not exist: %v", categoryPath, err)
	}

	// 4. Content route must exist (footer.about).
	contentPath := filepath.Join(outputDir, "content", "footer.about", "index.html")
	if _, err := os.Stat(contentPath); err != nil {
		t.Errorf("content route %s does not exist: %v", contentPath, err)
	}

	// 5. _headers must exist with CSP (B9).
	headersPath := filepath.Join(outputDir, "_headers")
	if _, err := os.Stat(headersPath); err != nil {
		t.Errorf("_headers file %s does not exist: %v", headersPath, err)
	}
}
