# Ephemeral local PostgreSQL gate

Change ID: ephemeral-postgres-local-gate
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: User authorized Agy to implement and Codex to independently validate the local PostgreSQL environment on 2026-08-15.
Repository baseline: 7e5aa90b92d23c8c316d44ca935be7af8d150a1c
Supersedes: none

## Outcome

Developers on Windows can run one repository command that starts an isolated, real PostgreSQL 16 process on the loopback interface, runs the existing PostgreSQL live-test gate with a process-scoped `TEST_DATABASE_URL`, and always stops the child process afterward.

## Scope

In scope:

- A Go tool under `server/tools/local-postgres-gate/` using a pinned `embedded-postgres` dependency.
- An isolated runtime/data directory outside the repository and a reusable user-cache directory for downloaded PostgreSQL binaries.
- Fixed-argument invocation of the existing `server/tools/postgres-live-gate` command.
- Documentation for the optional local gate, first-run download, port collision, offline behavior, and cleanup boundary.

Out of scope:

- Changing the application database default, production configuration, CI PostgreSQL service, migrations, tests, or `.env` files.
- Persisting a test database, changing a system PostgreSQL service, or configuring Supabase/Cloudflare.
- Treating a local successful run as a CI receipt or a production PostgreSQL validation.

## Requirements

### REQ-001: One-command real PostgreSQL execution

The repository SHALL provide a command that starts a real PostgreSQL 16 instance bound only to `127.0.0.1`, gives only its child gate process a generated `TEST_DATABASE_URL`, and invokes the existing `postgres-live-gate` without changing persistent application configuration.

#### AC-001: Local instance reaches the existing gate

- GIVEN no process listens on the selected loopback test port
- WHEN the developer runs the documented command from the repository root
- THEN it starts PostgreSQL 16, injects `TEST_DATABASE_URL` only into `postgres-live-gate`, and does not write an application `.env` profile.

### REQ-002: Failure-safe ephemeral lifecycle

The command SHALL stop its PostgreSQL child on both success and gate failure, remove temporary runtime/data paths, leave repository paths and dotenv profiles untouched, and return a non-zero result when startup or the live gate fails.

#### AC-002: Required PostgreSQL tests actually execute

- GIVEN the ephemeral instance is ready
- WHEN the command invokes `postgres-live-gate`
- THEN the gate proves each required `TestPostgresLive*` test executed and passed rather than skipped.

#### AC-004: Cleanup and failure are observable

- GIVEN either a successful gate or a controlled gate failure
- WHEN the launcher exits
- THEN PostgreSQL is stopped, temporary runtime/data paths are absent, no new repository paths exist, and the command exit reflects success or failure.

### REQ-003: Pinned, reviewable local dependency boundary

The Go module SHALL pin the launcher dependency and configure PostgreSQL major version 16 to match CI. Documentation SHALL identify the first-run binary download/cache boundary and provide non-secret remediation for offline startup and occupied ports.

#### AC-003: Supply-chain and local-process limits hold

- GIVEN the launcher is configured for a first run
- WHEN it resolves PostgreSQL binaries and starts the child process
- THEN the dependency and PostgreSQL major version are pinned, command arguments are fixed rather than shell-composed, the server binds only to loopback, no credential is written to repository configuration or output, and only the documented user-cache path persists.

## Security review findings to address

- Medium: `embedded-postgres` downloads executable PostgreSQL binaries on first run. Pin its Go module version and PostgreSQL 16; document its cache source and do not add unpinned installer scripts.
- Medium: a launcher that shell-composes commands or accepts arbitrary PostgreSQL arguments could execute unintended local commands. Use fixed `exec.CommandContext` arguments and validate any exposed port input.
- Medium: default test credentials must remain loopback-only and process-scoped. Do not write them to dotenv files, log a DSN containing the password, or expose the listener beyond `127.0.0.1`.
- Low: temporary runtime/data paths can become residue on failed cleanup. Use an OS temp root, attempt cleanup after `Stop`, and make residue detection explicit in the walkthrough.
