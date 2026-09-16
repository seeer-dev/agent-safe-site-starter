# Evidence

## Delivery Status

Revision 1 is `Accepted` against baseline `2605afefba0ad079353bc50b530b945ba5d1f682`. Implementation and all declared acceptance evidence are complete: SQLite tests, the live PostgreSQL gate (4 tests at the time of this change), red-before-green mutation testing (4 mutations), independent review, `verify`, `speccheck`, and `scopecheck` all passed. AC-005 required gate enrollment and rejection behavior, not a repository CI run. The repository later gained required CI PostgreSQL/live/stress gates; that later state does not alter this change's historical evidence contract.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Promo and payment boolean adapter corrected in store.go. GetActivePromoByCode uses enabled = ? bound to Go true; UpsertPromo binds p.Enabled directly; ListPaymentMethods scans enabled into bool directly; UpsertPaymentMethod binds pm.Enabled directly. SQLite tests pass (go test ./server/internal/modules/commerce -count=1). Live PostgreSQL mutation testing confirmed each adapter site is type-sensitive. Recorded in receipts/mutation-testing-rev1.md. |
| REQ-002 | passed | TestPostgresLiveCommerceBooleanAdaptersAndCheckout executed on live PostgreSQL 16 via local-postgres-gate. All checkout paths exercised from fresh migration. Recorded in receipts/live-run-rev1.md. |
| REQ-003 | passed | Test enrolled in postgres-live-gate RequiredLiveTests. Gate unit tests (TestGatePassesExactInventory etc.) pass with 4-test inventory. Red-before-green mutation testing confirmed adapter sensitivity. Recorded in receipts/mutation-testing-rev1.md. |
| AC-001 | passed | GetActivePromoByCode uses enabled = ? with Go true; UpsertPromo binds p.Enabled directly. Verified on live PostgreSQL: active promo found with Enabled=true, disabled/expired rejected. Mutation 1 and 3 confirmed type-sensitivity. Recorded in receipts/mutation-testing-rev1.md and receipts/live-run-rev1.md. |
| AC-002 | passed | ListPaymentMethods scans enabled into bool directly; UpsertPaymentMethod binds pm.Enabled directly. Verified on live PostgreSQL: enabled payment has Enabled=true, disabled has Enabled=false, validatePaymentMethod accepts enabled and rejects disabled. Mutation 2 and 4 confirmed type-sensitivity. Recorded in receipts/mutation-testing-rev1.md and receipts/live-run-rev1.md. |
| AC-003 | passed | Shipping-method adapter unchanged (already used direct bool bind/scan). Existing shipping tests pass on SQLite and live PostgreSQL. ListPublicShippingMethods returns only enabled methods. Recorded in receipts/live-run-rev1.md. |
| AC-004 | passed | Live PostgreSQL fresh-migration checkout round-trip observed and independently confirmed: promo (active/disabled/expired), payment (enabled/disabled), shipping (enabled/disabled + public filter), quote (subtotal=1000, discount=100, shipping=120, total=1020), create-order (same totals, stock 10->8, order persisted), disabled config produces no side effects (stock unchanged, order count=1). TestPostgresLiveCommerceBooleanAdaptersAndCheckout passed (0.36s). Independent reviewer replayed the live gate from scratch and confirmed all 4 required tests pass. Recorded in receipts/live-run-rev1.md and receipts/independent-review-rev1.md. |
| AC-005 | passed | TestPostgresLiveCommerceBooleanAdaptersAndCheckout added to RequiredLiveTests in server/tools/postgres-live-gate/main.go. Gate unit tests prove the enrollment and rejection behavior: TestGatePassesExactInventory includes the commerce test event and passes; TestGateRejectsMissingTest confirms a missing commerce test makes the gate exit non-zero; TestGateRejectsSkippedTest confirms a skipped test is rejected; TestGateRejectsFailedTest confirms a failed test is rejected; TestGateRejectsDuplicateRunWithoutPass confirms a duplicate run without pass is rejected. Local live gate (local-postgres-gate) confirms all 4 required tests execute and pass. This AC requires gate enrollment and rejection behavior, not a CI run. Recorded in receipts/live-run-rev1.md. |
| AC-006 | passed | Red-before-green mutation testing executed on live PostgreSQL and independently confirmed: 4 mutations (UpsertPromo int 0/1, ListPaymentMethods int scan, GetActivePromoByCode literal enabled=1, UpsertPaymentMethod int 0/1), each caused TestPostgresLiveCommerceBooleanAdaptersAndCheckout to fail with PostgreSQL type-semantics errors (OID 16 encode failure, bool-to-int scan error, boolean=int operator error). All restored with zero residue, final green pass confirmed. Independent reviewer replayed 3 of 4 mutations from scratch and confirmed each produces the expected type-semantics failure, restoration returns green, and git diff shows zero residue. Recorded in receipts/mutation-testing-rev1.md and receipts/independent-review-rev1.md. |

## Verification Commands Executed

| Command | Result |
|---|---|
| `go test ./server/internal/modules/commerce -count=1` | ok (2.217s) |
| `go test ./server/tools/postgres-live-gate -count=1 -v` | ok (all 6 gate unit tests pass) |
| `go run ./server/tools/local-postgres-gate` | ok (all 4 required live PostgreSQL tests passed) |
| `go run ./server/tools/speccheck` | ok (18 controlled specs, 4 protected changed files) |
| `go run ./server/tools/verify` | ok (archcheck, migration-parity, speccheck, scopecheck, go test ./..., concurrency stress -count=10, go vet) |
| Independent review (subagent replay) | ok (baseline green + 3 mutations red/green + zero residue) |
