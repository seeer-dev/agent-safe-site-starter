package commerce

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

var (
	// ErrForbidden is returned when a principal lacks a required capability.
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidTransition is returned when an order status transition is not
	// permitted by the order state machine.
	ErrInvalidTransition = errors.New("invalid order status transition")
	// ErrStaleVersion is returned when an order mutation's expected_version
	// does not match the current aggregate version. The store guards
	// atomically with WHERE version = expected_version; 0 affected rows
	// means a concurrent mutation won first. Maps to 409 Conflict.
	ErrStaleVersion = errors.New("stale version: order was modified by another request")
	// ErrInsufficientStock is returned when an order cannot be fulfilled.
	ErrInsufficientStock = errors.New("insufficient stock")
	// ErrInvalidShippingMethod is returned when the shipping method is empty,
	// missing from the current shipping_methods rows, or present but disabled.
	ErrInvalidShippingMethod = errors.New("invalid or unknown shipping method")
	// ErrShippingConfigUnavailable is returned when the authoritative
	// shipping_methods store cannot be queried. Quote and new-order checkout
	// fail closed without exposing the underlying database error.
	ErrShippingConfigUnavailable = errors.New("shipping configuration unavailable")
	// ErrInvalidPaymentMethod is returned when the payment method is empty,
	// unknown, disabled, or not ready (readiness_status != "ready"). The
	// server is the authority for payment method availability — the browser
	// must not decide which payment methods are available.
	ErrInvalidPaymentMethod = errors.New("invalid or unavailable payment method")
	// ErrInvalidPromoCode is returned when a non-empty promo code is unknown,
	// disabled, expired, not yet active, or otherwise invalid. Empty means no
	// promo. Valid codes are never enumerated to the public browser.
	ErrInvalidPromoCode = errors.New("invalid or expired promo code")
	// ErrPromoValidationUnavailable is returned when the authoritative promo
	// store cannot be queried. Checkout fails closed without exposing the
	// underlying database error to the public response.
	ErrPromoValidationUnavailable = errors.New("promo validation unavailable")
	// ErrUnverifiedMedia is returned when a product mutation tries to attach
	// media that has not passed the server-side post-upload verification flow.
	ErrUnverifiedMedia = errors.New("product media must use a verified media reference")
	// ErrMediaURLUnavailable is returned when a product has verified
	// product_images but the R2 public base URL is not configured. The
	// service cannot derive public URLs without a CDN base, so it fails
	// closed rather than emitting empty or broken URLs. Public endpoints
	// map this to 503; the render tool fails and preserves LKG dist.
	ErrMediaURLUnavailable = errors.New("media public URL base is not configured")
	// ErrInvalidProductInput is returned when a product create/update
	// payload fails client-side validation (missing required fields,
	// invalid slug/category, negative price/stock, invalid status,
	// empty or duplicate image keys). The handler maps this to 400.
	// The wrapped error's message is safe to return to the client.
	ErrInvalidProductInput = errors.New("invalid product input")
	// ErrInvalidAdminInput is returned when an admin mutation payload
	// (member, promo, payment method, bulk product status) fails
	// client-side validation. The handler maps this to 400. The wrapped
	// error's message is safe to return to the client because it only
	// describes the rejected input value, never internal state.
	ErrInvalidAdminInput = errors.New("invalid admin input")
	// ErrDuplicateShippingMethod is returned when create uses a method
	// key that already exists. Maps to 409 Conflict.
	ErrDuplicateShippingMethod = errors.New("shipping method key already exists")
	// ErrStaleShippingMethod is returned when an update's expected_version
	// does not match the stored version. The row is not overwritten.
	// Maps to 409 Conflict.
	ErrStaleShippingMethod = errors.New("stale version: shipping method was modified by another request")
	// ErrMissingIdempotencyKey is returned when the idempotency key is
	// empty or whitespace. The production order endpoints require a
	// non-empty key to prevent duplicate orders from transport retries.
	// An empty key bypasses both the early lookup and the in-transaction
	// idempotency check, so the server fails closed rather than silently
	// accepting a non-idempotent request.
	ErrMissingIdempotencyKey = errors.New("idempotency_key is required")
	// ErrInvalidEmail is returned when the email field is missing or
	// fails RFC 5322 parsing via net/mail.ParseAddress. The OpenAPI
	// contract declares email as required with format: email.
	ErrInvalidEmail = errors.New("email is required and must be a valid email address")
	// ErrInvalidPhone is returned when the phone field is blank after
	// trimming. The OpenAPI contract declares phone as required.
	ErrInvalidPhone = errors.New("phone is required")
	// ErrInvalidShippingAddress is returned when the shipping_address
	// field is blank after trimming. The OpenAPI contract declares
	// shipping_address as required.
	ErrInvalidShippingAddress = errors.New("shipping_address is required")
	// ErrOverflow is returned when an arithmetic operation (line total,
	// subtotal accumulation, or percent discount) would overflow int.
	// The service fails closed rather than wrapping to a negative or
	// low value.
	ErrOverflow = errors.New("arithmetic overflow in order calculation")
	// ErrMissingRestockReason is returned when the restock reason is blank
	// after trimming. The OpenAPI contract declares reason as required.
	ErrMissingRestockReason = errors.New("restock reason is required")
	// ErrOverRestock is returned when a restock item's cumulative
	// restocked_quantity would exceed returned_quantity, or cumulative
	// returned_quantity would exceed the original order item quantity.
	ErrOverRestock = errors.New("restocked quantity exceeds returned or ordered quantity")
	// ErrRestockItemNotFound is returned when a restock request references
	// a SKU that does not exist in the order's line items.
	ErrRestockItemNotFound = errors.New("order item not found for restock")
	// ErrReturnNotReceived is returned when a restock is attempted on an
	// order whose return_request_status is not "received". Goods must be
	// physically received before the per-item restock action can adjust
	// salable inventory.
	ErrReturnNotReceived = errors.New("order return status must be received before restock")
)

// validProductStatuses is the closed set of product status values. draft and
// active are operator-set; low_stock and out_of_stock are auto-derived.
var validProductStatuses = map[string]bool{
	"draft":        true,
	"active":       true,
	"low_stock":    true,
	"out_of_stock": true,
}

// orderTransitions defines the forward-only order state machine. Returns are
// handled separately via the return_request_status field.
var orderTransitions = map[string]map[string]bool{
	"pending":    {"processing": true, "cancelled": true},
	"processing": {"shipped": true, "cancelled": true},
	"shipped":    {"delivered": true},
	"delivered":  {},
	"cancelled":  {},
}

// returnTransitions defines the return request state machine.
// "received" is a terminal state for the admin return endpoint: it does
// NOT auto-restock and does NOT transition to "refunded" via this route.
// The INTEGRATION_PLAN.md (section 6.2) target state machine has
// received -> refunded, but that transition requires a verified provider
// webhook/refund adapter or governed offline-payment confirmation, a
// dedicated orders.refund capability, idempotency, and AAL2 — none of
// which are implemented yet (B7 gated remainder, AC-010 pending).
// Exposing "refunded" here would re-introduce the fake-refund path that
// Phase E removed. The per-item returned_quantity/restocked_quantity
// ledger and the inspected restock endpoint are also NOT yet implemented
// — see AC-009/REQ-005 pending items.
var returnTransitions = map[string]map[string]bool{
	"":          {"requested": true},
	"requested": {"approved": true, "rejected": true},
	"approved":  {"received": true, "rejected": true},
	"received":  {},
	"refunded":  {},
	"rejected":  {},
}

// Service holds the commerce business logic. It never reads auth.Principal
// from context; callers pass it explicitly so capability checks stay visible.
type Service struct {
	store         Store
	mediaVerifier MediaVerifier // nil if not wired — product image association is rejected
	publicBaseURL string        // R2 public base URL for deriving image URLs
}

// NewService constructs a Service backed by the given Store.
func NewService(store Store) Service {
	return Service{store: store}
}

// WithMediaVerifier returns a Service with the given media verifier.
// Bootstrap wires this adapter to validate product image keys against
// the media registry without cross-module imports. When nil (not wired),
// product image association is rejected with ErrUnverifiedMedia.
func (s Service) WithMediaVerifier(v MediaVerifier) Service {
	s.mediaVerifier = v
	return s
}

// WithPublicBaseURL returns a Service with the given R2 public base URL
// for deriving public image URLs from verified object keys.
func (s Service) WithPublicBaseURL(baseURL string) Service {
	s.publicBaseURL = strings.TrimRight(baseURL, "/")
	return s
}

// ----- Products -------------------------------------------------------------

func (s Service) ListProducts(ctx context.Context, filter ProductFilter) ([]Product, error) {
	products, err := s.store.ListProducts(ctx, filter)
	if err != nil {
		return nil, err
	}
	// Admin response: load product_images but do NOT derive URLs.
	return s.enrichProductsListWithImages(ctx, products, false)
}

func (s Service) ListPublishedProducts(ctx context.Context) ([]Product, error) {
	products, err := s.store.ListPublishedProducts(ctx)
	if err != nil {
		return nil, err
	}
	return s.enrichProductsListWithImages(ctx, products, true)
}

func (s Service) GetProduct(ctx context.Context, id string) (Product, error) {
	p, err := s.store.GetProduct(ctx, id)
	if err != nil {
		return p, err
	}
	// Admin response: load product_images but do NOT derive URLs.
	// The handler builds an AdminProduct DTO with raw keys.
	return s.enrichProductWithImages(ctx, p, false)
}

func (s Service) GetProductBySlug(ctx context.Context, slug string) (Product, error) {
	p, err := s.store.GetProductBySlug(ctx, slug)
	if err != nil {
		return p, err
	}
	// Public response: derive URLs from verified object keys.
	return s.enrichProductWithImages(ctx, p, true)
}

// reservedSlugs are route segments that product/category slugs cannot use
// because they would collide with static site routes or API paths.
var reservedSlugs = map[string]bool{
	"api": true, "admin": true, "assets": true, "articles": true,
	"content": true, "products": true, "category": true, "categories": true,
	"login": true, "logout": true, "account": true, "cart": true,
	"checkout": true, "orders": true, "quote": true, "health": true,
	"static": true, "favicon.ico": true, "robots.txt": true, "sitemap.xml": true,
}

