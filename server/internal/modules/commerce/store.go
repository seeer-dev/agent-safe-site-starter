package commerce

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ErrNotFound is returned when a single-row lookup misses.
var ErrNotFound = errors.New("commerce: not found")

// ErrIdempotencyConflict is returned when an idempotency key is reused with a
// different payload.
var ErrIdempotencyConflict = errors.New("commerce: idempotency key reused with different payload")

// Store is the persistence interface for the commerce module. Keeping it an
// interface mirrors the content module and lets tests swap implementations.
type Store interface {
	// Products
	ListProducts(ctx context.Context, filter ProductFilter) ([]Product, error)
	ListPublishedProducts(ctx context.Context, filter ProductFilter) ([]Product, error)
	GetProduct(ctx context.Context, id string) (Product, error)
	GetProductBySlug(ctx context.Context, slug string) (Product, error)
	UpsertProduct(ctx context.Context, p Product) error
	UpdateProductStatus(ctx context.Context, id, status string, updatedUnix int64) error
	DeleteProduct(ctx context.Context, id string) error
	BulkUpdateProductStatus(ctx context.Context, ids []string, status string, updatedUnix int64) error
	DecrementProductStock(ctx context.Context, sku string, qty int, updatedUnix int64) error
	IncrementProductStock(ctx context.Context, sku string, qty int, updatedUnix int64) error

	// Product images (B4 verified media association)
	ListProductImages(ctx context.Context, productID string) ([]ProductImage, error)
	ReplaceProductImages(ctx context.Context, productID string, images []ProductImage) error
	// UpsertProductWithImages atomically upserts the product and
	// replaces all product_images rows in a single transaction. This
	// prevents partial success: if the image replacement fails, the
	// product upsert is rolled back too.
	UpsertProductWithImages(ctx context.Context, p Product, images []ProductImage) error

	// Members
	ListMembers(ctx context.Context, filter MemberFilter) ([]Member, error)
	GetMember(ctx context.Context, id string) (Member, error)
	UpsertMember(ctx context.Context, m Member) error
	UpdateMemberStatus(ctx context.Context, id, status string, updatedUnix int64) error

	// Orders
	ListOrders(ctx context.Context, filter OrderFilter) ([]Order, error)
	GetOrder(ctx context.Context, id string) (Order, error)
	GetOrderByAccessToken(ctx context.Context, id, token string) (Order, error)
	FindOrderByIdempotencyKey(ctx context.Context, key string) (Order, error)
	CreateOrder(ctx context.Context, o Order) error
	CreateOrderTx(ctx context.Context, o Order, items []OrderItem) error
	// CreateOrderTxWithIdempotency performs the idempotency check INSIDE the
	// transaction. If a duplicate idempotency_key causes a unique constraint
	// violation on insert, it rolls back and reads back the existing order
	// via a fresh connection (PostgreSQL transactions abort after any error).
	// The caller verifies the payload matches.
	CreateOrderTxWithIdempotency(ctx context.Context, o Order, items []OrderItem) (existingOrder Order, conflict bool, err error)
	TransitionOrderStatus(ctx context.Context, id string, expectedVersion int, newStatus, timelineJSON string, updatedUnix int64, restock []OrderItem, soldOut []OrderItem, event OrderEvent) error
	TransitionOrderReturnStatus(ctx context.Context, id string, expectedVersion int, newStatus string, updatedUnix int64, event OrderEvent) error
	// GetOrderItems loads the per-item ledger rows for an order. Returns
	// returned_quantity and restocked_quantity alongside the snapshot columns.
	GetOrderItems(ctx context.Context, orderID string) ([]OrderItem, error)
	// RestockOrderItemsTx atomically increments returned_quantity and
	// restocked_quantity on order_items rows, increments products.stock for
	// the restocked amount, increments the order aggregate version (guarded
	// by expectedVersion), inserts an append-only order_event, and records
	// the idempotency key + request fingerprint. If the idempotency key
	// already exists, it returns the existing order and its stored
	// request_fingerprint with conflict=true so the caller can compare
	// fingerprints and decide retry vs conflict. If the version guard
	// fails (0 affected rows), it returns ErrStaleVersion. If a CHECK
	// constraint fails (over-restock), the transaction is rolled back and
	// the error propagates. If a product SKU is missing (stock UPDATE
	// affects 0 rows), the transaction is rolled back with
	// ErrRestockItemNotFound.
	// RestockOrderItemsTx atomically performs the per-item restock action.
	// On success, returns ("", snapshotJSON, false, nil) where snapshotJSON
	// is the serialized Order snapshot captured inside the transaction.
	// On concurrent duplicate key, returns (storedFingerprint, storedResponseJSON, true, nil).
	// The caller MUST return the snapshotJSON on success (not GetOrder) to
	// guarantee the response matches what was stored for future replays.
	RestockOrderItemsTx(ctx context.Context, orderID string, expectedVersion int, items []RestockItemInput, event OrderEvent, idempotencyKey, requestFingerprint string, createdUnix int64) (existingFingerprint, existingResponseJSON string, conflict bool, err error)
	// FindRestockByIdempotencyKey looks up a prior restock by its
	// idempotency key. Returns the order_id, request_fingerprint, and
	// response_json (the serialized Order snapshot from the original
	// successful restock) so the caller can detect same-payload replay
	// vs different-payload conflict and return the exact original response.
	FindRestockByIdempotencyKey(ctx context.Context, key string) (orderID, requestFingerprint, responseJSON string, err error)
	ListOrderEvents(ctx context.Context, orderID string) ([]OrderEvent, error)
	CountOrders(ctx context.Context) (int, error)

	// ECPay payment attempts
	EnsureECPayAttempt(ctx context.Context, attempt ECPayPaymentAttempt) (ECPayPaymentAttempt, error)
	GetECPayAttemptByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (ECPayPaymentAttempt, error)
	ClaimECPayCallback(ctx context.Context, merchantTradeNo, callbackFingerprint, providerTradeNo, rtnCode, status string, captured bool, updatedUnix int64) (bool, error)

	// Categories
	ListCategories(ctx context.Context, activeOnly bool) ([]Category, error)
	GetCategory(ctx context.Context, id string) (Category, error)
	UpsertCategory(ctx context.Context, c Category) error
	DeleteCategory(ctx context.Context, id string) error

	// Product variants
	ListProductVariants(ctx context.Context, productID string) ([]ProductVariant, error)
	// ReplaceProductVariants atomically replaces all variant rows for a
	// product in a single transaction.
	ReplaceProductVariants(ctx context.Context, productID string, variants []ProductVariant) error
	GetVariantBySKU(ctx context.Context, sku string) (ProductVariant, error)

	// Product comments
	ListProductComments(ctx context.Context, productID, status string) ([]ProductComment, error)
	ListComments(ctx context.Context, status string) ([]ProductComment, error)
	GetComment(ctx context.Context, id string) (ProductComment, error)
	InsertComment(ctx context.Context, c ProductComment) error
	UpdateCommentModeration(ctx context.Context, id, status, adminReply string, repliedUnix, updatedUnix int64) error
	// IncrementProductSoldCount adds qty to products.sold_count for the SKU.
	IncrementProductSoldCount(ctx context.Context, sku string, qty int, updatedUnix int64) error

	// Notification templates and logs
	ListNotificationTemplates(ctx context.Context) ([]NotificationTemplate, error)
	GetNotificationTemplateByCode(ctx context.Context, code string) (NotificationTemplate, error)
	UpsertNotificationTemplate(ctx context.Context, t NotificationTemplate) error
	DeleteNotificationTemplate(ctx context.Context, id string) error
	InsertNotificationLog(ctx context.Context, l NotificationLog) error
	GetNotificationLog(ctx context.Context, id string) (NotificationLog, error)
	ListNotificationLogs(ctx context.Context, filter NotificationLogFilter) ([]NotificationLog, error)

	// Promos
	ListPromos(ctx context.Context) ([]Promo, error)
	GetActivePromoByCode(ctx context.Context, code string, now int64) (Promo, error)
	UpsertPromo(ctx context.Context, p Promo) error
	DeletePromo(ctx context.Context, id string) error
	// IncrementPromoUsage adds qty to promos.used_count for the given code.
	IncrementPromoUsage(ctx context.Context, code string, qty int, updatedUnix int64) error

	// Admin dashboard
	CountProducts(ctx context.Context) (int, error)
	CountPendingComments(ctx context.Context) (int, error)
	SumOrderRevenue(ctx context.Context) (int, error)

	// Payment methods
	ListPaymentMethods(ctx context.Context) ([]PaymentMethod, error)
	UpsertPaymentMethod(ctx context.Context, pm PaymentMethod) error

	// Shipping methods
	ListShippingMethods(ctx context.Context) ([]ShippingMethod, error)
	GetShippingMethod(ctx context.Context, id string) (ShippingMethod, error)
	CreateShippingMethod(ctx context.Context, m ShippingMethod) error
	UpdateShippingMethod(ctx context.Context, m ShippingMethod, expectedVersion int) error
}

