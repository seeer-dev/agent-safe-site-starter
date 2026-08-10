package main

import (
	"context"
	"log"

	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	siterender "github.com/example/ai-site-starter/server/internal/render"
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
	store := content.NewSQLStore(db, dialect)
	articles, err := store.ListPublished(ctx)
	if err != nil {
		log.Fatal("run migrations first: ", err)
	}
	renderer := siterender.New(siterender.Config{
		SiteName:      "AI Site Starter",
		PublicSiteURL: cfg.PublicSiteURL,
		PublicAPIBase: cfg.PublicAPIBase,
		TemplateDir:   "site/templates",
		AssetDir:      "site/assets",
		OutputDir:     "dist",
	})
	if err := renderer.RenderAll(articles); err != nil {
		log.Fatal(err)
	}
	log.Printf("rendered %d article(s) into dist/", len(articles))
}
