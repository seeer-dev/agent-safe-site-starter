# Commerce Store Cohesion Delivery Plan

Change ID: commerce-store-cohesion
Revision: 1
Status: Accepted

Repository baseline: `7d44ebb42928ab13a8fa28864c1fa054e803b473`

## Scope lock

- `server/internal/modules/commerce/store.go`
- `server/internal/modules/commerce/store_catalog.go`
- `server/internal/modules/commerce/store_members.go`
- `server/internal/modules/commerce/store_orders.go`
- `server/internal/modules/commerce/store_promotions.go`
- `server/internal/modules/commerce/store_payment_methods.go`
- `server/internal/modules/commerce/store_shipping_methods.go`
- `server/internal/modules/commerce/shipping_methods_test.go`
- `specs/changes/commerce-store-cohesion/**`

## Delivery

Covers REQ-001, AC-001, REQ-002, AC-002.

- retain `Store`, `SQLStore`, `NewSQLStore`, and shared scan/helper functions in `store.go`;
- relocate existing product and product-image persistence to `store_catalog.go`;
- relocate member persistence to `store_members.go`;
- relocate order, checkout transaction, idempotency, return/restock, and order-event persistence to `store_orders.go`;
- relocate promotion, payment-method, and shipping-method persistence to their respective store files;
- update the static shipping driver-parity source test to inspect `store_shipping_methods.go` instead of assuming shipping persistence remains in `store.go`;
- preserve method signatures, SQL text, database semantics, transaction boundaries, error mapping, and package ownership;
- run the full repository CI chain before merge.

No Store subinterface split, top-level module, route, schema, migration, provider, or UI change is introduced.
