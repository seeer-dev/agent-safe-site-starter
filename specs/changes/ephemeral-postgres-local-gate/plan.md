# Implementation plan

## Controlled scope

- `go.mod`
- `go.sum`
- `server/tools/local-postgres-gate/**`
- `docs/environment-configuration.md`
- `specs/changes/ephemeral-postgres-local-gate/**`

## Implementation expansion context

Proposal revision: 1
Proposal status: Applying
Repository baseline: 7e5aa90b92d23c8c316d44ca935be7af8d150a1c
Observed HEAD: 7e5aa90b92d23c8c316d44ca935be7af8d150a1c

Pre-existing dirty paths:

- `.github/workflows/ci.yml`, PostgreSQL integration tests, `server/tools/postgres-live-gate/`, and `server/tools/verify/main_test.go` - belong to `postgres-lock-semantics-and-evidence`; preserve.
- `specs/changes/minimal-cart-integration/**` - current verification evidence; preserve.
- `specs/changes/harden-implementation-handoffs/**` - separate Draft; preserve.

Packet ID mapping:

- Slice 1 -> S01
- Slice 2 -> S02
- Slice 3 -> S03

### Packet S01: Pinned ephemeral PostgreSQL launcher

Status: complete
Covers: REQ-001, REQ-003, AC-001, AC-003
Hard dependencies: none
Outcome: A root-level Go command starts PostgreSQL 16 on loopback and injects a non-persistent test DSN only into the existing live gate.
Safe failure / rollback: Failure before readiness does not invoke the gate; temporary paths are cleaned and no application configuration is written.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Existing live-test owner | `server/tools/postgres-live-gate/main.go` | `RunLiveGate` | Runs fixed `go test -json -run ^TestPostgresLive ./server/...` and rejects missing/skipped required tests. |
| CI version contract | `.github/workflows/ci.yml` | `services.postgres.image` | CI uses `postgres:16-alpine` and passes `TEST_DATABASE_URL` only to the live-gate step. |
| Driver contract | `server/internal/platform/database/database.go` | `Open` postgres case | `postgres` resolves to the pgx SQL driver. |
| Dependency manifest | `go.mod` | direct require block | Existing direct dependencies include pgx but no embedded PostgreSQL launcher. |
| Local configuration boundary | `docs/environment-configuration.md` | Local development / deployment ownership | Local dotenv profiles are ignored and production values must not be stored in the repository. |

Read set:

- `go.mod` and `go.sum` - preserve direct dependency conventions.
- `server/tools/postgres-live-gate/main.go` and tests - reuse its exact gate rather than duplicate its inventory logic.
- `.github/workflows/ci.yml` - retain PostgreSQL 16 parity without changing CI.
- `docs/environment-configuration.md` - preserve dotenv and secret boundaries.

Modify set:

- `go.mod`, `go.sum` - add one pinned direct launcher dependency after approval.
- `server/tools/local-postgres-gate/main.go` - fixed-argument launcher, port validation, loopback-only PostgreSQL config, temporary paths, child environment, and deferred stop/cleanup.
- `server/tools/local-postgres-gate/main_test.go` - dependency-injected lifecycle tests; no network download or real database required for unit checks.

Must not modify:

- `.env*`, `server/internal/config/**`, `server/internal/platform/database/**` - the tool must not change application configuration or database ownership.
- `.github/workflows/ci.yml`, migration/test code, and `server/tools/postgres-live-gate/**` - existing CI/live-test semantics remain authoritative.

#### Behavior contract

Before:

- The live gate skips locally unless a user provisions `TEST_DATABASE_URL` manually.

After:

- `go run ./server/tools/local-postgres-gate` owns only a temporary PostgreSQL 16 process and runs the existing gate with a child-scoped DSN.

Preserved invariants:

- CI remains PostgreSQL 16 and unchanged.
- `postgres-live-gate` remains the sole authority for required live-test inventory.
- No `.env`, process-wide persistent environment, migration, or application database default changes.

Inputs and authority:

- The launcher owns the fixed local port or a validated explicit port flag.
- PostgreSQL binds `127.0.0.1` only.
- `TEST_DATABASE_URL` exists only in the child gate process environment.

Errors and edge states:

- Occupied/invalid port, download/startup failure, and gate failure produce non-zero exit and an actionable error without leaking a DSN password.
- No shell is invoked and no arbitrary command/database argument is accepted.

#### Ordered edits

1. `go.mod` / `go.sum` - direct dependency block
   - Change: add a pinned `github.com/fergusstrange/embedded-postgres` version compatible with the repository Go toolchain; pin PostgreSQL major version 16 in code.
   - Constraint: do not use `go get -u` or add a runtime installer script.
   - Result: the launcher has a reviewable Go module lock entry.
