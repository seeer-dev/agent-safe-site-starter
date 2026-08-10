package bootstrap

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/modules/contact"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/modules/media"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	mailplatform "github.com/example/ai-site-starter/server/internal/platform/mail"
	"github.com/example/ai-site-starter/server/internal/platform/storage"
)

type App struct {
	Handler http.Handler
	DB      *sql.DB
	Dialect database.Dialect
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, dialect, err := database.Open(ctx, cfg.DBDriver, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	app, err := NewWithDB(ctx, cfg, db, dialect)
	if err != nil {
		db.Close()
		return nil, err
	}
	return app, nil
}

func NewWithDB(ctx context.Context, cfg config.Config, db *sql.DB, dialect database.Dialect) (*App, error) {
	var verifier auth.Verifier
	switch cfg.AuthMode {
	case "supabase":
		verifier = auth.NewSupabaseVerifier(cfg.SupabaseURL, cfg.SupabasePublishableKey)
	default:
		verifier = auth.NewDevVerifier(cfg.DevAuthToken)
	}
	authenticator := auth.NewAuthenticator(verifier)

	var mailer mailplatform.Sender = mailplatform.LogSender{}
	if cfg.ResendAPIKey != "" {
		mailer = mailplatform.NewResendSender(cfg.ResendAPIKey, cfg.ResendFrom)
	}

	var objectStore storage.Store = storage.Disabled{}
	if cfg.R2Enabled() {
		r2, err := storage.NewR2(ctx, cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket)
		if err != nil {
			return nil, err
		}
		objectStore = r2
	}

	contentStore := content.NewSQLStore(db, dialect)
	contentHandler := content.NewHandler(content.NewService(contentStore), authenticator)
	contactHandler := contact.NewHandler(contact.NewService(contact.NewStore(db, dialect), mailer, cfg.ContactNotifyTo))
	mediaHandler := media.NewHandler(media.NewService(objectStore), authenticator)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /api/articles", contentHandler.ListPublished)
	mux.HandleFunc("POST /api/admin/articles", contentHandler.Publish)
	mux.HandleFunc("POST /api/contact", contactHandler.Submit)
	mux.HandleFunc("POST /api/media/presign", mediaHandler.Presign)

	return &App{Handler: withAccessLog(withCORS(cfg.SiteOrigin, mux)), DB: db, Dialect: dialect}, nil
}

func withCORS(origin string, next http.Handler) http.Handler {
	allowed := strings.TrimRight(origin, "/")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestOrigin := strings.TrimRight(r.Header.Get("Origin"), "/")
		if allowed != "" && requestOrigin == allowed {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("http %s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