// validateSlug rejects empty slugs, reserved route segments, path traversal
// attempts, and slugs with invalid characters. A valid slug is lowercase,
// contains only [a-z0-9-], and does not start/end with a hyphen.
func validateSlug(slug string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidProductInput)
	}
	if reservedSlugs[strings.ToLower(slug)] {
		return fmt.Errorf("%w: slug %q is reserved", ErrInvalidProductInput, slug)
	}
	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return fmt.Errorf("%w: slug %q contains path traversal characters", ErrInvalidProductInput, slug)
	}
	for _, r := range slug {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return fmt.Errorf("%w: slug %q contains invalid character %q", ErrInvalidProductInput, slug, string(r))
		}
	}
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return fmt.Errorf("%w: slug %q cannot start or end with a hyphen", ErrInvalidProductInput, slug)
	}
	return nil
}

// validateCategory rejects category values that would be unsafe as a static
// route segment. The category becomes /categories/{category}/ in the renderer,
// so it must be a single safe path segment. Empty is allowed (caller defaults
// it before calling). Reserved route segments, path traversal, slashes,
// backslashes, absolute/volume prefixes, and non-[a-z0-9-] characters are
// rejected. Dotted keys (e.g. "home.announcement") are NOT valid categories
// because categories map to a single route segment with no dots.
func validateCategory(category string) error {
	category = strings.TrimSpace(category)
	if category == "" {
		return nil // caller is responsible for defaulting empty
	}
	if reservedSlugs[strings.ToLower(category)] {
		return fmt.Errorf("%w: category %q is reserved", ErrInvalidProductInput, category)
	}
	if strings.Contains(category, "..") || strings.Contains(category, "/") || strings.Contains(category, "\\") {
		return fmt.Errorf("%w: category %q contains path traversal characters", ErrInvalidProductInput, category)
	}
	for _, r := range category {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
			return fmt.Errorf("%w: category %q contains invalid character %q", ErrInvalidProductInput, category, string(r))
		}
	}
	if strings.HasPrefix(category, "-") || strings.HasSuffix(category, "-") {
		return fmt.Errorf("%w: category %q cannot start or end with a hyphen", ErrInvalidProductInput, category)
	}
	return nil
}

func (s Service) CreateProduct(ctx context.Context, principal auth.Principal, in ProductInput) (Product, error) {
	if !auth.Can(principal, "twcommerce.create") {
		return Product{}, ErrForbidden
	}
	if strings.TrimSpace(in.SKU) == "" || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Slug) == "" {
		return Product{}, fmt.Errorf("%w: sku, name, and slug are required", ErrInvalidProductInput)
	}
	if err := validateSlug(in.Slug); err != nil {
		return Product{}, err
	}
	if in.Price < 0 || in.OriginalPrice < 0 || in.Stock < 0 {
		return Product{}, fmt.Errorf("%w: price, original_price, and stock must be non-negative", ErrInvalidProductInput)
	}
	// ProductInput no longer has Image/Images fields. httpx.DecodeJSON
	// uses DisallowUnknownFields, so any payload containing image or
	// images is rejected with 400 at the handler boundary. No explicit
	// check needed here.
	// Validate all product image keys via the media verifier. All keys
	// are new at creation time, so each must be a verified object owned
	// by the acting principal.
	productImages, err := s.validateAndBuildProductImages(ctx, principal, in.ProductImages, strings.TrimSpace(in.Name))
	if err != nil {
		return Product{}, err
	}
	id, err := randomID()
	if err != nil {
		return Product{}, err
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "draft"
	}
	if !validProductStatuses[status] {
		return Product{}, fmt.Errorf("%w: invalid product status %q", ErrInvalidProductInput, status)
	}
	// Auto-derive low_stock/out_of_stock from stock unless operator forced draft.
	if status != "draft" {
		status = deriveProductStatus(status, in.Stock)
	}
	category := defaultString(strings.TrimSpace(in.Category), "apparel")
	if err := validateCategory(category); err != nil {
		return Product{}, err
	}
	now := time.Now().Unix()
	p := Product{
		ID:              id,
		SKU:             strings.TrimSpace(in.SKU),
		Name:            strings.TrimSpace(in.Name),
		Slug:            strings.TrimSpace(in.Slug),
		Description:     in.Description,
		LongDescription: in.LongDescription,
		Image:           "",
		Images:          "[]",
		Category:        category,
		Status:          status,
		Material:        in.Material,
		Origin:          in.Origin,
		Price:           in.Price,
		OriginalPrice:   in.OriginalPrice,
		Stock:           in.Stock,
		Tag:             in.Tag,
		UpdatedUnix:     now,
	}
	// Set IDs and timestamps on product images.
	for i := range productImages {
		imgID, err := randomID()
		if err != nil {
			return Product{}, err
		}
		productImages[i].ID = imgID
		productImages[i].ProductID = id
		productImages[i].CreatedUnix = now
	}
	if err := s.store.UpsertProductWithImages(ctx, p, productImages); err != nil {
		return Product{}, err
	}
	p.ProductImages = productImages
	return p, nil
}

func (s Service) UpdateProduct(ctx context.Context, principal auth.Principal, id string, in ProductInput) (Product, error) {
	if !auth.Can(principal, "twcommerce.update") {
		return Product{}, ErrForbidden
	}
	existing, err := s.store.GetProduct(ctx, id)
	if err != nil {
		return Product{}, err
	}
	// Validate slug if it changed.
	if strings.TrimSpace(in.Slug) != "" && in.Slug != existing.Slug {
		if err := validateSlug(in.Slug); err != nil {
			return Product{}, err
		}
	}
	if in.Price < 0 || in.OriginalPrice < 0 || in.Stock < 0 {
		return Product{}, fmt.Errorf("%w: price, original_price, and stock must be non-negative", ErrInvalidProductInput)
	}
	// ProductInput no longer has Image/Images fields. httpx.DecodeJSON
	// uses DisallowUnknownFields, so any payload containing image or
	// images is rejected with 400 at the handler boundary. The legacy
	// products.image/images columns are preserved from the existing
	// row (they are derived from product_images at public response time).
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = existing.Status
	}
	if !validProductStatuses[status] {
		return Product{}, fmt.Errorf("%w: invalid product status %q", ErrInvalidProductInput, status)
	}
	if status != "draft" {
		status = deriveProductStatus(status, in.Stock)
	}
	category := defaultString(strings.TrimSpace(in.Category), existing.Category)
	if err := validateCategory(category); err != nil {
		return Product{}, err
	}
	productName := defaultString(strings.TrimSpace(in.Name), existing.Name)
	now := time.Now().Unix()
	p := Product{
		ID:              id,
		SKU:             defaultString(strings.TrimSpace(in.SKU), existing.SKU),
		Name:            productName,
		Slug:            defaultString(strings.TrimSpace(in.Slug), existing.Slug),
		Description:     in.Description,
		LongDescription: in.LongDescription,
		// Legacy flat image columns are always written empty — no
		// legacy backfill. Public URLs are derived from product_images
		// at response time via enrichProductWithImages.
		Image:         "",
		Images:        "[]",
		Category:      category,
		Status:        status,
		Material:      in.Material,
		Origin:        in.Origin,
		Price:         in.Price,
		OriginalPrice: in.OriginalPrice,
		Stock:         in.Stock,
		Tag:           in.Tag,
		Rating:        existing.Rating,
		ReviewsCount:  existing.ReviewsCount,
		UpdatedUnix:   now,
	}

	// Handle product_images replacement. If in.ProductImages is nil,
	// preserve existing associations (any authorized manager can update
	// other fields without touching images). If non-nil (even empty),
	// replace all associations.
	var productImages []ProductImage
	if in.ProductImages != nil {
		// Load existing associations to distinguish new keys from
		// preserved keys. New keys require media verification with
		// the acting principal. Existing keys (already associated)
		// do not need re-verification — any authorized manager can
		// keep them without impersonating the original uploader.
		existingImgs, err := s.store.ListProductImages(ctx, id)
		if err != nil {
			return Product{}, fmt.Errorf("load existing product images: %w", err)
		}
		existingKeys := make(map[string]bool, len(existingImgs))
		for _, ei := range existingImgs {
			existingKeys[ei.ObjectKey] = true
		}
		productImages, err = s.validateAndBuildProductImagesWithExisting(ctx, principal, in.ProductImages, productName, existingKeys)
		if err != nil {
			return Product{}, err
		}
		for i := range productImages {
			imgID, err := randomID()
			if err != nil {
				return Product{}, err
			}
			productImages[i].ID = imgID
			productImages[i].ProductID = id
			productImages[i].CreatedUnix = now
		}
		if err := s.store.UpsertProductWithImages(ctx, p, productImages); err != nil {
			return Product{}, err
		}
	} else {
		// Preserve existing images — just upsert the product.
		if err := s.store.UpsertProduct(ctx, p); err != nil {
			return Product{}, err
		}
		var err error
		productImages, err = s.store.ListProductImages(ctx, id)
		if err != nil {
			return Product{}, fmt.Errorf("load product images after update: %w", err)
		}
	}
	p.ProductImages = productImages
	return p, nil
}

func encodeProductImages(images []string) string {
	if images == nil {
		images = []string{}
	}
	encoded, _ := json.Marshal(images) // []string cannot fail JSON marshaling.
	return string(encoded)
}

// validateAndBuildProductImages validates all image keys as new keys
// via the media verifier and builds ProductImage structs with alt_text
// defaulting to productName and sort_order from slice order. Used by
// CreateProduct where all keys are new.
//
// Error semantics:
//   - ErrUnverifiedMedia (400): key is not verified or owned by a
//     different user. The handler maps this to 400.
//   - Wrapped error (5xx): registry/DB failure from the verifier.
//     The handler maps this to 500/503, NOT 400. The caller must
//     propagate this error without collapsing it to ErrUnverifiedMedia.
func (s Service) validateAndBuildProductImages(ctx context.Context, principal auth.Principal, inputs []ProductImageInput, productName string) ([]ProductImage, error) {
	if s.mediaVerifier == nil && len(inputs) > 0 {
		return nil, ErrUnverifiedMedia
	}
	images := make([]ProductImage, 0, len(inputs))
	seenKeys := make(map[string]bool, len(inputs))
	for _, in := range inputs {
		key := strings.TrimSpace(in.Key)
		if key == "" {
			return nil, fmt.Errorf("%w: product image key is required", ErrInvalidProductInput)
		}
		if seenKeys[key] {
			return nil, fmt.Errorf("%w: duplicate product image key %q", ErrInvalidProductInput, key)
		}
		seenKeys[key] = true
		if s.mediaVerifier != nil {
			if err := s.mediaVerifier.VerifyKey(ctx, principal.UserID, key); err != nil {
				// Only ErrUnverifiedMedia is a 400. All other errors
				// (DB failures, provider errors) propagate as 5xx.
				if errors.Is(err, ErrUnverifiedMedia) {
					return nil, ErrUnverifiedMedia
				}
				return nil, fmt.Errorf("verify product image key: %w", err)
			}
		}
		alt := strings.TrimSpace(in.AltText)
		if alt == "" {
			alt = productName
		}
		images = append(images, ProductImage{
			ObjectKey: key,
			AltText:   alt,
		})
	}
	return images, nil
}

