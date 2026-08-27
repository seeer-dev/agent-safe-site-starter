# Commerce Configuration Service Cohesion Delivery Plan

Change ID: commerce-configuration-service-cohesion
Revision: 1
Status: Accepted

Repository baseline: `0b64531287f95ce9963b8ab9506b9f0d72966663`

## Scope lock

- `server/internal/modules/commerce/service.go`
- `server/internal/modules/commerce/promotions.go`
- `server/internal/modules/commerce/payment_methods.go`
- `server/internal/modules/commerce/shipping_methods.go`
- `specs/changes/commerce-configuration-service-cohesion/**`

## Slice

Covers REQ-001, AC-001, REQ-002, AC-002, REQ-003, AC-003.

- move promotion CRUD and `calculateDiscount` from `service.go` to `promotions.go`;
- move public payment discovery, `validatePaymentMethod`, admin payment listing, and admin update behavior to `payment_methods.go`;
- move public shipping discovery and `computeShipping` to `shipping_methods.go`;
- keep quote/order orchestration, Service type construction, routes, Store contracts, database schema, migrations, API payloads, and UI unchanged;
- keep all relocated code in `package commerce` and preserve method bodies/validation order;
- run the full repository CI chain before squash merge to main and verify the resulting main push CI.

No new top-level module, provider abstraction, runtime registry, DI container, route, schema, or UI surface is introduced.
