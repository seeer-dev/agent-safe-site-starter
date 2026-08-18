# Frontend Contract Verification Gate Decision Plan

Change ID: verify-contract-checks
Revision: 3
Status: Verifying

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `.github/workflows/ci.yml`
- `Makefile`
- `README.md`
- `specs/changes/verify-contract-checks/**`

`server/tools/verify` is deliberately absent. The owner decision keeps it Go-only, so a contributor without Node is never blocked by the repository verifier. Package manifests and lockfiles are also absent: both scripts import only Node standard-library modules, so no install step is required.

## Dependency-ordered slices

### Slice 1: Local command

Outcome: `make verify-contracts` runs both scripts directly with `node`, and fails with an actionable message naming the minimum version when Node is missing or too old.

Acceptance evidence: the command run on this machine passes; a bounded mismatch introduced into one consumer makes the corresponding script fail and the command exit nonzero, restored afterwards. `go run ./server/tools/verify` is confirmed to invoke no Node. Covers REQ-002, AC-003, AC-004.

Rollback: remove the target; nothing else depends on it.

### Slice 2: Required CI gate with a pinned runtime

Outcome: CI runs the same command as a required step, after selecting an explicit Node version at or above 20.11.

Hard dependencies: Slice 1 defines the command CI invokes.

Acceptance evidence: the workflow names both checks and pins the version; the pin is at or above the floor that `import.meta.dirname` requires. Covers REQ-001, REQ-003, AC-001, AC-002, AC-005.

Rollback: remove the step and the setup action.

### Slice 3: Documented prerequisite

Outcome: README records the command and its Node floor.

Hard dependencies: Slices 1 and 2 fix the command name and the version.

Acceptance evidence: both appear in README. Covers REQ-002.

Rollback: revert the documentation only.

## Traceability matrix

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001, AC-002 | 2 | Injected contract mismatch fails the gate; both checks named in output |
| REQ-002, AC-003 | 1 | Command output on a missing or old Node |
| REQ-002, AC-004 | 1 | `verify` source and run contain no Node invocation |
| REQ-003, AC-005 | 2 | Pinned version at or above 20.11 in the workflow and the README |

## Risks and controls

- Risk: the gate passes without running anything. Control: AC-001 requires an injected mismatch to turn it red, not merely a green exit code.
- Risk: CI inherits a runner Node older than `import.meta.dirname`. Control: REQ-003 pins the version explicitly and AC-005 asserts the floor.
- Risk: scope creeps into `npm ci`. Control: the scripts' imports were inspected and are standard-library only; manifests and lockfiles are out of scope.
