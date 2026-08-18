# Mutation testing receipt — revision 1

Change ID: postgres-lock-semantics-and-evidence
Reviewer: Devin (GLM-5.2 High)
Date: 2026-08-17
Environment: local embedded PostgreSQL 16 via `go run ./server/tools/local-postgres-gate`
Baseline: 2605afefba0ad079353bc50b530b945ba5d1f682

## Scope

This receipt records red-before-green mutation testing for AC-007 on live
PostgreSQL. Each mutation temporarily disabled the guarded behavior, the
targeted assertion was observed failing, the mutation was restored, and a
final green pass confirmed zero residue.

The reviewer also modified the staff and media integration tests (schema
isolation fix documented in receipts/live-run-rev1.md). Per the spec's
Reviewer Independence invariant, independent-review evidence for AC-007
requires a separate replay by another reviewer.

## Mutation table

| # | Target | File | Mutation | Observed failure | Restored |
|---|---|---|---|---|---|
| 1 | Migration timestamp (AC-001/REQ-001) | `server/internal/migrate/migrate.go:60` | `EXTRACT(EPOCH FROM NOW())::BIGINT` → `0::BIGINT` | `TestPostgresLiveMigrateApplyAndTimestamp` FAIL: `migration "001_init.sql" applied_unix (0) is older than test start time (1786989102); must be newly generated` | yes, zero residue |
| 2 | Staff FOR UPDATE (AC-002/REQ-002) | `server/internal/modules/staff/store.go:185` | Removed `FOR UPDATE` from `SELECT id FROM staff_members WHERE status = 'active' AND role_label = 'owner'` | `TestPostgresLiveStaffLockActiveOwners` FAIL: `Op 2 completed prematurely before Tx1 released lock; active owner row lock failed to block competing operation` | yes, zero residue |
| 3 | Media FOR UPDATE (AC-003/REQ-002) | `server/internal/modules/media/store_sql.go:41` | `query += " FOR UPDATE"` → `query += ""` | `TestPostgresLiveMediaLifecycleAndLocks` FAIL: `expected deadline exceeded while waiting for SELECT ... FOR UPDATE row lock, got: media asset metadata mismatch` | yes, zero residue |
| 4 | Media SKIP LOCKED (AC-003/REQ-002) | `server/internal/modules/media/store_sql.go:197` | `FOR UPDATE SKIP LOCKED` → `FOR UPDATE` | `TestPostgresLiveMediaLifecycleAndLocks` FAIL (30.09s): `ClaimEligible with locked row failed: list media gc candidates: timeout: context deadline exceeded` | yes, zero residue |

## Residue verification

After all four mutations were restored:

```
git diff -- server/internal/migrate/migrate.go server/internal/modules/staff/store.go server/internal/modules/media/store_sql.go
```

Output: no changes (only CRLF line-ending warnings).

Final green pass: `go run ./server/tools/local-postgres-gate` → exit 0

```
TestPostgresLiveMigrateApplyAndTimestamp    --- PASS (0.37s)
TestPostgresLiveStaffLockActiveOwners       --- PASS (0.56s)
TestPostgresLiveMediaLifecycleAndLocks      --- PASS (0.69s)
postgres-live-gate: ok (all required live PostgreSQL tests passed)
local-postgres-gate: ok (all live tests passed and child stopped)
```

## Mutation specificity

Each mutation caused only the targeted test to fail while the other two
continued to pass, confirming that the assertions are specific to the
guarded behavior they name:

- Mutation 1 (timestamp): only migrate test failed; staff and media passed.
- Mutation 2 (staff FOR UPDATE): only staff test failed; migrate and media passed.
- Mutation 3 (media FOR UPDATE): only media test failed; migrate and staff passed.
- Mutation 4 (media SKIP LOCKED): only media test failed (30s timeout); migrate and staff passed.