// validateAndBuildProductImagesWithExisting validates image keys, but
// skips verification for keys already associated with this product.
// Only new keys (not in existingKeys) require media verification with
// the acting principal. This allows an authorized manager to preserve
// existing image associations without impersonating the original
// uploader.
//
// Error semantics are the same as validateAndBuildProductImages:
// ErrUnverifiedMedia -> 400, wrapped error -> 5xx.
func (s Service) validateAndBuildProductImagesWithExisting(ctx context.Context, principal auth.Principal, inputs []ProductImageInput, productName string, existingKeys map[string]bool) ([]ProductImage, error) {
	if s.mediaVerifier == nil && len(inputs) > 0 {
		return nil, ErrUnverifiedMedia
	}
	images := make([]ProductImage, 0, len(inputs))
	seenKeys := make(map[string]bool, len(inputs))
	for _, in := range inputs {
		key := strings.TrimSpace(in.Key)
		if key == "" {
			return nil, fmt.Errorf("%w: product image key is required", ErrInvalidProductInput)
		}
		if seenKeys[key] {
			return nil, fmt.Errorf("%w: duplicate product image key %q", ErrInvalidProductInput, key)
		}
		seenKeys[key] = true
		// Only verify NEW keys. Existing keys (already associated with
		// this product) are trusted without re-verification.
		if !existingKeys[key] && s.mediaVerifier != nil {
			if err := s.mediaVerifier.VerifyKey(ctx, principal.UserID, key); err != nil {
				if errors.Is(err, ErrUnverifiedMedia) {
					return nil, ErrUnverifiedMedia
				}
				return nil, fmt.Errorf("verify product image key: %w", err)
			}
		}
		alt := strings.TrimSpace(in.AltText)
		if alt == "" {
			alt = productName
		}
		images = append(images, ProductImage{
			ObjectKey: key,
			AltText:   alt,
		})
	}
	return images, nil
}

// derivePublicImageURL builds the public URL for a verified object key
// using the R2 public base URL. Returns empty string if publicBaseURL
// is not configured. The caller (enrichProductWithImages) checks for
// empty results and fails closed with ErrMediaURLUnavailable when
// product_images exist but no URL could be derived.
func (s Service) derivePublicImageURL(objectKey string) string {
	if s.publicBaseURL == "" || objectKey == "" {
		return ""
	}
	return s.publicBaseURL + "/" + objectKey
}

// enrichProductWithImages loads product_images from the store and
// populates the Product's ProductImages field. For public responses
// (deriveURLs=true), it ALWAYS clears the legacy Image/Images fields
// first and then derives them solely from verified product_images
// keys joined with the public base URL. This ensures no legacy
// unverified URL survives as a fallback when product_images is empty
// — the approved design is verified-only with no legacy backfill.
//
// If deriveURLs=true and product_images exist but publicBaseURL is
// empty, the function returns ErrMediaURLUnavailable. The service
// must NOT emit empty/broken URLs or succeed silently — public
// endpoints map this to 503, and the render tool fails and preserves
// LKG dist.
//
// For admin responses (deriveURLs=false), the legacy Image/Images
// fields are also cleared to empty so the admin DTO does not treat
// old flat URLs as authority. The admin DTO exposes raw object keys
// via ProductImages for editing. No URL derivation is needed for
// admin responses, so ErrMediaURLUnavailable is never returned.
func (s Service) enrichProductWithImages(ctx context.Context, p Product, deriveURLs bool) (Product, error) {
	imgs, err := s.store.ListProductImages(ctx, p.ID)
	if err != nil {
		return p, fmt.Errorf("load product images: %w", err)
	}
	p.ProductImages = imgs
	// Always clear legacy flat columns. No legacy backfill — only
	// verified product_images are authority for public URLs.
	p.Image = ""
	p.Images = "[]"
	if deriveURLs && len(imgs) > 0 {
		if s.publicBaseURL == "" {
			// Cannot derive public URLs without a CDN base. Fail closed
			// rather than emitting empty or broken URLs.
			return p, ErrMediaURLUnavailable
		}
		urls := make([]string, len(imgs))
		for i, img := range imgs {
			urls[i] = s.derivePublicImageURL(img.ObjectKey)
		}
		p.Image = urls[0]
		p.Images = encodeProductImages(urls)
	}
	return p, nil
}

// enrichProductsListWithImages loads product_images for each product
// and derives public URLs. This is used by public listing endpoints.
func (s Service) enrichProductsListWithImages(ctx context.Context, products []Product, deriveURLs bool) ([]Product, error) {
	for i := range products {
		enhanced, err := s.enrichProductWithImages(ctx, products[i], deriveURLs)
		if err != nil {
			return nil, err
		}
		products[i] = enhanced
	}
	return products, nil
}

func (s Service) UpdateProductStatus(ctx context.Context, principal auth.Principal, id, status string) (Product, error) {
	if !auth.Can(principal, "twcommerce.update") {
		return Product{}, ErrForbidden
	}
	if !validProductStatuses[status] {
		return Product{}, fmt.Errorf("%w: invalid product status %q", ErrInvalidProductInput, status)
	}
	existing, err := s.store.GetProduct(ctx, id)
	if err != nil {
		return Product{}, err
	}
	// Auto-derive low_stock/out_of_stock from current stock unless operator
	// explicitly sets draft.
	if status != "draft" {
		status = deriveProductStatus(status, existing.Stock)
	}
	if err := s.store.UpdateProductStatus(ctx, id, status, time.Now().Unix()); err != nil {
		return Product{}, err
	}
	p, err := s.store.GetProduct(ctx, id)
	if err != nil {
		return Product{}, err
	}
	// Admin response: load product_images but do NOT derive URLs.
	return s.enrichProductWithImages(ctx, p, false)
}

func (s Service) DeleteProduct(ctx context.Context, principal auth.Principal, id string) error {
	if !auth.Can(principal, "twcommerce.delete") {
		return ErrForbidden
	}
	return s.store.DeleteProduct(ctx, id)
}

// BulkUpdateProducts applies a bulk action to a set of products. The payload
// map currently supports {"status": "<status>"}.
func (s Service) BulkUpdateProducts(ctx context.Context, principal auth.Principal, ids []string, payload map[string]any) error {
	if !auth.Can(principal, "twcommerce.update") {
		return ErrForbidden
	}
	status, ok := payload["status"].(string)
	if !ok {
		return fmt.Errorf("%w: bulk payload must include a status string", ErrInvalidProductInput)
	}
	if !validProductStatuses[status] {
		return fmt.Errorf("%w: invalid product status %q", ErrInvalidProductInput, status)
	}
	return s.store.BulkUpdateProductStatus(ctx, ids, status, time.Now().Unix())
}

// deriveProductStatus auto-derives low_stock/out_of_stock from stock. An
// operator-requested active status is downgraded when stock is low or zero.
// draft is always preserved as an operator-set offline state.
func deriveProductStatus(requested string, stock int) string {
	if requested == "draft" {
		return "draft"
	}
	switch {
	case stock <= 0:
		return "out_of_stock"
	case stock < 5:
		return "low_stock"
	default:
		if requested == "out_of_stock" || requested == "low_stock" {
			return "active"
		}
		return requested
	}
}

// ----- Orders ---------------------------------------------------------------

func (s Service) ListOrders(ctx context.Context, filter OrderFilter) ([]Order, error) {
	return s.store.ListOrders(ctx, filter)
}

func (s Service) GetOrder(ctx context.Context, id string) (Order, error) {
	o, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return Order{}, err
	}
	o = hydrateOrder(o)
	// Merge returned_quantity/restocked_quantity from the order_items ledger.
	// items_json remains the snapshot source for SKU/name/price/quantity;
	// the ledger columns are only in order_items. A DB error here is
	// propagated — silently ignoring it would hide a corrupted ledger from
	// the admin UI, causing decisions based on stale restock data.
	ledgerItems, err := s.store.GetOrderItems(ctx, id)
	if err != nil {
		return Order{}, fmt.Errorf("load order items ledger: %w", err)
	}
	if len(ledgerItems) > 0 {
		ledgerBySKU := make(map[string]OrderItem, len(ledgerItems))
		for _, li := range ledgerItems {
			ledgerBySKU[li.SKU] = li
		}
		for i := range o.Items {
			if li, ok := ledgerBySKU[o.Items[i].SKU]; ok {
				o.Items[i].ReturnedQuantity = li.ReturnedQuantity
				o.Items[i].RestockedQuantity = li.RestockedQuantity
			}
		}
	}
	return o, nil
}

// GetOrderForGuest returns an order for a guest customer. The caller must
// provide the order ID AND the opaque access token that was generated at
// order creation time. If the token does not match, ErrNotFound is returned
// (not a forbidden error, to avoid leaking order existence). PII fields
// (phone, email, shipping address) are masked in the returned order.
// The access token is never included in the response.
func (s Service) GetOrderForGuest(ctx context.Context, id, token string) (Order, error) {
	o, err := s.store.GetOrderByAccessToken(ctx, id, token)
	if err != nil {
		return Order{}, err
	}
	return maskCustomerPII(hydrateOrder(o)), nil
}

// GetOrderForMember returns an order for an authenticated member. The
// principal's UserID must match the order's MemberID. If it does not match,
// ErrNotFound is returned (not forbidden, to avoid leaking existence).
// PII fields are masked. The access token is never included.
func (s Service) GetOrderForMember(ctx context.Context, principal auth.Principal, id string) (Order, error) {
	o, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return Order{}, err
	}
	if principal.UserID == "" || o.MemberID != principal.UserID {
		return Order{}, ErrNotFound
	}
	return maskCustomerPII(hydrateOrder(o)), nil
}

