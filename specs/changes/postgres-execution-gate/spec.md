# PostgreSQL Execution Gate Specification

Change ID: postgres-execution-gate
Revision: 2
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner issued plain apply instruction on 2026-08-14 to implement revision 2.
Repository baseline: f48b13774be42b08b226af894d6804111d4c4316
Supersedes: none

## Outcome

Establish real PostgreSQL execution, migration parity verification, schema governance, and an honest concurrency stress gate. Dedicated live-PostgreSQL integration tests execute the identified PostgreSQL-only branches; the existing SQLite suite remains intact. The change prevents migration drift, brings database schema changes under controlled-spec governance, and repeats only concurrency tests whose expected semantics are currently approved.

## Scope

In scope:
- GitHub Actions CI workflow update (`.github/workflows/ci.yml`) to provision an ephemeral PostgreSQL container service (`services: postgres`) without requiring production credentials.
- Application of all PostgreSQL migrations from `db/migrations/postgres/*.sql` via `server/tools/migrate` against the ephemeral PostgreSQL instance in CI.
- Dedicated integration tests that open the ephemeral PostgreSQL instance and exercise the identified PostgreSQL-only branches; existing module tests continue to use SQLite.
- A standalone migration-parity tool enforcing identical migration filename sets and version numbering between `db/migrations/sqlite/` and `db/migrations/postgres/`, invoked by repository verification and CI without changing migration files.
- Addition of `db/` to the protected prefix list in `server/tools/speccheck/main.go` so schema/migration edits require an authorized controlled change.
- Concurrency verification enhancement in `server/tools/verify/main.go` and CI to execute the approved concurrency test set with repeated iterations (`-count >= 10`).
- Explicit source-level quarantine of `TestConcurrentOrderCancellationRestocksOnce`, with a skip message pointing to the unresolved loser-error product decision. The test is not represented as deterministic while quarantined.

Out of scope:
- Product semantics fixes for the loser error in `TestConcurrentOrderCancellationRestocksOnce` (which requires product owner decision on whether concurrent losers return `ErrStaleVersion` or `ErrInvalidTransition`).
- Database schema or migration content changes. The parity tool reads the current directories and tests against temporary fixtures; this change does not authorize edits under `db/`.
- Removing SQLite support or changing local-first SQLite development workflows.
- Introducing third-party ORMs or external migration engines.

## Decisions and Invariants

- Ephemeral CI Database: PostgreSQL in CI is provisioned strictly as an ephemeral GitHub Actions service container (`postgres:16-alpine`), using non-secret local test credentials (`test_user`/`test_password`). No production secrets or external database connections are used.
- Symmetric Dual-Driver Parity: Every migration version must exist in both `db/migrations/sqlite/` and `db/migrations/postgres/` with identical filenames. A standalone migration-parity tool, invoked by `verify` and CI, rejects mismatched sets. `archcheck` remains responsible only for architectural import boundaries.
- Schema Governance Gate: Changes to `db/` are protected under `speccheck`, aligning tool enforcement with the governance rule in `AGENTS.md` requiring controlled specs for database schema changes.
- Local-First Invariant: Local development continues to default to SQLite without requiring a local PostgreSQL daemon. PostgreSQL execution is required in CI and optional locally via environment variables (`DB_DRIVER=postgres`, `DATABASE_URL`).
- Live-Branch Evidence: Setting `DB_DRIVER=postgres` does not retarget existing tests because their helpers open SQLite explicitly. Dedicated environment-gated integration tests MUST connect to the ephemeral service and prove the seven identified branches: PostgreSQL `database.Open`, `database.Bind`, PostgreSQL migration timestamp insertion, staff owner-row `FOR UPDATE`, both media asset `FOR UPDATE` branches, and media GC `FOR UPDATE SKIP LOCKED`.
- Stress Concurrency Gate: Each test in the approved concurrency set must execute with `-count >= 10`. A test blocked by unresolved product semantics must be quarantined explicitly in source, linked to the decision blocker, and excluded from deterministic claims rather than being silently omitted by a runner pattern.

## Requirements

### REQ-001: Ephemeral PostgreSQL CI Service and Live Execution
GitHub Actions CI workflows MUST provision an ephemeral PostgreSQL service, apply the entire `db/migrations/postgres/` migration sequence, preserve the existing SQLite test suite, and execute dedicated integration tests against the live PostgreSQL database, ensuring all seven identified PostgreSQL-specific branches are actively exercised.

#### AC-001: CI Ephemeral PostgreSQL Migration and Execution
- GIVEN a pull request or push trigger in `.github/workflows/ci.yml`
- WHEN the CI workflow executes
- THEN it MUST start a PostgreSQL service container, run `go run ./server/tools/migrate` to apply all migrations under `db/migrations/postgres/`, run the existing SQLite suite, and execute the dedicated live-PostgreSQL integration tests with exit code 0.

#### AC-002: PostgreSQL Dialect and Lock Branch Execution
- GIVEN the Go backend test suite executing against PostgreSQL
- WHEN database operations and transactional locks run
- THEN the evidence MUST trace live execution of PostgreSQL `database.Open`, `database.Bind` placeholder substitution, PostgreSQL migration timestamp insertion, staff owner-row `FOR UPDATE`, both media asset `FOR UPDATE` branches, and media GC `FOR UPDATE SKIP LOCKED`, without SQL syntax or constraint errors.

### REQ-002: Mechanical Migration Parity Verification
The repository verification tooling MUST mechanically verify that the set of migration filenames in `db/migrations/sqlite/` exactly matches the set in `db/migrations/postgres/`, failing the build if any asymmetry exists.

#### AC-003: Migration Filename Parity Gate
- GIVEN the migration directories `db/migrations/sqlite/` and `db/migrations/postgres/`
- WHEN the standalone migration-parity tool, repository verification, or CI runs
- THEN it MUST compare the sorted lists of filenames in both directories and fail with a clear descriptive error if any migration file is present in one directory but missing in the other.

### REQ-003: Speccheck Governance Protection for Database Schema
The controlled-spec verification tool (`server/tools/speccheck`) MUST classify `db/` as a protected surface, requiring that any modification to database migrations or schema files is governed by an authorized controlled change.

#### AC-004: Speccheck Protection for `db/` Directory
- GIVEN any modification, addition, or deletion of files under `db/`
- WHEN `go run ./server/tools/speccheck` runs
- THEN `speccheck` MUST treat `db/` as a protected path and reject changes not covered by an authorized controlled specification.

### REQ-004: Concurrency Verification Stress Testing and Explicit Quarantine
Verification tooling and CI MUST execute each test in the approved concurrency set with repeated iterations (`-count >= 10`). Tests blocked by unresolved product semantics MUST be explicitly quarantined in source with a decision reference and MUST NOT be counted as deterministic coverage.

#### AC-005: Repeated Concurrency Test Iterations
- GIVEN the approved concurrency test set and the known loser-error ambiguity in `TestConcurrentOrderCancellationRestocksOnce`
- WHEN `go run ./server/tools/verify` or CI runs
- THEN every approved test MUST execute with `-count >= 10` and complete deterministically, while `TestConcurrentOrderCancellationRestocksOnce` MUST contain an explicit `t.Skip` message pointing to the unresolved product decision and MUST be reported as quarantined rather than passed coverage.

## Amendments

### Revision 2 (2026-08-14)
- Clarified that setting `DB_DRIVER=postgres` does not retarget existing module tests because helpers hardcode SQLite; introduced dedicated live-PostgreSQL integration tests.
- Replaced mixing parity checks into `archcheck` with a standalone `server/tools/migration-parity` tool invoked by `verify` and CI.
- Removed `db/**` from authorized implementation paths.
- Added explicit source-level quarantine (`t.Skip`) for `TestConcurrentOrderCancellationRestocksOnce`.
- Reallocated receipt requirements: AC-002 requires `consumer-reachability` and `independent-review`; AC-004 requires `security-review`.
