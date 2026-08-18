# Evidence

## Delivery Status

Revision 1 is `Verifying` against baseline `f8053f8ea3ac640586384e24667933e5f556b689`. Local live gates passed on embedded PostgreSQL 16 via `go run ./server/tools/local-postgres-gate`: AC-001 (consumer-reachability), AC-004, AC-006 are fully passed. AC-002, AC-003 have consumer-reachability observed but independent-review still pending because the reviewer (Devin) modified the staff and media integration tests (schema isolation fix). AC-007 mutation testing was executed but independent-review is pending for the same reason. AC-005 remains pending until a real PostgreSQL CI run is recorded with run identity.

## Test isolation fix

The staff and media PostgreSQL integration tests raced on `CREATE TABLE IF NOT EXISTS order_events` when run in parallel (different packages, same default schema). Both tests now create a unique PostgreSQL schema per run, use `search_path` in the DSN, and set `SetMaxOpenConns(1)` on each pool. This mirrors the migrate test's existing isolation pattern. No lock-semantic assertions, store SQL, transaction boundaries, or migration files were changed. Details in receipts/live-run-rev1.md.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Live PostgreSQL fresh-schema migration execution observed. TestPostgresLiveMigrateApplyAndTimestamp passed (0.37s) on embedded PostgreSQL 16 via local-postgres-gate. All migrations applied in isolated namespace; applied_unix values are current. Recorded in receipts/live-run-rev1.md. |
| REQ-002 | passed | Live PostgreSQL lock semantic execution observed across staff and media. TestPostgresLiveStaffLockActiveOwners passed (0.56s) and TestPostgresLiveMediaLifecycleAndLocks passed (0.69s) on embedded PostgreSQL 16. Recorded in receipts/live-run-rev1.md. |
| REQ-003 | pending | AC-005 still pending: requires observed live CI run identity. The gate mechanism (AC-004) is proven, but CI execution has not been observed. |
| REQ-004 | passed | Mutation-sensitive live verification and exact verifier assertion. AC-006 was previously passed (verifier exact argv). AC-007 mutation testing executed on live PostgreSQL; independent-review pending. Recorded in receipts/mutation-testing-rev1.md. |
| AC-001 | passed | Fresh-schema consumer reachability evidence on live PostgreSQL 16. TestPostgresLiveMigrateApplyAndTimestamp observed passing with current timestamps in isolated schema. Recorded in receipts/live-run-rev1.md. |
| AC-002 | pending | Consumer-reachability observed on live PostgreSQL 16: two competing owner demotions overlapped; FOR UPDATE blocked Op 2; exactly one active owner remained; losing operation returned canonical ErrLastOwner. Independent-review still pending: reviewer modified the staff integration test (schema isolation fix), so another reviewer must replay it. Recorded in receipts/live-run-rev1.md. |
| AC-003 | pending | Consumer-reachability observed on live PostgreSQL 16: FOR UPDATE bounded blocking confirmed (Tx2 blocked on locked row, returned DeadlineExceeded); SKIP LOCKED non-blocking confirmed (ClaimEligible skipped held row, claimed unlocked row, no duplication). Independent-review still pending: reviewer modified the media integration test (schema isolation fix), so another reviewer must replay it. Recorded in receipts/live-run-rev1.md. |
| AC-004 | passed | Exact named-test gate implemented in server/tools/postgres-live-gate. Reviewer verified both rejection branches by mutation rather than by report: with no TEST_DATABASE_URL the gate exits 1 naming all three required tests as skipped, and renaming TestPostgresLiveStaffLockActiveOwners produced a distinct missing-from-execution-stream diagnostic; the rename was restored with zero residue. Unit coverage: TestGateRejectsMissingTest, TestGateRejectsSkippedTest, TestGateRejectsFailedTest, TestGatePassesExactInventory. Recorded in receipts/reviewer-acceptance-rev1.md. |
| AC-005 | pending | Pending observed live CI run, consumer reachability, and independent review. Local live run does not satisfy this AC; spec requires run identity from repository CI. |
| AC-006 | passed | Verifier concurrency step wiring assertion in server/tools/verify/main_test.go compares the complete argument vector including packages and repeat count. Reviewer mutation-verified it: changing -count=10 to -count=5 in verify/main.go turned TestVerifyConcurrencyStepExactArgv red, and restoration returned it to green with zero residue. Recorded in receipts/reviewer-acceptance-rev1.md. |
| AC-007 | pending | Red-before-green mutation testing executed on live PostgreSQL by reviewer: 4 mutations (migration timestamp to 0, staff FOR UPDATE removed, media FOR UPDATE removed, media SKIP LOCKED removed), each caused only the targeted assertion to fail, all restored with zero residue, final green pass confirmed. Independent-review still pending: reviewer modified the staff and media integration tests (schema isolation fix), so another reviewer must replay the mutation testing. Recorded in receipts/mutation-testing-rev1.md. |

## Known Historical Gap

The superseded receipt under `postgres-execution-gate` explicitly stated that live PostgreSQL had not been observed while its AC-002 record was marked passed. Its migration test also ran after the shared CI migration step, and its staff/media lock tests did not observe concurrency semantics. This proposal does not rewrite that immutable history; it replaces the affected evidence claim with falsifiable tests and an observed live-run requirement.

## Remaining blockers

1. **AC-005**: Requires observed CI run identity. The repository has no git remote and no `gh` CLI configured locally. A CI run requires pushing to a remote with PostgreSQL service configured.
2. **Independent-review for AC-002, AC-003, AC-007**: The reviewer (Devin) modified the staff and media integration tests (schema isolation fix). Per the spec's Reviewer Independence invariant, a separate replay by another reviewer is required before these ACs can fully close.
