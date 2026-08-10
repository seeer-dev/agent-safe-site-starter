package main

import (
	"context"
	"log"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/platform/database"
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
	count, err := store.Count(ctx)
	if err != nil {
		log.Fatal("run migrations first: ", err)
	}
	if count > 0 {
		log.Printf("seed skipped: articles already exist")
		return
	}
	service := content.NewService(store)
	_, err = service.Publish(ctx, auth.Principal{UserID: "seed", Role: "admin"}, content.UpsertInput{
		Slug:      "hello-starter",
		Title:     "Hello from the starter",
		Excerpt:   "This article came from SQLite and was rendered into static HTML.",
		BodyHTML:  "<p>Edit content through the API or your future CMS, then render and publish again.</p>",
		Published: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("seeded sample article")
}
