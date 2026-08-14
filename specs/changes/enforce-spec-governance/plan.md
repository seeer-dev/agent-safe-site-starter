# Enforce spec governance plan

## Repository reality

- `server/tools/verify` currently runs archcheck, scopecheck, Go tests, and vet.
- CI runs archcheck, tests, and vet directly and does not run scopecheck or a spec gate.
- `.ai/scope.json` is an ignored local allowlist and cannot provide durable spec traceability by itself.
- The target repo has no existing `specs/` or external spec CLI dependency.
- The first workflow draft exposed revision names, slice selection, and a separate walkthrough step as normal user operations even though these are agent-owned control mechanics.

## Scope lock

- `.ai/scope.example.json`
- `.github/workflows/ci.yml`
- `AGENTS.md`
- `README.md`
- `architecture.yaml`
- `server/tools/speccheck/**`
- `server/tools/verify/main.go`
- `skills/site/**`
- `specs/changes/enforce-spec-governance/**`
- `workflows/safe-change.md`

## Vertical slices

### Slice 1: Machine-readable spec contract

Outcome: controlled change metadata and Markdown artifacts have one deterministic validation model.

Coverage: REQ-002, AC-003, AC-004.

Proof: targeted Go tests cover valid controls, metadata/ID drift, scenario structure, and invalid Accepted evidence.

### Slice 2: Protected-diff enforcement

Outcome: every changed protected file maps to exactly one implementation-authorizing controlled change.

Coverage: REQ-001, REQ-004, AC-001, AC-002, AC-007.

Proof: targeted Go tests cover missing, Draft, duplicate, and governance-path coverage.

### Slice 3: Local and hosted gates

Outcome: normal local verification and CI execute speccheck against the relevant diff.

Coverage: REQ-003, AC-005, AC-006.

Proof: inspect verifier step order, run the full verifier from a clean task worktree, and inspect CI baseline wiring.

### Slice 4: Two-step operator workflow

Outcome: the normal user interaction is `propose <outcome>`, review a concise summary, then plain `apply`; controlled-spec state, all dependency-ordered slices, required walkthroughs, and validation remain agent-owned.

Coverage: REQ-005, AC-008, AC-009, AC-010.

Proof: inspect the site router, workflow usage, SDD, delivery lifecycle, agent rules, and README for one consistent contract; run the skill validator and full repository verifier from the clean task worktree.

### Slice 5: Evidence integrity gate

Outcome: `speccheck` rejects contradictory passed proof and, under strict evidence mode, stale revisions or missing required receipt artifacts.

Coverage: REQ-006, AC-011, AC-012.

Proof: targeted Go tests cover contradiction phrases, revision binding, required receipt kinds, unsafe paths, missing files, and valid current receipts.

### Slice 6: Reachability, recovery, production, and independent acceptance rules

Outcome: site references define real consumer reachability, fail-closed protected UI behavior, recovery threat-model requirements, approved-claim audits, structured walkthrough receipts, and independent replay.

Coverage: REQ-007, REQ-008, REQ-009, AC-013 through AC-018.

Proof: inspect routed site references for the new invariants, run the skill validator, and exercise the new evidence gate with receipt-backed proof.

## Traceability

| REQ/AC | Slice | Surface or consumer | Verification |
|---|---|---|---|
| REQ-001 | Slice 2 | changed protected paths | speccheck coverage tests |
| AC-001 | Slice 2 | uncovered path failure | targeted Go test |
| AC-002 | Slice 2 | Draft/Superseded state | targeted Go test |
| REQ-002 | Slice 1 | control/spec/plan/evidence artifacts | parser and consistency tests |
| AC-003 | Slice 1 | drift diagnostics | targeted Go test |
| AC-004 | Slice 1 | Accepted evidence gate | targeted Go test |
| REQ-003 | Slice 3 | local verifier and GitHub Actions | full verify and workflow inspection |
| AC-005 | Slice 3 | `server/tools/verify` | full verify output |
| AC-006 | Slice 3 | `.github/workflows/ci.yml` | workflow baseline inspection |
| REQ-004 | Slice 2 | governance paths | protected-path tests |
| AC-007 | Slice 2 | governance edit coverage | targeted Go test |
| REQ-005 | Slice 4 | user-facing site delivery workflow | routing/reference consistency inspection and full verify |
| AC-008 | Slice 4 | propose interaction | workflow usage inspection |
| AC-009 | Slice 4 | plain apply interaction | workflow usage inspection |
| AC-010 | Slice 4 | implementation discovery and stop boundary | site/agent rule inspection |
| REQ-006 | Slice 5 | control evidence model and `speccheck` | targeted semantic and receipt tests |
| AC-011 | Slice 5 | passed proof semantics | contradiction rejection tests |
| AC-012 | Slice 5 | strict evidence revision and receipts | stale/missing/valid receipt tests |
| REQ-007 | Slice 6 | API clients, auth/session producers, UI consumers | reachability and failure-state reference inspection |
| AC-013 | Slice 6 | producer-to-real-consumer trace | integration-planning inspection |
| AC-014 | Slice 6 | protected empty/error/authorization states | integration-planning and walkthrough inspection |
| REQ-008 | Slice 6 | account/order recovery trust boundaries | auth recovery checklist inspection plus security-review receipt |
| AC-015 | Slice 6 | recovery threat model | auth and evidence requirements inspection |
| REQ-009 | Slice 6 | production claims, walkthroughs, reviewer handoff | source/output audit and receipt contract inspection |
| AC-016 | Slice 6 | source, fresh client build, fresh rendered output | production-claim audit contract inspection |
| AC-017 | Slice 6 | runtime surface/persona/state journey | structured walkthrough receipt validation |
| AC-018 | Slice 6 | implementer-to-reviewer handoff | independent replay contract inspection |

## Baseline and safe failure

The main worktree contains thousands of unrelated changes, including existing edits under `server/tools/dev` and `server/tools/render`. Validate in a clean temporary worktree containing only this controlled change. A missing or malformed control artifact must fail closed with paths and reasons; it must never skip because Git inspection failed.