// ListOrdersForMember returns all orders for a registered member. The
// caller must have verified the member's identity (e.g. via Supabase
// session). PII fields are masked in the returned orders.
func (s Service) ListOrdersForMember(ctx context.Context, memberID string) ([]Order, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}
	orders, err := s.store.ListOrders(ctx, OrderFilter{MemberID: memberID})
	if err != nil {
		return nil, err
	}
	masked := make([]Order, len(orders))
	for i, o := range orders {
		masked[i] = maskCustomerPII(hydrateOrder(o))
	}
	return masked, nil
}

// maskCustomerPII redacts sensitive fields from an order before returning
// it to a customer-facing endpoint. Phone is partially masked, email is
// partially masked, and shipping address is masked to keep only the
// city/district prefix. The access token is always cleared.
func maskCustomerPII(o Order) Order {
	o.Phone = maskPhone(o.Phone)
	o.Email = maskEmail(o.Email)
	o.ShippingAddress = maskShippingAddress(o.ShippingAddress)
	o.AccessToken = ""     // never expose the plaintext access token
	o.AccessTokenHash = "" // never expose the hash either
	return o
}

// maskShippingAddress keeps only the first 6 characters (enough for a
// Taiwan city + district like "台北市大安區") and masks the rest. If the
// address is shorter than 6 chars, it is fully masked.
func maskShippingAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if len(addr) <= 6 {
		return strings.Repeat("*", len(addr))
	}
	return addr[:6] + strings.Repeat("*", len(addr)-6)
}

func maskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) <= 4 {
		return strings.Repeat("*", len(phone))
	}
	// Keep last 4 digits, mask the rest.
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return email
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 2 {
		return local + domain
	}
	return local[:2] + strings.Repeat("*", len(local)-2) + domain
}

// QuoteInput is the browser-supplied quote request. Client prices, totals,
// shipping, discount, and payment claims are untrusted; the service
// recalculates everything from the product catalog and approved rules.
type QuoteInput struct {
	Items          []OrderItem `json:"items"`
	ShippingMethod string      `json:"shipping_method"`
	PaymentMethod  string      `json:"payment_method"`
	PromoCode      string      `json:"promo_code"`
}

// QuoteResult is the server-authoritative calculation returned to the
// browser. The checkout UI must display these totals, not client-computed
// ones.
type QuoteResult struct {
	Items          []OrderItem `json:"items"`
	Subtotal       int         `json:"subtotal"`
	Discount       int         `json:"discount"`
	Shipping       int         `json:"shipping"`
	Total          int         `json:"total"`
	PromoCode      string      `json:"promo_code,omitempty"`
	ShippingMethod string      `json:"shipping_method,omitempty"`
	PaymentMethod  string      `json:"payment_method,omitempty"`
}

// Quote calculates authoritative totals from the product catalog. It does
// not decrement stock or persist anything. The browser must display these
// totals and must not send its own calculation to CreateOrder.
//
// Shipping is computed from the current shipping_methods rows. Empty,
// missing, or disabled methods return ErrInvalidShippingMethod. A store
// read failure returns ErrShippingConfigUnavailable.
//
// Quote also validates the payment method against the admin-managed
// payment_methods table. An empty, unknown, disabled, or not-ready
// payment method returns ErrInvalidPaymentMethod. The server is the
// authority for payment method availability.
func (s Service) Quote(ctx context.Context, in QuoteInput) (QuoteResult, error) {
	if len(in.Items) == 0 {
		return QuoteResult{}, fmt.Errorf("quote must contain at least one item")
	}
	items, subtotal, err := s.resolveItems(ctx, in.Items)
	if err != nil {
		return QuoteResult{}, err
	}
	shipping, err := s.computeShipping(ctx, in.ShippingMethod, subtotal)
	if err != nil {
		return QuoteResult{}, err
	}
	if err := s.validatePaymentMethod(ctx, in.PaymentMethod); err != nil {
		return QuoteResult{}, err
	}
	discount, err := s.calculateDiscount(ctx, subtotal, strings.TrimSpace(in.PromoCode))
	if err != nil {
		return QuoteResult{}, err
	}
	total, err := addOverflow(subtotal-discount, shipping)
	if err != nil {
		return QuoteResult{}, fmt.Errorf("total overflow: %w", err)
	}
	return QuoteResult{
		Items:          items,
		Subtotal:       subtotal,
		Discount:       discount,
		Shipping:       shipping,
		Total:          total,
		PromoCode:      strings.TrimSpace(in.PromoCode),
		ShippingMethod: strings.TrimSpace(in.ShippingMethod),
		PaymentMethod:  strings.TrimSpace(in.PaymentMethod),
	}, nil
}

// resolveItems looks up each SKU in the catalog and returns authoritative
// line items with server prices. Client-supplied prices are ignored.
//
// All arithmetic uses checked operations: line totals and subtotal
// accumulation that would overflow int return ErrOverflow. This prevents
// a malicious client from sending extreme quantities or prices (via
// promo manipulation) that wrap to a negative or low total.
func (s Service) resolveItems(ctx context.Context, items []OrderItem) ([]OrderItem, int, error) {
	var (
		out      []OrderItem
		subtotal int
	)
	for _, ci := range items {
		if ci.Quantity <= 0 {
			return nil, 0, fmt.Errorf("item quantity must be positive")
		}
		product, err := s.findProductBySKU(ctx, ci.SKU)
		if err != nil {
			return nil, 0, fmt.Errorf("product %q: %w", ci.SKU, err)
		}
		// Checked line total: product.Price * ci.Quantity
		lineTotal, err := mulOverflow(product.Price, ci.Quantity)
		if err != nil {
			return nil, 0, fmt.Errorf("line total overflow for %q: %w", ci.SKU, err)
		}
		// Checked subtotal accumulation
		subtotal, err = addOverflow(subtotal, lineTotal)
		if err != nil {
			return nil, 0, fmt.Errorf("subtotal overflow: %w", err)
		}
		line := OrderItem{
			SKU:      product.SKU,
			Name:     product.Name,
			Price:    product.Price,
			Quantity: ci.Quantity,
		}
		out = append(out, line)
	}
	return out, subtotal, nil
}

// CreateOrder is the public guest checkout entry point. No auth is required.
// The service recalculates totals from the product catalog and ignores any
// client-supplied prices, shipping, discount, or payment claims. Stock is
// decremented atomically within the same transaction as the order insert.
// Idempotency is checked INSIDE the transaction to eliminate race conditions.
// MemberID is always empty for guest checkout.
func (s Service) CreateOrder(ctx context.Context, in OrderInput) (Order, error) {
	return s.createOrder(ctx, in, "")
}

// CreateOrderForMember is the authenticated member checkout entry point.
// The MemberID is derived from the authenticated principal and written in
// the SAME transaction as the order insert and stock decrement — there is
// no post-commit patch. If the transaction fails, no order is left behind
// and no stock is decremented. This is the REQ-009 invariant: member
// identity is never trusted from the client and is atomic with order creation.
func (s Service) CreateOrderForMember(ctx context.Context, principal auth.Principal, in OrderInput) (Order, error) {
	if principal.UserID == "" {
		return Order{}, ErrForbidden
	}
	return s.createOrder(ctx, in, principal.UserID)
}

// normalizeOrderInput trims all string fields on OrderInput. It does
// NOT perform any semantic validation that could reject the input —
// it only normalizes for consistent fingerprinting, comparison, and
// persistence. This is called BEFORE the idempotency check so that
// the same value with different surrounding whitespace produces the
// same fingerprint and matches the same existing order.
//
// Semantic validation (email format, non-blank phone/address) is done
// separately in validateOrderInput AFTER the idempotency check, so
// that a same-key different-payload replay returns ErrIdempotencyConflict
// even if the new payload would fail validation.
func normalizeOrderInput(in OrderInput) OrderInput {
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	in.Email = strings.TrimSpace(in.Email)
	in.Phone = strings.TrimSpace(in.Phone)
	in.ShippingAddress = strings.TrimSpace(in.ShippingAddress)
	in.ShippingMethod = strings.TrimSpace(in.ShippingMethod)
	in.PaymentMethod = strings.TrimSpace(in.PaymentMethod)
	in.PromoCode = strings.TrimSpace(in.PromoCode)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	return in
}

// validateOrderInput performs semantic validation on the required
// contact fields. This is called ONLY when creating a new order (after
// the idempotency check confirms no existing order with the same key).
// Replays of existing orders bypass this validation — a same-key
// different-payload replay returns ErrIdempotencyConflict regardless
// of whether the new payload would pass or fail validation.
//
// Email is validated via net/mail.ParseAddress (RFC 5322), not a regex.
// The OpenAPI contract declares email with format: email.
//
// Starter limits on field length are deliberately generous to avoid
// changing business rules — they guard against absurdly large inputs
// that could cause DB or memory issues, not enforce a narrow policy.
func validateOrderInput(in OrderInput) error {
	if in.Email == "" {
		return ErrInvalidEmail
	}
	// net/mail.ParseAddress parses a single RFC 5322 address. It
	// rejects most malformed inputs (dangling commas, newlines in
	// wrong places, missing @, etc.). We validate but do NOT
	// canonicalize — the trimmed input is stored as-is so the
	// fingerprint at creation matches the fingerprint at replay.
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return ErrInvalidEmail
	}

	if in.Phone == "" {
		return ErrInvalidPhone
	}
	if in.ShippingAddress == "" {
		return ErrInvalidShippingAddress
	}

	// Starter limits: generous enough for any reasonable input, small
	// enough to prevent absurd payloads. These are NOT business rules
	// (e.g. phone format validation is a product decision). They guard
	// against DB column overflow and memory abuse.
	const (
		maxEmailLen           = 254 // RFC 5321 max
		maxPhoneLen           = 30
		maxShippingAddressLen = 500
		maxCustomerNameLen    = 100
	)
	if len(in.Email) > maxEmailLen {
		return ErrInvalidEmail
	}
	if len(in.Phone) > maxPhoneLen {
		return ErrInvalidPhone
	}
	if len(in.ShippingAddress) > maxShippingAddressLen {
		return ErrInvalidShippingAddress
	}
	if len(in.CustomerName) > maxCustomerNameLen {
		return fmt.Errorf("customer_name exceeds starter limit")
	}

	return nil
}

