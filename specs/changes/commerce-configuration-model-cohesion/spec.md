# Commerce Configuration Model Cohesion Specification

Change ID: commerce-configuration-model-cohesion
Revision: 1
Status: Accepted
Decision authority: Repository owner
Approval basis: Owner approved continued Phase 2 commerce cohesion refactoring on 2026-08-27 with behavior-preserving slices merged to main after verification.
Repository baseline: dd03b9049d3235c7fab1a05b37dba93b4fb7c45a
Supersedes: none

### REQ-001: Promotion and payment models remain commerce-owned but gain cohesive files

The implementation MUST keep promotion and payment configuration models in `package commerce` while colocating each group in its own behavior-oriented file without changing exported contracts.

#### AC-001: Promotion and payment model contracts are unchanged

- GIVEN existing callers compile against `Promo`, `PromoInput`, `PaymentMethod`, and `PaymentMethodInput`
- WHEN the cohesion refactor is applied
- THEN the exported names, field types, and JSON tags remain unchanged and no new top-level module is introduced

### REQ-002: Shipping models are colocated with shipping behavior

The implementation MUST colocate shipping configuration models with `shipping_methods.go` while preserving existing route, service, storage, database, and JSON contracts.

#### AC-002: Shipping behavior remains contract-compatible

- GIVEN existing shipping admin/public flows use the current commerce package contracts
- WHEN the shipping models move into the shipping cohesion file
- THEN `ShippingMethod`, `ShippingMethodInput`, and `ShippingMethodUpdateInput` remain contract-compatible and no behavior or schema change is introduced
