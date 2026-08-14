package commerce

import (
	"errors"
	"net/http"
	"strings"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/httpx"
)

// Handler exposes the commerce module over HTTP. Admin endpoints require an
// authenticated principal with the relevant capability; public endpoints are
// open. auth.Principal is always resolved at the handler boundary and passed
// explicitly into the service — never via context.Context.
type Handler struct {
	service Service
	auth    auth.Authenticator
}

// NewHandler constructs a Handler.
func NewHandler(service Service, authenticator auth.Authenticator) Handler {
	return Handler{service: service, auth: authenticator}
}

// adminProductImage is the admin-facing product image DTO. It exposes
// the raw verified object key so the admin form can edit associations
// without treating public URLs as authority. This is ONLY output by
// authorized admin endpoints, never by public catalog endpoints.
type adminProductImage struct {
	ID          string `json:"id"`
	ObjectKey   string `json:"object_key"`
	AltText     string `json:"alt_text"`
	SortOrder   int    `json:"sort_order"`
	CreatedUnix int64  `json:"created_unix"`
}

// adminProductResponse is the admin-facing product DTO. It exposes
// raw product_images with object keys for editing. The legacy
// image/images fields are NOT included — admin uses only
// product_images. This is ONLY output by authorized admin endpoints.
type adminProductResponse struct {
	ID              string              `json:"id"`
	SKU             string              `json:"sku"`
	Name            string              `json:"name"`
	Slug            string              `json:"slug"`
	Description     string              `json:"description"`
	LongDescription string              `json:"long_description"`
	Category        string              `json:"category"`
	Status          string              `json:"status"`
	Material        string              `json:"material"`
	Origin          string              `json:"origin"`
	Price           int                 `json:"price"`
	OriginalPrice   int                 `json:"original_price"`
	Stock           int                 `json:"stock"`
	Tag             string              `json:"tag"`
	Rating          float64             `json:"rating"`
	ReviewsCount    int                 `json:"reviews_count"`
	UpdatedUnix     int64               `json:"updated_unix"`
	ProductImages   []adminProductImage `json:"product_images"`
}

// toAdminProductResponse builds an adminProductResponse from a Product.
// The Product's ProductImages (json:"-") is never serialized directly;
// this DTO explicitly exposes the raw object keys for admin editing.
// The legacy image/images fields are NOT included — admin uses only
// product_images.
func toAdminProductResponse(p Product) adminProductResponse {
	imgs := make([]adminProductImage, 0, len(p.ProductImages))
	for _, img := range p.ProductImages {
		imgs = append(imgs, adminProductImage{
			ID:          img.ID,
			ObjectKey:   img.ObjectKey,
			AltText:     img.AltText,
			SortOrder:   img.SortOrder,
			CreatedUnix: img.CreatedUnix,
		})
	}
	return adminProductResponse{
		ID:              p.ID,
		SKU:             p.SKU,
		Name:            p.Name,
		Slug:            p.Slug,
		Description:     p.Description,
		LongDescription: p.LongDescription,
		Category:        p.Category,
		Status:          p.Status,
		Material:        p.Material,
		Origin:          p.Origin,
		Price:           p.Price,
		OriginalPrice:   p.OriginalPrice,
		Stock:           p.Stock,
		Tag:             p.Tag,
		Rating:          p.Rating,
		ReviewsCount:    p.ReviewsCount,
		UpdatedUnix:     p.UpdatedUnix,
		ProductImages:   imgs,
	}
}

// toAdminProductListResponse builds a list of adminProductResponse.
func toAdminProductListResponse(products []Product) []adminProductResponse {
	out := make([]adminProductResponse, 0, len(products))
	for _, p := range products {
		out = append(out, toAdminProductResponse(p))
	}
	return out
}

// ----- Public endpoints (no auth) -------------------------------------------

func (h Handler) ListPublishedProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.ListPublishedProducts(r.Context())
	if err != nil {
		if errors.Is(err, ErrMediaURLUnavailable) {
			httpx.Error(w, http.StatusServiceUnavailable, "media public URL is not configured")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"products": products})
}

