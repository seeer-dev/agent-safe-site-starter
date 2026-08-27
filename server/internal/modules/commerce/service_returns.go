package commerce

import (
	"context"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

// UpdateOrderReturnStatus advances the return request state machine using
// expected_version optimistic concurrency. Recording receipt does not make
// returned goods saleable; an inspected, quantity-aware inventory adjustment
// is a separate controlled action.
func (s Service) UpdateOrderReturnStatus(ctx context.Context, principal auth.Principal, id string, expectedVersion int, newStatus string) (Order, error) {
	return s.UpdateOrderReturnStatusWithNote(ctx, principal, id, expectedVersion, newStatus, "")
}

func (s Service) UpdateOrderReturnStatusWithNote(ctx context.Context, principal auth.Principal, id string, expectedVersion int, newStatus, note string) (Order, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return Order{}, ErrForbidden
	}
	existing, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return Order{}, err
	}
	// Validate the return transition is legal using the ACTUAL current
	// return_request_status. The store's version guard catches concurrent
	// mutations between this load and the UPDATE.
	allowed, ok := returnTransitions[existing.ReturnRequestStatus]
	if !ok || !allowed[newStatus] {
		return Order{}, ErrInvalidTransition
	}
	if existing.Status != "delivered" {
		return Order{}, ErrInvalidTransition
	}

	now := time.Now().Unix()
	note = strings.TrimSpace(note)
	event, err := newOrderEvent(id, "return_status", principal.UserID, existing.ReturnRequestStatus, newStatus, note, now)
	if err != nil {
		return Order{}, err
	}
	if err := s.store.TransitionOrderReturnStatus(ctx, id, expectedVersion, newStatus, now, event); err != nil {
		return Order{}, err
	}
	return s.GetOrder(ctx, id)
}