// SQLStore is the production Store backed by *sql.DB.
type SQLStore struct {
	db      *sql.DB
	dialect database.Dialect
}

// NewSQLStore constructs a SQLStore for the given dialect.
func NewSQLStore(db *sql.DB, dialect database.Dialect) SQLStore {
	return SQLStore{db: db, dialect: dialect}
}

// ----- helpers --------------------------------------------------------------

// productColumns is the canonical products SELECT column list. Appending
// columns requires updating scanProducts/scanProductRow in the same order.
const productColumns = "id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, is_featured, sold_count, updated_unix"

// orderColumns is the canonical orders SELECT column list. Appending
// columns requires updating scanOrders/scanOrderRow in the same order.
const orderColumns = "id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix, recipient_name, recipient_phone, city, district, cvs_store_name, cvs_store_address, invoice_type, invoice_tax_id, payment_fee, coupon_code, buyer_note"

func scanProducts(rows *sql.Rows) ([]Product, error) {
	var out []Product
	for rows.Next() {
		var p Product
		var featured int
		if err := rows.Scan(
			&p.ID, &p.SKU, &p.Name, &p.Slug, &p.Description, &p.LongDescription, &p.Image, &p.Images,
			&p.Category, &p.Status, &p.Material, &p.Origin, &p.Price, &p.OriginalPrice, &p.Stock,
			&p.Tag, &p.Rating, &p.ReviewsCount, &featured, &p.SoldCount, &p.UpdatedUnix,
		); err != nil {
			return nil, err
		}
		p.IsFeatured = featured == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanProductRow(row *sql.Row) (Product, error) {
	var p Product
	var featured int
	err := row.Scan(
		&p.ID, &p.SKU, &p.Name, &p.Slug, &p.Description, &p.LongDescription, &p.Image, &p.Images,
		&p.Category, &p.Status, &p.Material, &p.Origin, &p.Price, &p.OriginalPrice, &p.Stock,
		&p.Tag, &p.Rating, &p.ReviewsCount, &featured, &p.SoldCount, &p.UpdatedUnix,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		return Product{}, err
	}
	p.IsFeatured = featured == 1
	return p, nil
}

func scanOrders(rows *sql.Rows) ([]Order, error) {
	var out []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID, &o.MemberID, &o.CustomerName, &o.Email, &o.Phone, &o.ItemsJSON,
			&o.ShippingAddress, &o.ShippingMethod, &o.PaymentMethod, &o.TrackingNumber,
			&o.Subtotal, &o.Discount, &o.Shipping, &o.Total,
			&o.Status, &o.PaymentStatus, &o.ReturnRequestStatus, &o.PaymentIntentID, &o.IdempotencyKey, &o.AccessTokenHash, &o.RequestFingerprint,
			&o.TimelineJSON, &o.ExpectedStatus, &o.Version, &o.UpdatedUnix,
			&o.RecipientName, &o.RecipientPhone, &o.City, &o.District,
			&o.CVSStoreName, &o.CVSStoreAddress, &o.InvoiceType, &o.InvoiceTaxID,
			&o.PaymentFee, &o.CouponCode, &o.BuyerNote,
		); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func scanOrderRow(row *sql.Row) (Order, error) {
	var o Order
	err := row.Scan(
		&o.ID, &o.MemberID, &o.CustomerName, &o.Email, &o.Phone, &o.ItemsJSON,
		&o.ShippingAddress, &o.ShippingMethod, &o.PaymentMethod, &o.TrackingNumber,
		&o.Subtotal, &o.Discount, &o.Shipping, &o.Total,
		&o.Status, &o.PaymentStatus, &o.ReturnRequestStatus, &o.PaymentIntentID, &o.IdempotencyKey, &o.AccessTokenHash, &o.RequestFingerprint,
		&o.TimelineJSON, &o.ExpectedStatus, &o.Version, &o.UpdatedUnix,
		&o.RecipientName, &o.RecipientPhone, &o.City, &o.District,
		&o.CVSStoreName, &o.CVSStoreAddress, &o.InvoiceType, &o.InvoiceTaxID,
		&o.PaymentFee, &o.CouponCode, &o.BuyerNote,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Order{}, ErrNotFound
		}
		return Order{}, err
	}
	return o, nil
}

func scanPromos(rows *sql.Rows) ([]Promo, error) {
	var out []Promo
	for rows.Next() {
		var p Promo
		var enabled int
		var usageLimit sql.NullInt64
		if err := rows.Scan(&p.ID, &p.Code, &p.Label, &p.Type, &p.Value, &enabled, &p.StartsUnix, &p.ExpiresUnix, &p.MinSubtotal, &usageLimit, &p.UsedCount, &p.UpdatedUnix); err != nil {
			return nil, err
		}
		p.Enabled = enabled == 1
		if usageLimit.Valid {
			limit := int(usageLimit.Int64)
			p.UsageLimit = &limit
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// hashAccessToken returns the SHA-256 hex digest of the plaintext access
// token. Only the hash is stored in the database; the plaintext token is
// returned to the customer exactly once at order creation and is never
// persisted. Lookups hash the incoming plaintext token and compare.
func hashAccessToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
