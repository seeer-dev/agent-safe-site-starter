package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/httpx"
	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/modules/contact"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/modules/media"
	"github.com/example/ai-site-starter/server/internal/modules/sitecontent"
	"github.com/example/ai-site-starter/server/internal/modules/staff"
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

	// Staff store is needed for capability resolution in supabase mode.
	staffStore := staff.NewSQLStore(db, dialect)

	var authenticator auth.Authenticator
	if cfg.AuthMode == "supabase" {
		resolver := auth.NewStaffCapabilityResolver(newStaffLookupAdapter(staffStore))
		authenticator = auth.NewAuthenticatorWithResolver(verifier, resolver)
	} else {
		authenticator = auth.NewAuthenticator(verifier)
	}

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

	// B4: media verification registry store is shared between the media
	// verify service and the commerce MediaVerifier adapter. The adapter
	// is defined here in bootstrap to preserve module isolation: commerce
	// never imports media.
	mediaRegistry := media.NewSQLRegistryStore(db, dialect)
	verifyHandler := media.NewVerifyHandler(media.NewVerifyService(objectStore, mediaRegistry), authenticator)

	commerceStore := commerce.NewSQLStore(db, dialect)
	commerceService := commerce.NewService(commerceStore).
		WithMediaVerifier(mediaVerifierAdapter{registry: mediaRegistry}).
		WithPublicBaseURL(cfg.R2PublicBaseURL)
	commerceHandler := commerce.NewHandler(commerceService, authenticator)

	siteContentStore := sitecontent.NewSQLStore(db, dialect)
	siteContentHandler := sitecontent.NewHandler(sitecontent.NewService(siteContentStore), authenticator)

	staffHandler := staff.NewHandler(staff.NewService(staffStore), authenticator)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	// Public content endpoints
	mux.HandleFunc("GET /api/articles", contentHandler.ListPublished)
	mux.HandleFunc("POST /api/contact", contactHandler.Submit)
	mux.HandleFunc("POST /api/media/presign", mediaHandler.Presign)
	mux.HandleFunc("POST /api/media/verify", verifyHandler.Verify)

	// Admin session endpoint: returns the validated principal and server-
	// derived capabilities. The admin UI must use this to confirm the
	// session is valid and to obtain the canonical capability list.
	mux.HandleFunc("GET /api/admin/me", func(w http.ResponseWriter, r *http.Request) {
		principal, err := authenticator.Principal(r)
		if err != nil {
			auth.WriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"user_id":      principal.UserID,
			"staff_id":     principal.StaffID,
			"email":        principal.Email,
			"role":         principal.Role,
			"capabilities": principal.Capabilities,
		})
	})

	// Admin content endpoints
	mux.HandleFunc("POST /api/admin/articles", contentHandler.Publish)

	// Public commerce endpoints (no auth)
	mux.HandleFunc("GET /api/products", commerceHandler.ListPublishedProducts)
	mux.HandleFunc("GET /api/products/{slug}", commerceHandler.GetProductBySlug)
	mux.HandleFunc("GET /api/shipping-methods", commerceHandler.ListPublicShippingMethods)
	mux.HandleFunc("GET /api/payment-methods", commerceHandler.ListPublicPaymentMethods)
	mux.HandleFunc("POST /api/quote", commerceHandler.Quote)
	mux.HandleFunc("POST /api/orders", commerceHandler.CreateOrder)
	mux.HandleFunc("POST /api/orders/mine", commerceHandler.CreateOrderForMember)
	mux.HandleFunc("GET /api/orders/{id}", commerceHandler.GetOrderForGuest)
	mux.HandleFunc("GET /api/orders/mine", commerceHandler.ListMyOrders)
	mux.HandleFunc("GET /api/orders/mine/{id}", commerceHandler.GetMyOrder)

	// Public site content endpoint (no auth)
	mux.HandleFunc("GET /api/site-content/published", siteContentHandler.ListPublished)

	// Admin commerce endpoints — products
	mux.HandleFunc("GET /api/admin/products", commerceHandler.ListProducts)
	mux.HandleFunc("GET /api/admin/products/{id}", commerceHandler.GetProduct)
	mux.HandleFunc("POST /api/admin/products", commerceHandler.CreateProduct)
	mux.HandleFunc("PUT /api/admin/products/{id}", commerceHandler.UpdateProduct)
	mux.HandleFunc("DELETE /api/admin/products/{id}", commerceHandler.DeleteProduct)
	mux.HandleFunc("PATCH /api/admin/products/{id}/status", commerceHandler.UpdateProductStatus)
	mux.HandleFunc("POST /api/admin/products/bulk", commerceHandler.BulkUpdateProducts)

	// Admin commerce endpoints — orders
	mux.HandleFunc("GET /api/admin/orders", commerceHandler.ListOrders)
	mux.HandleFunc("GET /api/admin/orders/{id}", commerceHandler.GetOrder)
	mux.HandleFunc("PATCH /api/admin/orders/{id}/status", commerceHandler.UpdateOrderStatus)
	mux.HandleFunc("PATCH /api/admin/orders/{id}/return", commerceHandler.UpdateOrderReturnStatus)
	mux.HandleFunc("POST /api/admin/orders/{id}/restock", commerceHandler.RestockOrderItems)

	// Admin commerce endpoints — members
	mux.HandleFunc("GET /api/admin/members", commerceHandler.ListMembers)
	mux.HandleFunc("PUT /api/admin/members/{id}", commerceHandler.UpdateMember)
	mux.HandleFunc("PATCH /api/admin/members/{id}/status", commerceHandler.UpdateMemberStatus)

	// Admin commerce endpoints — promos
	mux.HandleFunc("GET /api/admin/promos", commerceHandler.ListPromos)
	mux.HandleFunc("POST /api/admin/promos", commerceHandler.CreatePromo)
	mux.HandleFunc("PUT /api/admin/promos/{id}", commerceHandler.UpdatePromo)
	mux.HandleFunc("DELETE /api/admin/promos/{id}", commerceHandler.DeletePromo)

	// Admin commerce endpoints — payment methods
	mux.HandleFunc("GET /api/admin/payment-methods", commerceHandler.ListPaymentMethods)
	mux.HandleFunc("PUT /api/admin/payment-methods/{id}", commerceHandler.UpdatePaymentMethod)

	// Admin commerce endpoints — shipping methods (list/create/update; no delete)
	mux.HandleFunc("GET /api/admin/shipping-methods", commerceHandler.ListShippingMethods)
	mux.HandleFunc("POST /api/admin/shipping-methods", commerceHandler.CreateShippingMethod)
	mux.HandleFunc("PUT /api/admin/shipping-methods/{id}", commerceHandler.UpdateShippingMethod)

	// Admin site content endpoints. Create/update only save draft. Approve
	// records an approval for the current draft version (content.approve).
	// Publish requires content.publish AND a current non-stale non-expired
	// approval; the public renderer keeps serving the last published snapshot.
	mux.HandleFunc("GET /api/admin/site-content", siteContentHandler.ListAll)
	mux.HandleFunc("POST /api/admin/site-content", siteContentHandler.Create)
	mux.HandleFunc("PUT /api/admin/site-content/{id}", siteContentHandler.Update)
	mux.HandleFunc("POST /api/admin/site-content/{id}/approve", siteContentHandler.Approve)
	mux.HandleFunc("POST /api/admin/site-content/{id}/publish", siteContentHandler.Publish)
	mux.HandleFunc("DELETE /api/admin/site-content/{id}", siteContentHandler.Delete)

	// Admin staff endpoints
	mux.HandleFunc("GET /api/admin/staff", staffHandler.List)
	mux.HandleFunc("POST /api/admin/staff", staffHandler.Create)
	mux.HandleFunc("PUT /api/admin/staff/{id}", staffHandler.Update)
	mux.HandleFunc("PATCH /api/admin/staff/{id}/status", staffHandler.UpdateStatus)
	mux.HandleFunc("DELETE /api/admin/staff/{id}", staffHandler.Delete)

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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
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

// mediaVerifierAdapter implements commerce.MediaVerifier by wrapping
// the media.RegistryStore. It is defined here in bootstrap to preserve
// module isolation: commerce defines the interface (consumer), media
// defines the registry (provider), and bootstrap wires them together.
type mediaVerifierAdapter struct {
	registry media.RegistryStore
}

// VerifyKey validates that the given object key is a verified media
// object owned by the given userID. Returns:
//   - nil if the key is a verified media object owned by userID.
//   - commerce.ErrUnverifiedMedia if the key is not in the registry
//     (media.ErrObjectNotFound) or is owned by a different user.
//   - A wrapped error for any other registry/DB failure. The caller
//     must NOT collapse this into ErrUnverifiedMedia — a DB outage
//     is a 5xx, not a 400. The commerce service checks errors.Is
//     (err, ErrUnverifiedMedia) to distinguish 400 from 5xx.
func (a mediaVerifierAdapter) VerifyKey(ctx context.Context, userID, objectKey string) error {
	obj, err := a.registry.GetByObjectKey(ctx, objectKey)
	if err != nil {
		// Only "not found" is a verification failure (400). All other
		// errors (DB connection lost, query error, etc.) are provider
		// failures that must propagate as 5xx.
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
