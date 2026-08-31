# Commerce Boolean Adapter and Live PostgreSQL Checkout Evidence Delivery Plan

Change ID: commerce-boolean-adapter-and-live-evidence
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository Reality

| Area | Current state | Required correction |
|---|---|---|
| `GetActivePromoByCode` | Uses literal `enabled = 1` in WHERE clause | Use `enabled = ?` bound to Go `true` via `database.Bind` |
| `UpsertPromo` | Converts `p.Enabled` to int 0/1 before binding | Bind `p.Enabled` directly |
| `ListPaymentMethods` | Scans `enabled` into `int`, converts to bool | Scan into Go `bool` directly |
| `UpsertPaymentMethod` | Converts `pm.Enabled` to int 0/1 before binding | Bind `pm.Enabled` directly |
| Shipping methods | Already binds/scans `enabled` as Go `bool` | No change; reference implementation |
| Live commerce test | No PostgreSQL integration test for commerce | Add `TestPostgresLiveCommerceBooleanAdaptersAndCheckout` |
| `postgres-live-gate` | Requires migrate, staff, media tests | Add commerce test to `RequiredLiveTests` |

## Scope Lock

- `server/internal/modules/commerce/store.go`
- `server/internal/modules/commerce/postgres_integration_test.go` (new file)
- `server/tools/postgres-live-gate/main.go`
- `server/tools/postgres-live-gate/main_test.go`
- `specs/changes/commerce-boolean-adapter-and-live-evidence/**`

Production behavior in `store.go` may be temporarily mutated only for red-before-green evidence and must be restored. No migration files are touched.

## Overlap Note

The `commerce-module-file-split` Draft change has `applies_to: server/internal/modules/commerce/**`. This change touches `server/internal/modules/commerce/store.go` and adds a new test file in the same package. Since `commerce-module-file-split` is `Draft` (not `Applying` or `Verifying`), it does not authorize product edits and does not create an active ownership conflict. If the file-split change moves to `Applying` before this change completes, the repository owner must reconcile the two scopes.

## Dependency-Ordered Slices

### Slice 1: Boolean Adapter Correction
- Fix `GetActivePromoByCode`: replace `enabled = 1` with `enabled = ?` and bind Go `true`.
- Fix `UpsertPromo`: remove int conversion, bind `p.Enabled` directly.
- Fix `ListPaymentMethods`: scan into `bool` directly, remove int intermediary.
- Fix `UpsertPaymentMethod`: remove int conversion, bind `pm.Enabled` directly.
- Run `go test ./server/internal/modules/commerce -count=1` to confirm SQLite still passes.
- Covers: REQ-001, AC-001, AC-002, AC-003.

### Slice 2: Live PostgreSQL Integration Test
- Create `server/internal/modules/commerce/postgres_integration_test.go` with `TestPostgresLiveCommerceBooleanAdaptersAndCheckout`.
- The test creates an isolated schema, applies migrations, and exercises:
  - Enabled/disabled promo upsert and active lookup; expired promo rejection.
  - Enabled/disabled payment method upsert, list, and validation.
  - Enabled/disabled shipping method read/write and public filtering.
  - A quote using enabled payment/shipping and active promo.
  - A create-order using the same settings, with persisted result and amount checks.
  - Disabled or missing configuration producing no order side effects.
- Run the test locally with `TEST_DATABASE_URL` via `local-postgres-gate`.
- Covers: REQ-002, AC-004.

### Slice 3: Gate Enrollment and Mutation Evidence
- Add `TestPostgresLiveCommerceBooleanAdaptersAndCheckout` to `RequiredLiveTests` in `server/tools/postgres-live-gate/main.go` under the commerce package.
- Run `local-postgres-gate` to confirm all four required tests pass.
- Red-before-green: temporarily revert one bool bind (e.g., `UpsertPromo`) back to int 0/1, confirm the live test fails on PostgreSQL, restore, confirm zero residue.
- Run `go run ./server/tools/verify` and `go run ./server/tools/speccheck`.
- Covers: REQ-003, AC-005, AC-006.

## Verification Commands

```text
工作目錄：repository root
go test ./server/internal/modules/commerce -count=1
go test ./server/internal/modules/commerce -run '^TestPostgresLiveCommerceBooleanAdaptersAndCheckout$' -count=1 -v
go run ./server/tools/local-postgres-gate
go run ./server/tools/speccheck
go run ./server/tools/verify
```

The second command requires `TEST_DATABASE_URL` set to a live PostgreSQL instance.
