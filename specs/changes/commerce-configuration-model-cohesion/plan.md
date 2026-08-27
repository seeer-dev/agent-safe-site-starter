# Commerce Configuration Model Cohesion Delivery Plan

Change ID: commerce-configuration-model-cohesion
Revision: 1
Status: Accepted

Repository baseline: `dd03b9049d3235c7fab1a05b37dba93b4fb7c45a`

## Scope lock

- `server/internal/modules/commerce/model.go`
- `server/internal/modules/commerce/promotions.go`
- `server/internal/modules/commerce/payment_methods.go`
- `server/internal/modules/commerce/shipping_methods.go`
- `specs/changes/commerce-configuration-model-cohesion/**`

## Slice

Covers REQ-001, AC-001, REQ-002, AC-002.

- remove promotion, payment-method, and shipping-method declarations from the central commerce model file;
- add promotion and payment configuration cohesion files inside `package commerce`;
- colocate shipping configuration declarations with the existing shipping behavior file;
- preserve all exported names, field types, JSON tags, routes, storage calls, migrations, and runtime behavior;
- run full repository CI before merge.

No new top-level module, provider abstraction, route, schema, or UI surface is introduced.
