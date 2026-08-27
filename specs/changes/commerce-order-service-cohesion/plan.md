# Commerce Order Service Cohesion Delivery Plan

Change ID: commerce-order-service-cohesion
Revision: 1
Status: Accepted

Repository baseline: `2ae36c9476b134806f4be81e7bbdb222a9b00298`

## Scope lock

- `server/internal/modules/commerce/service.go`
- `server/internal/modules/commerce/service_orders.go`
- `server/internal/modules/commerce/service_checkout.go`
- `server/internal/modules/commerce/service_returns.go`
- `server/internal/modules/commerce/service_restock.go`
- `specs/changes/commerce-order-service-cohesion/**`

## Delivery

Covers REQ-001, AC-001, REQ-002, AC-002.

- retain the existing `Service` facade, constructor, product/member behavior, and shared helpers in `service.go`;
- relocate order lookup, customer masking, and order-status mutation behavior to `service_orders.go`;
- relocate quote, order input validation, item resolution, create-order orchestration, and idempotent checkout behavior to `service_checkout.go`;
- relocate return-request status transitions to `service_returns.go`;
- relocate inspected per-item restock and restock idempotency behavior to `service_restock.go`;
- preserve method signatures, authorization checks, validation ordering, state machines, idempotency semantics, Store calls, and error mapping;
- run the full repository CI chain before merge.

No Store interface split, top-level module, route, API contract, schema, migration, provider, UI, or refund-capability change is introduced.