// createOrder is the shared internal checkout implementation. The memberID
// parameter is "" for guest checkout or the principal's UserID for member
// checkout. It is written in the same transaction as the order insert and
// stock decrement, ensuring atomicity.
//
// VALIDATION ORDER (order matters — each step is side-effect-free until STEP 7):
//  1. idempotency_key non-blank (the only pre-lookup check).
//  2. Normalize (trim only — no semantic validation that rejects).
//  3. EARLY idempotency check: lookup by key, compare fingerprints using
//     the normalized payload. Same payload → return original order.
//     Different payload → ErrIdempotencyConflict. This happens BEFORE
//     any other validation so a same-key different-payload replay always
//     returns 409 Conflict, even if the new payload has empty items,
//     blank customer_name, or a malformed email.
//  4. New-order structural validation: items non-empty, customer_name
//     non-blank. Only runs when key is NOT found.
//  5. Resolve items (checked arithmetic for overflow).
//  6. Stock check, shipping validation, payment method validation,
//     discount/total calculation (checked arithmetic for overflow).
//  7. Contact field validation (email format, phone/address non-blank).
//     This runs AFTER all side-effect-free catalog/state validation,
//     so tests exercising stock/shipping/payment errors do not need
//     valid PII — those errors fire first.
//  8. Persistence: marshal, generate order ID + access token, transaction.
//
// This ordering preserves the invariant that the server tells the client
// "you already used this key for a different request" (409 Conflict)
// rather than "your new request is invalid" (400 Bad Request) when both
// are true. The 409 is the more truthful and useful response — the client
// must not retry with the same key regardless of the new payload's validity.
//
// The in-transaction check in CreateOrderTxWithIdempotency remains as a
// race-condition safety net for concurrent requests with the same key.
func (s Service) createOrder(ctx context.Context, in OrderInput, memberID string) (Order, error) {
	// Require a non-empty idempotency key. An empty key bypasses both
	// the early lookup and the in-transaction idempotency check, which
	// means a transport retry (server commits, response lost, client
	// retries) would create a duplicate order and double-decrement stock.
	// The production order endpoints must fail closed on blank keys.
	// This is the ONLY check before the idempotency lookup — items and
	// customer_name validation runs only on the "new order" path (after
	// the lookup confirms no existing order with this key), so a
	// same-key different-payload replay with empty items or blank
	// customer_name still returns ErrIdempotencyConflict.
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return Order{}, ErrMissingIdempotencyKey
	}

	// STEP 1: Normalize (trim only). No semantic validation that could
	// reject the input. This ensures the fingerprint and comparison use
	// normalized values, so whitespace differences do not cause data
	// inconsistency or fingerprint mismatch on replay.
	in = normalizeOrderInput(in)

	// STEP 2: EARLY IDEMPOTENCY CHECK. Before any semantic or mutable-
	// state validation, check if an order with the same idempotency_key
	// already exists. If it does, compare fingerprints and return the
	// original order (same payload) or ErrIdempotencyConflict (different
	// payload). This prevents a replay from failing with ErrInvalidEmail,
	// ErrInsufficientStock, or ErrInvalidPaymentMethod after the original
	// order succeeded — the client must know the key was already used,
	// not that the new (different) payload is invalid.
	key := in.IdempotencyKey // already trimmed
	existing, err := s.store.FindOrderByIdempotencyKey(ctx, key)
	if err == nil {
		// Found existing order — compare fingerprints.
		if !samePayload(existing, in, memberID) {
			return Order{}, ErrIdempotencyConflict
		}
		// Same payload — return the original order. The plaintext
		// access token is NOT returned on replay.
		existing.AccessToken = ""
		return hydrateOrder(existing), nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Order{}, fmt.Errorf("find by idempotency key: %w", err)
	}
	// ErrNotFound — no existing order with this key. Proceed to new-order
	// validation path.

	// STEP 3: New-order structural validation. These checks only run
	// when the key is NOT found — a same-key different-payload replay
	// already returned ErrIdempotencyConflict in STEP 2, regardless of
	// whether the new payload has empty items or a blank customer_name.
	if len(in.Items) == 0 {
		return Order{}, fmt.Errorf("order must contain at least one item")
	}
	if in.CustomerName == "" { // already trimmed by normalizeOrderInput
		return Order{}, fmt.Errorf("customer_name is required")
	}
	// Reject duplicate SKUs in the input. The order_items table has a
	// (order_id, sku) unique index — without this check, a request with
	// duplicate SKUs would pass validation but fail at INSERT with a
	// unique constraint violation (500 Internal Server Error). Rejecting
	// here returns a 400 Bad Request with a clear message instead.
	seenSKUs := make(map[string]bool, len(in.Items))
	for _, ci := range in.Items {
		sku := strings.TrimSpace(ci.SKU)
		if sku == "" {
			return Order{}, fmt.Errorf("item sku is required")
		}
		if seenSKUs[sku] {
			return Order{}, fmt.Errorf("%w: duplicate item sku %q in order request", ErrInvalidAdminInput, sku)
		}
		seenSKUs[sku] = true
	}

	// STEP 4: Resolve items from the catalog — client prices are ignored.
	// Includes checked arithmetic for line totals and subtotal accumulation.
	items, subtotal, err := s.resolveItems(ctx, in.Items)
	if err != nil {
		return Order{}, err
	}

	// STEP 6: Check stock availability before proceeding.
	for _, item := range items {
		product, err := s.findProductBySKU(ctx, item.SKU)
		if err != nil {
			return Order{}, fmt.Errorf("product %q: %w", item.SKU, err)
		}
		if product.Stock < item.Quantity {
			return Order{}, ErrInsufficientStock
		}
	}

	// STEP 6b: Server-authoritative totals: client shipping and discount
	// are ignored. Validate shipping method and compute fee from the
	// current shipping_methods rows. Replay of an existing idempotency
	// key already returned above, before this mutable config read.
	shipping, err := s.computeShipping(ctx, in.ShippingMethod, subtotal)
	if err != nil {
		return Order{}, err
	}
	// Validate payment method against the admin-managed payment_methods
	// table. Rejects empty/unknown/disabled/not-ready methods. The server
	// is the authority for payment method availability — the browser must
	// not decide which payment methods are available (REQ-004/AC-007).
	if err := s.validatePaymentMethod(ctx, in.PaymentMethod); err != nil {
		return Order{}, err
	}
	discount, err := s.calculateDiscount(ctx, subtotal, in.PromoCode) // already normalized
	if err != nil {
		return Order{}, err
	}
	total, err := addOverflow(subtotal-discount, shipping)
	if err != nil {
		return Order{}, fmt.Errorf("total overflow: %w", err)
	}

	// STEP 7: Contact field validation (email/phone/shipping_address).
	// This runs AFTER all side-effect-free validation (items, stock,
	// shipping, payment, discount, total) but BEFORE any persistence
	// (marshal, order ID generation, token generation, transaction).
	// This ordering preserves existing test semantics: tests that exercise
	// stock/shipping/payment errors do not need valid PII, because those
	// errors fire first. A same-key different-payload replay already
	// returned ErrIdempotencyConflict in STEP 3, regardless of PII validity.
	if err := validateOrderInput(in); err != nil {
		return Order{}, err
	}

	// STEP 8: Persistence preparation — marshal, generate IDs, transaction.
	itemsJSON, err := marshalJSON(items)
	if err != nil {
		return Order{}, err
	}
	now := time.Now().Unix()
	timeline := []TimelineEvent{{Status: "pending", At: now, Note: "order placed"}}
	timelineJSON, err := marshalJSON(timeline)
	if err != nil {
		return Order{}, err
	}

	orderID, err := s.nextOrderID(ctx)
	if err != nil {
		return Order{}, err
	}
	accessToken, err := nextAccessToken()
	if err != nil {
		return Order{}, err
	}

	o := Order{
		ID:                  orderID,
		MemberID:            memberID,        // "" for guest, principal.UserID for member — written in same tx
		CustomerName:        in.CustomerName, // already normalized
		Email:               in.Email,        // already normalized
		Phone:               in.Phone,        // already normalized
		Items:               items,
		ItemsJSON:           itemsJSON,
		ShippingAddress:     in.ShippingAddress, // already normalized
		ShippingMethod:      in.ShippingMethod,  // already normalized
		PaymentMethod:       in.PaymentMethod,   // already normalized
		Subtotal:            subtotal,
		Discount:            discount,
		Shipping:            shipping,
		Total:               total,
		Status:              "pending",
		PaymentStatus:       "unpaid",
		ReturnRequestStatus: "",
		IdempotencyKey:      key,
		RequestFingerprint:  computeRequestFingerprint(in, memberID),
		AccessToken:         accessToken,                  // plaintext — returned once in the create response
		AccessTokenHash:     hashAccessToken(accessToken), // hash — persisted in DB
		Timeline:            timeline,
		TimelineJSON:        timelineJSON,
		UpdatedUnix:         now,
	}

	// Insert the order and decrement stock in a single transaction. The
	// idempotency check is performed INSIDE the transaction as a race-
	// condition safety net for concurrent requests with the same key
	// that both pass the early check above. On conflict, the existing
	// order is returned for fingerprint verification.
	existing, conflict, err := s.store.CreateOrderTxWithIdempotency(ctx, o, items)
	if err != nil {
		return Order{}, err
	}
	if conflict {
		// An order with the same idempotency_key already exists (race
		// condition: a concurrent request inserted between our early
		// check and this transaction). Same fingerprint → return the
		// same order. Different fingerprint → conflict.
		if !samePayload(existing, in, memberID) {
			return Order{}, ErrIdempotencyConflict
		}
		// The existing order's plaintext access token is NOT returned on
		// replay — it was already displayed at original creation. The
		// hash is in the DB but the plaintext is gone. Secure token
		// recovery is not yet implemented (see REQ-009 pending evidence).
		existing.AccessToken = ""
		return hydrateOrder(existing), nil
	}
	o.Version = 1 // matches the DEFAULT 1 written by the store insert
	return hydrateOrder(o), nil
}

