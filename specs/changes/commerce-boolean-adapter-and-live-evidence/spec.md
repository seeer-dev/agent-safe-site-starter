# Commerce Boolean Adapter and Live PostgreSQL Checkout Evidence Specification

Change ID: commerce-boolean-adapter-and-live-evidence
Revision: 1
Status: Verifying
Decision authority: Repository owner/user
Approval basis: Repository owner approved opening and applying the 2026-08-17 proposal summary derived from the WP1 work package in docs/優化開發執行清單.md.
Repository baseline: 2605afefba0ad079353bc50b530b945ba5d1f682
Supersedes: none

## Outcome

Fix the commerce module's promo and payment-method boolean adapter so that `enabled` is bound and scanned as Go `bool` on both SQLite and PostgreSQL, matching the shipping-method adapter that already works. Add a named live PostgreSQL integration test that exercises promo, payment, shipping, quote, and create-order paths from a fresh migration, and enroll that test in the `postgres-live-gate` exact inventory so CI mechanically proves its execution.

The shipping-method adapter (`CreateShippingMethod`, `UpdateShippingMethod`, `scanShippingMethod`) already binds `m.Enabled` directly and scans into `bool`. This change brings promo and payment methods to the same standard.

## Scope

In scope:
- `SQLStore.GetActivePromoByCode`: replace literal `enabled = 1` with `enabled = ?` bound to Go `true`.
- `SQLStore.UpsertPromo`: bind `p.Enabled` directly instead of converting to int 0/1.
- `SQLStore.ListPaymentMethods`: scan `enabled` into Go `bool` directly instead of int intermediary.
- `SQLStore.UpsertPaymentMethod`: bind `pm.Enabled` directly instead of converting to int 0/1.
- A new live PostgreSQL integration test in `server/internal/modules/commerce/postgres_integration_test.go` that verifies enabled/disabled promo, payment, and shipping methods, a quote, and a create-order round-trip from fresh migration.
- Adding the new test name to `server/tools/postgres-live-gate/main.go` `RequiredLiveTests`.

Out of scope:
- SQLite or PostgreSQL migration changes. The existing `enabled BOOLEAN` (PostgreSQL) and `enabled INTEGER CHECK (enabled IN (0, 1))` (SQLite) column definitions already accept Go `bool` via pgx and modernc.org/sqlite respectively.
- Splitting `store.go`, changing API shape, changing checkout rules, or reorganizing other SQL.
- The `TestConcurrentOrderCancellationRestocksOnce` test (currently `t.Skip`); it is not evidence for this change.
- The `commerce-module-file-split` Draft change; this change does not split any files.

## Decisions and Invariants

- Bool Adapter Parity: promo and payment methods must use the same binding pattern as shipping methods—bind Go `bool` directly, scan into Go `bool` directly. The int 0/1 intermediary is removed.
- Portable SQL: `enabled = ?` with Go `true` works on both drivers. pgx encodes `true` as PostgreSQL `TRUE`; modernc.org/sqlite encodes `true` as `1`. No driver-specific SQL is introduced.
- Live Evidence: the named test must start from a fresh migration in an isolated schema, exercise all three adapter paths (promo, payment, shipping), complete a quote and a create-order, and assert that disabled or missing configuration produces no order side effects.
- Gate Enrollment: the test name must appear in `postgres-live-gate`'s `RequiredLiveTests`. A missing, skipped, or failed execution must make the gate exit non-zero.
- Mutation Sensitivity: temporarily reverting one bool bind back to int 0/1 must cause the live test to fail on PostgreSQL due to type semantics, confirming the test is sensitive to the adapter change.
- No Migration Changes: if live PostgreSQL evidence proves the bool adapter cannot satisfy both drivers simultaneously, stop with `SPEC_DRIFT` and propose paired migration changes as a new decision. Do not modify migrations in this change.

## Requirements

### REQ-001: Portable Boolean Adapter for Promo and Payment Methods
The promo and payment-method store functions MUST bind and scan `enabled` as Go `bool` on both SQLite and PostgreSQL, matching the shipping-method adapter.

#### AC-001: Promo Boolean Adapter
- GIVEN `SQLStore.GetActivePromoByCode` and `SQLStore.UpsertPromo` running on PostgreSQL with `enabled BOOLEAN`
- WHEN an enabled promo is upserted and looked up by code
- THEN the query must use `enabled = ?` bound to Go `true` (not literal `enabled = 1`), and `UpsertPromo` must bind `p.Enabled` directly (not int 0/1)

#### AC-002: Payment Method Boolean Adapter
- GIVEN `SQLStore.ListPaymentMethods` and `SQLStore.UpsertPaymentMethod` running on PostgreSQL with `enabled BOOLEAN`
- WHEN a payment method is upserted and listed
- THEN `ListPaymentMethods` must scan `enabled` into Go `bool` directly (not int intermediary), and `UpsertPaymentMethod` must bind `pm.Enabled` directly (not int 0/1)

#### AC-003: Shipping Method Adapter Unchanged
- GIVEN the existing shipping-method adapter that already binds and scans `enabled` as Go `bool`
- WHEN the promo and payment adapters are corrected
- THEN the shipping-method adapter must remain unchanged and continue to pass its existing tests

### REQ-002: Live PostgreSQL Checkout Integration Test
A named live PostgreSQL integration test MUST exercise the full commerce checkout path from fresh migration and prove all three boolean adapters work on PostgreSQL.

#### AC-004: Fresh Migration Checkout Round-Trip
- GIVEN a live PostgreSQL server with an isolated empty schema
- WHEN the integration test applies migrations and exercises enabled/disabled promo, payment, and shipping methods, completes a quote, and creates an order
- THEN the test must verify: enabled promo is active and expired/disabled promo is rejected; enabled payment method validates and disabled does not; enabled shipping method is listed publicly and disabled is not; a quote with enabled settings returns correct amounts; a create-order with enabled settings persists the order with correct totals; disabled or missing configuration produces no order side effects

### REQ-003: Gate Enrollment and Mutation Evidence
The live test MUST be enrolled in the gate, and the adapter change MUST be mutation-sensitive.

#### AC-005: Gate Enrollment
- GIVEN the `postgres-live-gate` exact test inventory
- WHEN the new commerce live test is added to `RequiredLiveTests`
- THEN a missing, skipped, or failed execution of that test must make the gate exit non-zero

#### AC-006: Red-Before-Green Boolean Adapter Mutation
- GIVEN the corrected boolean adapter on PostgreSQL
- WHEN one bool bind is temporarily reverted to int 0/1
- THEN the named live test must be observed failing due to PostgreSQL type semantics; after restoration the test must pass, and the temporary mutation must not remain in the diff

## Amendments

None.