func (h Handler) GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		httpx.Error(w, http.StatusBadRequest, "slug is required")
		return
	}
	product, err := h.service.GetProductBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, ErrMediaURLUnavailable) {
			httpx.Error(w, http.StatusServiceUnavailable, "media public URL is not configured")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to get product")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"product": product})
}

// ListPublicShippingMethods returns enabled admin-managed shipping methods.
// Empty configuration is an empty array. Store failure fails closed.
func (h Handler) ListPublicShippingMethods(w http.ResponseWriter, r *http.Request) {
	methods, err := h.service.ListPublicShippingMethods(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list shipping methods")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"shipping_methods": methods})
}

// ListPublicPaymentMethods returns the admin-managed payment methods that
// are enabled and ready for customer use. The browser must use this
// endpoint, not hardcoded constants, to determine payment availability.
func (h Handler) ListPublicPaymentMethods(w http.ResponseWriter, r *http.Request) {
	methods, err := h.service.ListPublicPaymentMethods(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list payment methods")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"payment_methods": methods})
}

func (h Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var in OrderInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.service.CreateOrder(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, ErrInsufficientStock):
			httpx.Error(w, http.StatusConflict, "insufficient stock")
		case errors.Is(err, ErrIdempotencyConflict):
			httpx.Error(w, http.StatusConflict, "idempotency key reused with different payload")
		case errors.Is(err, ErrInvalidShippingMethod):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrShippingConfigUnavailable):
			httpx.Error(w, http.StatusServiceUnavailable, ErrShippingConfigUnavailable.Error())
		case errors.Is(err, ErrInvalidPaymentMethod):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInvalidPromoCode):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrPromoValidationUnavailable):
			httpx.Error(w, http.StatusServiceUnavailable, ErrPromoValidationUnavailable.Error())
		case errors.Is(err, ErrMissingIdempotencyKey):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		default:
			httpx.Error(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"order": order})
}

// CreateOrderForMember is the authenticated member checkout endpoint.
// It is identical to CreateOrder except the MemberID is derived from the
// authenticated principal (not from the client payload). The client
// cannot set MemberID — it is always the principal's UserID.
func (h Handler) CreateOrderForMember(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if principal.UserID == "" {
		httpx.Error(w, http.StatusForbidden, "member identity required")
		return
	}
	var in OrderInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.service.CreateOrderForMember(r.Context(), principal, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrInsufficientStock):
			httpx.Error(w, http.StatusConflict, "insufficient stock")
		case errors.Is(err, ErrIdempotencyConflict):
			httpx.Error(w, http.StatusConflict, "idempotency key reused with different payload")
		case errors.Is(err, ErrInvalidShippingMethod):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrShippingConfigUnavailable):
			httpx.Error(w, http.StatusServiceUnavailable, ErrShippingConfigUnavailable.Error())
		case errors.Is(err, ErrInvalidPaymentMethod):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInvalidPromoCode):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrPromoValidationUnavailable):
			httpx.Error(w, http.StatusServiceUnavailable, ErrPromoValidationUnavailable.Error())
		case errors.Is(err, ErrMissingIdempotencyKey):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrForbidden):
			httpx.Error(w, http.StatusForbidden, "forbidden")
		default:
			httpx.Error(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	httpx.JSON(w, http.StatusCreated, order)
}

// GetOrderForGuest is the public guest order lookup endpoint. It requires
// the order ID in the path and an opaque access token in the
// X-Order-Access-Token header (not the query string, to avoid logging
// the token in URLs, browser history, and Referer headers). If the token
// does not match, 404 is returned (not 403, to avoid leaking order
// existence). PII fields (phone, email, shipping address) are masked.
// The access token is never included in the response.
func (h Handler) GetOrderForGuest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "order id is required")
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-Order-Access-Token"))
	if token == "" {
		httpx.Error(w, http.StatusBadRequest, "X-Order-Access-Token header is required")
		return
	}
	order, err := h.service.GetOrderForGuest(r.Context(), id, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "order not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to get order")
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

