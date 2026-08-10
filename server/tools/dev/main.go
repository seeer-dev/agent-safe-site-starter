package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/bootstrap"
	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	siterender "github.com/example/ai-site-starter/server/internal/render"
)

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
	store := content.NewSQLStore(db, dialect)
	count, err := store.Count(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 {
		service := content.NewService(store)
		_, err := service.Publish(ctx, auth.Principal{UserID: "dev", Role: "admin"}, content.UpsertInput{
			Slug:      "hello-starter",
			Title:     "Hello from the starter",
			Excerpt:   "SQLite → Go renderer → static HTML. No frontend framework required.",
			BodyHTML:  "<p>This page was rendered from the local SQLite database.</p>",
			Published: true,
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	articles, err := store.ListPublished(ctx)
	if err != nil {
		log.Fatal(err)
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

	app, err := bootstrap.NewWithDB(ctx, cfg, db, dialect)
	if err != nil {
		log.Fatal(err)
	}
	apiServer := &http.Server{Addr: cfg.HTTPAddr, Handler: app.Handler, ReadHeaderTimeout: 5 * time.Second}
	siteServer := &http.Server{Addr: cfg.SiteAddr, Handler: http.FileServer(http.Dir("dist")), ReadHeaderTimeout: 5 * time.Second}

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
