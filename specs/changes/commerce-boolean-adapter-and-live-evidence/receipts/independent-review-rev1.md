# Independent Review Receipt — commerce-boolean-adapter-and-live-evidence (rev1)

## Reviewer identity

- **Role:** Independent reviewer (subagent, separate execution context from the implementer).
- **Method:** From-scratch verification. No prior receipt (`live-run-rev1.md`,
  `mutation-testing-rev1.md`) was trusted. Every claim was re-executed against a
  freshly started ephemeral PostgreSQL 16 instance.
- **Scope:** AC-004 (consumer-reachability of the live commerce checkout
  round-trip) and AC-006 (mutation-sensitivity of the 4 boolean adapter sites).

## Environment

- **Date:** 2026-08-18
- **OS:** Windows (PowerShell)
- **Go:** go1.26.4 windows/amd64
- **PostgreSQL:** ephemeral embedded PostgreSQL 16 (via `go run ./server/tools/local-postgres-gate`, port 127.0.0.1:5433)
- **Repo:** `D:\Projects\AI-go-starter` (HEAD `f704c76`)
- **Command authority:** `go run ./server/tools/local-postgres-gate` →
  `postgres-live-gate` → `go test -json -run '^TestPostgresLive' ./server/...`

## Task 1 — Baseline green

Command: `go run ./server/tools/local-postgres-gate` (output captured to
`gate-baseline.log`).

Result: **EXIT=0**, final line
`postgres-live-gate: ok (all required live PostgreSQL tests passed)`.

All 4 required live tests executed and passed (no skip, no fail on the required
set). Elapsed times taken from the `go test -json` `pass` events:

| Test | Package | Action | Elapsed (s) |
|------|---------|--------|-------------|
| TestPostgresLiveMigrateApplyAndTimestamp | server/internal/migrate | pass | 0.37 |
| TestPostgresLiveStaffLockActiveOwners | server/internal/modules/staff | pass | 0.56 |
| TestPostgresLiveMediaLifecycleAndLocks | server/internal/modules/media | pass | 0.69 |
| TestPostgresLiveCommerceBooleanAdaptersAndCheckout | server/internal/modules/commerce | pass | 0.36 |

The only `skip` events in the stream were package-level skips (no `Test` field)
for packages without a `TestPostgresLive*` match (e.g. `server/cmd/api`,
`server/internal/httpx`, `server/tools/archcheck`). None of the 4 required
tests were skipped or missing.

**AC-004 consumer-reachability:** Independently confirmed. The test
`TestPostgresLiveCommerceBooleanAdaptersAndCheckout`
(`server/internal/modules/commerce/postgres_integration_test.go`) runs from a
fresh `migrate.Apply` in an isolated schema and exercises:
1. Promo boolean adapter — upsert active/disabled/expired promos, `GetActivePromoByCode` finds active, rejects disabled/expired.
2. Payment-method boolean adapter — upsert enabled/disabled, `ListPaymentMethods` scans `Enabled` correctly, `validatePaymentMethod` accepts enabled / rejects disabled.
3. Shipping-method boolean adapter — create enabled/disabled, `ListShippingMethods` + `ListPublicShippingMethods` (public filter keeps only enabled).
4. Quote with enabled settings + active promo (subtotal 1000, discount 100, shipping 120, total 1020).
5. Create-order round-trip with stock decrement (10 → 8), persisted order total 1020, status `pending`.
6. No-side-effects: disabled payment/shipping/promo each rejected with the correct sentinel error; stock unchanged; order count stays 1.

## Task 2 — Mutation replay

Three of the four adapter sites were mutated (exceeds the required minimum of
two). Each mutation temporarily reverted the corrected site back to the int 0/1
pattern, ran the full gate, observed the failure, restored the correction
exactly, and re-confirmed green.

### Mutation 1 — `GetActivePromoByCode` (store.go:1106-1109)

**Before (authorized correction):**
```go
query := database.Bind(s.dialect, `SELECT ... FROM promos WHERE code = ? AND enabled = ? AND starts_unix <= ? AND (expires_unix = 0 OR expires_unix >= ?) LIMIT 1`)
var p Promo
if err := s.db.QueryRowContext(ctx, query, code, true, now, now).Scan(
```

**After (mutation — reverted to int literal):**
```go
query := database.Bind(s.dialect, `SELECT ... FROM promos WHERE code = ? AND enabled = 1 AND starts_unix <= ? AND (expires_unix = 0 OR expires_unix >= ?) LIMIT 1`)
var p Promo
if err := s.db.QueryRowContext(ctx, query, code, now, now).Scan(
```

**Gate result:** EXIT=1. `TestPostgresLiveCommerceBooleanAdaptersAndCheckout`
FAILED. Exact failure from the test assertion
(`postgres_integration_test.go:153`):

```
GetActivePromoByCode(active): ERROR: operator does not exist: boolean = integer (SQLSTATE 42883)
```

This is a PostgreSQL type-semantics error (comparing a `boolean` column to an
`integer` literal), surfaced by the test assertion — not a compile or setup
error. The other 3 required tests still passed; only the commerce test failed.

**Restoration:** Reverted exactly to the authorized correction. Re-ran the
gate → EXIT=0, all 4 required tests pass.

### Mutation 3 — `ListPaymentMethods` (store.go:1160-1166)