// ListMyOrders is the authenticated member order listing endpoint. It
// requires a Bearer token; the principal's UserID scopes the query to
// only the member's own orders. PII fields are masked.
func (h Handler) ListMyOrders(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if principal.UserID == "" {
		httpx.Error(w, http.StatusForbidden, "member identity required")
		return
	}
	orders, err := h.service.ListOrdersForMember(r.Context(), principal.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list orders")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": orders})
}

// GetMyOrder is the authenticated member single-order endpoint. It
// requires a Bearer token; the principal's UserID must match the order's
// MemberID. Cross-user access returns 404 (not 403, to avoid leaking
// existence). PII fields are masked.
func (h Handler) GetMyOrder(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if principal.UserID == "" {
		httpx.Error(w, http.StatusForbidden, "member identity required")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "order id is required")
		return
	}
	order, err := h.service.GetOrderForMember(r.Context(), principal, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "order not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to get order")
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

// Quote returns server-authoritative totals for a cart. The browser must
// call this before showing the final checkout total and must display these
// totals, not client-computed ones. Client-supplied prices, shipping, and
// discount are ignored by the service.
func (h Handler) Quote(w http.ResponseWriter, r *http.Request) {
	var in QuoteInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.Quote(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidShippingMethod):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrShippingConfigUnavailable):
			httpx.Error(w, http.StatusServiceUnavailable, ErrShippingConfigUnavailable.Error())
		case errors.Is(err, ErrInvalidPaymentMethod):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInvalidPromoCode):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrPromoValidationUnavailable):
			httpx.Error(w, http.StatusServiceUnavailable, ErrPromoValidationUnavailable.Error())
		default:
			httpx.Error(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

// ----- Admin: products ------------------------------------------------------

func (h Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	filter := ProductFilter{
		Status:   r.URL.Query().Get("status"),
		Category: r.URL.Query().Get("category"),
	}
	products, err := h.service.ListProducts(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"products": toAdminProductListResponse(products)})
}

func (h Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	product, err := h.service.GetProduct(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "product not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to get product")
		return
	}
	httpx.JSON(w, http.StatusOK, toAdminProductResponse(product))
}

func (h Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in ProductInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	product, err := h.service.CreateProduct(r.Context(), principal, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toAdminProductResponse(product))
}

func (h Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in ProductInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	product, err := h.service.UpdateProduct(r.Context(), principal, r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toAdminProductResponse(product))
}

