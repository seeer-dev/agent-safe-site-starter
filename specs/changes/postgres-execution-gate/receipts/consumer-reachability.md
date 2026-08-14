# Consumer Reachability Receipt: PostgreSQL Execution Gate

- **Change ID**: `postgres-execution-gate`
- **Revision**: 2
- **Date**: 2026-08-14
- **Auditor**: Antigravity Implementation Agent

## 1. Traceability of 7 Live PostgreSQL Branches

Each of the 7 previously unexercised PostgreSQL-only branches in `server/internal/` has been wired to a dedicated environment-gated integration test executing against live PostgreSQL:

| Branch # | File & Lines | Implementation Code | Test File & Entry Point | Live Execution Trace |
|---|---|---|---|---|
| 1 | `server/internal/platform/database/database.go:33-35` | `case "postgres": sqlDriver = "pgx"; dialect = Postgres` | `server/internal/platform/database/postgres_integration_test.go:TestPostgresLiveOpenAndBind` | Opens pgx connection pool and successfully pings live PostgreSQL container |
| 2 | `server/internal/platform/database/database.go:58-73` | `Bind(Postgres, ...)` query placeholder transformation `?` -> `$1, $2...` | `server/internal/platform/database/postgres_integration_test.go:TestPostgresLiveOpenAndBind` | Translates `SELECT ?::int + ?::int` to `SELECT $1::int + $2::int` and executes query returning computed sum |
| 3 | `server/internal/migrate/migrate.go:59-61` | `INSERT INTO schema_migrations ... EXTRACT(EPOCH FROM NOW())::BIGINT` | `server/internal/migrate/postgres_integration_test.go:TestPostgresLiveMigrateApplyAndTimestamp` | Executes all 16 migrations on live PostgreSQL and verifies `applied_unix` timestamps recorded from `EXTRACT(EPOCH FROM NOW())` |
| 4 | `server/internal/modules/staff/store.go:179-182` | `SELECT id FROM staff_members WHERE status = 'active' AND role_label = 'owner' FOR UPDATE` | `server/internal/modules/staff/postgres_integration_test.go:TestPostgresLiveStaffLockActiveOwners` | Opens transaction, acquires `FOR UPDATE` row lock on active owner rows, and validates last-owner guard |
| 5 | `server/internal/modules/media/store_sql.go:40-42` | `SELECT state, ... FROM media_assets WHERE object_key = ? FOR UPDATE` | `server/internal/modules/media/postgres_integration_test.go:TestPostgresLiveMediaLifecycleAndLocks` | Opens transaction and acquires `FOR UPDATE` lock during asset reservation |
| 6 | `server/internal/modules/media/store_sql.go:102-104` | `SELECT state, reservation_token FROM media_assets WHERE object_key = ? FOR UPDATE` | `server/internal/modules/media/postgres_integration_test.go:TestPostgresLiveMediaLifecycleAndLocks` | Opens transaction and acquires `FOR UPDATE` lock during asset activation |
| 7 | `server/internal/modules/media/store_sql.go:196-198` | `SELECT ma.object_key ... FOR UPDATE SKIP LOCKED` | `server/internal/modules/media/postgres_integration_test.go:TestPostgresLiveMediaLifecycleAndLocks` | Calls `ClaimEligible` which executes `listEligible(lock=true)` with `FOR UPDATE SKIP LOCKED` to lock candidate assets for garbage collection |

## 2. Environment Gating Strategy

- **Local Development**: When `TEST_DATABASE_URL` is unset, all live integration tests skip gracefully via `t.Skip("TEST_DATABASE_URL not set; skipping live PostgreSQL integration test")`, preserving fast, dependency-free local developer workflows with SQLite.
- **CI Environment**: `.github/workflows/ci.yml` provides `TEST_DATABASE_URL` connected to the ephemeral `services: postgres` instance.
