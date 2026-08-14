# PostgreSQL Lock Semantics and Execution Evidence Delivery Plan

Change ID: postgres-lock-semantics-and-evidence
Revision: 1
Status: Ready

Normative specification: [`spec.md`](spec.md)

## Repository Reality

| Area | Current state | Required correction |
|---|---|---|
| Migration test | CI applies migrations before `TestPostgresLiveMigrateApplyAndTimestamp`; its second `Apply` observes existing records | Apply into a unique empty namespace and assert newly inserted versions/timestamps |
| Staff lock test | Sequentially demotes owner 1, then rejects demoting owner 2 | Use deterministic overlap across distinct operations and assert one winner, one `ErrLastOwner`, one remaining owner |
| Media lock test | Sequential lifecycle calls execute lock-bearing SQL but do not observe blocking or skip behavior | Hold locks from a separate transaction and observe blocking/timeout and `SKIP LOCKED` selection |
| CI live step | `go test -v -run '^TestPostgresLive' ./server/...` relies on package exit | Parse `go test -json`, propagate runner failure, and require an exact pass inventory with zero missing/skip |
| Verifier unit test | Finds `go test` and `-count=10` independently | Compare the complete concurrency-step argv |
| Prior receipt | Marks AC-002 passed while stating live PostgreSQL was not observed | Keep this change pending until a named real CI run is recorded |

## Scope Lock

- `.github/workflows/ci.yml`
- `server/internal/migrate/migrate.go`
- `server/internal/migrate/postgres_integration_test.go`
- `server/internal/modules/media/postgres_integration_test.go`
- `server/internal/modules/media/store_sql.go`
- `server/internal/modules/staff/postgres_integration_test.go`
- `server/internal/modules/staff/store.go`
- `server/tools/postgres-live-gate/**`
- `server/tools/verify/main_test.go`
- `specs/changes/postgres-lock-semantics-and-evidence/**`

Production behavior in `migrate.go`, `media/store_sql.go`, and `staff/store.go` may be temporarily mutated only for red-before-green evidence and must be restored. A final production edit is limited to a minimal unexported synchronization seam when required for deterministic tests; it must be inert when unset and may not alter domain outcomes.

## Dependency-Ordered Slices

### Slice 1: Fresh Migration and Exact Event Gate
- Isolate the migration test from the pre-migrated CI namespace and assert the exact migration filename set plus newly inserted positive timestamps.
- Add `server/tools/postgres-live-gate` using only the standard library. It runs or consumes `go test -json`, propagates process failure, and validates the exact package/test pass inventory with negative fixtures for missing, skipped, and failing events.
- Covers: REQ-001, REQ-003, AC-001, AC-004.

### Slice 2: Staff and Media Semantic Lock Tests
- Replace the staff test's sequential lock claim with deterministic overlapping owner updates over live PostgreSQL. Preserve any useful sequential last-owner regression under a truthful name.
- Add media lock tests that hold a row from a separate transaction, observe the competing `FOR UPDATE` effect within a bound, and prove `SKIP LOCKED` omits the held row while avoiding duplicate claims.
- Add only the smallest inert test seam if direct database coordination cannot prove deterministic overlap.
- Covers: REQ-002, AC-002, AC-003.

### Slice 3: CI Wiring, Mutation Replay, and Independent Evidence
- Replace the CI live-test command with the exact event gate while preserving the ephemeral service, migrations, and SQLite suite.
- Strengthen `server/tools/verify/main_test.go` to compare the complete concurrency command.
- For each mapped branch, temporarily disable the behavior, record the targeted failure, restore it, and replay green.
- Agy hands off implementation without accepting evidence. Claude reviews the diff and negative cases. Any Claude remediation is replayed by Codex.
- Keep status `Verifying` until the real CI run receipt names the run and shows every required test passed without skip.
- Covers: REQ-003, REQ-004, AC-004, AC-005, AC-006, AC-007.

## Validation Plan

```text
go test ./server/tools/postgres-live-gate -count=20
CI=true go test -run '^TestPostgresLive' ./server/...          # must fail without TEST_DATABASE_URL
go test -json -run '^TestPostgresLive' ./server/...            # live environment; exact gate consumes events
go test ./server/tools/verify -count=20
go run ./server/tools/archcheck
go run ./server/tools/speccheck
go run ./server/tools/scopecheck
go run ./server/tools/verify
```

The live PostgreSQL commands are evidence only when `TEST_DATABASE_URL` targets an isolated test service. Do not print the DSN. No status advances to Accepted from local skips or source inspection.
