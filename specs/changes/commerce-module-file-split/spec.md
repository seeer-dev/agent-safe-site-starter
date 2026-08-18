# Commerce Production File Decomposition Specification

Change ID: commerce-module-file-split
Revision: 3
Status: Draft
Decision authority: Repository owner/user
Approval basis: Pending repository owner approval of the corrected 2026-08-15 proposal summary.
Repository baseline: 6a3c113e9d4957617a532ab2d10701a6fbc66db9
Supersedes: none

## Outcome

Decompose the oversized commerce production files by existing aggregates while retaining one `commerce` package, exactly the same interfaces, routes, SQL, authorization, errors, and behavior. Existing tests remain physically untouched and prove the move.

## Scope

In scope:
- Move production declarations out of `service.go`, `store.go`, and `http.go` into aggregate-named files.
- Follow the existing same-package precedent in `shipping_methods.go`.
- Keep shared constructors, interfaces, errors, arithmetic, IDs, serialization helpers, and cross-aggregate order helpers in stable shared files when moving them would obscure ownership.

Out of scope:
- Splitting `commerce` into new packages or modules.
- Editing migrations, OpenAPI, bootstrap routes, exported symbols, SQL text, or product behavior.
- Moving, renaming, deleting, skipping, or weakening any `_test.go` content.
- Opportunistic cleanup, pagination, logging, or validation changes.

## Requirements

### REQ-001: Same-Package Aggregate Decomposition
Production declarations in `service.go`, `store.go`, and `http.go` MUST be moved into cohesive catalog, order, member, promo, and payment/shipping configuration files while all files remain `package commerce`. Existing `shipping_methods.go` SHALL remain the precedent rather than being folded back into a monolith.

#### AC-001: Service and HTTP Ownership Is Discoverable
- GIVEN product, order, member, promo, and payment methods are interleaved in the current service and handler files
- WHEN decomposition is complete
- THEN each aggregate's service and handler methods MUST be discoverable in aggregate-named files without changing method signatures or route registration.

#### AC-002: Store Ownership Is Discoverable
- GIVEN product, order, member, promo, payment, and shipping SQL methods are interleaved in `store.go`
- WHEN decomposition is complete
- THEN each aggregate's store methods MUST be discoverable in aggregate-named files with SQL text and transaction boundaries unchanged.

### REQ-002: Zero Behavioral and Test Churn
The change MUST preserve exported symbols, HTTP behavior, authorization, error mapping, transaction semantics, and the complete existing test inventory. Existing `_test.go` files SHALL remain byte-for-byte unchanged.

#### AC-003: Existing Tests Prove the Move
- GIVEN 208 existing named commerce tests at the inspected baseline
- WHEN the production declarations are moved
- THEN every existing `Test*` name MUST still execute and pass, and the final diff MUST contain no commerce `_test.go` modification.

## Amendments

Revision 2 removes the earlier test-file split. Moving 208 tests would create review and blame churn without improving runtime behavior; untouched tests are a stronger preservation boundary for this refactor.

Revision 3 rebases the proposal onto baseline 6a3c113. Seventeen commits landed since revision 2. Scope and requirements are unchanged.

One note for the implementer, recorded here because it is easy to lose during decomposition: `commerce` writes `media_assets.unassociated_since_unix` (`server/internal/modules/commerce/store.go:434,440`), a table the `media` module owns. `architecture.yaml` now records it under `known_cross_module_data_writes` with `detected_by: none`, because `archcheck` inspects Go imports and does not read SQL. Splitting files must not move that write somewhere it is harder to see.
