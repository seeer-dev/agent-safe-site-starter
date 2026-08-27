package commerce

import (
	"context"
	"encoding/json"
)

// Product is a sellable item in the commerce catalog. Prices and stock are
// stored as integers because the storefront prices everything in TWD.
type Product struct {
	ID              string  `json:"id"`
	SKU             string  `json:"sku"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	Description     string  `json:"description"`
	LongDescription string  `json:"long_description"`
	Image           string  `json:"image"`
	Images          string  `json:"-"` // JSON array persisted as TEXT; API marshals it as string[].
	Category        string  `json:"category"`
	Status          string  `json:"status"` // draft|active|low_stock|out_of_stock
	Material        string  `json:"material"`
	Origin          string  `json:"origin"`
	Price           int     `json:"price"`
	OriginalPrice   int     `json:"original_price"`
	Stock           int     `json:"stock"`
	Tag             string  `json:"tag"`
	Rating          float64 `json:"rating"`
	ReviewsCount    int     `json:"reviews_count"`
	UpdatedUnix     int64   `json:"updated_unix"`
	// ProductImages is loaded from the product_images table (not the
	// products table). For public responses, the service derives
	// image/images URLs from this field. For admin responses, the
	// raw object keys and alt_text are exposed so the form can edit
	// them without treating public URLs as authority.
	// ProductImages is loaded from the product_images table (not the
	// products table). It is NEVER serialized in public responses
	// (json:"-") to avoid leaking verified object keys. For public
	// responses, the service derives image/images URL fields from
	// these keys. For admin responses, the handler builds an explicit
	// AdminProduct DTO that includes the raw keys for editing.
	ProductImages []ProductImage `json:"-"`
}

// MarshalJSON keeps the database representation private and exposes images as
// a real JSON array at the API boundary. If ProductImages is populated,
// the image/images fields are derived from the verified object keys
// joined with the public base URL.
func (p Product) MarshalJSON() ([]byte, error) {
	type productAlias Product
	images := []string{}
	if p.Images != "" {
		if err := json.Unmarshal([]byte(p.Images), &images); err != nil {
			return nil, err
		}
	}
	return json.Marshal(struct {
		productAlias
		Images []string `json:"images"`
	}{productAlias: productAlias(p), Images: images})
}

// ProductImage is a verified image associated with a product. The
// ObjectKey points to a verified object in R2 (under
// verified/product-images/{userID}/{sha256}.{ext}). The public URL
// is derived at render time as R2PublicBaseURL + "/" + ObjectKey.
// Commerce owns this table; the media module owns the verification
// registry. Commerce validates ObjectKey against the media registry
// via the MediaVerifier interface (wired in bootstrap).
type ProductImage struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ObjectKey   string `json:"object_key"`
	AltText     string `json:"alt_text"`
	SortOrder   int    `json:"sort_order"`
	CreatedUnix int64  `json:"created_unix"`
}

// ProductImageInput is the browser-supplied payload for product image
// association. The Key must be a verified object key returned by the
// media verify endpoint. AltText defaults to the product name if
// empty. SortOrder is derived from the array order.
type ProductImageInput struct {
	Key     string `json:"key"`
	AltText string `json:"alt_text"`
}

// MediaVerifier is the interface commerce uses to validate that a
// product image reference points to a real verified media object
// owned by the acting principal. The interface is defined in the
// commerce package (consumer) and implemented by a bootstrap adapter
// that wraps the media registry store. This preserves module
// isolation: commerce never imports media.
type MediaVerifier interface {
	// VerifyKey validates that the given object key is a verified
	// media object owned by the given userID. Returns nil if valid,
	// ErrUnverifiedMedia if the key is not in the registry or is
	// owned by a different user.
	VerifyKey(ctx context.Context, userID, objectKey string) error
}

// ProductInput is the browser-supplied payload for create/update operations.
// Browser-supplied prices and status are untrusted; the service re-derives
// status from stock and ignores client totals where relevant.
//
// Image/Images string fields are intentionally ABSENT. The legacy
// products.image/images columns are NOT an authority — public URLs are
// derived from verified product_images keys at response time. Since
// httpx.DecodeJSON uses DisallowUnknownFields, any payload containing
// image or images fields is rejected with 400. This prevents arbitrary
// URL injection through the product create/update API.
type ProductInput struct {
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
	ProductImages   []ProductImageInput `json:"product_images"`
}

// ProductFilter narrows product listings by status and/or category.
type ProductFilter struct {
	Status   string
	Category string
}

// Member is a registered customer record.
type Member struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Status      string `json:"status"` // active|locked
	Tier        string `json:"tier"`   // regular|vip
	Tags        string `json:"tags"`
	Notes       string `json:"notes"`
	TotalOrders int    `json:"total_orders"`
	TotalSpent  int    `json:"total_spent"`
	UpdatedUnix int64  `json:"updated_unix"`
}

// MemberInput is the browser-supplied payload for member updates.
type MemberInput struct {
	Email  string `json:"email"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Tier   string `json:"tier"`
	Tags   string `json:"tags"`
	Notes  string `json:"notes"`
}

// MemberFilter narrows member listings by status and/or tier.
type MemberFilter struct {
	Status string
	Tier   string
}

// OrderItem is a single line item embedded in an order's items_json and
// persisted in the order_items table. ReturnedQuantity and RestockedQuantity
// are persisted only in order_items (not in items_json); they default to 0
// and are omitted from JSON when 0 so existing payloads stay backward-compatible.
type OrderItem struct {
	SKU               string `json:"sku"`
	Name              string `json:"name"`
	Price             int    `json:"price"`
	Quantity          int    `json:"quantity"`
	ReturnedQuantity  int    `json:"returned_quantity,omitempty"`
	RestockedQuantity int    `json:"restocked_quantity,omitempty"`
}

