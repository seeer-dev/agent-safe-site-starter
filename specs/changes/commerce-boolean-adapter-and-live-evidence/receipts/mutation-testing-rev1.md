# Mutation Testing Receipt — Commerce Boolean Adapter

Change ID: commerce-boolean-adapter-and-live-evidence
Revision: 1
AC: AC-006
Date: 2026-08-17
Environment: local embedded PostgreSQL 16 via `go run ./server/tools/local-postgres-gate`
Command: `go run ./server/tools/local-postgres-gate`

## Method

For each of the four corrected boolean adapter sites in `server/internal/modules/commerce/store.go`, the adapter was temporarily reverted to the pre-change int 0/1 intermediary pattern. The live PostgreSQL gate was then run. Each mutation was expected to produce a PostgreSQL type-semantics failure in `TestPostgresLiveCommerceBooleanAdaptersAndCheckout` (the assertion that protects the claimed bool-adapter behavior). After observing the red failure, the mutation was restored and the gate was re-run to confirm green. The working tree was inspected after restoration to confirm zero mutation residue.

## Mutations

### Mutation 1: UpsertPromo — revert `p.Enabled` direct bind to int 0/1

Mutation: re-introduced `enabled := 0; if p.Enabled { enabled = 1 }` and bound `enabled` instead of `p.Enabled`.

Red observation:
```
postgres_integration_test.go:141: upsert active promo: upsert promo: failed to encode args[5]: unable to encode 1 into binary format for bool (OID 16): cannot find encode plan
--- FAIL: TestPostgresLiveCommerceBooleanAdaptersAndCheckout (0.34s)
```
pgx rejects encoding int `1` into a PostgreSQL `boolean` column (OID 16). This is the exact type-semantics failure the bool adapter prevents.

Restoration: removed the int intermediary, restored `p.Enabled` direct bind.
Green: `--- PASS: TestPostgresLiveCommerceBooleanAdaptersAndCheckout (0.36s)`; `postgres-live-gate: ok`.
Residue: none.

### Mutation 2: ListPaymentMethods — revert direct bool scan to int intermediary

Mutation: re-introduced `var enabled int` scan and `pm.Enabled = enabled == 1` conversion.

Red observation:
```
postgres_integration_test.go:203: ListPaymentMethods: sql: Scan error on column index 5, name "enabled": converting driver.Value type bool ("true") to a int: invalid syntax
--- FAIL: TestPostgresLiveCommerceBooleanAdaptersAndCheckout (0.36s)
```
pgx returns `bool` for a PostgreSQL `boolean` column; the `database/sql` driver layer cannot scan `bool` into `int`. This is the exact type-semantics failure the direct bool scan prevents.

Restoration: removed the int intermediary, restored direct `&pm.Enabled` scan.
Green: `--- PASS: TestPostgresLiveCommerceBooleanAdaptersAndCheckout (0.36s)`; `postgres-live-gate: ok`.
Residue: none.

### Mutation 3: GetActivePromoByCode — revert `enabled = ?` bound true to literal `enabled = 1`

Mutation: replaced `enabled = ?` (bound to Go `true`) with literal `enabled = 1` in the WHERE clause.

Red observation:
```
postgres_integration_test.go:153: GetActivePromoByCode(active): ERROR: operator does not exist: boolean = integer (SQLSTATE 42883)
--- FAIL: TestPostgresLiveCommerceBooleanAdaptersAndCheckout (0.35s)
```
PostgreSQL rejects comparing a `boolean` column to an `integer` literal. This is the exact type-semantics failure the parameterized bool bind prevents.

Restoration: restored `enabled = ?` with Go `true` bind.
Green: `--- PASS: TestPostgresLiveCommerceBooleanAdaptersAndCheckout (0.36s)`; `postgres-live-gate: ok`.
Residue: none.

### Mutation 4: UpsertPaymentMethod — revert `pm.Enabled` direct bind to int 0/1

Mutation: re-introduced `enabled := 0; if pm.Enabled { enabled = 1 }` and bound `enabled` instead of `pm.Enabled`.

Red observation:
```
postgres_integration_test.go:195: upsert enabled payment: upsert payment method: failed to encode args[5]: unable to encode 1 into binary format for bool (OID 16): cannot find encode plan
--- FAIL: TestPostgresLiveCommerceBooleanAdaptersAndCheckout (0.34s)
```
pgx rejects encoding int `1` into a PostgreSQL `boolean` column (OID 16). Same type-semantics failure as Mutation 1, on the payment-method adapter.

Restoration: removed the int intermediary, restored `pm.Enabled` direct bind.
Green: `--- PASS: TestPostgresLiveCommerceBooleanAdaptersAndCheckout (0.36s)`; `postgres-live-gate: ok`.
Residue: none.

## Final Green Replay

After all four mutations were restored, a final green replay was run:
```
go run ./server/tools/local-postgres-gate
```
Result: `postgres-live-gate: ok (all required live PostgreSQL tests passed)` — all 4 required tests passed (migrate, staff, media, commerce).

## Residue Check

`git diff --stat server/internal/modules/commerce/store.go` after all restorations:
```
 server/internal/modules/commerce/store.go | 20 +++++-------------
 1 file changed, 5 insertions(+), 15 deletions(-)
```
The diff shows only the authorized adapter correction (removal of 4 int intermediaries, addition of direct bool binds). No mutation residue remains.

## Conclusion

All four mutations produced the expected PostgreSQL type-semantics failure in the targeted assertion. Each restoration returned the test to green with zero residue. The live test is mutation-sensitive to the boolean adapter change.
