# Commerce Production File Decomposition Plan

Change ID: commerce-module-file-split
Revision: 3
Status: Draft

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `server/internal/modules/commerce/**`
- `specs/changes/commerce-module-file-split/**`

## Implementation expansion context

Proposal revision: 2
Proposal status: Draft
Repository baseline: `7e5aa90b92d23c8c316d44ca935be7af8d150a1c`
Observed HEAD: `7e5aa90b92d23c8c316d44ca935be7af8d150a1c`
Pre-existing dirty paths:
- `.github/workflows/ci.yml`, staff/media/migrate integration work, `server/tools/verify/main_test.go`, `server/tools/postgres-live-gate/**`, and their controlled specs — preserve; unrelated active changes.
- `Makefile` — direct unprotected cleanup from this architecture-review task.
Packet ID mapping:
- Slice 1 -> Packet S01
- Slice 2 -> Packet S02
- Slice 3 -> Packet S03

### Packet S01: Decompose service ownership

Status: ENVIRONMENT_BLOCKED
Covers: REQ-001, REQ-002, AC-001, AC-003
Hard dependencies: none
Outcome: Service behavior is grouped by aggregate without changing the `Service` type or any method signature.
Safe failure / rollback: File moves can be reverted independently; all declarations stay in one Go package.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Shared service owner | `server/internal/modules/commerce/service.go` | `Service`, `NewService`, shared errors/helpers | 2358-line file contains catalog, order, member, promo, and payment behavior. |
| Existing split precedent | `server/internal/modules/commerce/shipping_methods.go` | `ListShippingMethods`, `CreateShippingMethod`, `UpdateShippingMethod` | Same-package aggregate file already compiles against shared `Service` and store interfaces. |
| Catalog seam | `server/internal/modules/commerce/service.go` | `ListProducts` through `BulkUpdateProducts` | Cohesive product/image methods. |
| Order seam | `server/internal/modules/commerce/service.go` | `ListOrders` through `RestockOrderItems`, plus order helpers | Order aggregate and transactional state-machine behavior. |
| Member/promo/payment seams | `server/internal/modules/commerce/service.go` | `ListMembers`, `ListPromos`, `ListPaymentMethods` method groups | Cohesive administrative aggregates. |
| Existing proof | `server/internal/modules/commerce/*_test.go` | 208 `Test*` functions | Broad behavior and concurrency coverage; files must not change. |

Read set:
- `server/internal/modules/commerce/model.go`, `shipping_methods.go`, and all existing `_test.go` files — determine shared types and proof boundaries.

Modify set:
- `server/internal/modules/commerce/service.go` — retain shared `Service`, constructor/options, errors, and genuinely shared helpers.
- New `service_catalog.go`, `service_orders.go`, `service_members.go`, `service_promos.go`, and `service_payment_methods.go` — receive declarations without semantic edits.

Must not modify:
- `server/internal/modules/commerce/*_test.go` — preservation oracle.
- `server/internal/bootstrap/app.go`, `contracts/openapi.yaml`, `db/**` — no consumer, contract, or schema change.

#### Behavior contract

Before:
- Unrelated aggregate methods are interleaved in a 2358-line file.

After:
- Methods are grouped in same-package aggregate files; callers and runtime behavior are identical.

Preserved invariants:
- `Service`, store interfaces, exported signatures, capability checks, errors, arithmetic, idempotency, optimistic locking, and transaction semantics remain byte-equivalent except for declaration location.

Inputs and authority:
- Existing method receivers and module interfaces remain authoritative; no new dependency seam is introduced.

Errors and edge states:
- Any required semantic edit is `SPEC_DRIFT` and stops this refactor packet.

#### Ordered edits