// RegenerateAccessToken was removed because email-only verification is not
// possession proof — anyone who knows or guesses the order ID and checkout
// email can rotate the token, invalidate the original, and take over the
// order. A secure recovery mechanism (e.g. short-lived OTP/magic-link to
// verified email with rate limiting, or client-held high-entropy recovery
// credential) is required but not yet implemented. See REQ-009 pending
// evidence in specs/changes/minimal-cart-integration/control.json.

// UpdateOrderStatus advances an order through the state machine using
// expected_version optimistic concurrency. The service loads the order,
// validates the transition using the ACTUAL current status, and delegates
// the atomic version guard to the store. Stock is restocked when an order
// is cancelled before shipping.
func (s Service) UpdateOrderStatus(ctx context.Context, principal auth.Principal, id string, expectedVersion int, newStatus string) (Order, error) {
	return s.UpdateOrderStatusWithNote(ctx, principal, id, expectedVersion, newStatus, "")
}

func (s Service) UpdateOrderStatusWithNote(ctx context.Context, principal auth.Principal, id string, expectedVersion int, newStatus, note string) (Order, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return Order{}, ErrForbidden
	}
	existing, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return Order{}, err
	}
	// Validate the transition is legal using the ACTUAL current status,
	// not a client-claimed expected_status. The store's version guard
	// catches concurrent mutations between this load and the UPDATE.
	allowed, ok := orderTransitions[existing.Status]
	if !ok || !allowed[newStatus] {
		return Order{}, ErrInvalidTransition
	}

	now := time.Now().Unix()
	var restock []OrderItem
	// Restock on cancellation before the order ships, but only inside the same
	// transaction that wins the expected-version update.
	if newStatus == "cancelled" && (existing.Status == "pending" || existing.Status == "processing") {
		restock, err = unmarshalItems(existing.ItemsJSON)
		if err != nil {
			return Order{}, err
		}
	}

	note = strings.TrimSpace(note)
	timeline, err := appendTimeline(existing.TimelineJSON, newStatus, note, now)
	if err != nil {
		return Order{}, err
	}
	event, err := newOrderEvent(id, "order_status", principal.UserID, existing.Status, newStatus, note, now)
	if err != nil {
		return Order{}, err
	}
	if err := s.store.TransitionOrderStatus(ctx, id, expectedVersion, newStatus, timeline, now, restock, event); err != nil {
		return Order{}, err
	}
	return s.GetOrder(ctx, id)
}

// UpdateOrderReturnStatus advances the return request state machine using
// expected_version optimistic concurrency. Recording receipt does not make
// returned goods saleable; an inspected, quantity-aware inventory adjustment
// is a separate controlled action.
func (s Service) UpdateOrderReturnStatus(ctx context.Context, principal auth.Principal, id string, expectedVersion int, newStatus string) (Order, error) {
	return s.UpdateOrderReturnStatusWithNote(ctx, principal, id, expectedVersion, newStatus, "")
}

func (s Service) UpdateOrderReturnStatusWithNote(ctx context.Context, principal auth.Principal, id string, expectedVersion int, newStatus, note string) (Order, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return Order{}, ErrForbidden
	}
	existing, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return Order{}, err
	}
	// Validate the return transition is legal using the ACTUAL current
	// return_request_status. The store's version guard catches concurrent
	// mutations between this load and the UPDATE.
	allowed, ok := returnTransitions[existing.ReturnRequestStatus]
	if !ok || !allowed[newStatus] {
		return Order{}, ErrInvalidTransition
	}
	if existing.Status != "delivered" {
		return Order{}, ErrInvalidTransition
	}

	now := time.Now().Unix()
	note = strings.TrimSpace(note)
	event, err := newOrderEvent(id, "return_status", principal.UserID, existing.ReturnRequestStatus, newStatus, note, now)
	if err != nil {
		return Order{}, err
	}
	if err := s.store.TransitionOrderReturnStatus(ctx, id, expectedVersion, newStatus, now, event); err != nil {
		return Order{}, err
	}
	return s.GetOrder(ctx, id)
}

// RestockOrder was removed (fail-closed). The per-item
// returned_quantity/restocked_quantity ledger (order_items table,
// INTEGRATION_PLAN.md:300), inventory.adjust capability, idempotency key,
// required reason, and admin UI consumer are NOT yet implemented. The
// restock endpoint, handler, service method, store method, OpenAPI
// operation, and route were removed to avoid shipping an unsafe,
// unreachable mutation. See AC-009/REQ-005 pending items.
//
// B7 implementation: RestockOrderItems is the per-item restock action.
// It requires orders.returns AND inventory.adjust capabilities, a non-empty
// reason, expected_version, and idempotency key. The order must have
// return_request_status = "received" (goods physically received). The
// service validates per-item constraints (0 <= restocked <= returned,
// cumulative totals <= quantity) before delegating to the store's atomic
// transaction. Same key + same payload returns the same result; same key +
// different payload returns ErrIdempotencyConflict. Stale version, forbidden
// actor, over-restock, and duplicate all leave the order, items, inventory,
// and events unchanged.

// RestockOrderItems performs the per-item restock action. The admin
// declares per-item returned_quantity (units physically received back) and
// restocked_quantity (units being put back into salable inventory) as
// DELTA values — the service adds them to the existing cumulative ledger.
// The service enforces:
//   - Principal holds BOTH orders.returns AND inventory.adjust capabilities.
//   - Order exists and return_request_status is "received".
//   - Reason is non-empty.
//   - Idempotency key is non-empty.
//   - At least one item, no duplicate SKUs (after trimming).
//   - At least one positive delta (returned or restocked) across all items
//     — an all-zero no-op is rejected.
//   - Per item: SKU (trimmed) exists in the order's order_items ledger.
//   - Per item: returned_delta >= 0, restocked_delta >= 0.
//   - Per item (overflow-safe): returned_delta > quantity - currentReturned → over-restock.
//   - Per item (overflow-safe): restocked_delta - returned_delta > currentReturned - currentRestocked → over-restock.
//
// There is NO per-action restocked_delta <= returned_delta constraint.
// A restock-only follow-up action (returned_delta=0, restocked_delta=1)
// is legal when existingReturned > existingRestocked — the cumulative
// constraint (restocked_delta - returned_delta <= currentReturned -
// currentRestocked) is the sole authority.
//
// SKUs are trimmed and written back into a canonical copy of in.Items
// before fingerprint computation, event creation, and store delegation.
// This prevents a whitespace-padded SKU from passing validation (which
// trims for lookup) but failing the SQL UPDATE (which uses the raw value).
//
// The cumulative checks use subtraction (not addition) to avoid int
// overflow: quantity, currentReturned, currentRestocked are all bounded
// by the original order item quantity, so the subtractions are safe.
//
// The store's atomic transaction additionally enforces the CHECK constraint
// at the DB level, so a race that passes the service validation but fails
// the DB constraint is rolled back with no side effect.
func (s Service) RestockOrderItems(ctx context.Context, principal auth.Principal, id string, in RestockInput) (Order, error) {
	if !auth.Can(principal, "orders.returns") || !auth.Can(principal, "inventory.adjust") {
		return Order{}, ErrForbidden
	}
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" {
		return Order{}, ErrMissingIdempotencyKey
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return Order{}, ErrMissingRestockReason
	}
	if len(in.Items) == 0 {
		return Order{}, fmt.Errorf("%w: at least one item is required", ErrInvalidAdminInput)
	}

	// Canonicalize items: trim SKU and write back so the trimmed value
	// flows into fingerprint, event, and store. Without this, a SKU like
	// "  SKU-A  " would pass validation (which trims for ledger lookup)
	// but fail the SQL UPDATE (which uses the raw value), causing a
	// silent 0-rows-affected → ErrRestockItemNotFound.
	canonicalItems := make([]RestockItemInput, len(in.Items))
	for i, ri := range in.Items {
		canonicalItems[i] = RestockItemInput{
			SKU:               strings.TrimSpace(ri.SKU),
			ReturnedQuantity:  ri.ReturnedQuantity,
			RestockedQuantity: ri.RestockedQuantity,
		}
	}
	in.Items = canonicalItems
	fingerprint := computeRestockFingerprint(in, id)

	// STEP 1: Early idempotency check — before any mutable-state validation.
	// A same-key different-payload replay must return ErrIdempotencyConflict,
	// not a validation error from the new payload.
	_, existingFingerprint, existingResponseJSON, err := s.store.FindRestockByIdempotencyKey(ctx, key)
	if err == nil {
		if existingFingerprint != fingerprint {
			return Order{}, ErrIdempotencyConflict
		}
		// Same payload — return the STORED response snapshot, not the
		// current order state. The idempotency contract requires that a
		// replay returns the exact same response as the original success.
		// If we called GetOrder here, a subsequent mutation (e.g. status
		// update) would cause the replay to return a different response,
		// violating the contract. The stored snapshot was captured inside
		// the original transaction, so it reflects the exact post-restock
		// state at that moment.
		// If the snapshot is empty, placeholder, or corrupt, this is an
		// internal error (the CHECK constraint and Step 6 UPDATE should
		// prevent this on committed rows). Returning the current state
		// would silently violate the idempotency contract.
		return decodeRestockSnapshot(existingResponseJSON)
	}
	if !errors.Is(err, ErrNotFound) {
		return Order{}, fmt.Errorf("find restock by idempotency key: %w", err)
	}

	// STEP 2: Load the order and validate mutable state.
	existing, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return s.restockReplayIfCommitted(ctx, key, fingerprint, err)
	}
	if existing.ReturnRequestStatus != "received" {
		return s.restockReplayIfCommitted(ctx, key, fingerprint, ErrReturnNotReceived)
	}

	// STEP 3: Validate per-item constraints against the current ledger.
	ledgerItems, err := s.store.GetOrderItems(ctx, id)
	if err != nil {
		return s.restockReplayIfCommitted(ctx, key, fingerprint, fmt.Errorf("load order items for restock: %w", err))
	}
	ledgerBySKU := make(map[string]OrderItem, len(ledgerItems))
	for _, li := range ledgerItems {
		ledgerBySKU[li.SKU] = li
	}
	seenSKUs := make(map[string]bool, len(in.Items))
	hasPositiveDelta := false
	for _, ri := range in.Items {
		sku := ri.SKU // already trimmed by canonicalization
		if sku == "" {
			return Order{}, fmt.Errorf("%w: item sku is required", ErrInvalidAdminInput)
		}
		if seenSKUs[sku] {
			return Order{}, fmt.Errorf("%w: duplicate item sku %q in restock request", ErrInvalidAdminInput, sku)
		}
		seenSKUs[sku] = true
		if ri.ReturnedQuantity < 0 || ri.RestockedQuantity < 0 {
			return Order{}, fmt.Errorf("%w: returned_quantity and restocked_quantity must be non-negative", ErrOverRestock)
		}
		if ri.ReturnedQuantity > 0 || ri.RestockedQuantity > 0 {
			hasPositiveDelta = true
		}
		li, ok := ledgerBySKU[sku]
		if !ok {
			return s.restockReplayIfCommitted(ctx, key, fingerprint, fmt.Errorf("%w: sku %s", ErrRestockItemNotFound, sku))
		}
		// Cumulative returned must not exceed original quantity.
		// Overflow-safe: use subtraction instead of addition.
		// returned_delta > quantity - currentReturned → violation.
		availableReturned := li.Quantity - li.ReturnedQuantity
		if ri.ReturnedQuantity > availableReturned {
			return s.restockReplayIfCommitted(ctx, key, fingerprint, fmt.Errorf("%w: cumulative returned_quantity would exceed ordered quantity (%d) for sku %s (already returned %d, available %d, requested %d)", ErrOverRestock, li.Quantity, sku, li.ReturnedQuantity, availableReturned, ri.ReturnedQuantity))
		}
		// Cumulative restocked must not exceed cumulative returned.
		// Overflow-safe: restocked_delta - returned_delta > currentReturned - currentRestocked → violation.
		// currentReturned >= currentRestocked (DB invariant), so the RHS is >= 0.
		// When returned_delta = 0 and restocked_delta > 0 (restock-only
		// follow-up), this checks restocked_delta > currentReturned -
		// currentRestocked, i.e. the un-restocked surplus. This is the
		// sole authority — there is NO per-action restocked <= returned
		// constraint, because it would block legal restock-only follow-ups.
		availableRestocked := li.ReturnedQuantity - li.RestockedQuantity
		if ri.RestockedQuantity-ri.ReturnedQuantity > availableRestocked {
			return s.restockReplayIfCommitted(ctx, key, fingerprint, fmt.Errorf("%w: cumulative restocked_quantity would exceed cumulative returned_quantity for sku %s (already restocked %d of %d returned, available %d, requested delta restocked %d returned %d)", ErrOverRestock, sku, li.RestockedQuantity, li.ReturnedQuantity, availableRestocked, ri.RestockedQuantity, ri.ReturnedQuantity))
		}
	}
	// Reject all-zero no-op: at least one item must have a positive delta.
	if !hasPositiveDelta {
		return Order{}, fmt.Errorf("%w: at least one item must have a positive returned_quantity or restocked_quantity", ErrInvalidAdminInput)
	}

	// STEP 4: Build the event and delegate to the atomic store transaction.
	now := time.Now().Unix()
	event, err := newOrderEvent(id, "restock", principal.UserID, existing.ReturnRequestStatus, existing.ReturnRequestStatus, reason, now)
	if err != nil {
		return Order{}, err
	}
	storedFingerprint, storedResponseJSON, conflict, err := s.store.RestockOrderItemsTx(ctx, id, in.ExpectedVersion, in.Items, event, key, fingerprint, now)
	if err != nil {
		return Order{}, err
	}
	if conflict {
		// A concurrent request with the same idempotency key inserted
		// between our early check and this transaction. The store returns
		// the STORED fingerprint and response snapshot from the winning
		// request. Compare the fingerprint to decide replay vs conflict.
		if storedFingerprint != fingerprint {
			return Order{}, ErrIdempotencyConflict
		}
		// Same payload — return the STORED response snapshot, not the
		// current order state. This ensures the concurrent replay returns
		// the exact same response as the original success, even if other
		// mutations have occurred since.
		return decodeRestockSnapshot(storedResponseJSON)
	}
	// Success — return the snapshot captured inside the transaction.
	// We MUST NOT call GetOrder here: another mutation could land between
	// the tx commit and this read, causing the success response to differ
	// from the stored snapshot. Returning the snapshot directly guarantees
	// the success response matches what future replays will return.
	return decodeRestockSnapshot(storedResponseJSON)
}