2. `server/tools/local-postgres-gate/main.go` - `main`, `Run`, and launcher configuration builder
   - Change: create temp runtime/data paths outside the repository, use a durable user-cache path only for downloaded binaries, bind loopback, construct a redacted process-scoped DSN, run `go run ./server/tools/postgres-live-gate` through `exec.CommandContext`, and defer `Stop` plus cleanup.
   - Constraint: no shell composition, no application dotenv read/write, no CI changes, no logged password.
   - Result: one command owns startup, gate invocation, shutdown, and cleanup.
3. `server/tools/local-postgres-gate/main_test.go` - lifecycle seams
   - Change: use fakes for database startup/stop and command execution to prove startup failure does not execute the gate, success/failure both stop the child, only the child receives the DSN, and cleanup paths are attempted.
   - Constraint: tests do not download or launch PostgreSQL.
   - Result: cleanup and command-boundary behavior fail before implementation and pass after restoration.

#### Integration trace

`go run ./server/tools/local-postgres-gate`
  -> validated local port and OS temp/user-cache paths
  -> embedded PostgreSQL 16 bound to `127.0.0.1`
  -> child-only `TEST_DATABASE_URL`
  -> `go run ./server/tools/postgres-live-gate`
  -> `RunLiveGate`
  -> exact `TestPostgresLive*` inventory
  -> gate result and child shutdown

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Lifecycle does not need a real download | repository root | `go test ./server/tools/local-postgres-gate -count=1` | named fake lifecycle tests | pass | startup/gate error | fake start or gate error | `git status --short`; OS temp path absent |
| Command uses existing gate | repository root | `go test ./server/tools/local-postgres-gate -count=1` | argv/environment assertion | exact `go run ./server/tools/postgres-live-gate`; child-only DSN | arbitrary/shell argument absent | fake runner captures argv | test asserts no DSN/password is sent to output |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] PostgreSQL major version remains 16 and loopback-only.
- [ ] Existing live-gate inventory is reused, not copied.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.

### Packet S02: Document the optional local gate

Status: complete
Covers: REQ-003, AC-003
Hard dependencies: S01 because documentation must name the delivered command and cleanup behavior.
Outcome: Developers can run and troubleshoot the local PostgreSQL gate without writing secrets or confusing it with production configuration.
Safe failure / rollback: Documentation-only change; no environment file is generated.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Local config guidance | `docs/environment-configuration.md` | Local development section | Documents dotenv precedence and forbids repository production secrets. |
| Existing manual variable | `server/internal/migrate/postgres_integration_test.go` | `getTestPostgresDSN` | Tests use `TEST_DATABASE_URL` and skip only outside CI when it is absent. |

Read set:

- `docs/environment-configuration.md` - add optional local test guidance without changing application ownership tables.

Modify set:

- `docs/environment-configuration.md` - add a concise local PostgreSQL test-gate subsection.

Must not modify:

- `.env*` and production configuration docs - no persistent credentials or new deployment behavior.

#### Ordered edits

1. `docs/environment-configuration.md` - Local development section
   - Change: document the single command, first-run binary cache, PostgreSQL 16 parity, loopback-only/process-scoped DSN, clean stop behavior, occupied-port remediation, and offline-cache limitation.
   - Constraint: do not show or request production credentials; do not present local success as CI or production proof.
   - Result: local test setup remains opt-in and non-secret.

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Docs match the tool | repository root | `rg -n "local-postgres-gate|TEST_DATABASE_URL|PostgreSQL 16" docs/environment-configuration.md` | documented command/limits | every required boundary present | stale command/version | mutate expected string in review only | restore before completion; `git diff --check` |

#### Completion gate

- [ ] Documentation names no secret value or persistent dotenv edit.
- [ ] Documentation distinguishes local, CI, and production evidence.
- [ ] Documentation matches delivered flags/paths exactly.

### Packet S03: Execute and independently inspect the real local run

Status: complete
Covers: REQ-001, REQ-002, AC-001, AC-002, AC-004
Hard dependencies: S01 because live execution requires the delivered launcher.
Outcome: A local PostgreSQL 16 instance runs the exact existing live-test gate and leaves no residue.
Safe failure / rollback: Any live test failure remains a failed acceptance result; the tool must stop the child and report it.

#### Execution result

Codex independently completed the first binary download and two successful live runs on 2026-08-15. The launcher started real PostgreSQL 16 at `127.0.0.1:5433`, the existing gate executed its required live tests, and post-run inspection found neither a listener on port 5433 nor a remaining `postgres` process. See the receipts directory for exact evidence.

## Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass |
| Every AC maps to proof in at least one packet | pass |
| Every packet maps back to REQ/AC | pass |
| Baseline, observed HEAD, dirty paths, and packet ID mapping are explicit | pass |
| All paths and symbols were inspected | pass |
| Hard dependency graph has no unexplained edge or cycle | pass |
| Scope covers every modify path and no unrelated path | pass |
| Product/authority decisions are approved or blocked | pass |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |
