# PostgreSQL Execution Gate Delivery Plan

Change ID: postgres-execution-gate
Revision: 2
Status: Applying

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Area | Observed state at proposal time | Delivery implication |
|---|---|---|
| PostgreSQL Live Testing | Existing module test helpers explicitly call `database.Open(..., "sqlite", ...)`; setting `DB_DRIVER=postgres` does not retarget them | Add dedicated environment-gated live-PostgreSQL tests for each identified branch while retaining the SQLite suite |
| PostgreSQL branch inventory | Seven branches require live proof: `database.Open`, `Bind`, migration timestamp insertion, staff `FOR UPDATE`, two media `FOR UPDATE` branches, and media `FOR UPDATE SKIP LOCKED` | Map each branch to an integration test and record consumer-reachability plus independent-review receipts |
| Migration Drift Check | Migration parity is tested only in ad-hoc tests for isolated files (012, 015); `archcheck` currently checks only import boundaries | Add a standalone migration-parity tool and invoke it from verify/CI; do not mix filesystem parity into `archcheck` |
| Speccheck Governance | `server/tools/speccheck/main.go` omits `db/` from `requiresControlledSpec` | Add `"db/"` to protected prefixes in `speccheck` |
| Concurrency Gate | `verify` and CI execute `go test ./...` with default count=1; `TestConcurrentOrderCancellationRestocksOnce` alternates between pass and `ErrInvalidTransition` because loser semantics are unresolved | Quarantine that exact test explicitly in source and repeat only the approved concurrency set; do not claim the quarantine as passing coverage |

## Scope lock

The controlled change applies strictly to the following paths:

- `.github/workflows/ci.yml`
- `server/internal/migrate/postgres_integration_test.go`
- `server/internal/modules/commerce/service_test.go`
- `server/internal/modules/media/postgres_integration_test.go`
- `server/internal/modules/staff/postgres_integration_test.go`
- `server/internal/platform/database/postgres_integration_test.go`
- `server/tools/migration-parity/**`
- `server/tools/speccheck/main.go`
- `server/tools/speccheck/main_test.go`
- `server/tools/verify/main.go`
- `server/tools/verify/main_test.go`
- `specs/changes/postgres-execution-gate/**`

No file under `db/` is authorized. Parity negative tests use temporary fixture directories, and AC-004 is proved by a speccheck unit test rather than by leaving a migration diff in the working tree.

## Dependency-ordered slices

### Slice 1: Speccheck and Migration Parity Governance
- Update `server/tools/speccheck/main.go` to add `"db/"` to `requiresControlledSpec` protected prefix list.
- Add the `db/` case to `server/tools/speccheck/main_test.go` and prove uncovered database changes fail the controlled-spec gate.
- Add a standalone `server/tools/migration-parity` command with temporary-fixture tests for matching, SQLite-only, and PostgreSQL-only migration sets.
- Invoke the standalone parity command from `server/tools/verify`; keep `server/tools/archcheck` unchanged.
- Covers: REQ-002, REQ-003, AC-003, AC-004.

### Slice 2: Ephemeral PostgreSQL CI Service & Live Testing
- Update `.github/workflows/ci.yml` to include `services: postgres` with `postgres:16-alpine`.
- Add CI migration step running `go run ./server/tools/migrate` against the PostgreSQL service.
- Preserve the existing SQLite `go test ./...` step.
- Add environment-gated live-PostgreSQL integration tests under `database`, `migrate`, `staff`, and `media`. Local runs skip only when the dedicated test database URL is absent; CI supplies it and treats any skip as a failure in the integration step.
- Trace the seven identified branches from test entry point through the live PostgreSQL connection and SQL operation; record `receipts/consumer-reachability.md` and an independent replay in `receipts/independent-review.md`.
- Covers: REQ-001, AC-001, AC-002.

### Slice 3: Concurrency Stress Verification
- Add an explicit `t.Skip` at `TestConcurrentOrderCancellationRestocksOnce` with a message naming the unresolved loser-error decision and the follow-up requirement to restore the test. Do not change service behavior or bless either `ErrStaleVersion` or `ErrInvalidTransition`.
- Update `server/tools/verify/main.go` and CI to run each test in the approved concurrency set with `-count >= 10`; add a verifier wiring test so the repeat gate cannot disappear silently.
- Report the cancellation test as quarantined, not passed. Acceptance cannot use its stock assertion as current repeated evidence.
- Verify full toolchain: `go run ./server/tools/archcheck`, `go run ./server/tools/speccheck`, `go run ./server/tools/verify`.
- Covers: REQ-004, AC-005.

## Traceability matrix

| REQ / AC | Slice | Target / Consumer | Verification Method | Expected Evidence |
|---|---|---|---|---|
| REQ-001 | Slice 2 | `.github/workflows/ci.yml`, dedicated integration tests, `server/tools/migrate` | CI workflow execution | Existing SQLite suite plus migration and dedicated live-PostgreSQL suite pass |
| REQ-002 | Slice 1 | `server/tools/migration-parity/**`, `server/tools/verify/main.go` | Standalone tool and verifier execution | Tool fails if migration filenames differ between SQLite and PostgreSQL |
| REQ-003 | Slice 1 | `server/tools/speccheck/main.go` | `speccheck` execution | Tool rejects `db/` edits lacking authorized controlled spec |
| REQ-004 | Slice 3 | `server/internal/modules/commerce/service_test.go`, verify/CI | Source review and repeated execution | Known semantic blocker is explicitly quarantined; approved tests repeat at least 10 times |
| AC-001 | Slice 2 | `.github/workflows/ci.yml` | CI log inspection | PostgreSQL container boots, migrations apply, SQLite suite and live integration suite pass |
| AC-002 | Slice 2 | Four dedicated integration-test files | Live PostgreSQL trace plus two receipts | All seven enumerated dialect/migration/lock branches execute without error and receive independent replay |
| AC-003 | Slice 1 | Standalone migration-parity tool | Temporary-fixture unit tests and command run | Asymmetric migration filenames trigger a descriptive failure without modifying `db/` |
| AC-004 | Slice 1 | `server/tools/speccheck/main.go` and test | Unit test, speccheck run, security-review receipt | `db/` changes are recognized as a protected governance surface |
| AC-005 | Slice 3 | Commerce test, verify/CI repeat set | Source assertion and `-count >= 10` runs | Quarantine is visible; every non-quarantined selected test completes deterministically |
