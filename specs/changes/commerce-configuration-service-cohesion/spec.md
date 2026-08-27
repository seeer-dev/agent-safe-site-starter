# Commerce Configuration Service Cohesion Specification

Change ID: commerce-configuration-service-cohesion
Revision: 1
Status: Accepted
Decision authority: Repository owner
Approval basis: Owner approved continuing Phase 2 through a complete behavior-preserving commerce service cohesion segment on 2026-08-27, with verified slices squashed back to main.
Repository baseline: 0b64531287f95ce9963b8ab9506b9f0d72966663
Supersedes: none

### REQ-001: Promotion service behavior is colocated with promotion models

The implementation MUST move promotion administration and discount rule behavior out of the central commerce service file into `promotions.go` while keeping `package commerce`, existing Service contracts, Store calls, validation, authorization, error semantics, and arithmetic behavior unchanged.

#### AC-001: Promotion behavior remains contract-compatible

- GIVEN existing callers use promotion CRUD and checkout discount behavior
- WHEN the promotion service cohesion refactor is applied
- THEN `ListPromos`, `CreatePromo`, `UpdatePromo`, `DeletePromo`, and `calculateDiscount` retain their existing behavior and no route, schema, provider, or top-level module is introduced

### REQ-002: Payment configuration service behavior is colocated with payment models

The implementation MUST colocate public payment discovery, server-side payment validation, admin listing, and admin update behavior in `payment_methods.go` without changing availability rules or authorization semantics.

#### AC-002: Payment discovery and validation remain authoritative

- GIVEN checkout and public discovery use the current payment configuration rules
- WHEN payment service logic moves from `service.go` to `payment_methods.go`
- THEN enabled/readiness filtering, accepted identifiers, errors, Store calls, and admin authorization remain unchanged

### REQ-003: Shipping public and fee rules are colocated with shipping behavior

The implementation MUST colocate public shipping discovery and authoritative shipping-fee calculation in `shipping_methods.go` while leaving quote/order orchestration in the central checkout/order flow.

#### AC-003: Shipping relocation does not change checkout behavior

- GIVEN quote and order creation call the existing shipping helpers
- WHEN shipping public discovery and `computeShipping` move into the shipping cohesion file
- THEN method availability, fee/free-threshold behavior, failure semantics, routes, storage, schema, and API contracts remain unchanged
