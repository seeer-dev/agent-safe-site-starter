package main

import (
	"context"
	"log"

	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	siterender "github.com/example/ai-site-starter/server/internal/render"
	"github.com/example/ai-site-starter/server/tools/internal/rendercompose"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	db, dialect, err := database.Open(ctx, cfg.DBDriver, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Default to the minimal-cart theme if no theme is configured.
	siteTheme := cfg.SiteTheme
	if siteTheme == "" {
		siteTheme = "minimal-cart"
	}

	renderer := siterender.New(rendererConfig(cfg, siteTheme))

	// Compose the full minimal-cart public render input (articles,
	// published products, categories, site content) via the shared
	// composition package used by both render and dev tools.
	input, err := rendercompose.Compose(ctx, db, dialect, cfg.R2PublicBaseURL)
	if err != nil {
		log.Fatalf("compose render input: %v", err)
	}

	// Render everything into a staging directory, then atomically promote.
	// On any failure, the existing dist is preserved (last-known-good) and
	// the tool exits with a non-zero status code.
	if err := renderer.RenderAllFull(input.Articles, input.Products, input.Categories, input.CategoryLabels, input.ProductsByCategory, input.ContentBlocks); err != nil {
		log.Fatalf("render failed (dist preserved): %v", err)
	}
	log.Printf("rendered %d article(s), %d product(s), %d categor(y/ies), %d content page(s) into dist/",
		len(input.Articles), len(input.Products), len(input.Categories), len(input.ContentBlocks))
}

// rendererConfig builds a siterender.Config from the app config and the
// resolved site theme. This is a separate helper so the config-to-renderer
// wiring can be tested without a database. R2PublicBaseURL is passed
// through so the renderer can include the R2 CDN origin in the CSP
// img-src directive of the generated _headers file.
func siteNameForTheme(siteTheme string) string {
	if siteTheme == "minimal-cart" {
		return "質物選物"
	}
	return "AI Site Starter"
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