// TimelineEvent records a status transition appended to an order's timeline.
type TimelineEvent struct {
	Status string `json:"status"`
	At     int64  `json:"at"`
	Note   string `json:"note,omitempty"`
}

// OrderEvent is the append-only audit record written in the same transaction
// as an order state mutation.
type OrderEvent struct {
	ID          string `json:"id"`
	OrderID     string `json:"order_id"`
	EventType   string `json:"event_type"`
	Sequence    int    `json:"sequence"`
	ActorUserID string `json:"actor_user_id"`
	FromStatus  string `json:"from_status"`
	ToStatus    string `json:"to_status"`
	Reason      string `json:"reason"`
	CreatedUnix int64  `json:"created_unix"`
}

// Order is a customer order. Totals are server-calculated from product prices;
// client-supplied totals are never trusted.
type Order struct {
	ID                  string      `json:"id"`
	MemberID            string      `json:"member_id"`
	CustomerName        string      `json:"customer_name"`
	Email               string      `json:"email"`
	Phone               string      `json:"phone"`
	Items               []OrderItem `json:"items"`
	ItemsJSON           string      `json:"-"` // raw column value
	ShippingAddress     string      `json:"shipping_address"`
	ShippingMethod      string      `json:"shipping_method"`
	PaymentMethod       string      `json:"payment_method"`
	TrackingNumber      string      `json:"tracking_number"`
	Subtotal            int         `json:"subtotal"`
	Discount            int         `json:"discount"`
	Shipping            int         `json:"shipping"`
	Total               int         `json:"total"`
	Status              string      `json:"status"` // pending|processing|shipped|delivered|cancelled
	PaymentStatus       string      `json:"payment_status"`
	ReturnRequestStatus string      `json:"return_request_status"`
	PaymentIntentID     string      `json:"payment_intent_id"`
	IdempotencyKey      string      `json:"idempotency_key,omitempty"`
	// AccessToken is the plaintext access token. It is ONLY set in the
	// create-order response (one-time display to the customer). It is
	// never populated from the DB (the DB stores a hash) and is always
	// cleared by maskCustomerPII for all other responses.
	AccessToken string `json:"access_token,omitempty"`
	// AccessTokenHash is the SHA-256 hash stored in the DB. It is never
	// serialized to JSON (json:"-"). Used for lookup via
	// GetOrderByAccessToken which hashes the incoming plaintext token
	// and compares.
	AccessTokenHash string `json:"-"`
	// RequestFingerprint is a SHA-256 hash of the canonical normalized
	// OrderInput + memberID, stored at creation time. It is used for
	// idempotency replay validation: an early lookup compares the incoming
	// request's fingerprint against the stored one, so a replay succeeds
	// even if mutable state (stock, payment method config, fee schedule)
	// has changed since the original creation. This prevents a same-key
	// same-payload retry from failing with ErrInsufficientStock or
	// ErrInvalidPaymentMethod after the original order already succeeded.
	RequestFingerprint string          `json:"-"`
	Timeline           []TimelineEvent `json:"timeline,omitempty"`
	TimelineJSON       string          `json:"-"`
	ExpectedStatus     string          `json:"expected_status,omitempty"`
	// Version is the aggregate optimistic-concurrency version. Each order
	// mutation (fulfillment, return, payment transition) increments it.
	// Clients send expected_version in mutation requests; the store guards
	// atomically with WHERE version = expected_version and returns 409
	// (ErrInvalidTransition) when the version is stale.
	Version     int   `json:"version"`
	UpdatedUnix int64 `json:"updated_unix"`
}

// OrderInput is the browser-supplied checkout payload. Totals and item prices
// here are untrusted; the service recalculates from the product catalog.
// MemberID is intentionally absent — member identity is derived from the
// authenticated principal at the handler boundary, never from client input.
type OrderInput struct {
	CustomerName    string      `json:"customer_name"`
	Email           string      `json:"email"`
	Phone           string      `json:"phone"`
	Items           []OrderItem `json:"items"`
	ShippingAddress string      `json:"shipping_address"`
	ShippingMethod  string      `json:"shipping_method"`
	PaymentMethod   string      `json:"payment_method"`
	Shipping        int         `json:"shipping"`
	Discount        int         `json:"discount"`
	PromoCode       string      `json:"promo_code"`
	IdempotencyKey  string      `json:"idempotency_key"`
}

// OrderFilter narrows order listings by status, payment status, and/or member.
type OrderFilter struct {
	Status        string
	PaymentStatus string
	MemberID      string
}

// RestockItemInput is the per-item restock declaration. The admin confirms
// that ReturnedQuantity units were physically received back, and chooses to
// restock RestockedQuantity of them (the remainder may be damaged). The
// service enforces 0 <= RestockedQuantity <= ReturnedQuantity and that the
// cumulative totals do not exceed the original order item quantity.
type RestockItemInput struct {
	SKU               string `json:"sku"`
	ReturnedQuantity  int    `json:"returned_quantity"`
	RestockedQuantity int    `json:"restocked_quantity"`
}

// RestockInput is the browser-supplied payload for the per-item restock
// action. It requires a non-empty reason, expected_version for optimistic
// concurrency, a non-empty idempotency key, and at least one item.
type RestockInput struct {
	ExpectedVersion int                `json:"expected_version"`
	IdempotencyKey  string             `json:"idempotency_key"`
	Reason          string             `json:"reason"`
	Items           []RestockItemInput `json:"items"`
}
