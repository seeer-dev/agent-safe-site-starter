# Live Run Receipt — Commerce Boolean Adapters and Checkout

Change ID: commerce-boolean-adapter-and-live-evidence
Revision: 1
AC: AC-004, AC-005
Date: 2026-08-17
Environment: local embedded PostgreSQL 16 via `go run ./server/tools/local-postgres-gate`
Command: `go run ./server/tools/local-postgres-gate`

## Execution

The `local-postgres-gate` tool starts an ephemeral PostgreSQL 16 instance on `127.0.0.1:5433`, sets `TEST_DATABASE_URL`, and invokes `postgres-live-gate` which runs `go test -json -run '^TestPostgresLive' ./server/...` and validates the exact required test inventory.

## Required Test Inventory

| Package | Test | Result | Elapsed |
|---|---|---|---|
| server/internal/migrate | TestPostgresLiveMigrateApplyAndTimestamp | PASS | 0.37s |
| server/internal/modules/staff | TestPostgresLiveStaffLockActiveOwners | PASS | 0.56s |
| server/internal/modules/media | TestPostgresLiveMediaLifecycleAndLocks | PASS | 0.69s |
| server/internal/modules/commerce | TestPostgresLiveCommerceBooleanAdaptersAndCheckout | PASS | 0.36s |

## Gate Result

```
postgres-live-gate: ok (all required live PostgreSQL tests passed)
local-postgres-gate: ok (all live tests passed and child stopped)
```

All 4 required live PostgreSQL tests executed and passed with no skip or missing entries.

## TestPostgresLiveCommerceBooleanAdaptersAndCheckout Coverage

The test creates an isolated schema (`commerce_test_<unixnano>`), applies all migrations, and exercises:

1. **Promo boolean adapter (AC-001)**: upserts active/disabled/expired promos via `UpsertPromo` (direct `p.Enabled` bind); looks up active promo via `GetActivePromoByCode` (`enabled = ?` bound to Go `true`); asserts enabled promo found with `Enabled = true`, disabled and expired promos return `ErrNotFound`.

2. **Payment method boolean adapter (AC-002)**: upserts enabled/disabled payment methods via `UpsertPaymentMethod` (direct `pm.Enabled` bind); lists via `ListPaymentMethods` (direct `bool` scan); asserts enabled method has `Enabled = true`, disabled has `Enabled = false`; `validatePaymentMethod` accepts enabled, rejects disabled with `ErrInvalidPaymentMethod`.

3. **Shipping method boolean adapter (AC-003)**: creates enabled/disabled shipping methods via `CreateShippingMethod` (unchanged, already direct bool); lists via `ListShippingMethods`; asserts enabled method has `Enabled = true`; `ListPublicShippingMethods` returns only the enabled method.

4. **Quote (AC-004)**: quotes 2x PG-SKU at 500 each with active promo (fixed 100 discount) and home_delivery (120 fee); asserts subtotal=1000, discount=100, shipping=120, total=1020.

5. **Create order (AC-004)**: creates order with same settings; asserts subtotal=1000, discount=100, shipping=120, total=1020, status=pending; verifies stock decremented from 10 to 8; verifies order persisted with correct total.

6. **No side effects (AC-004)**: attempts create-order with disabled payment (rejected with `ErrInvalidPaymentMethod`), disabled shipping (rejected with `ErrInvalidShippingMethod`), disabled promo (rejected with `ErrInvalidPromoCode`); asserts stock unchanged after failures; asserts order count = 1 (only the successful order).

## CI Observation

AC-005 requires a real CI run with run identity. This local run does not satisfy AC-005's CI requirement; it satisfies the local live PostgreSQL execution requirement for AC-004. AC-005 remains pending until a repository CI run is recorded.
