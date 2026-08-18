package commerce

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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
	ListPublishedProducts(ctx context.Context) ([]Product, error)
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
	TransitionOrderStatus(ctx context.Context, id string, expectedVersion int, newStatus, timelineJSON string, updatedUnix int64, restock []OrderItem, event OrderEvent) error
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

	// Promos
	ListPromos(ctx context.Context) ([]Promo, error)
	GetActivePromoByCode(ctx context.Context, code string, now int64) (Promo, error)
	UpsertPromo(ctx context.Context, p Promo) error
	DeletePromo(ctx context.Context, id string) error

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

// ----- Products -------------------------------------------------------------

func (s SQLStore) ListProducts(ctx context.Context, filter ProductFilter) ([]Product, error) {
	var (
		clauses []string
		args    []any
	)
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Category != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, filter.Category)
	}
	query := `SELECT id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix FROM products`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY updated_unix DESC"
	query = database.Bind(s.dialect, query)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

func (s SQLStore) ListPublishedProducts(ctx context.Context) ([]Product, error) {
	query := database.Bind(s.dialect, `SELECT id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix
		FROM products WHERE status IN ('active','low_stock','out_of_stock') ORDER BY updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

func (s SQLStore) GetProduct(ctx context.Context, id string) (Product, error) {
	query := database.Bind(s.dialect, `SELECT id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix
		FROM products WHERE id = ? LIMIT 1`)
	return scanProductRow(s.db.QueryRowContext(ctx, query, id))
}

func (s SQLStore) GetProductBySlug(ctx context.Context, slug string) (Product, error) {
	query := database.Bind(s.dialect, `SELECT id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix
		FROM products WHERE slug = ? AND status IN ('active','low_stock','out_of_stock') LIMIT 1`)
	return scanProductRow(s.db.QueryRowContext(ctx, query, slug))
}

func (s SQLStore) UpsertProduct(ctx context.Context, p Product) error {
	query := database.Bind(s.dialect, `INSERT INTO products
		(id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			sku = excluded.sku,
			name = excluded.name,
			slug = excluded.slug,
			description = excluded.description,
			long_description = excluded.long_description,
			image = excluded.image,
			images = excluded.images,
			category = excluded.category,
			status = excluded.status,
			material = excluded.material,
			origin = excluded.origin,
			price = excluded.price,
			original_price = excluded.original_price,
			stock = excluded.stock,
			tag = excluded.tag,
			rating = excluded.rating,
			reviews_count = excluded.reviews_count,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.SKU, p.Name, p.Slug, p.Description, p.LongDescription, p.Image, p.Images,
		p.Category, p.Status, p.Material, p.Origin, p.Price, p.OriginalPrice, p.Stock,
		p.Tag, p.Rating, p.ReviewsCount, p.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert product: %w", err)
	}
	return nil
}

func (s SQLStore) UpdateProductStatus(ctx context.Context, id, status string, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE products SET status = ?, updated_unix = ? WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, status, updatedUnix, id)
	if err != nil {
		return fmt.Errorf("update product status: %w", err)
	}
	return requireAffected(res)
}

func (s SQLStore) DeleteProduct(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete product: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	oldKeys, err := productImageKeysTx(ctx, tx, s.dialect, id)
	if err != nil {
		return err
	}
	query := database.Bind(s.dialect, `DELETE FROM products WHERE id = ?`)
	res, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if err := requireAffected(res); err != nil {
		return err
	}
	if err := markMediaAssociationsTx(ctx, tx, s.dialect, oldKeys, nil, time.Now().Unix()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete product: %w", err)
	}
	return nil
}

func (s SQLStore) BulkUpdateProductStatus(ctx context.Context, ids []string, status string, updatedUnix int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+2)
	args = append(args, status, updatedUnix)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := database.Bind(s.dialect, `UPDATE products SET status = ?, updated_unix = ? WHERE id IN (`+strings.Join(placeholders, ", ")+`)`)
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("bulk update product status: %w", err)
	}
	return nil
}

func (s SQLStore) DecrementProductStock(ctx context.Context, sku string, qty int, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE products SET stock = stock - ?, updated_unix = ? WHERE sku = ? AND stock >= ?`)
	res, err := s.db.ExecContext(ctx, query, qty, updatedUnix, sku, qty)
	if err != nil {
		return fmt.Errorf("decrement stock: %w", err)
	}
	return requireAffected(res)
}

func (s SQLStore) IncrementProductStock(ctx context.Context, sku string, qty int, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE products SET stock = stock + ?, updated_unix = ? WHERE sku = ?`)
	res, err := s.db.ExecContext(ctx, query, qty, updatedUnix, sku)
	if err != nil {
		return fmt.Errorf("increment stock: %w", err)
	}
	return requireAffected(res)
}

// ----- Product images -------------------------------------------------------

func (s SQLStore) ListProductImages(ctx context.Context, productID string) ([]ProductImage, error) {
	query := database.Bind(s.dialect, `SELECT id, product_id, object_key, alt_text, sort_order, created_unix
		FROM product_images WHERE product_id = ? ORDER BY sort_order, created_unix`)
	rows, err := s.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("list product images: %w", err)
	}
	defer rows.Close()
	var out []ProductImage
	for rows.Next() {
		var img ProductImage
		if err := rows.Scan(&img.ID, &img.ProductID, &img.ObjectKey, &img.AltText, &img.SortOrder, &img.CreatedUnix); err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

// ReplaceProductImages deletes all existing product_images rows for the
// given product_id and inserts the new set in a single transaction.
// Sort order is derived from the slice order. This is atomic: if any
// insert fails, the delete is rolled back.
func (s SQLStore) ReplaceProductImages(ctx context.Context, productID string, images []ProductImage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace product images tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	oldKeys, err := productImageKeysTx(ctx, tx, s.dialect, productID)
	if err != nil {
		return err
	}

	deleteQuery := database.Bind(s.dialect, `DELETE FROM product_images WHERE product_id = ?`)
	if _, err := tx.ExecContext(ctx, deleteQuery, productID); err != nil {
		return fmt.Errorf("delete old product images: %w", err)
	}

	insertQuery := database.Bind(s.dialect, `INSERT INTO product_images (id, product_id, object_key, alt_text, sort_order, created_unix) VALUES (?, ?, ?, ?, ?, ?)`)
	for i, img := range images {
		if _, err := tx.ExecContext(ctx, insertQuery, img.ID, productID, img.ObjectKey, img.AltText, i, img.CreatedUnix); err != nil {
			if database.IsForeignKeyViolation(err) {
				return ErrUnverifiedMedia
			}
			return fmt.Errorf("insert product image %d: %w", i, err)
		}
	}
	if err := markMediaAssociationsTx(ctx, tx, s.dialect, oldKeys, images, time.Now().Unix()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace product images: %w", err)
	}
	return nil
}

// UpsertProductWithImages atomically upserts the product and replaces
// all product_images in a single transaction. If the image replacement
// fails, the product upsert is rolled back. This prevents partial
// success where a product is updated but its images are not.
func (s SQLStore) UpsertProductWithImages(ctx context.Context, p Product, images []ProductImage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert product with images tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	upsertQuery := database.Bind(s.dialect, `INSERT INTO products
		(id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			sku = excluded.sku,
			name = excluded.name,
			slug = excluded.slug,
			description = excluded.description,
			long_description = excluded.long_description,
			image = excluded.image,
			images = excluded.images,
			category = excluded.category,
			status = excluded.status,
			material = excluded.material,
			origin = excluded.origin,
			price = excluded.price,
			original_price = excluded.original_price,
			stock = excluded.stock,
			tag = excluded.tag,
			rating = excluded.rating,
			reviews_count = excluded.reviews_count,
			updated_unix = excluded.updated_unix`)
	if _, err := tx.ExecContext(ctx, upsertQuery,
		p.ID, p.SKU, p.Name, p.Slug, p.Description, p.LongDescription, p.Image, p.Images,
		p.Category, p.Status, p.Material, p.Origin, p.Price, p.OriginalPrice, p.Stock,
		p.Tag, p.Rating, p.ReviewsCount, p.UpdatedUnix); err != nil {
		return fmt.Errorf("upsert product in tx: %w", err)
	}
	oldKeys, err := productImageKeysTx(ctx, tx, s.dialect, p.ID)
	if err != nil {
		return err
	}

	deleteQuery := database.Bind(s.dialect, `DELETE FROM product_images WHERE product_id = ?`)
	if _, err := tx.ExecContext(ctx, deleteQuery, p.ID); err != nil {
		return fmt.Errorf("delete old product images in tx: %w", err)
	}

	insertQuery := database.Bind(s.dialect, `INSERT INTO product_images (id, product_id, object_key, alt_text, sort_order, created_unix) VALUES (?, ?, ?, ?, ?, ?)`)
	for i, img := range images {
		if _, err := tx.ExecContext(ctx, insertQuery, img.ID, p.ID, img.ObjectKey, img.AltText, i, img.CreatedUnix); err != nil {
			if database.IsForeignKeyViolation(err) {
				return ErrUnverifiedMedia
			}
			return fmt.Errorf("insert product image %d in tx: %w", i, err)
		}
	}
	associationUnix := time.Now().Unix()
	if err := markMediaAssociationsTx(ctx, tx, s.dialect, oldKeys, images, associationUnix); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert product with images: %w", err)
	}
	return nil
}

func productImageKeysTx(ctx context.Context, tx *sql.Tx, dialect database.Dialect, productID string) ([]string, error) {
	query := database.Bind(dialect, `SELECT object_key FROM product_images WHERE product_id = ?`)
	rows, err := tx.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("list old product image keys: %w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func markMediaAssociationsTx(ctx context.Context, tx *sql.Tx, dialect database.Dialect, oldKeys []string, images []ProductImage, nowUnix int64) error {
	markLinked := database.Bind(dialect, `UPDATE media_assets SET unassociated_since_unix = 0 WHERE object_key = ? AND state = 'active'`)
	for _, img := range images {
		if _, err := tx.ExecContext(ctx, markLinked, img.ObjectKey); err != nil {
			return fmt.Errorf("protect associated media: %w", err)
		}
	}
	markUnlinked := database.Bind(dialect, `UPDATE media_assets SET unassociated_since_unix = ?
		WHERE object_key = ? AND state = 'active'
		AND NOT EXISTS (SELECT 1 FROM product_images pi WHERE pi.object_key = media_assets.object_key)`)
	for _, key := range oldKeys {
		if _, err := tx.ExecContext(ctx, markUnlinked, nowUnix, key); err != nil {
			return fmt.Errorf("start media gc grace period: %w", err)
		}
	}
	return nil
}

// ----- Members --------------------------------------------------------------

func (s SQLStore) ListMembers(ctx context.Context, filter MemberFilter) ([]Member, error) {
	var (
		clauses []string
		args    []any
	)
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Tier != "" {
		clauses = append(clauses, "tier = ?")
		args = append(args, filter.Tier)
	}
	query := `SELECT id, email, name, status, tier, tags, notes, total_orders, total_spent, updated_unix FROM members`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY updated_unix DESC"
	query = database.Bind(s.dialect, query)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.Name, &m.Status, &m.Tier, &m.Tags, &m.Notes, &m.TotalOrders, &m.TotalSpent, &m.UpdatedUnix); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s SQLStore) GetMember(ctx context.Context, id string) (Member, error) {
	query := database.Bind(s.dialect, `SELECT id, email, name, status, tier, tags, notes, total_orders, total_spent, updated_unix
		FROM members WHERE id = ? LIMIT 1`)
	var m Member
	err := s.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.Email, &m.Name, &m.Status, &m.Tier, &m.Tags, &m.Notes, &m.TotalOrders, &m.TotalSpent, &m.UpdatedUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Member{}, ErrNotFound
		}
		return Member{}, err
	}
	return m, nil
}

func (s SQLStore) UpsertMember(ctx context.Context, m Member) error {
	query := database.Bind(s.dialect, `INSERT INTO members
		(id, email, name, status, tier, tags, notes, total_orders, total_spent, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email = excluded.email,
			name = excluded.name,
			status = excluded.status,
			tier = excluded.tier,
			tags = excluded.tags,
			notes = excluded.notes,
			total_orders = excluded.total_orders,
			total_spent = excluded.total_spent,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		m.ID, m.Email, m.Name, m.Status, m.Tier, m.Tags, m.Notes, m.TotalOrders, m.TotalSpent, m.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert member: %w", err)
	}
	return nil
}

func (s SQLStore) UpdateMemberStatus(ctx context.Context, id, status string, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE members SET status = ?, updated_unix = ? WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, status, updatedUnix, id)
	if err != nil {
		return fmt.Errorf("update member status: %w", err)
	}
	return requireAffected(res)
}

// ----- Orders ---------------------------------------------------------------

func (s SQLStore) ListOrders(ctx context.Context, filter OrderFilter) ([]Order, error) {
	var (
		clauses []string
		args    []any
	)
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.PaymentStatus != "" {
		clauses = append(clauses, "payment_status = ?")
		args = append(args, filter.PaymentStatus)
	}
	if filter.MemberID != "" {
		clauses = append(clauses, "member_id = ?")
		args = append(args, filter.MemberID)
	}
	query := `SELECT id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix FROM orders`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY updated_unix DESC"
	query = database.Bind(s.dialect, query)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

func (s SQLStore) GetOrder(ctx context.Context, id string) (Order, error) {
	query := database.Bind(s.dialect, `SELECT id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix
		FROM orders WHERE id = ? LIMIT 1`)
	return scanOrderRow(s.db.QueryRowContext(ctx, query, id))
}

// GetOrderByAccessToken looks up an order by both its ID and its opaque
// access token. The token is hashed (SHA-256) before comparison — the
// DB stores only the hash, never the plaintext token. This is the guest
// customer order access path: the token is unguessable and prevents
// enumeration by order ID alone.
func (s SQLStore) GetOrderByAccessToken(ctx context.Context, id, token string) (Order, error) {
	if token == "" {
		return Order{}, ErrNotFound
	}
	tokenHash := hashAccessToken(token)
	query := database.Bind(s.dialect, `SELECT id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix
		FROM orders WHERE id = ? AND access_token = ? LIMIT 1`)
	return scanOrderRow(s.db.QueryRowContext(ctx, query, id, tokenHash))
}

func (s SQLStore) FindOrderByIdempotencyKey(ctx context.Context, key string) (Order, error) {
	if key == "" {
		return Order{}, ErrNotFound
	}
	query := database.Bind(s.dialect, `SELECT id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix
		FROM orders WHERE idempotency_key = ? LIMIT 1`)
	return scanOrderRow(s.db.QueryRowContext(ctx, query, key))
}

func (s SQLStore) CreateOrder(ctx context.Context, o Order) error {
	return s.CreateOrderTx(ctx, o, nil)
}

// CreateOrderTx inserts the order and decrements stock for each line item
// within a single database transaction. If any stock decrement fails (e.g.
// insufficient stock) the entire transaction is rolled back and no order
// is persisted. The unique index on idempotency_key (migration 003) makes
// a concurrent duplicate-key insert fail at the DB layer, which the caller
// maps to ErrIdempotencyConflict on retry.
func (s SQLStore) CreateOrderTx(ctx context.Context, o Order, items []OrderItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin order tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	insertQuery := database.Bind(s.dialect, `INSERT INTO orders
		(id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if _, err := tx.ExecContext(ctx, insertQuery,
		o.ID, o.MemberID, o.CustomerName, o.Email, o.Phone, o.ItemsJSON,
		o.ShippingAddress, o.ShippingMethod, o.PaymentMethod, o.TrackingNumber,
		o.Subtotal, o.Discount, o.Shipping, o.Total,
		o.Status, o.PaymentStatus, o.ReturnRequestStatus, o.PaymentIntentID, o.IdempotencyKey, o.AccessTokenHash, o.RequestFingerprint,
		o.TimelineJSON, o.ExpectedStatus, 1, o.UpdatedUnix); err != nil {
		return fmt.Errorf("create order tx: %w", err)
	}
	if err := s.insertOrderEvent(ctx, tx, OrderEvent{
		ID: o.ID + ":created", OrderID: o.ID, EventType: "order_status",
		Sequence: 1, ActorUserID: orderCreationActor(o), FromStatus: "", ToStatus: o.Status, Reason: "order placed", CreatedUnix: o.UpdatedUnix,
	}); err != nil {
		return err
	}
	if err := s.insertOrderItems(ctx, tx, o.ID, items, o.UpdatedUnix); err != nil {
		return err
	}

	decrementQuery := database.Bind(s.dialect, `UPDATE products SET stock = stock - ?, updated_unix = ? WHERE sku = ? AND stock >= ?`)
	for _, item := range items {
		res, err := tx.ExecContext(ctx, decrementQuery, item.Quantity, o.UpdatedUnix, item.SKU, item.Quantity)
		if err != nil {
			return fmt.Errorf("decrement stock for %s: %w", item.SKU, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrInsufficientStock
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit order tx: %w", err)
	}
	return nil
}

// CreateOrderTxWithIdempotency performs the idempotency check INSIDE the
// transaction. It first tries to find an existing order with the same
// idempotency_key within the transaction. If found, it returns the existing
// order with conflict=true (no insert, no stock decrement). If not found,
// it inserts the new order and decrements stock atomically. If a concurrent
// insert causes a unique constraint violation, it rolls back the aborted
// transaction and reads back the existing order via a fresh connection.
//
// PostgreSQL transactions enter an aborted state after any error, so the
// read-back MUST use a new connection (s.db), not the aborted tx.
// SQLite also benefits from this approach since the tx is rolled back.
func (s SQLStore) CreateOrderTxWithIdempotency(ctx context.Context, o Order, items []OrderItem) (Order, bool, error) {
	key := strings.TrimSpace(o.IdempotencyKey)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, false, fmt.Errorf("begin order tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Step 1: Check for existing order with the same key INSIDE the tx.
	if key != "" {
		findQuery := database.Bind(s.dialect, `SELECT id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix
			FROM orders WHERE idempotency_key = ? LIMIT 1`)
		existing, err := scanOrderRow(tx.QueryRowContext(ctx, findQuery, key))
		if err == nil {
			// Found existing order with same key — return it for payload verification.
			return existing, true, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return Order{}, false, fmt.Errorf("find by idempotency key: %w", err)
		}
	}

	// Step 2: Insert the new order.
	insertQuery := database.Bind(s.dialect, `INSERT INTO orders
		(id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err = tx.ExecContext(ctx, insertQuery,
		o.ID, o.MemberID, o.CustomerName, o.Email, o.Phone, o.ItemsJSON,
		o.ShippingAddress, o.ShippingMethod, o.PaymentMethod, o.TrackingNumber,
		o.Subtotal, o.Discount, o.Shipping, o.Total,
		o.Status, o.PaymentStatus, o.ReturnRequestStatus, o.PaymentIntentID, o.IdempotencyKey, o.AccessTokenHash, o.RequestFingerprint,
		o.TimelineJSON, o.ExpectedStatus, 1, o.UpdatedUnix)
	if err != nil {
		// Unique constraint violation — a concurrent request inserted first.
		// The transaction is now aborted (PostgreSQL) or must be rolled back
		// (SQLite). Roll back and read the existing order via a FRESH
		// connection from the pool, not the aborted tx.
		if database.IsUniqueViolation(err) && key != "" {
			_ = tx.Rollback()
			findQuery := database.Bind(s.dialect, `SELECT id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix
				FROM orders WHERE idempotency_key = ? LIMIT 1`)
			existing, findErr := scanOrderRow(s.db.QueryRowContext(ctx, findQuery, key))
			if findErr == nil {
				return existing, true, nil
			}
		}
		return Order{}, false, fmt.Errorf("create order tx: %w", err)
	}
	if err := s.insertOrderEvent(ctx, tx, OrderEvent{
		ID: o.ID + ":created", OrderID: o.ID, EventType: "order_status",
		Sequence: 1, ActorUserID: orderCreationActor(o), FromStatus: "", ToStatus: o.Status, Reason: "order placed", CreatedUnix: o.UpdatedUnix,
	}); err != nil {
		return Order{}, false, err
	}
	if err := s.insertOrderItems(ctx, tx, o.ID, items, o.UpdatedUnix); err != nil {
		return Order{}, false, err
	}

	// Step 3: Decrement stock for each item.
	decrementQuery := database.Bind(s.dialect, `UPDATE products SET stock = stock - ?, updated_unix = ? WHERE sku = ? AND stock >= ?`)
	for _, item := range items {
		res, err := tx.ExecContext(ctx, decrementQuery, item.Quantity, o.UpdatedUnix, item.SKU, item.Quantity)
		if err != nil {
			return Order{}, false, fmt.Errorf("decrement stock for %s: %w", item.SKU, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return Order{}, false, err
		}
		if affected == 0 {
			return Order{}, false, ErrInsufficientStock
		}
	}

	if err := tx.Commit(); err != nil {
		return Order{}, false, fmt.Errorf("commit order tx: %w", err)
	}
	return o, false, nil
}

func (s SQLStore) insertOrderEvent(ctx context.Context, tx *sql.Tx, event OrderEvent) error {
	query := database.Bind(s.dialect, `INSERT INTO order_events (id, order_id, event_type, sequence, actor_user_id, from_status, to_status, reason, created_unix) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if _, err := tx.ExecContext(ctx, query, event.ID, event.OrderID, event.EventType, event.Sequence, event.ActorUserID, event.FromStatus, event.ToStatus, event.Reason, event.CreatedUnix); err != nil {
		return fmt.Errorf("insert order event: %w", err)
	}
	return nil
}

// insertOrderItems writes the per-item ledger rows for an order within
// the same transaction as the order insert. line_total is computed as
// price * quantity. The (order_id, sku) unique index ensures one row
// per SKU per order.
func (s SQLStore) insertOrderItems(ctx context.Context, tx *sql.Tx, orderID string, items []OrderItem, createdUnix int64) error {
	insertQuery := database.Bind(s.dialect, `INSERT INTO order_items (id, order_id, sku, name, price, quantity, line_total, returned_quantity, restocked_quantity, created_unix) VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, ?)`)
	for i, item := range items {
		itemID, err := randomID()
		if err != nil {
			return fmt.Errorf("generate order_item id: %w", err)
		}
		lineTotal := item.Price * item.Quantity
		if _, err := tx.ExecContext(ctx, insertQuery, itemID, orderID, item.SKU, item.Name, item.Price, item.Quantity, lineTotal, createdUnix); err != nil {
			return fmt.Errorf("insert order_item %d (%s): %w", i, item.SKU, err)
		}
	}
	return nil
}

// GetOrderItems loads the per-item ledger rows for an order, including
// returned_quantity and restocked_quantity. Returns items in insertion
// order (by id ascending is not guaranteed; callers should not rely on
// ordering beyond what the query provides).
func (s SQLStore) GetOrderItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	query := database.Bind(s.dialect, `SELECT sku, name, price, quantity, returned_quantity, restocked_quantity
		FROM order_items WHERE order_id = ? ORDER BY created_unix, id`)
	rows, err := s.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	defer rows.Close()
	var out []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.SKU, &item.Name, &item.Price, &item.Quantity, &item.ReturnedQuantity, &item.RestockedQuantity); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// FindRestockByIdempotencyKey looks up a prior restock request by its
// idempotency key. Returns ErrNotFound if no prior restock with this key.
func (s SQLStore) FindRestockByIdempotencyKey(ctx context.Context, key string) (string, string, string, error) {
	if key == "" {
		return "", "", "", ErrNotFound
	}
	query := database.Bind(s.dialect, `SELECT order_id, request_fingerprint, response_json FROM restock_idempotency WHERE idempotency_key = ? LIMIT 1`)
	var orderID, fingerprint, responseJSON string
	err := s.db.QueryRowContext(ctx, query, key).Scan(&orderID, &fingerprint, &responseJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", "", ErrNotFound
		}
		return "", "", "", err
	}
	return orderID, fingerprint, responseJSON, nil
}

// RestockOrderItemsTx atomically performs the per-item restock action:
//  1. Insert idempotency record with empty response_json (unique constraint
//     catches concurrent dupes).
//  2. Update order version (WHERE version = expectedVersion → 0 rows = stale).
//  3. For each item: UPDATE order_items SET returned_quantity += ?, restocked_quantity += ?
//     WHERE order_id = ? AND sku = ? — the CHECK constraint enforces
//     0 <= restocked_quantity <= returned_quantity <= quantity.
//  4. For each restocked item: UPDATE products SET stock += ? WHERE sku = ?.
//  5. Insert append-only order_event.
//  6. Read back the order + order_items inside the tx, merge, serialize to
//     JSON, and UPDATE restock_idempotency.response_json with the snapshot.
//     This ensures the committed row always has a non-empty response_json
//     (enforced by CHECK response_json <> ”).
//
// If the idempotency key already exists (concurrent duplicate), returns
// (storedFingerprint, storedResponseJSON, true, nil) — the caller compares
// the fingerprint and returns the stored response on match. If the version
// guard fails, returns ErrStaleVersion. If a CHECK constraint fails
// (over-restock), the transaction is rolled back and the error propagates.
func (s SQLStore) RestockOrderItemsTx(ctx context.Context, orderID string, expectedVersion int, items []RestockItemInput, event OrderEvent, idempotencyKey, requestFingerprint string, createdUnix int64) (string, string, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", false, fmt.Errorf("begin restock tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Step 1: Insert idempotency record with a placeholder response_json.
	// The real snapshot is written in Step 6 before commit. We use '{}'
	// (not '') as the placeholder because the CHECK constraint
	// response_json <> '' rejects empty strings on INSERT. The placeholder
	// is never visible to callers — if the transaction fails after this
	// INSERT, the row is rolled back with the tx.
	idemQuery := database.Bind(s.dialect, `INSERT INTO restock_idempotency (idempotency_key, order_id, request_fingerprint, response_json, created_unix) VALUES (?, ?, ?, '{}', ?)`)
	if _, err := tx.ExecContext(ctx, idemQuery, idempotencyKey, orderID, requestFingerprint, createdUnix); err != nil {
		if database.IsUniqueViolation(err) {
			_ = tx.Rollback()
			// Read the stored fingerprint and response snapshot via a fresh
			// connection (the current transaction is aborted on PostgreSQL).
			_, storedFingerprint, storedResponseJSON, fpErr := s.FindRestockByIdempotencyKey(ctx, idempotencyKey)
			if fpErr == nil {
				return storedFingerprint, storedResponseJSON, true, nil
			}
		}
		return "", "", false, fmt.Errorf("insert restock idempotency: %w", err)
	}

	// Step 2: Update order version (optimistic concurrency guard).
	versionQuery := database.Bind(s.dialect, `UPDATE orders SET version = version + 1, updated_unix = ? WHERE id = ? AND version = ?`)
	res, err := tx.ExecContext(ctx, versionQuery, createdUnix, orderID, expectedVersion)
	if err != nil {
		return "", "", false, fmt.Errorf("update order version for restock: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return "", "", false, err
	}
	if affected == 0 {
		return "", "", false, ErrStaleVersion
	}

	// Step 3: Update order_items ledger rows. The CHECK constraint on
	// order_items enforces 0 <= restocked_quantity <= returned_quantity
	// <= quantity. If the cumulative update violates the constraint,
	// the UPDATE affects 0 rows (SQLite) or raises an error (PostgreSQL).
	// We check affected rows to catch the SQLite case.
	itemQuery := database.Bind(s.dialect, `UPDATE order_items
		SET returned_quantity = returned_quantity + ?,
		    restocked_quantity = restocked_quantity + ?
		WHERE order_id = ? AND sku = ?`)
	for _, ri := range items {
		res, err := tx.ExecContext(ctx, itemQuery, ri.ReturnedQuantity, ri.RestockedQuantity, orderID, ri.SKU)
		if err != nil {
			return "", "", false, fmt.Errorf("restock order_item %s: %w", ri.SKU, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return "", "", false, err
		}
		if n == 0 {
			return "", "", false, fmt.Errorf("%w: sku %s", ErrRestockItemNotFound, ri.SKU)
		}
	}

	// Step 4: Increment product stock for the restocked amount. Each UPDATE
	// must affect exactly 1 row — if the product SKU is missing (0 rows),
	// the entire transaction is rolled back with ErrRestockItemNotFound.
	// This prevents a partial restock where the order_items ledger is
	// updated but inventory is not.
	stockQuery := database.Bind(s.dialect, `UPDATE products SET stock = stock + ?, updated_unix = ? WHERE sku = ?`)
	for _, ri := range items {
		if ri.RestockedQuantity <= 0 {
			continue
		}
		res, err := tx.ExecContext(ctx, stockQuery, ri.RestockedQuantity, createdUnix, ri.SKU)
		if err != nil {
			return "", "", false, fmt.Errorf("increment stock for %s: %w", ri.SKU, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return "", "", false, err
		}
		if n != 1 {
			return "", "", false, fmt.Errorf("%w: product sku %s missing during stock increment", ErrRestockItemNotFound, ri.SKU)
		}
	}

	// Step 5: Insert append-only order_event.
	sequenceQuery := database.Bind(s.dialect, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM order_events WHERE order_id = ?`)
	if err := tx.QueryRowContext(ctx, sequenceQuery, event.OrderID).Scan(&event.Sequence); err != nil {
		return "", "", false, fmt.Errorf("allocate restock event sequence: %w", err)
	}
	if err := s.insertOrderEvent(ctx, tx, event); err != nil {
		return "", "", false, err
	}

	// Step 6: Build the response snapshot inside the transaction.
	// Read back the order row and order_items, merge the ledger columns
	// into the items, serialize to JSON, and UPDATE the idempotency row.
	// This ensures the committed row has a non-empty response_json that
	// exactly reflects the post-restock state at commit time.
	snapshot, err := s.buildRestockSnapshot(ctx, tx, orderID)
	if err != nil {
		return "", "", false, fmt.Errorf("build restock response snapshot: %w", err)
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return "", "", false, fmt.Errorf("marshal restock response snapshot: %w", err)
	}
	updateQuery := database.Bind(s.dialect, `UPDATE restock_idempotency SET response_json = ? WHERE idempotency_key = ?`)
	updateRes, err := tx.ExecContext(ctx, updateQuery, string(snapshotJSON), idempotencyKey)
	if err != nil {
		return "", "", false, fmt.Errorf("store restock response snapshot: %w", err)
	}
	n, err := updateRes.RowsAffected()
	if err != nil {
		return "", "", false, err
	}
	if n != 1 {
		return "", "", false, fmt.Errorf("store restock response snapshot: expected 1 row updated, got %d", n)
	}

	if err := tx.Commit(); err != nil {
		return "", "", false, fmt.Errorf("commit restock tx: %w", err)
	}
	return "", string(snapshotJSON), false, nil
}

// buildRestockSnapshot reads the order and order_items inside the given
// transaction and returns a hydrated Order with ledger columns merged
// into Items. This is the exact state that will be returned to the caller
// and stored as the idempotency response snapshot.
func (s SQLStore) buildRestockSnapshot(ctx context.Context, tx *sql.Tx, orderID string) (Order, error) {
	orderQuery := database.Bind(s.dialect, `SELECT id, member_id, customer_name, email, phone, items_json, shipping_address, shipping_method, payment_method, tracking_number, subtotal, discount, shipping, total, status, payment_status, return_request_status, payment_intent_id, idempotency_key, access_token, request_fingerprint, timeline_json, expected_status, version, updated_unix
		FROM orders WHERE id = ? LIMIT 1`)
	o, err := scanOrderRow(tx.QueryRowContext(ctx, orderQuery, orderID))
	if err != nil {
		return Order{}, err
	}
	o = hydrateOrder(o)
	// Merge returned_quantity/restocked_quantity from the order_items ledger.
	itemsQuery := database.Bind(s.dialect, `SELECT sku, name, price, quantity, returned_quantity, restocked_quantity
		FROM order_items WHERE order_id = ? ORDER BY created_unix, id`)
	rows, err := tx.QueryContext(ctx, itemsQuery, orderID)
	if err != nil {
		return Order{}, fmt.Errorf("query order_items for snapshot: %w", err)
	}
	defer rows.Close()
	var ledgerItems []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.SKU, &item.Name, &item.Price, &item.Quantity, &item.ReturnedQuantity, &item.RestockedQuantity); err != nil {
			return Order{}, err
		}
		ledgerItems = append(ledgerItems, item)
	}
	if err := rows.Err(); err != nil {
		return Order{}, err
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

func (s SQLStore) transitionOrder(ctx context.Context, updateQuery string, args []any, restock []OrderItem, event OrderEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin order transition: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, database.Bind(s.dialect, updateQuery), args...)
	if err != nil {
		return fmt.Errorf("update order transition: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrStaleVersion
	}

	stockQuery := database.Bind(s.dialect, `UPDATE products SET stock = stock + ?, updated_unix = ? WHERE sku = ?`)
	for _, item := range restock {
		res, err := tx.ExecContext(ctx, stockQuery, item.Quantity, event.CreatedUnix, item.SKU)
		if err != nil {
			return fmt.Errorf("restock %s: %w", item.SKU, err)
		}
		if err := requireAffected(res); err != nil {
			return fmt.Errorf("restock %s: %w", item.SKU, err)
		}
	}
	sequenceQuery := database.Bind(s.dialect, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM order_events WHERE order_id = ?`)
	if err := tx.QueryRowContext(ctx, sequenceQuery, event.OrderID).Scan(&event.Sequence); err != nil {
		return fmt.Errorf("allocate order event sequence: %w", err)
	}
	if err := s.insertOrderEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit order transition: %w", err)
	}
	return nil
}

func (s SQLStore) TransitionOrderStatus(ctx context.Context, id string, expectedVersion int, newStatus, timelineJSON string, updatedUnix int64, restock []OrderItem, event OrderEvent) error {
	return s.transitionOrder(ctx,
		`UPDATE orders SET status = ?, version = version + 1, timeline_json = ?, updated_unix = ? WHERE id = ? AND version = ?`,
		[]any{newStatus, timelineJSON, updatedUnix, id, expectedVersion}, restock, event)
}

func (s SQLStore) TransitionOrderReturnStatus(ctx context.Context, id string, expectedVersion int, newStatus string, updatedUnix int64, event OrderEvent) error {
	return s.transitionOrder(ctx,
		`UPDATE orders SET return_request_status = ?, version = version + 1, updated_unix = ? WHERE id = ? AND version = ?`,
		[]any{newStatus, updatedUnix, id, expectedVersion}, nil, event)
}

func (s SQLStore) ListOrderEvents(ctx context.Context, orderID string) ([]OrderEvent, error) {
	query := database.Bind(s.dialect, `SELECT id, order_id, event_type, sequence, actor_user_id, from_status, to_status, reason, created_unix FROM order_events WHERE order_id = ? ORDER BY sequence`)
	rows, err := s.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []OrderEvent
	for rows.Next() {
		var event OrderEvent
		if err := rows.Scan(&event.ID, &event.OrderID, &event.EventType, &event.Sequence, &event.ActorUserID, &event.FromStatus, &event.ToStatus, &event.Reason, &event.CreatedUnix); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func orderCreationActor(order Order) string {
	if order.MemberID != "" {
		return order.MemberID
	}
	return "guest"
}

func (s SQLStore) CountOrders(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders`).Scan(&n)
	return n, err
}

// ----- Promos ---------------------------------------------------------------

func (s SQLStore) ListPromos(ctx context.Context) ([]Promo, error) {
	query := database.Bind(s.dialect, `SELECT id, code, label, type, value, enabled, starts_unix, expires_unix, updated_unix FROM promos ORDER BY updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPromos(rows)
}

func (s SQLStore) GetActivePromoByCode(ctx context.Context, code string, now int64) (Promo, error) {
	query := database.Bind(s.dialect, `SELECT id, code, label, type, value, enabled, starts_unix, expires_unix, updated_unix
		FROM promos WHERE code = ? AND enabled = ? AND starts_unix <= ? AND (expires_unix = 0 OR expires_unix >= ?) LIMIT 1`)
	var p Promo
	if err := s.db.QueryRowContext(ctx, query, code, true, now, now).Scan(
		&p.ID, &p.Code, &p.Label, &p.Type, &p.Value, &p.Enabled, &p.StartsUnix, &p.ExpiresUnix, &p.UpdatedUnix,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Promo{}, ErrNotFound
		}
		return Promo{}, err
	}
	return p, nil
}

func (s SQLStore) UpsertPromo(ctx context.Context, p Promo) error {
	query := database.Bind(s.dialect, `INSERT INTO promos
		(id, code, label, type, value, enabled, starts_unix, expires_unix, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			code = excluded.code,
			label = excluded.label,
			type = excluded.type,
			value = excluded.value,
			enabled = excluded.enabled,
			starts_unix = excluded.starts_unix,
			expires_unix = excluded.expires_unix,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.Code, p.Label, p.Type, p.Value, p.Enabled, p.StartsUnix, p.ExpiresUnix, p.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert promo: %w", err)
	}
	return nil
}

func (s SQLStore) DeletePromo(ctx context.Context, id string) error {
	query := database.Bind(s.dialect, `DELETE FROM promos WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete promo: %w", err)
	}
	return requireAffected(res)
}

// ----- Payment methods ------------------------------------------------------

func (s SQLStore) ListPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	query := database.Bind(s.dialect, `SELECT id, method, provider_label, environment, readiness_status, enabled, updated_unix FROM payment_methods ORDER BY updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PaymentMethod
	for rows.Next() {
		var pm PaymentMethod
		if err := rows.Scan(&pm.ID, &pm.Method, &pm.ProviderLabel, &pm.Environment, &pm.ReadinessStatus, &pm.Enabled, &pm.UpdatedUnix); err != nil {
			return nil, err
		}
		out = append(out, pm)
	}
	return out, rows.Err()
}

func (s SQLStore) UpsertPaymentMethod(ctx context.Context, pm PaymentMethod) error {
	query := database.Bind(s.dialect, `INSERT INTO payment_methods
		(id, method, provider_label, environment, readiness_status, enabled, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			method = excluded.method,
			provider_label = excluded.provider_label,
			environment = excluded.environment,
			readiness_status = excluded.readiness_status,
			enabled = excluded.enabled,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		pm.ID, pm.Method, pm.ProviderLabel, pm.Environment, pm.ReadinessStatus, pm.Enabled, pm.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert payment method: %w", err)
	}
	return nil
}

// ----- Shipping methods -----------------------------------------------------

func (s SQLStore) ListShippingMethods(ctx context.Context) ([]ShippingMethod, error) {
	query := database.Bind(s.dialect, `SELECT id, method, label, description, fee, free_threshold, enabled, sort_order, version, updated_unix
		FROM shipping_methods ORDER BY sort_order ASC, method ASC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ShippingMethod, 0)
	for rows.Next() {
		m, err := scanShippingMethod(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s SQLStore) GetShippingMethod(ctx context.Context, id string) (ShippingMethod, error) {
	query := database.Bind(s.dialect, `SELECT id, method, label, description, fee, free_threshold, enabled, sort_order, version, updated_unix
		FROM shipping_methods WHERE id = ?`)
	row := s.db.QueryRowContext(ctx, query, id)
	m, err := scanShippingMethod(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ShippingMethod{}, ErrNotFound
		}
		return ShippingMethod{}, err
	}
	return m, nil
}

func (s SQLStore) CreateShippingMethod(ctx context.Context, m ShippingMethod) error {
	var threshold any
	if m.FreeThreshold != nil {
		threshold = *m.FreeThreshold
	}
	query := database.Bind(s.dialect, `INSERT INTO shipping_methods
		(id, method, label, description, fee, free_threshold, enabled, sort_order, version, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.db.ExecContext(ctx, query,
		m.ID, m.Method, m.Label, m.Description, m.Fee, threshold, m.Enabled, m.SortOrder, m.Version, m.UpdatedUnix)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return ErrDuplicateShippingMethod
		}
		return fmt.Errorf("create shipping method: %w", err)
	}
	return nil
}

func (s SQLStore) UpdateShippingMethod(ctx context.Context, m ShippingMethod, expectedVersion int) error {
	var threshold any
	if m.FreeThreshold != nil {
		threshold = *m.FreeThreshold
	}
	// method is immutable: it is not included in the SET list.
	query := database.Bind(s.dialect, `UPDATE shipping_methods SET
		label = ?, description = ?, fee = ?, free_threshold = ?, enabled = ?,
		sort_order = ?, version = version + 1, updated_unix = ?
		WHERE id = ? AND version = ?`)
	res, err := s.db.ExecContext(ctx, query,
		m.Label, m.Description, m.Fee, threshold, m.Enabled, m.SortOrder, m.UpdatedUnix, m.ID, expectedVersion)
	if err != nil {
		return fmt.Errorf("update shipping method: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		_, getErr := s.GetShippingMethod(ctx, m.ID)
		if errors.Is(getErr, ErrNotFound) {
			return ErrNotFound
		}
		if getErr != nil {
			return getErr
		}
		return ErrStaleShippingMethod
	}
	return nil
}

type shippingMethodScanner interface {
	Scan(dest ...any) error
}

func scanShippingMethod(row shippingMethodScanner) (ShippingMethod, error) {
	var m ShippingMethod
	var threshold sql.NullInt64
	var enabled bool
	if err := row.Scan(
		&m.ID, &m.Method, &m.Label, &m.Description, &m.Fee, &threshold,
		&enabled, &m.SortOrder, &m.Version, &m.UpdatedUnix,
	); err != nil {
		return ShippingMethod{}, err
	}
	m.Enabled = enabled
	if threshold.Valid {
		v := int(threshold.Int64)
		m.FreeThreshold = &v
	}
	return m, nil
}

// ----- helpers --------------------------------------------------------------

func scanProducts(rows *sql.Rows) ([]Product, error) {
	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID, &p.SKU, &p.Name, &p.Slug, &p.Description, &p.LongDescription, &p.Image, &p.Images,
			&p.Category, &p.Status, &p.Material, &p.Origin, &p.Price, &p.OriginalPrice, &p.Stock,
			&p.Tag, &p.Rating, &p.ReviewsCount, &p.UpdatedUnix,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanProductRow(row *sql.Row) (Product, error) {
	var p Product
	err := row.Scan(
		&p.ID, &p.SKU, &p.Name, &p.Slug, &p.Description, &p.LongDescription, &p.Image, &p.Images,
		&p.Category, &p.Status, &p.Material, &p.Origin, &p.Price, &p.OriginalPrice, &p.Stock,
		&p.Tag, &p.Rating, &p.ReviewsCount, &p.UpdatedUnix,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		return Product{}, err
	}
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
		if err := rows.Scan(&p.ID, &p.Code, &p.Label, &p.Type, &p.Value, &enabled, &p.StartsUnix, &p.ExpiresUnix, &p.UpdatedUnix); err != nil {
			return nil, err
		}
		p.Enabled = enabled == 1
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
