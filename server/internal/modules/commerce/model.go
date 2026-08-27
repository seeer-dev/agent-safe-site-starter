package commerce

import (
	"context"
	"encoding/json"
)

// Product is a sellable item in the commerce catalog. Prices and stock are
// stored as integers because the storefront prices everything in TWD.
type Product struct {
	ID              string         `json:"id"`
	SKU             string         `json:"sku"`
	Name            string         `json:"name"`
	Slug            string         `json:"slug"`
	Description     string         `json:"description"`
	LongDescription string         `json:"long_description"`
	Image           string         `json:"image"`
	Images          string         `json:"-"`
	Category        string         `json:"category"`
	Status          string         `json:"status"`
	Material        string         `json:"material"`
	Origin          string         `json:"origin"`
	Price           int            `json:"price"`
	OriginalPrice   int            `json:"original_price"`
	Stock           int            `json:"stock"`
	Tag             string         `json:"tag"`
	Rating          float64        `json:"rating"`
	ReviewsCount    int            `json:"reviews_count"`
	UpdatedUnix     int64          `json:"updated_unix"`
	ProductImages   []ProductImage `json:"-"`
}

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

type ProductImage struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ObjectKey   string `json:"object_key"`
	AltText     string `json:"alt_text"`
	SortOrder   int    `json:"sort_order"`
	CreatedUnix int64  `json:"created_unix"`
}

type ProductImageInput struct {
	Key     string `json:"key"`
	AltText string `json:"alt_text"`
}

type MediaVerifier interface {
	VerifyKey(ctx context.Context, userID, objectKey string) error
}

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

type ProductFilter struct {
	Status   string
	Category string
}

type Member struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Tier        string `json:"tier"`
	Tags        string `json:"tags"`
	Notes       string `json:"notes"`
	TotalOrders int    `json:"total_orders"`
	TotalSpent  int    `json:"total_spent"`
	UpdatedUnix int64  `json:"updated_unix"`
}

type MemberInput struct {
	Email  string `json:"email"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Tier   string `json:"tier"`
	Tags   string `json:"tags"`
	Notes  string `json:"notes"`
}

type MemberFilter struct {
	Status string
	Tier   string
}

type OrderItem struct {
	SKU               string `json:"sku"`
	Name              string `json:"name"`
	Price             int    `json:"price"`
	Quantity          int    `json:"quantity"`
	ReturnedQuantity  int    `json:"returned_quantity,omitempty"`
	RestockedQuantity int    `json:"restocked_quantity,omitempty"`
}

type TimelineEvent struct {
	Status string `json:"status"`
	At     int64  `json:"at"`
	Note   string `json:"note,omitempty"`
}

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

type Order struct {
	ID                  string          `json:"id"`
	MemberID            string          `json:"member_id"`
	CustomerName        string          `json:"customer_name"`
	Email               string          `json:"email"`
	Phone               string          `json:"phone"`
	Items               []OrderItem     `json:"items"`
	ItemsJSON           string          `json:"-"`
	ShippingAddress     string          `json:"shipping_address"`
	ShippingMethod      string          `json:"shipping_method"`
	PaymentMethod       string          `json:"payment_method"`
	TrackingNumber      string          `json:"tracking_number"`
	Subtotal            int             `json:"subtotal"`
	Discount            int             `json:"discount"`
	Shipping            int             `json:"shipping"`
	Total               int             `json:"total"`
	Status              string          `json:"status"`
	PaymentStatus       string          `json:"payment_status"`
	ReturnRequestStatus string          `json:"return_request_status"`
	PaymentIntentID     string          `json:"payment_intent_id"`
	IdempotencyKey      string          `json:"idempotency_key,omitempty"`
	AccessToken         string          `json:"access_token,omitempty"`
	AccessTokenHash     string          `json:"-"`
	RequestFingerprint  string          `json:"-"`
	Timeline            []TimelineEvent `json:"timeline,omitempty"`
	TimelineJSON        string          `json:"-"`
	ExpectedStatus      string          `json:"expected_status,omitempty"`
	Version             int             `json:"version"`
	UpdatedUnix         int64           `json:"updated_unix"`
}

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

type OrderFilter struct {
	Status        string
	PaymentStatus string
	MemberID      string
}

type RestockItemInput struct {
	SKU               string `json:"sku"`
	ReturnedQuantity  int    `json:"returned_quantity"`
	RestockedQuantity int    `json:"restocked_quantity"`
}

type RestockInput struct {
	ExpectedVersion int                `json:"expected_version"`
	IdempotencyKey  string             `json:"idempotency_key"`
	Reason          string             `json:"reason"`
	Items           []RestockItemInput `json:"items"`
}
