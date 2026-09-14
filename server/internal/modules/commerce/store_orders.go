package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

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
	query := `SELECT ` + orderColumns + ` FROM orders`
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
	query := database.Bind(s.dialect, `SELECT ` + orderColumns + `
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
	query := database.Bind(s.dialect, `SELECT ` + orderColumns + `
		FROM orders WHERE id = ? AND access_token = ? LIMIT 1`)
	return scanOrderRow(s.db.QueryRowContext(ctx, query, id, tokenHash))
}

func (s SQLStore) FindOrderByIdempotencyKey(ctx context.Context, key string) (Order, error) {
	if key == "" {
		return Order{}, ErrNotFound
	}
	query := database.Bind(s.dialect, `SELECT ` + orderColumns + `
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
		(` + orderColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if _, err := tx.ExecContext(ctx, insertQuery,
		o.ID, o.MemberID, o.CustomerName, o.Email, o.Phone, o.ItemsJSON,
		o.ShippingAddress, o.ShippingMethod, o.PaymentMethod, o.TrackingNumber,
		o.Subtotal, o.Discount, o.Shipping, o.Total,
		o.Status, o.PaymentStatus, o.ReturnRequestStatus, o.PaymentIntentID, o.IdempotencyKey, o.AccessTokenHash, o.RequestFingerprint,
		o.TimelineJSON, o.ExpectedStatus, 1, o.UpdatedUnix,
		o.RecipientName, o.RecipientPhone, o.City, o.District,
		o.CVSStoreName, o.CVSStoreAddress, o.InvoiceType, o.InvoiceTaxID,
		o.PaymentFee, o.CouponCode, o.BuyerNote); err != nil {
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

	for _, item := range items {
		if err := s.decrementStockTx(ctx, tx, item.SKU, item.Quantity, o.UpdatedUnix); err != nil {
			return err
		}
	}
	if err := s.incrementPromoUsageTx(ctx, tx, o.CouponCode, o.UpdatedUnix); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit order tx: %w", err)
	}
	return nil
}

// decrementStockTx decrements stock for the given SKU inside tx. The SKU
// may address a products row or a product_variants row — the product is
// tried first, then the variant. Zero rows on both means the item was
// unavailable or unknown at commit time → ErrInsufficientStock.
func (s SQLStore) decrementStockTx(ctx context.Context, tx *sql.Tx, sku string, qty int, now int64) error {
	res, err := tx.ExecContext(ctx, database.Bind(s.dialect,
		`UPDATE products SET stock = stock - ?, updated_unix = ? WHERE sku = ? AND stock >= ?`),
		qty, now, sku, qty)
	if err != nil {
		return fmt.Errorf("decrement stock for %s: %w", sku, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	res, err = tx.ExecContext(ctx, database.Bind(s.dialect,
		`UPDATE product_variants SET stock = stock - ?, updated_unix = ? WHERE sku = ? AND stock >= ?`),
		qty, now, sku, qty)
	if err != nil {
		return fmt.Errorf("decrement variant stock for %s: %w", sku, err)
	}
	affected, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrInsufficientStock
	}
	return nil
}

// incrementStockTx returns stock to the products row or the
// product_variants row addressed by sku. Used by cancel/restock paths.
// Zero rows on both returns ErrNotFound so a missing stock row is never
// silently skipped.
func (s SQLStore) incrementStockTx(ctx context.Context, tx *sql.Tx, sku string, qty int, now int64) error {
	res, err := tx.ExecContext(ctx, database.Bind(s.dialect,
		`UPDATE products SET stock = stock + ?, updated_unix = ? WHERE sku = ?`),
		qty, now, sku)
	if err != nil {
		return fmt.Errorf("increment stock for %s: %w", sku, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	res, err = tx.ExecContext(ctx, database.Bind(s.dialect,
		`UPDATE product_variants SET stock = stock + ?, updated_unix = ? WHERE sku = ?`),
		qty, now, sku)
	if err != nil {
		return fmt.Errorf("increment variant stock for %s: %w", sku, err)
	}
	affected, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// incrementPromoUsageTx records one redemption of the order's coupon code
// inside the order transaction so usage limits are enforced atomically.
// Empty code is a no-op.
func (s SQLStore) incrementPromoUsageTx(ctx context.Context, tx *sql.Tx, code string, now int64) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	res, err := tx.ExecContext(ctx, database.Bind(s.dialect,
		`UPDATE promos SET used_count = used_count + 1, updated_unix = ? WHERE code = ?`),
		now, code)
	if err != nil {
		return fmt.Errorf("increment promo usage: %w", err)
	}
	// The code was validated before persistence; a missing row here means
	// the promo was deleted between validation and commit. Treat as
	// non-fatal: the order is already priced server-side, usage tracking
	// is best-effort bookkeeping.
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return nil
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
		findQuery := database.Bind(s.dialect, `SELECT ` + orderColumns + `
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
		(` + orderColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err = tx.ExecContext(ctx, insertQuery,
		o.ID, o.MemberID, o.CustomerName, o.Email, o.Phone, o.ItemsJSON,
		o.ShippingAddress, o.ShippingMethod, o.PaymentMethod, o.TrackingNumber,
		o.Subtotal, o.Discount, o.Shipping, o.Total,
		o.Status, o.PaymentStatus, o.ReturnRequestStatus, o.PaymentIntentID, o.IdempotencyKey, o.AccessTokenHash, o.RequestFingerprint,
		o.TimelineJSON, o.ExpectedStatus, 1, o.UpdatedUnix,
		o.RecipientName, o.RecipientPhone, o.City, o.District,
		o.CVSStoreName, o.CVSStoreAddress, o.InvoiceType, o.InvoiceTaxID,
		o.PaymentFee, o.CouponCode, o.BuyerNote)
	if err != nil {
		// Unique constraint violation — a concurrent request inserted first.
		// The transaction is now aborted (PostgreSQL) or must be rolled back
		// (SQLite). Roll back and read the existing order via a FRESH
		// connection from the pool, not the aborted tx.
		if database.IsUniqueViolation(err) && key != "" {
			_ = tx.Rollback()
			findQuery := database.Bind(s.dialect, `SELECT ` + orderColumns + `
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

	// Step 3: Decrement stock for each item (product or variant SKU).
	for _, item := range items {
		if err := s.decrementStockTx(ctx, tx, item.SKU, item.Quantity, o.UpdatedUnix); err != nil {
			return Order{}, false, err
		}
	}
	if err := s.incrementPromoUsageTx(ctx, tx, o.CouponCode, o.UpdatedUnix); err != nil {
		return Order{}, false, err
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

	// Step 4: Increment stock for the restocked amount. The SKU may
	// address a products row or a product_variants row. A SKU missing
	// from both rolls back with ErrRestockItemNotFound — this prevents
	// a partial restock where the order_items ledger is updated but
	// inventory is not.
	for _, ri := range items {
		if ri.RestockedQuantity <= 0 {
			continue
		}
		if err := s.incrementStockTx(ctx, tx, ri.SKU, ri.RestockedQuantity, createdUnix); err != nil {
			if errors.Is(err, ErrNotFound) {
				return "", "", false, fmt.Errorf("%w: product sku %s missing during stock increment", ErrRestockItemNotFound, ri.SKU)
			}
			return "", "", false, fmt.Errorf("increment stock for %s: %w", ri.SKU, err)
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
	orderQuery := database.Bind(s.dialect, `SELECT ` + orderColumns + `
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

func (s SQLStore) transitionOrder(ctx context.Context, updateQuery string, args []any, restock []OrderItem, soldOut []OrderItem, event OrderEvent) error {
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

	for _, item := range restock {
		if err := s.incrementStockTx(ctx, tx, item.SKU, item.Quantity, event.CreatedUnix); err != nil {
			return fmt.Errorf("restock %s: %w", item.SKU, err)
		}
	}
	// soldOut increments products.sold_count when the order reaches a
	// fulfilled state. A variant SKU credits the parent product.
	soldQuery := database.Bind(s.dialect, `UPDATE products SET sold_count = sold_count + ?, updated_unix = ?
		WHERE sku = ? OR id = (SELECT product_id FROM product_variants WHERE sku = ?)`)
	for _, item := range soldOut {
		if _, err := tx.ExecContext(ctx, soldQuery, item.Quantity, event.CreatedUnix, item.SKU, item.SKU); err != nil {
			return fmt.Errorf("increment sold count for %s: %w", item.SKU, err)
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

func (s SQLStore) TransitionOrderStatus(ctx context.Context, id string, expectedVersion int, newStatus, timelineJSON string, updatedUnix int64, restock []OrderItem, soldOut []OrderItem, event OrderEvent) error {
	return s.transitionOrder(ctx,
		`UPDATE orders SET status = ?, version = version + 1, timeline_json = ?, updated_unix = ? WHERE id = ? AND version = ?`,
		[]any{newStatus, timelineJSON, updatedUnix, id, expectedVersion}, restock, soldOut, event)
}

func (s SQLStore) TransitionOrderReturnStatus(ctx context.Context, id string, expectedVersion int, newStatus string, updatedUnix int64, event OrderEvent) error {
	return s.transitionOrder(ctx,
		`UPDATE orders SET return_request_status = ?, version = version + 1, updated_unix = ? WHERE id = ? AND version = ?`,
		[]any{newStatus, updatedUnix, id, expectedVersion}, nil, nil, event)
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

// SumOrderRevenue totals non-cancelled orders for the admin dashboard.
func (s SQLStore) SumOrderRevenue(ctx context.Context) (int, error) {
	var n sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total), 0) FROM orders WHERE status <> 'cancelled'`).Scan(&n)
	if err != nil {
		return 0, err
	}
	return int(n.Int64), nil
}
