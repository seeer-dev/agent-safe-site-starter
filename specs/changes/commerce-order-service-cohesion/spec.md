# Commerce Order Service Cohesion Specification

Change ID: commerce-order-service-cohesion
Revision: 1
Status: Accepted

### REQ-001: Preserve commerce order behavior while relocating ownership

The implementation MUST preserve existing order-facing behavior, contracts, authorization, state machines, idempotency rules, persistence semantics, and fail-closed safety while moving code between files inside `package commerce`.

#### AC-001: Existing order service contracts remain unchanged

- GIVEN the current commerce `Service` facade and Store contract
- WHEN order query/masking, checkout, order status, return status, and restock implementations are relocated
- THEN exported method signatures, call ordering, error semantics, authorization checks, state transitions, idempotency behavior, and persistence behavior remain unchanged

### REQ-002: Order service responsibilities are colocated by cohesion

The implementation MUST reduce the central `service.go` ownership surface by colocating existing order responsibilities in dedicated files without introducing new module or abstraction boundaries.

#### AC-002: Dedicated order service files own existing responsibilities

- GIVEN the existing contiguous order implementation in `service.go`
- WHEN this change is applied
- THEN order lookup/status behavior is owned by `service_orders.go`, quote/create-order behavior by `service_checkout.go`, return transitions by `service_returns.go`, and per-item restock/idempotency behavior by `service_restock.go`, all within `package commerce`
