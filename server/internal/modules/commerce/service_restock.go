package commerce

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

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
