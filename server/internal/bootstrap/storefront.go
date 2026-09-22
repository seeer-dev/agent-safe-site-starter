package bootstrap

import (
	"net/http"

	"github.com/example/ai-site-starter/server/internal/httpx"
	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/modules/sitecontent"
)

// storefrontHandler implements GET /api/storefront/bootstrap — the single
// aggregation call the storefront makes on first load. It composes the
// commerce, content, and sitecontent services here in the bootstrap layer
// so no business module imports another module.
type storefrontHandler struct {
	commerce         commerce.Service
	content          content.Service
	sitecontent      sitecontent.Service
	turnstileSiteKey string
}

func newStorefrontHandler(commerceService commerce.Service, contentService content.Service, siteContentService sitecontent.Service, turnstileSiteKey string) storefrontHandler {
	return storefrontHandler{commerce: commerceService, content: contentService, sitecontent: siteContentService, turnstileSiteKey: turnstileSiteKey}
}

// Get returns everything the storefront shell needs in one round trip:
// published store settings, active categories, published site-content
// blocks, and the customer-usable payment/shipping methods. Every piece
// is public or published-only data — drafts never leave this endpoint.
func (h storefrontHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	settings, err := h.sitecontent.GetPublishedStoreSettings(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load store settings")
		return
	}
	categories, err := h.commerce.ListActiveCategories(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load categories")
		return
	}
	blocks, err := h.sitecontent.ListPublished(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load site content")
		return
	}
	paymentMethods, err := h.commerce.ListPublicPaymentMethods(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load payment methods")
		return
	}
	shippingMethods, err := h.commerce.ListPublicShippingMethods(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load shipping methods")
		return
	}
	announcements, err := h.content.ListPublished(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load announcements")
		return
	}

	if categories == nil {
		categories = []commerce.Category{}
	}
	if blocks == nil {
		blocks = []sitecontent.SiteContent{}
	}
	if paymentMethods == nil {
		paymentMethods = []commerce.PublicPaymentMethod{}
	}
	if shippingMethods == nil {
		shippingMethods = []commerce.PublicShippingMethod{}
	}
	if announcements == nil {
		announcements = []content.Article{}
	}
	if settings == nil {
		settings = []byte("{}")
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"settings":           settings,
		"categories":         categories,
		"site_content":       blocks,
		"payment_methods":    paymentMethods,
		"shipping_methods":   shippingMethods,
		"announcements":      announcements,
		"turnstile_site_key": h.turnstileSiteKey,
	})
}
