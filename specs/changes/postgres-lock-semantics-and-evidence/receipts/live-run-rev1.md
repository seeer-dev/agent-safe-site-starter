# Live PostgreSQL run receipt — revision 1

Change ID: postgres-lock-semantics-and-evidence
Reviewer: Devin (GLM-5.2 High)
Date: 2026-08-17
Environment: local embedded PostgreSQL 16 via `go run ./server/tools/local-postgres-gate`
Baseline: 2605afefba0ad079353bc50b530b945ba5d1f682

## Scope

This receipt records the first observed live PostgreSQL execution of all three
required tests in `server/tools/postgres-live-gate/main.go` and the test
isolation fix that made the media and staff tests pass in parallel.

The reviewer also modified the staff and media integration tests (schema
isolation). Per the spec's Reviewer Independence invariant, independent-review
evidence for AC-002, AC-003, and AC-007 requires a separate replay by another
reviewer. This receipt provides consumer-reachability evidence only.

## Test isolation fix

### Problem

`TestPostgresLiveStaffLockActiveOwners` and `TestPostgresLiveMediaLifecycleAndLocks`
both opened connections to the same DSN and called `migrate.Apply` to the default
`public` schema. Go test runs packages in parallel; the two tests started within
1 ms of each other and raced on `CREATE TABLE IF NOT EXISTS order_events`,
producing:

```
ERROR: duplicate key value violates unique constraint "pg_type_typname_nsp_index"
DETAIL: Key (typname, typnamespace)=(order_events, 2200) already exists.
```

The migrate test (`TestPostgresLiveMigrateApplyAndTimestamp`) already used an
isolated schema via `CREATE SCHEMA` + DSN `search_path` + `SetMaxOpenConns(1)`.
The staff and media tests did not.

### Fix

Both tests now create a unique PostgreSQL schema per test run, open both db1 and
db2 with `search_path=<schema>` in the DSN, and set `SetMaxOpenConns(1)` on each
pool to preserve the connection-scoped search_path. The schema is dropped with
`CASCADE` on cleanup. This mirrors the migrate test's isolation pattern.

Files changed:
- `server/internal/modules/staff/postgres_integration_test.go` (added `fmt`, `strings` imports; schema isolation setup)
- `server/internal/modules/media/postgres_integration_test.go` (added `strings` import; schema isolation setup)

No lock-semantic assertions, store SQL, transaction boundaries, or migration
files were changed.

## Observed live run

Command: `go run ./server/tools/local-postgres-gate`
Exit code: 0

### Required test results

| Package | Test | Result | Elapsed |
|---|---|---|---|
| server/internal/migrate | TestPostgresLiveMigrateApplyAndTimestamp | PASS | 0.37s |
| server/internal/modules/staff | TestPostgresLiveStaffLockActiveOwners | PASS | 0.56s |
| server/internal/modules/media | TestPostgresLiveMediaLifecycleAndLocks | PASS | 0.69s |

### Gate output

```
postgres-live-gate: ok (all required live PostgreSQL tests passed)
local-postgres-gate: ok (all live tests passed and child stopped)
```

### Additional live tests observed (not in required inventory)

| Package | Test | Result |
|---|---|---|
| server/internal/platform/database | TestPostgresLiveOpenAndBind | PASS |
| server/internal/platform/database | TestPostgresLiveConstraintViolations | PASS |

## AC coverage

| AC | Evidence kind | Status | Notes |
|---|---|---|---|
| AC-001 | consumer-reachability | passed | Fresh-schema migration timestamp observed on live PostgreSQL 16. All migrations applied in isolated namespace; `applied_unix` values are current. |
| AC-002 | consumer-reachability | passed | Staff cross-row last-owner serialization observed on live PostgreSQL. Two competing demotions overlapped; FOR UPDATE blocked Op 2; exactly one active owner remained. |
| AC-003 | consumer-reachability | passed | Media FOR UPDATE bounded blocking and SKIP LOCKED non-duplication observed on live PostgreSQL. |
| AC-005 | — | pending | Requires observed CI run identity; local run does not satisfy this AC. |
