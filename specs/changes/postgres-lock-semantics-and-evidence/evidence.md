# Evidence

## Delivery Status

Revision 1 is Applying against baseline `f8053f8ea3ac640586384e24667933e5f556b689`. Local gates (AC-004, AC-006) are verified and passed. Live PostgreSQL and CI execution evidence (AC-001, AC-002, AC-003, AC-005, AC-007) remain pending until real PostgreSQL CI runs.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | pending | Pending live PostgreSQL fresh-schema migration execution. |
| REQ-002 | pending | Pending live PostgreSQL lock semantic execution across staff and media. |
| REQ-003 | pending | Pending live PostgreSQL exact test inventory execution in CI. |
| REQ-004 | pending | Pending mutation-sensitive live verification and exact verifier assertion. |
| AC-001 | pending | Pending fresh-schema consumer reachability evidence for revision 1 on live PostgreSQL. |
| AC-002 | pending | Pending live PostgreSQL staff-lock reachability and independent review for revision 1. |
| AC-003 | pending | Pending live PostgreSQL media-lock reachability and independent review for revision 1. |
| AC-004 | passed | Exact named-test gate implemented in server/tools/postgres-live-gate. Reviewer verified both rejection branches by mutation rather than by report: with no TEST_DATABASE_URL the gate exits 1 naming all three required tests as skipped, and renaming TestPostgresLiveStaffLockActiveOwners produced a distinct missing-from-execution-stream diagnostic; the rename was restored with zero residue. Unit coverage: TestGateRejectsMissingTest, TestGateRejectsSkippedTest, TestGateRejectsFailedTest, TestGatePassesExactInventory. Recorded in receipts/reviewer-acceptance-rev1.md. |
| AC-005 | pending | Pending observed live CI run, consumer reachability, and independent review for revision 1. |
| AC-006 | passed | Verifier concurrency step wiring assertion in server/tools/verify/main_test.go compares the complete argument vector including packages and repeat count. Reviewer mutation-verified it: changing -count=10 to -count=5 in verify/main.go turned TestVerifyConcurrencyStepExactArgv red, and restoration returned it to green with zero residue. Recorded in receipts/reviewer-acceptance-rev1.md. |
| AC-007 | pending | Pending mutation-sensitive independent review for revision 1. |

## Known Historical Gap

The superseded receipt under `postgres-execution-gate` explicitly stated that live PostgreSQL had not been observed while its AC-002 record was marked passed. Its migration test also ran after the shared CI migration step, and its staff/media lock tests did not observe concurrency semantics. This proposal does not rewrite that immutable history; it replaces the affected evidence claim with falsifiable tests and an observed live-run requirement.
