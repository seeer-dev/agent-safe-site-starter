package commerce

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

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