1. Move catalog/image service declarations to `service_catalog.go` without editing bodies.
2. Move order/quote/return/restock declarations and their private order-only helpers to `service_orders.go` without editing bodies.
3. Move member, promo, and payment-method declarations to their named files; retain existing `shipping_methods.go`.
4. Run `gofmt` only on production files and inspect the diff for body changes beyond relocation/import normalization.

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Service move preserves behavior | repository root | `go test -json ./server/internal/modules/commerce` | Every baseline `Test*` emits an Action=`pass` event | All 208 names remain present and pass. | Missing/skipped test or semantic failure. | Existing tests are unchanged; temporarily altering a moved capability check must fail its existing selected test before restoration. | Restore body, run `git diff -- server/internal/modules/commerce/*_test.go` and require empty output. |

#### Expansion stop

Marker: ENVIRONMENT_BLOCKED
Claim that cannot be executed: Produce attributable full-gate acceptance for the production-file move in the current shared worktree.
Repository evidence: Existing unrelated changes currently make `speccheck` report multiple controlled-spec owners for staff/media/migrate paths, and `scopecheck` reports those paths outside this proposal scope.
Impact: Packet S01 and dependent S02/S03 cannot be accepted in the current worktree even though their repository anchors are known.
Smallest unblock action: Let the active controlled changes settle or execute the approved refactor in a dedicated isolated worktree with an attributable clean baseline.

#### Completion gate

- [ ] Method bodies changed only as required for import normalization.
- [ ] Every baseline test name appears and passes.
- [ ] No `_test.go`, consumer, contract, or schema file changed.
- [ ] Temporary mutation is restored.

### Packet S02: Decompose store ownership

Status: executable
Covers: REQ-001, REQ-002, AC-002, AC-003
Hard dependencies: S01 establishes aggregate filenames and helper ownership so store files use the same grouping.
Outcome: SQL methods are grouped by aggregate with query text and transaction boundaries preserved.
Safe failure / rollback: Same-package declaration moves are reversible; failure leaves no migration or data effect.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Store monolith | `server/internal/modules/commerce/store.go` | `SQLStore` methods | 1409-line file interleaves catalog, member, order, promo, payment, and shipping SQL. |
| Transaction boundary | `server/internal/modules/commerce/store.go` | `CreateOrderTxWithIdempotency`, `RestockOrderItemsTx`, `transitionOrder` | Must move together with private transaction helpers. |
| Catalog boundary | `server/internal/modules/commerce/store.go` | `ListProducts` through `UpsertProductWithImages` | Product and image persistence group. |

Read set:
- `server/internal/modules/commerce/store.go` and store-related interfaces in production files.

Modify set:
- `store.go` — retain `SQLStore`, constructor, common scanners/helpers used by multiple aggregates.
- New `store_catalog.go`, `store_orders.go`, `store_members.go`, `store_promos.go`, and `store_checkout_methods.go` — receive existing declarations and imports.

Must not modify:
- SQL strings, transaction ordering, lock statements, error conversion, migrations, or tests.

#### Behavior contract

Before: persistence methods are interleaved in one file.
After: persistence methods are grouped by aggregate with byte-equivalent SQL and control flow.
Preserved invariants: SQLite/PostgreSQL portability, idempotency, optimistic version guards, event insertion, and stock transitions.
Inputs and authority: Existing `SQLStore` and dialect binding remain authoritative.
Errors and edge states: A helper used by two aggregates stays shared; duplicating it is forbidden.

#### Ordered edits

1. Move catalog/product-image store methods together.
2. Move all order transaction methods and private helpers as one unit.
3. Move members, promos, payment, and shipping store groups.
4. Compare extracted SQL literals and transaction calls against baseline before accepting formatting-only differences.

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Store move preserves SQLite semantics | repository root | `go test -json ./server/internal/modules/commerce` | Order idempotency, concurrency, stock, product image, and shipping tests | All selected named tests pass. | Query/transaction drift. | Temporarily alter one moved `WHERE version = ?` guard and confirm the existing stale-version test fails, then restore. | Search diff for altered SQL and require no test-file changes. |
| Import architecture remains valid | repository root | `go run ./server/tools/archcheck` | `archcheck: ok` | Exit 0. | New cross-module import. | Existing archcheck proof; no new check introduced. | No temporary mutation. |