func (h Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.service.DeleteProduct(r.Context(), principal, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id")})
}

func (h Handler) UpdateProductStatus(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	product, err := h.service.UpdateProductStatus(r.Context(), principal, r.PathValue("id"), body.Status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toAdminProductResponse(product))
}

func (h Handler) BulkUpdateProducts(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		IDs     []string       `json:"ids"`
		Payload map[string]any `json:"payload"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.BulkUpdateProducts(r.Context(), principal, body.IDs, body.Payload); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": len(body.IDs)})
}

// ----- Admin: orders --------------------------------------------------------

func (h Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	filter := OrderFilter{
		Status:        r.URL.Query().Get("status"),
		PaymentStatus: r.URL.Query().Get("payment_status"),
	}
	orders, err := h.service.ListOrders(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list orders")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	order, err := h.service.GetOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "order not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to get order")
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

func (h Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		ExpectedVersion int    `json:"expected_version"`
		NewStatus       string `json:"new_status"`
		Note            string `json:"note"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.service.UpdateOrderStatusWithNote(r.Context(), principal, r.PathValue("id"), body.ExpectedVersion, body.NewStatus, body.Note)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

func (h Handler) UpdateOrderReturnStatus(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		ExpectedVersion int    `json:"expected_version"`
		NewStatus       string `json:"new_status"`
		Note            string `json:"note"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.service.UpdateOrderReturnStatusWithNote(r.Context(), principal, r.PathValue("id"), body.ExpectedVersion, body.NewStatus, body.Note)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

func (h Handler) RestockOrderItems(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body RestockInput
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.service.RestockOrderItems(r.Context(), principal, r.PathValue("id"), body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

// ----- Admin: members -------------------------------------------------------

func (h Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	filter := MemberFilter{
		Status: r.URL.Query().Get("status"),
		Tier:   r.URL.Query().Get("tier"),
	}
	members, err := h.service.ListMembers(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"members": members})
}

func (h Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in MemberInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	member, err := h.service.UpdateMember(r.Context(), principal, r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, member)
}

func (h Handler) UpdateMemberStatus(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	member, err := h.service.UpdateMemberStatus(r.Context(), principal, r.PathValue("id"), body.Status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, member)
}

// ----- Admin: promos --------------------------------------------------------

func (h Handler) ListPromos(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	promos, err := h.service.ListPromos(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list promos")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"promos": promos})
}

func (h Handler) CreatePromo(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in PromoInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	promo, err := h.service.CreatePromo(r.Context(), principal, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, promo)
}

func (h Handler) UpdatePromo(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in PromoInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	promo, err := h.service.UpdatePromo(r.Context(), principal, r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, promo)
}

func (h Handler) DeletePromo(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.service.DeletePromo(r.Context(), principal, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id")})
}

// ----- Admin: payment methods ----------------------------------------------

func (h Handler) ListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	methods, err := h.service.ListPaymentMethods(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list payment methods")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"payment_methods": methods})
}

func (h Handler) UpdatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in PaymentMethodInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	pm, err := h.service.UpdatePaymentMethod(r.Context(), principal, r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, pm)
}

// ----- Admin: shipping methods ---------------------------------------------

func (h Handler) ListShippingMethods(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	methods, err := h.service.ListShippingMethods(r.Context(), principal)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"shipping_methods": methods})
}

func (h Handler) CreateShippingMethod(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in ShippingMethodInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.service.CreateShippingMethod(r.Context(), principal, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, m)
}

func (h Handler) UpdateShippingMethod(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in ShippingMethodUpdateInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.service.UpdateShippingMethod(r.Context(), principal, r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

// ----- error mapping --------------------------------------------------------

// writeServiceError maps well-known service errors to HTTP status codes.
// Unknown errors default to 500 (not 400) to avoid leaking internal
// state as a client error. Only explicitly mapped sentinel errors
// get 4xx status codes.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not found")
	case errors.Is(err, ErrInvalidTransition):
		httpx.Error(w, http.StatusConflict, "invalid status transition")
	case errors.Is(err, ErrStaleVersion):
		httpx.Error(w, http.StatusConflict, "stale version: order was modified by another request")
	case errors.Is(err, ErrStaleShippingMethod):
		httpx.Error(w, http.StatusConflict, ErrStaleShippingMethod.Error())
	case errors.Is(err, ErrDuplicateShippingMethod):
		httpx.Error(w, http.StatusConflict, ErrDuplicateShippingMethod.Error())
	case errors.Is(err, ErrInsufficientStock):
		httpx.Error(w, http.StatusConflict, "insufficient stock")
	case errors.Is(err, ErrIdempotencyConflict):
		httpx.Error(w, http.StatusConflict, "idempotency key reused with different payload")
	case errors.Is(err, ErrUnverifiedMedia):
		httpx.Error(w, http.StatusBadRequest, ErrUnverifiedMedia.Error())
	case errors.Is(err, ErrInvalidProductInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrInvalidAdminInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrMediaURLUnavailable):
		httpx.Error(w, http.StatusServiceUnavailable, "media public URL is not configured")
	case errors.Is(err, ErrMissingRestockReason):
		httpx.Error(w, http.StatusBadRequest, ErrMissingRestockReason.Error())
	case errors.Is(err, ErrOverRestock):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrRestockItemNotFound):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrReturnNotReceived):
		httpx.Error(w, http.StatusConflict, ErrReturnNotReceived.Error())
	default:
		// Unknown errors are internal failures (DB, provider, etc.).
		// Return 500 without exposing the internal error message.
		httpx.Error(w, http.StatusInternalServerError, "internal error")
	}
}
