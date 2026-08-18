# Commerce Production File Decomposition Specification
| 3 | none | Proposal targeted baseline 7e5aa90 | Proposal targets baseline 3ecc1b6 | Seventeen commits landed since revision 2, including the live PostgreSQL gate, the commerce boolean adapter fix, request observability, and connection pool bounds. Evidence binds to the baseline, so a stale one cannot carry acceptance. | Owner instruction to refresh draft baselines on 2026-08-18 | No evidence existed to invalidate; all records remain pending. |
| 3 | scope | — | unchanged | The scope already named server/internal/modules/commerce/** and no requirement changed. Note for the implementer: commerce writes media_assets.unassociated_since_unix (store.go:434,440), a table media owns, recorded in architecture.yaml under known_cross_module_data_writes. Decomposition must not make that coupling harder to see. | same | — |

Change ID: commerce-module-file-split
Revision: 3
Status: Draft
Decision authority: Repository owner/user
Approval basis: Pending repository owner approval of the corrected 2026-08-15 proposal summary.
Repository baseline: 3ecc1b63b85f13be773af79453a2c29f868903e4
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
