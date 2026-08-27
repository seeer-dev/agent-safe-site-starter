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