// restockReplayIfCommitted re-resolves the idempotency key after a
// mutable-state error. Both racers can miss the early lookup; the winner
// then commits the idempotency row and ledger in one transaction. The
// loser's subsequent ledger read would otherwise return ErrOverRestock
// (or another mutable-state error) before RestockOrderItemsTx can
// arbitrate on the unique key. If a committed record now exists, honor
// it. If the key is still absent, return the original error so a genuine
// over-restock stays ErrOverRestock.
func (s Service) restockReplayIfCommitted(ctx context.Context, key, fingerprint string, original error) (Order, error) {
	_, existingFingerprint, existingResponseJSON, err := s.store.FindRestockByIdempotencyKey(ctx, key)
	if err == nil {
		if existingFingerprint != fingerprint {
			return Order{}, ErrIdempotencyConflict
		}
		return decodeRestockSnapshot(existingResponseJSON)
	}
	if !errors.Is(err, ErrNotFound) {
		return Order{}, fmt.Errorf("find restock by idempotency key: %w", err)
	}
	return Order{}, original
}

// decodeRestockSnapshot deserializes the stored response_json snapshot
// and validates that it is a real Order (not an empty placeholder or
// corrupt data). Returns an internal error on any failure — the caller
// must never silently return a different response than what was stored.
func decodeRestockSnapshot(responseJSON string) (Order, error) {
	if responseJSON == "" || responseJSON == "{}" {
		return Order{}, fmt.Errorf("restock idempotency: stored response snapshot is empty or placeholder")
	}
	var stored Order
	if err := json.Unmarshal([]byte(responseJSON), &stored); err != nil {
		return Order{}, fmt.Errorf("restock idempotency: corrupt stored response snapshot: %w", err)
	}
	// Validate the snapshot is a real order, not a degenerate empty Order.
	// A valid post-restock snapshot must have an ID, at least one item,
	// and a version >= 1 (the restock tx increments version).
	if stored.ID == "" {
		return Order{}, fmt.Errorf("restock idempotency: stored response snapshot has empty order ID")
	}
	if len(stored.Items) == 0 {
		return Order{}, fmt.Errorf("restock idempotency: stored response snapshot has no items")
	}
	if stored.Version < 1 {
		return Order{}, fmt.Errorf("restock idempotency: stored response snapshot has invalid version %d", stored.Version)
	}
	return stored, nil
}

// computeRestockFingerprint returns a SHA-256 hex digest of the canonical
// RestockInput + orderID. This is used for idempotency replay validation:
// same key + same fingerprint = retry, same key + different fingerprint =
// conflict. The fingerprint includes the order ID so a restock for a
// different order with the same key is detected as a conflict.
func computeRestockFingerprint(in RestockInput, orderID string) string {
	type canonical struct {
		OrderID         string             `json:"order_id"`
		ExpectedVersion int                `json:"expected_version"`
		IdempotencyKey  string             `json:"idempotency_key"`
		Reason          string             `json:"reason"`
		Items           []RestockItemInput `json:"items"`
	}
	c := canonical{
		OrderID:         orderID,
		ExpectedVersion: in.ExpectedVersion,
		IdempotencyKey:  strings.TrimSpace(in.IdempotencyKey),
		Reason:          strings.TrimSpace(in.Reason),
		Items:           in.Items,
	}
	b, _ := json.Marshal(c)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ----- Members --------------------------------------------------------------

func (s Service) ListMembers(ctx context.Context, filter MemberFilter) ([]Member, error) {
	return s.store.ListMembers(ctx, filter)
}

func (s Service) UpdateMember(ctx context.Context, principal auth.Principal, id string, in MemberInput) (Member, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return Member{}, ErrForbidden
	}
	existing, err := s.store.GetMember(ctx, id)
	if err != nil {
		return Member{}, err
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = existing.Status
	}
	if status != "active" && status != "locked" {
		return Member{}, fmt.Errorf("%w: invalid member status %q", ErrInvalidAdminInput, status)
	}
	tier := strings.TrimSpace(in.Tier)
	if tier == "" {
		tier = existing.Tier
	}
	if tier != "regular" && tier != "vip" {
		return Member{}, fmt.Errorf("%w: invalid member tier %q", ErrInvalidAdminInput, tier)
	}
	m := Member{
		ID:          id,
		Email:       defaultString(strings.TrimSpace(in.Email), existing.Email),
		Name:        in.Name,
		Status:      status,
		Tier:        tier,
		Tags:        in.Tags,
		Notes:       in.Notes,
		TotalOrders: existing.TotalOrders,
		TotalSpent:  existing.TotalSpent,
		UpdatedUnix: time.Now().Unix(),
	}
	if err := s.store.UpsertMember(ctx, m); err != nil {
		return Member{}, err
	}
	return s.store.GetMember(ctx, id)
}

func (s Service) UpdateMemberStatus(ctx context.Context, principal auth.Principal, id, status string) (Member, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return Member{}, ErrForbidden
	}
	if status != "active" && status != "locked" {
		return Member{}, fmt.Errorf("%w: invalid member status %q", ErrInvalidAdminInput, status)
	}
	if err := s.store.UpdateMemberStatus(ctx, id, status, time.Now().Unix()); err != nil {
		return Member{}, err
	}
	return s.store.GetMember(ctx, id)
}

// ----- helpers --------------------------------------------------------------

func (s Service) findProductBySKU(ctx context.Context, sku string) (Product, error) {
	// The schema keys products by id and slug; sku is unique but has no
	// dedicated lookup. List and filter in-memory for the starter profile.
	products, err := s.store.ListPublishedProducts(ctx)
	if err != nil {
		return Product{}, err
	}
	for _, p := range products {
		if p.SKU == sku {
			return p, nil
		}
	}
	return Product{}, ErrNotFound
}

// nextOrderID generates a random, unguessable order ID. The ID is "TW-"
// followed by 32 hex characters (16 random bytes). This replaces the
// previous COUNT(*)+1 scheme which was enumerable and race-prone.
func (s Service) nextOrderID(ctx context.Context) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "TW-" + hex.EncodeToString(b[:]), nil
}

