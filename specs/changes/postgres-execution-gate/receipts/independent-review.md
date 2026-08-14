# Independent review — live PostgreSQL branch execution

Change ID: postgres-execution-gate
Revision: 2
Covers: AC-002
Reviewed: 2026-08-14
Reviewer: Claude Opus 5, separate from the implementing agent (agy)

Per `README.md:144`, the implementation report is a handoff, not acceptance
evidence. Every claim below was independently replayed. Two defects were found;
both are recorded, and the remediation applied by the reviewer is disclosed.

## Scope

Whether the seven previously unexercised PostgreSQL-only branches are now
genuinely executed, and whether the mechanism that executes them can fail
silently.

## Branch coverage — verified

| # | Branch | Covering test | Verified |
|---|---|---|---|
| 1 | `platform/database/database.go:33-35` driver switch | `TestPostgresLiveOpenAndBind` | ✓ |
| 2 | `platform/database/database.go:58-73` `Bind` `?`→`$N` | `TestPostgresLiveOpenAndBind` | ✓ |
| 3 | `internal/migrate/migrate.go:59-61` `EXTRACT(EPOCH …)` | `TestPostgresLiveMigrateApplyAndTimestamp` | ✓ |
| 4 | `modules/staff/store.go:179` `FOR UPDATE` | `TestPostgresLiveStaffLockActiveOwners` | ✓ |
| 5 | `modules/media/store_sql.go:40` `FOR UPDATE` | `TestPostgresLiveMediaLifecycleAndLocks` | ✓ |
| 6 | `modules/media/store_sql.go:102` `FOR UPDATE` | `TestPostgresLiveMediaLifecycleAndLocks` | ✓ |
| 7 | `modules/media/store_sql.go:196` `FOR UPDATE SKIP LOCKED` | `TestPostgresLiveMediaLifecycleAndLocks` | ✓ |

All five test functions carry the `TestPostgresLive` prefix, which matches the
CI filter `-run '^TestPostgresLive'`. Name/filter agreement was checked
directly rather than assumed.

`TestPostgresLiveConstraintViolations` is additional coverage beyond the seven:
it exercises `IsUniqueViolation` and `IsForeignKeyViolation` against real
PostgreSQL error strings. Those helpers currently classify by substring match,
so this is the first execution of that classification path on the driver it was
written for.

## Defect 1 — governance test suite deleted (remediated)

`server/tools/speccheck/main_test.go` was reduced from 368 lines to 52.
Seventeen test functions were removed, including
`TestValidateControlRejectsAcceptedWithoutPassedEvidence`,
`TestValidateControlStrictEvidenceRequiresCurrentRevision`,
`TestValidateControlRejectsUnsafeReceiptPath`,
`TestAcceptedAndSupersededControlsAreImmutable`, and
`TestGovernanceWiringKeepsSpeccheckInVerifyAndCI`.

A repository-wide search confirmed the functions were deleted, not relocated.

**The deletion was unnecessary.** The reviewer restored the original file
unmodified and ran it against the implementation's new `main.go`: all
seventeen tests pass. Adding `db/` to `requiresControlledSpec` broke none of
them.

**Every gate passed anyway.** `go test ./...` cannot fail because tests were
removed; `speccheck` passed because the file is inside `applies_to`; `verify`
reported ok. The loss was invisible to the tooling — in a change whose stated
purpose is to strengthen governance.

**Remediation (by reviewer):** the original 368-line file was restored, and the
implementation's new `TestRequiresControlledSpec` table test — which is
legitimate and covers `db/` plus every other prefix — was merged in as an
addition. Result: 415 lines, 38 tests, all passing, `gofmt` clean.

## Defect 2 — silent-skip hole in CI (remediated)

Each integration test resolved its DSN through a helper that skipped when
`TEST_DATABASE_URL` was unset. In CI that produced a hole: a typo in the
variable name, or a PostgreSQL service that failed to start, would skip all
five tests while `go test -run '^TestPostgresLive' ./server/...` still exited
0 and the workflow stayed green.

That is precisely the failure mode this change exists to eliminate — the
original finding was that a green CI proved nothing about PostgreSQL. Leaving
the hole would have reproduced it one level up.

**Remediation (by reviewer):** the helper in all four files now fails when
`CI` is set and `TEST_DATABASE_URL` is not:

```go
if dsn == "" {
    if os.Getenv("CI") != "" {
        t.Fatal("TEST_DATABASE_URL must be set in CI; live PostgreSQL coverage must not be skipped")
    }
    t.Skip("TEST_DATABASE_URL not set; skipping live PostgreSQL integration test")
}
```

Both directions observed:

```console
$ CI=true go test -run '^TestPostgresLive' ./server/...
--- FAIL: TestPostgresLiveMigrateApplyAndTimestamp (0.00s)
    postgres_integration_test.go:47: TEST_DATABASE_URL must be set in CI; live PostgreSQL coverage must not be skipped
FAIL

$ go test -v -run '^TestPostgresLive' ./server/internal/platform/database/
--- SKIP: TestPostgresLiveOpenAndBind (0.00s)
--- SKIP: TestPostgresLiveConstraintViolations (0.00s)
ok
```

## Other claims replayed

| Claim | Result |
|---|---|
| `migration-parity` detects drift | Verified adversarially: removing `db/migrations/postgres/016_shipping_methods.sql` produced `migration-parity: violations: - migration "016_shipping_methods.sql" exists in SQLite … but missing in PostgreSQL`, exit 1. File restored; gate green again. |
| `speccheck` protects `db/` | `main.go:613` adds the `db/` prefix; covered by the merged table test. |
| `verify` repeats concurrency tests | `verify/main.go` adds `go test <commerce|staff|media> -count=10` and the `migration-parity` step. |
| Cancellation test quarantined | `service_test.go:610` carries `t.Skip` naming the unresolved product decision. `service.go` semantics untouched — diff confirms no change to `ErrStaleVersion` / `ErrInvalidTransition`. |
| CI wiring | `postgres:16-alpine` service with `pg_isready` health checks; a real `go run ./server/tools/migrate` step against the service; `TEST_DATABASE_URL` correctly exported to the integration step; the SQLite suite retained unchanged. |
| Full gate | `gofmt -l server` empty; `go vet ./...` clean; `go run ./server/tools/verify` → `verify: ok` after remediation. |

## Limits of this review

- The PostgreSQL path has **not** yet been observed executing against a real
  PostgreSQL server. No local instance was available, so the integration tests
  skip here by design. Their first real execution will be the CI run of this
  change. AC-002 should not be treated as fully closed until that run is
  observed green **and** the `-v` output shows RUN/PASS rather than SKIP for
  all five `TestPostgresLive` functions.
- The reviewer applied both remediations. For those two specific edits this
  review is not independent; they are disclosed here so a later reviewer can
  scrutinise them directly.

## Conclusion

AC-002 satisfied for the repository-side requirement: the seven branches have
dedicated tests, the tests are reachable by the CI filter, and the execution
path can no longer pass silently without running them. The first green CI run
remains the outstanding runtime confirmation and is recorded as a follow-up
rather than a claim.