#### Completion gate

- [ ] SQL and transaction semantics are unchanged.
- [ ] Shared helpers are not duplicated.
- [ ] Existing tests remain untouched and pass.

### Packet S03: Decompose HTTP ownership and close proof

Status: executable
Covers: REQ-001, REQ-002, AC-001, AC-003
Hard dependencies: S01/S02 establish aggregate grouping; handlers follow the same map last to minimize merge conflicts.
Outcome: Handlers are grouped by aggregate while bootstrap route registration remains unchanged.
Safe failure / rollback: Same-package file moves do not alter route ownership; rollback is file restoration.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Handler monolith | `server/internal/modules/commerce/http.go` | `Handler` methods | 887-line file contains public and admin handlers for all commerce aggregates. |
| Real route consumer | `server/internal/bootstrap/app.go` | `/api/products`, `/api/orders`, `/api/admin/*` registrations | Calls exported handler methods; must not change. |
| HTTP proof | `server/internal/modules/commerce/http_test.go` | existing handler/error tests | Must remain untouched. |

Read set:
- `server/internal/bootstrap/app.go`, `server/internal/modules/commerce/http.go`, `http_test.go`.

Modify set:
- `http.go` — retain shared `Handler`, constructor, and shared response/error helpers.
- New `http_catalog.go`, `http_orders.go`, `http_members.go`, `http_promos.go`, and `http_checkout_methods.go` — receive existing handler methods.

Must not modify:
- `server/internal/bootstrap/app.go`, route patterns, response envelopes, status mapping, OpenAPI, or tests.

#### Behavior contract

Before: handler methods are interleaved.
After: handler methods are grouped by aggregate with identical exported method names and responses.
Preserved invariants: explicit `auth.Principal` retrieval, capability checks, safe generic 5xx responses, and JSON shapes.
Inputs and authority: Bootstrap route registration remains the real entry point.
Errors and edge states: Any route or response change is out of scope and stops apply.

#### Ordered edits

1. Move public/admin product handlers to `http_catalog.go`.
2. Move quote/order/return/restock handlers to `http_orders.go`.
3. Move member, promo, payment, and shipping handlers to named files.
4. Preserve shared HTTP helpers in `http.go`; run formatter and inspect moved bodies.

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Routes still reach handlers | repository root | `go test -json ./server/internal/modules/commerce` | Existing `*HTTP*` tests | Every existing HTTP test emits pass. | Missing method, changed envelope/status, auth regression. | Temporarily change one moved safe error status and confirm its existing exact HTTP test fails, then restore. | `git diff -- server/internal/modules/commerce/*_test.go` is empty. |
| Whole repository remains valid | repository root | `go run ./server/tools/speccheck`; `go run ./server/tools/scopecheck`; `go run ./server/tools/verify` | Gate outputs | All exit 0 in an isolated worktree after unrelated changes settle. | Scope or unrelated dirty failure. | Scope gate provides bounded path falsification during apply. | No scratch paths or test edits remain. |

#### Completion gate

- [ ] Bootstrap and OpenAPI are unchanged.
- [ ] All 208 baseline tests appear and pass.
- [ ] No test was moved, renamed, skipped, or weakened.
- [ ] Full verifier passes in an isolated, attributable state.

## Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass |
| Every AC maps to proof in at least one packet | pass |
| Every packet maps back to REQ/AC | pass |
| Baseline, observed HEAD, dirty paths, and packet mapping are explicit | pass |
| All named production seams and consumers were inspected | pass |
| Hard dependency graph has no unexplained edge or cycle | pass |
| Scope covers every modify path and no unrelated path | pass |
| Product/authority decisions are approved or blocked | pending repository owner approval |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |
