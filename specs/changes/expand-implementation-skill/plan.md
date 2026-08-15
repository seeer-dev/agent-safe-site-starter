# Expand implementation skill delivery plan

Change ID: expand-implementation-skill
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality

- `skills/site/SKILL.md` is the single user-intent router and owns the current `propose`/`apply` workflow.
- `skills/site/references/integration-planning.md` already requires repository reality, consumer traces, vertical slices, dependency order, and proof, but it leaves the per-slice implementation instruction format implicit.
- `skills/site/references/workflow-usage.md` keeps proposal artifacts internal and exposes only `propose <outcome>` and plain `apply` to the user.
- `README.md` currently describes `skills/site/` as one router skill and does not describe a reusable implementation-expansion skill.
- No repository-local skill currently defines a low-inference implementation blueprint or a drift-stop protocol.
- The working tree already contains unrelated, uncommitted `postgres-lock-semantics-and-evidence` work. This proposal must not modify or absorb it.

## Scope lock

- `README.md`
- `server/tools/speccheck/main_test.go`
- `skills/expand-implementation/**`
- `skills/site/SKILL.md`
- `skills/site/references/integration-planning.md`
- `skills/site/references/workflow-usage.md`
- `specs/changes/expand-implementation-skill/**`

## Dependency-ordered slices

### Slice 1: Define the reusable expansion contract

Outcome: A repository-local skill tells an agent how to turn controlled proposal content and inspected code into deterministic implementation packets without considering executor identity.

- Initialize `skills/expand-implementation/` according to the skill-creator conventions.
- Keep `SKILL.md` concise and imperative; route detailed field definitions to `skills/expand-implementation/references/blueprint-format.md`.
- Define mandatory inputs, repository inspection, slice expansion, traceability, ambiguity handling, and the `SPEC_DRIFT`/decision-blocker stop protocol.
- Explicitly exclude agent selection, delegation, scheduling, and provider/model-specific behavior.
- Covers REQ-001, REQ-002, AC-001, AC-002, AC-003.

### Slice 2: Make proof executable and falsifiable

Outcome: Each implementation packet states how to prove the behavior and how to demonstrate that its check is sensitive to the claimed defect.

- Require exact commands, named tests/assertions or observable artifacts, negative cases, expected results, bounded failure triggers, and restoration checks.
- Prohibit vague completion language such as “ensure”, “handle correctly”, or “verify manually” without a concrete observation.
- Require proposal-to-slice-to-REQ/AC coverage and identify uncovered IDs as a blocker.
- Covers REQ-003, AC-004.

### Slice 3: Wire expansion into the existing two-step workflow

Outcome: Non-trivial proposals gain expanded plans without adding a user command or a competing artifact format.

- Update `skills/site/SKILL.md` to load the new skill during non-trivial proposal preparation.
- Update `skills/site/references/integration-planning.md` and `skills/site/references/workflow-usage.md` so expansion populates the existing `plan.md` and remains read-only for product code.
- Update `README.md` to describe the router skill plus reusable expansion skill.
- Extend `server/tools/speccheck/main_test.go` with exact source-wiring assertions that preserve `propose`/plain `apply` and reject an unreferenced expansion skill.
- Covers REQ-004, AC-005.

### Slice 4: Validate transferability

Outcome: Independent replay shows the format is useful across different change shapes without model-specific instructions.

- Run the skill validator against `skills/expand-implementation/`.
- Replay expansion, read-only, against two materially different existing controlled proposals; recommended fixtures are one backend concurrency/governance change and one cross-surface integration change.
- Confirm every REQ/AC maps to a packet or an explicit blocker and inspect for provider/agent names.
- Record an independent-review receipt without treating implementer output as acceptance.
- Run the targeted governance test, `go run ./server/tools/speccheck`, `go run ./server/tools/scopecheck`, and `go run ./server/tools/verify` when the unrelated worktree baseline permits an attributable result.
- Covers REQ-001, REQ-002, REQ-003, REQ-004, AC-001, AC-002, AC-003, AC-004, AC-005, AC-006.

## Traceability

| ID | Slice | Surface | Expected verification |
|---|---|---|---|
| REQ-001 | 1, 4 | `skills/expand-implementation/**` | Source inspection and independent replay show no agent/provider branching |
| AC-001 | 1, 4 | Expansion contract | Same mandatory fields and stops for both replay fixtures; agent/provider-name scan is empty |
| REQ-002 | 1, 4 | Skill procedure and blueprint reference | Each replay produces complete repository-backed packets or explicit blockers |
| AC-002 | 1, 4 | Per-slice blueprint | Required-field inventory is complete for every executable slice |
| AC-003 | 1, 4 | Drift protocol | A deliberately unsupported proposal claim produces `SPEC_DRIFT` or a decision blocker, not guessed implementation |
| REQ-003 | 2, 4 | Completion and falsification fields | Each behavioral claim maps to an exact check and bounded failure trigger |
| AC-004 | 2, 4 | Per-slice proof | Review confirms command, assertion/artifact, negative case, mutation/failure trigger, and restoration |
| REQ-004 | 3, 4 | Site router and workflow references | Governance test and source inspection preserve the two-step lifecycle |
| AC-005 | 3, 4 | Existing `plan.md` workflow | No new user verb, status, or controlled artifact is introduced |
| AC-006 | 4 | Two independent replay fixtures | Independent-review receipt records coverage, drift handling, and agent-neutral usability |

## Baseline and safe failure

- Baseline: `1f09ab78b15bab0e3e18597537b6ee0b85a3c980`.
- Preserve all unrelated dirty PostgreSQL lock-semantics files exactly as found.
- If the new skill needs a path outside the scope lock, return the proposal to Draft and revise scope before editing.
- If expansion needs to decide product behavior, permissions, data handling, cost, or a trust boundary, record a blocker and do not encode a guess as an implementation instruction.
- If independent replay only succeeds with model-specific hints or leaked expected answers, keep AC-006 pending and revise the skill.
