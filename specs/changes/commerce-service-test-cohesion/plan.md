# Commerce Service Test Cohesion Delivery Plan

Change ID: commerce-service-test-cohesion
Revision: 1
Status: Accepted

Repository baseline: `8c93e03f6f0721f7be061fb539d50e3a7cf2c27e`

## Scope lock

- `server/internal/modules/commerce/service_test.go`
- `server/internal/modules/commerce/service_checkout_test.go`
- `server/internal/modules/commerce/service_orders_test.go`
- `server/internal/modules/commerce/service_returns_test.go`
- `server/internal/modules/commerce/service_restock_test.go`
- `specs/changes/commerce-service-test-cohesion/**`

## Delivery

Covers REQ-001, AC-001, REQ-002, AC-002.

- keep shared test DB setup, principals, seed helpers, counters, and non-order tests in `service_test.go`;
- relocate checkout/create-order/idempotency/fingerprint/overflow tests to `service_checkout_test.go`;
- relocate order lookup, member/guest access, masking, and order-state tests to `service_orders_test.go`;
- relocate return-state tests to `service_returns_test.go`;
- relocate per-item restock and restock-idempotency tests to `service_restock_test.go`;
- preserve test bodies and assertions; only file ownership/import placement changes;
- require the full repository verification chain before merge when CI capacity is available.

No production source or runtime contract is changed.