**Before (authorized correction):**
```go
for rows.Next() {
    var pm PaymentMethod
    if err := rows.Scan(&pm.ID, &pm.Method, &pm.ProviderLabel, &pm.Environment, &pm.ReadinessStatus, &pm.Enabled, &pm.UpdatedUnix); err != nil {
        return nil, err
    }
    out = append(out, pm)
}
```

**After (mutation — reverted to int scan + `enabled == 1`):**
```go
for rows.Next() {
    var pm PaymentMethod
    var enabled int
    if err := rows.Scan(&pm.ID, &pm.Method, &pm.ProviderLabel, &pm.Environment, &pm.ReadinessStatus, &enabled, &pm.UpdatedUnix); err != nil {
        return nil, err
    }
    pm.Enabled = enabled == 1
    out = append(out, pm)
}
```

**Gate result:** EXIT=1. `TestPostgresLiveCommerceBooleanAdaptersAndCheckout`
FAILED. Exact failure from the test assertion
(`postgres_integration_test.go:203`):

```
ListPaymentMethods: sql: Scan error on column index 5, name "enabled": converting driver.Value type bool ("true") to a int: invalid syntax
```

This is a PostgreSQL type-semantics error (the driver returns a Go `bool` for
the `boolean` column; scanning into `int` fails), surfaced by the test
assertion — not a compile or setup error.

**Restoration:** Reverted exactly to the authorized correction. Re-ran the
gate → EXIT=0, all 4 required tests pass.

### Mutation 4 — `UpsertPaymentMethod` (store.go:1170-1182)

**Before (authorized correction):**
```go
func (s SQLStore) UpsertPaymentMethod(ctx context.Context, pm PaymentMethod) error {
    query := database.Bind(s.dialect, `INSERT INTO payment_methods ... VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET ...`)
    _, err := s.db.ExecContext(ctx, query,
        pm.ID, pm.Method, pm.ProviderLabel, pm.Environment, pm.ReadinessStatus, pm.Enabled, pm.UpdatedUnix)
```

**After (mutation — reverted to int 0/1 intermediary):**
```go
func (s SQLStore) UpsertPaymentMethod(ctx context.Context, pm PaymentMethod) error {
    enabled := 0
    if pm.Enabled {
        enabled = 1
    }
    query := database.Bind(s.dialect, `INSERT INTO payment_methods ... VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET ...`)
    _, err := s.db.ExecContext(ctx, query,
        pm.ID, pm.Method, pm.ProviderLabel, pm.Environment, pm.ReadinessStatus, enabled, pm.UpdatedUnix)
```

**Gate result:** EXIT=1. `TestPostgresLiveCommerceBooleanAdaptersAndCheckout`
FAILED. Exact failure from the test assertion
(`postgres_integration_test.go:195`):

```
upsert enabled payment: upsert payment method: failed to encode args[5]: unable to encode 1 into binary format for bool (OID 16): cannot find encode plan
```

This is a PostgreSQL type-semantics error (binding an `int` value to a `boolean`
column OID 16 fails at encode time), surfaced by the test assertion — not a
compile or setup error.

**Restoration:** Reverted exactly to the authorized correction. Re-ran the
gate → EXIT=0, all 4 required tests pass.

### Mutation 2 — `UpsertPromo` (not replayed)

The fourth site (`UpsertPromo`, int 0/1 intermediary → direct `p.Enabled` bind)
was not individually replayed. Three of four sites were replayed (exceeding the
required minimum of two), and all three produced the expected PostgreSQL
type-semantics failure. The `UpsertPromo` site uses the identical int→bool
bind pattern as `UpsertPaymentMethod` (mutation 4), which already demonstrated
the `unable to encode 1 into binary format for bool (OID 16)` failure class.

## Residue check

After all mutations were restored:

```
git diff --stat HEAD -- server/internal/modules/commerce/store.go
 server/internal/modules/commerce/store.go | 20 +++++---------------
 1 file changed, 5 insertions(+), 15 deletions(-)
```

The diff versus `HEAD` shows **only** the authorized adapter correction: 5
insertions, 15 deletions (removal of 4 int intermediaries and the `enabled = 1`
literal). No mutation residue remains in `store.go`. The full `git diff`
content matches the authorized change described in the change spec exactly
(GetActivePromoByCode `enabled = ?` + `true` bind; UpsertPromo direct
`p.Enabled`; ListPaymentMethods direct `&pm.Enabled` scan; UpsertPaymentMethod
direct `pm.Enabled`).

A final post-restoration gate run confirmed green: EXIT=0,
`postgres-live-gate: ok (all required live PostgreSQL tests passed)`, with all
4 required tests passing at the same elapsed times as the baseline.

## Conclusion

- **AC-004 (consumer-reachability):** Independently confirmed.
  `TestPostgresLiveCommerceBooleanAdaptersAndCheckout` runs a full commerce
  checkout round-trip (promo, payment, shipping, quote, create-order,
  no-side-effects) on live PostgreSQL 16 from a fresh migration in an isolated
  schema, and passes.
- **AC-006 (mutation-sensitivity):** Independently confirmed for 3 of the 4
  boolean adapter sites (GetActivePromoByCode, ListPaymentMethods,
  UpsertPaymentMethod). Each reverted site caused
  `TestPostgresLiveCommerceBooleanAdaptersAndCheckout` to fail with a
  PostgreSQL type-semantics error surfaced by the test assertion (not a compile
  or setup error), and restoration returned the gate to green. The 4th site
  (UpsertPromo) shares the identical bind pattern as the confirmed
  UpsertPaymentMethod site.

Both acceptance criteria are independently verified from scratch.
