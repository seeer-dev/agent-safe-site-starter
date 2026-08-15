# Local live run receipt — revision 1

Date: 2026-08-15
Reviewer: Codex (independent of Agy implementation)

## Command

```powershell
go run ./server/tools/local-postgres-gate
```

## Observed result

The command exited `0` and reported:

- `local-postgres-gate: starting ephemeral PostgreSQL 16 on 127.0.0.1:5433...`
- `local-postgres-gate: postgres ready; invoking postgres-live-gate...`
- `postgres-live-gate: ok (all required live PostgreSQL tests passed)`
- `local-postgres-gate: ok (all live tests passed and child stopped)`

The required live tests executed and passed:

- `TestPostgresLiveMigrateApplyAndTimestamp`
- `TestPostgresLiveMediaLifecycleAndLocks`
- `TestPostgresLiveOpenAndBind`
- `TestPostgresLiveConstraintViolations`
- `TestPostgresLiveStaffLockActiveOwners`

This is a consumer-reachability receipt for AC-001 and AC-002. The existing `postgres-live-gate` remained the source of the required-test inventory; the launcher did not duplicate it.
