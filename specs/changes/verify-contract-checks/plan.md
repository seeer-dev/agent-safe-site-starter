# Frontend Contract Verification Gate Decision Plan

Change ID: verify-contract-checks
Revision: 2
Status: Draft

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `specs/changes/verify-contract-checks/**`

No CI, verifier, manifest, lockfile, admin, theme, or contract path is authorized by this revision.

## Implementation expansion context

Proposal revision: 1
Proposal status: Draft
Repository baseline: `7e5aa90b92d23c8c316d44ca935be7af8d150a1c`
Observed HEAD: `7e5aa90b92d23c8c316d44ca935be7af8d150a1c`
Pre-existing dirty paths:
- `.github/workflows/ci.yml` and `server/tools/verify/main_test.go` are actively modified by another controlled change; preserve and do not overwrite.
- Other PostgreSQL/staff verification paths and controlled artifacts are unrelated.
- `Makefile` and architecture-review proposal directories belong to this review task.
Packet ID mapping:
- Slice 1 -> Packet S01

### Packet S01: Select the mandatory frontend-gate entrypoint

Status: DECISION_REQUIRED
Covers: REQ-001, REQ-002, AC-001, AC-002, AC-003
Hard dependencies: active CI/verify governance change must settle before this proposal rebases and selects exact modify anchors.
Outcome: Owner chooses either a Node-enabled full verifier or a separate mandatory frontend-contract command, with identical CI enforcement and honest prerequisite failures.
Safe failure / rollback: Existing scripts remain available but non-mandatory; no active CI/verify work is overwritten.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Go verifier | `server/tools/verify/main.go` | `defaultSteps` | Runs Go architecture, migration, spec, scope, tests, stress tests, and vet; no Node steps. |
| Baseline CI | `.github/workflows/ci.yml` | `verify` job | Sets up Go/PostgreSQL only and does not run npm contract scripts. Current worktree version is concurrently modified and must not be attributed here. |
| Admin check | `admin/package.json` | `check:resource-contracts` | Runs `node scripts/check-resource-contracts.mjs`. |
| Theme check | `site/themes/minimal-cart/package.json` | `check:openapi-contracts` | Runs `node scripts/check-openapi-contracts.mjs`. |
| Contract authority | `contracts/openapi.yaml` | route schemas | Shared source inspected by the existing scripts. |
| Declared starter prerequisite | `README.md` | Run the local demo | Currently advertises Go as the required local-demo tool; unconditional Node in the full verifier changes that contract. |

Read set:
- `.github/workflows/ci.yml`, `server/tools/verify/main.go`, `server/tools/verify/main_test.go`, both package manifests/locks, both contract scripts, `contracts/openapi.yaml`, `README.md`.

Modify set:
- None in revision 1. Revision 2 must select exact CI, command, test, and documentation paths after concurrent verifier work settles.

Must not modify:
- Active `.github/workflows/ci.yml` or `server/tools/verify/main_test.go` work.
- Frontend consumer code or OpenAPI merely to make a check green.

#### Behavior contract

Before:
- Two repository-specific consumer checks exist but neither baseline CI nor the Go verifier executes them.

After the future executable revision:
- CI and one documented local command execute both checks.
- Missing prerequisites fail honestly.

Preserved invariants:
- Contract drift is fixed at the incorrect producer/consumer, not hidden by weakening scripts.
- Package locks remain authoritative and verification does not mutate them.
- Existing Go verification remains intact.

Inputs and authority:
- Repository owner decides whether Node is part of the full verifier contract.
- `contracts/openapi.yaml` remains the API contract authority.

Errors and edge states:
- Missing Node, missing `node_modules`, lock mismatch, script failure, and concurrent CI changes need explicit handling.

#### Candidate decisions

1. Full verifier requires Node: add setup/install instructions, make `server/tools/verify` invoke both checks, and provision Node/dependencies in CI.
2. Separate mandatory frontend gate: add one deterministic local command (for example a Make target or tool) and require it as a named CI step while keeping Go verifier prerequisites unchanged.

Both options must fail nonzero when prerequisites are missing; silent skip is forbidden.

#### Expansion stop

Marker: DECISION_REQUIRED
Claim that cannot be executed: Select command ownership and exact paths for REQ-001/REQ-002.
Repository evidence: Baseline CI has no Node setup, README advertises Go for the local demo, and active unrelated work is modifying CI and verifier tests.
Impact: All requirements and ACs; no governance path is authorized.
Smallest unblock action: Let the active CI/verify change settle, then repository owner chooses option 1 or 2. Rebase revision 2, add exact protected paths, inventory existing verifier tests, and require red-before-green script mismatch evidence.

## Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass |
| Every AC maps to proof in at least one packet | blocked by command-ownership decision |
| Every packet maps back to REQ/AC | pass |
| Baseline, observed HEAD, dirty paths, and packet mapping are explicit | pass |
| All current repository anchors were inspected | pass |
| Hard dependency graph has no unexplained edge or cycle | pass |
| Scope covers every modify path and no unrelated path | pass; no product edits authorized |
| Product/authority decisions are approved or blocked | DECISION_REQUIRED |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |
