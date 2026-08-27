package commerce

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

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