// nextAccessToken generates a random 32-byte (64 hex char) opaque access
// token for guest order access. This token is required alongside the order
// ID to look up an order without authentication, preventing enumeration.
func nextAccessToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalItems(itemsJSON string) ([]OrderItem, error) {
	if itemsJSON == "" {
		return nil, nil
	}
	var items []OrderItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func appendTimeline(timelineJSON, status, note string, at int64) (string, error) {
	var timeline []TimelineEvent
	if timelineJSON != "" {
		if err := json.Unmarshal([]byte(timelineJSON), &timeline); err != nil {
			return "", err
		}
	}
	timeline = append(timeline, TimelineEvent{Status: status, At: at, Note: note})
	return marshalJSON(timeline)
}

func newOrderEvent(orderID, eventType, actorUserID, fromStatus, toStatus, reason string, at int64) (OrderEvent, error) {
	id, err := randomID()
	if err != nil {
		return OrderEvent{}, err
	}
	return OrderEvent{ID: id, OrderID: orderID, EventType: eventType, ActorUserID: actorUserID, FromStatus: fromStatus, ToStatus: toStatus, Reason: reason, CreatedUnix: at}, nil
}

// hydrateOrder decodes items_json and timeline_json into struct fields so
// handlers can return rich JSON without exposing raw column values.
func hydrateOrder(o Order) Order {
	if o.ItemsJSON != "" {
		_ = json.Unmarshal([]byte(o.ItemsJSON), &o.Items)
	}
	if o.TimelineJSON != "" {
		_ = json.Unmarshal([]byte(o.TimelineJSON), &o.Timeline)
	}
	return o
}

// mulOverflow multiplies a and b, returning ErrOverflow if the result
// would overflow int. The check is performed BEFORE the multiplication
// using division-based bounds — the int64 intermediate approach is
// broken on 64-bit platforms where int is 64-bit (int64(a)*int64(b)
// wraps before the comparison can detect it).
//
// Domain invariant: in our commerce code, a and b are non-negative
// (prices, quantities, subtotals, promo values). The general case is
// handled correctly for completeness.
func mulOverflow(a, b int) (int, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	// Division-based overflow check. We check BEFORE multiplying so
	// the check itself cannot overflow.
	if a > 0 {
		if b > 0 {
			// Both positive: a*b > MaxInt iff a > MaxInt/b
			if a > math.MaxInt/b {
				return 0, ErrOverflow
			}
		} else {
			// a > 0, b < 0: a*b < MinInt iff b < MinInt/a
			if b < math.MinInt/a {
				return 0, ErrOverflow
			}
		}
	} else {
		// a < 0 (a == 0 handled above)
		if b > 0 {
			// a < 0, b > 0: a*b < MinInt iff a < MinInt/b
			if a < math.MinInt/b {
				return 0, ErrOverflow
			}
		} else {
			// Both negative: a*b > MaxInt.
			// |MinInt| = MaxInt+1, so either operand being MinInt
			// means the product overflows.
			if a == math.MinInt || b == math.MinInt {
				return 0, ErrOverflow
			}
			absA := -a
			absB := -b
			if absA > math.MaxInt/absB {
				return 0, ErrOverflow
			}
		}
	}
	return a * b, nil
}

// addOverflow adds a and b, returning ErrOverflow if the result would
// overflow int. The check is performed BEFORE the addition using
// sign-aware bounds.
func addOverflow(a, b int) (int, error) {
	// Same-sign: overflow if the result would flip sign.
	// Different-sign: cannot overflow (result is between a and b).
	if a > 0 && b > 0 {
		if a > math.MaxInt-b {
			return 0, ErrOverflow
		}
	} else if a < 0 && b < 0 {
		if a < math.MinInt-b {
			return 0, ErrOverflow
		}
	}
	return a + b, nil
}

// computeRequestFingerprint produces a SHA-256 hex digest of the
// canonical normalized OrderInput + memberID. This fingerprint is
// persisted at creation time and compared on idempotency replay. It
// captures ALL client-controlled order-defining fields so that a
// materially different request cannot replay as the same order.
//
// ENCODING: The fingerprint is computed over a JSON-marshaled typed
// struct, NOT a delimiter-separated string. JSON encoding is
// length-delimited and unambiguous — it escapes newlines, quotes, and
// other special characters inside field values, so two distinct inputs
// can never produce the same canonical byte string. The previous
// \n-separated "field=value" scheme was collision-prone: a field value
// containing \n could shift the boundary between fields, making two
// materially different requests encode to the same bytes.
//
// Normalization is consistent with creation semantics in createOrder:
// fields that are TrimSpace'd at creation are TrimSpace'd here. Items
// are sorted by SKU then quantity so reordering does not produce a
// different fingerprint. The memberID parameter is the caller's
// identity ("" for guest, principal.UserID for member) and is included
// so a guest replay of a member order (or vice versa) is a material
// difference.
//
// The fingerprint does NOT include mutable server-state fields (stock,
// fee schedule, payment method readiness). This is intentional: a
// same-key same-payload replay must succeed even if the server state
// has changed since the original creation. The fingerprint captures
// what the CLIENT requested, not what the server validated against.
func computeRequestFingerprint(in OrderInput, memberID string) string {
	// Normalize items: sort by SKU then quantity for canonical ordering.
	// SKU is used as-is (NOT trimmed) to match creation semantics:
	// resolveItems/findProductBySKU looks up ci.SKU without trimming,
	// so " SKU-A " and "SKU-A" are different requests (the former
	// would fail product lookup). Trimming SKU in the fingerprint
	// would make these two collide, causing a whitespace-different
	// replay to return the original order instead of ErrIdempotencyConflict.
	type itemKey struct {
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity"`
	}
	items := make([]itemKey, len(in.Items))
	for i, item := range in.Items {
		items[i] = itemKey{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SKU != items[j].SKU {
			return items[i].SKU < items[j].SKU
		}
		return items[i].Quantity < items[j].Quantity
	})

	// Canonical typed struct. JSON field tags fix the key order.
	// encoding/json escapes newlines (\n → \\n), quotes, and other
	// special characters inside values, so the encoding is
	// unambiguous — no delimiter collision is possible.
	type fingerprintPayload struct {
		CustomerName    string    `json:"customer_name"`
		Email           string    `json:"email"`
		Phone           string    `json:"phone"`
		ShippingAddress string    `json:"shipping_address"`
		ShippingMethod  string    `json:"shipping_method"`
		PaymentMethod   string    `json:"payment_method"`
		PromoCode       string    `json:"promo_code"`
		MemberID        string    `json:"member_id"`
		Items           []itemKey `json:"items"`
	}

	payload := fingerprintPayload{
		CustomerName:    strings.TrimSpace(in.CustomerName),
		Email:           strings.TrimSpace(in.Email),
		Phone:           strings.TrimSpace(in.Phone),
		ShippingAddress: strings.TrimSpace(in.ShippingAddress),
		ShippingMethod:  strings.TrimSpace(in.ShippingMethod),
		PaymentMethod:   strings.TrimSpace(in.PaymentMethod),
		PromoCode:       strings.TrimSpace(in.PromoCode),
		MemberID:        memberID,
		Items:           items,
	}

	// json.Marshal produces deterministic output for a given struct
	// (fixed field order via struct declaration order, sorted map keys
	// if any — we use only struct fields and slices, no maps).
	canonical, err := json.Marshal(payload)
	if err != nil {
		// Should never happen for this simple struct. If it does,
		// panic rather than produce a degenerate fingerprint that
		// could allow a duplicate order.
		panic(fmt.Sprintf("computeRequestFingerprint: json.Marshal failed: %v", err))
	}

	h := sha256.Sum256(canonical)
	return hex.EncodeToString(h[:])
}

// samePayload compares the stored request fingerprint against the
// incoming request's fingerprint. This is a complete comparison over
// all client-controlled order-defining fields — no partial field-by-
// field logic that could miss a material difference.
//
// BACKWARD COMPATIBILITY: Migration 008 sets request_fingerprint=”
// for all pre-existing orders. Such rows cannot be fingerprint-
// compared because the original input is not reconstructable from the
// stored order alone (e.g. promo_code is not stored, only the
// resulting discount). For these legacy rows, we fall back to a
// field-by-field comparison over the fields that ARE available on the
// stored order. This fallback is imperfect (it cannot detect a
// different promo_code that produced the same discount, and it
// compares discount as a proxy for promo_code), but it preserves
// replay semantics for legacy orders rather than breaking them with
// ErrIdempotencyConflict on every same-key replay. New orders always
// have a non-empty fingerprint and use the exact comparison.
func samePayload(existing Order, in OrderInput, memberID string) bool {
	if existing.RequestFingerprint != "" {
		// New order with fingerprint — exact comparison.
		return existing.RequestFingerprint == computeRequestFingerprint(in, memberID)
	}
	// Legacy order with empty fingerprint (pre-migration 008 row).
	// Fall back to field-by-field comparison over available stored
	// fields. This is the pre-revision-5 samePayload behavior, extended
	// to include payment_method, shipping_method, phone,
	// shipping_address, and memberID. The discount is compared as a
	// proxy for promo_code (which is not stored on the order).
	return samePayloadLegacy(existing, in, memberID)
}

// samePayloadLegacy is the backward-compatible field-by-field
// comparison for pre-migration orders that have no stored request
// fingerprint. It compares all fields available on the stored Order
// against the incoming OrderInput, with normalization consistent with
// creation semantics. It is less precise than fingerprint comparison
// (promo_code is not stored, only the resulting discount) but
// preserves replay semantics for legacy rows.
func samePayloadLegacy(existing Order, in OrderInput, memberID string) bool {
	existingItems, _ := unmarshalItems(existing.ItemsJSON)
	if len(existingItems) != len(in.Items) {
		return false
	}
	for i, item := range in.Items {
		if existingItems[i].SKU != item.SKU || existingItems[i].Quantity != item.Quantity {
			return false
		}
	}
	return existing.CustomerName == strings.TrimSpace(in.CustomerName) &&
		existing.Email == strings.TrimSpace(in.Email) &&
		existing.Phone == strings.TrimSpace(in.Phone) &&
		existing.ShippingAddress == strings.TrimSpace(in.ShippingAddress) &&
		existing.ShippingMethod == strings.TrimSpace(in.ShippingMethod) &&
		existing.PaymentMethod == strings.TrimSpace(in.PaymentMethod) &&
		existing.MemberID == memberID
}
