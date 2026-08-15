# Lifecycle walkthrough — revision 1

Date: 2026-08-15
Reviewer: Codex

## Successful lifecycle

After the independent real run, both checks were clear:

```text
PORT_5433_CLEAR
POSTGRES_PROCESS_CLEAR
REPO_RUNTIME_DIR_CLEAR
```

The PostgreSQL binary cache exists only under the Windows local application-data cache as documented; it contains the downloaded `bin`, `lib`, and `share` assets rather than repository runtime data.

## Controlled failure paths

`go test ./server/tools/local-postgres-gate -count=1` passed, including:

- `TestLauncherStartupFailureCleansUpAndDoesNotRunGate`: a failed start cleans the temporary directory and does not invoke the gate.
- `TestLauncherGateFailureStopsDBAndCleansUp`: a downstream gate error stops the database and cleans the temporary directory.
- `TestLauncherStopWarningPreservesGateError`: a stop warning does not hide the gate failure.
- `TestLauncherRejectsPositionalArguments`: rejects unexpected arguments before creating a temporary directory or starting PostgreSQL.

This provides the required AC-004 walkthrough. The forced failure is deterministic and dependency-injected, so it does not weaken or alter the real live-test inventory.
