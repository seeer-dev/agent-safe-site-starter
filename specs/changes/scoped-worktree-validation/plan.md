# Scoped Worktree Validation Delivery Plan

Change ID: scoped-worktree-validation
Revision: 2
Status: Verifying

Normative specification: [`spec.md`](spec.md)

## Repository Reality

- `server/tools/scopecheck/main.go` reads one repository-global `.ai/scope.json` and compares it with `git diff --name-only HEAD` plus every untracked path. With `SCOPE_BASE_REF`, it substitutes `<base>...HEAD` for the tracked diff but still adds all untracked paths.
- `changedFiles` has no writer or task attribution. `allowed` can answer whether a path matches a pattern, but it cannot determine which concurrent task created a path.
- `server/tools/scopecheck` has no test file. Its process-exit behavior, Git failures, legacy compatibility, and shared-worktree false positives lack regression coverage.
- `server/tools/verify/main.go` invokes `go run ./server/tools/scopecheck`; child commands inherit the caller environment, so `SCOPE_CHANGE_ID` requires no verifier edit.
- `README.md`, `AGENTS.md`, `workflows/safe-change.md`, and routed site references describe `.ai/scope.json` as the task-local scope mechanism.
- `skills/expand-implementation/references/handoff-protocol.md` already requires baseline-relative postflight audits, OS-temp transport, preservation of pre-existing dirt, and incident accounting, but does not require an isolated linked worktree.
- A shared mutable filesystem cannot securely infer writer attribution after the fact. Filtering the global dirty set to allowed paths would hide real out-of-scope edits and is forbidden.
- CodeGraph is unavailable. Repository inspection used exact source and documentation searches.

## Design Selection

| Option | Parallel false positives | True out-of-scope detection | Trust/complexity | Decision |
|---|---|---|---|---|
| A. Isolated linked worktree + selected control | Eliminated because foreign dirt is absent | Complete baseline/HEAD/local/untracked diff remains checked | Git-derived; agent-managed lifecycle | Owner approved |
| B. Pre/post hash snapshot | Foreign post-snapshot writes still collide | Cannot attribute concurrent writers or deleted temporary residue | Supplemental audit only | Reject as sole authority |
| C. Alternate index/patch manifest | Can isolate listed paths | Omitted unstaged changes can evade the gate | Relies on implementer-selected set | Reject as sole authority |

Decision: The owner approved Option A on 2026-08-15. It remains local-only and forbidden in CI; the packets below are executable under this controlled change.

## Scope Lock

- `AGENTS.md`
- `README.md`
- `server/tools/scopecheck/main.go`
- `server/tools/scopecheck/main_test.go`
- `skills/expand-implementation/references/handoff-protocol.md`
- `skills/site/SKILL.md`
- `skills/site/references/change-safety.md`
- `skills/site/references/delivery-lifecycle.md`
- `skills/site/references/integration-planning.md`
- `skills/site/references/user-walkthrough.md`
- `workflows/safe-change.md`
- `specs/changes/scoped-worktree-validation/**`

Must not modify:
- `.ai/scope.json` and `.ai/scope.example.json` - preserve legacy task-local configuration.
- `.github/workflows/ci.yml` - hosted full-diff governance must not opt into a selected subset.
- `server/tools/verify/**` - environment inheritance already reaches `scopecheck`; avoid unrelated verifier churn.
- `server/tools/speccheck/**` - controlled-spec authority remains unchanged.
- Product code, database, frontend, and generated output - no product behavior changes.

## Implementation expansion context

Proposal revision: 2
Proposal status: Applying
Repository baseline: `bc1d17f10d258c337efab975466949c92a5ec956`
Observed HEAD: `bc1d17f10d258c337efab975466949c92a5ec956`
Pre-existing dirty paths:
- `Makefile` - preserve; not attributable to this proposal.
- `specs/changes/commerce-module-file-split/**` - preserve; concurrent proposal.
- `specs/changes/public-endpoint-rate-limit/**` - preserve; concurrent proposal.
- `specs/changes/supabase-jwks-verifier/**` - preserve; concurrent proposal.
- `specs/changes/verify-contract-checks/**` - preserve; concurrent proposal.
Packet ID mapping:
- New decision boundary -> Packet S00.
- New core validator slice -> Packet S01.
- New regression slice -> Packet S02.
- New workflow/handoff slice -> Packet S03.

### Packet S00: Approve a verifiable attribution source

Status: approved by owner
Covers: REQ-001, REQ-002, REQ-004, AC-001, AC-003, AC-005, AC-007, AC-008
Hard dependencies: none
Outcome: The owner-approved trust model determines which Git state belongs to a selected task without hiding actual out-of-scope edits.
Safe failure / rollback: Keep the change Draft; current scopecheck behavior remains authoritative and no implementation path changes.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Behavior owner | `server/tools/scopecheck/main.go` | `main`, `changedFiles`, `allowed` | One global scope and one global dirty set; no task attribution |
| Durable authority | `server/tools/speccheck/main.go` | `Control.RepositoryBaseline`, `Control.AppliesTo` | Controlled changes already store baseline and exact scope |
| Handoff contract | `skills/expand-implementation/references/handoff-protocol.md` | `Postflight Scope Inspection & Incident Accounting` | Requires baseline-relative audit but lacks isolated execution |
| Consumer | `server/tools/verify/main.go` | `defaultSteps` scopecheck entry | Normal verification invokes scopecheck and inherits environment |

Read set:
- `server/tools/scopecheck/main.go` - current input, Git, and matching behavior.
- `server/tools/speccheck/main.go` - existing controlled metadata shape and lifecycle vocabulary.
- `skills/expand-implementation/references/handoff-protocol.md` - current attribution and residue contract.

Modify set:
- `specs/changes/scoped-worktree-validation/**` - record owner decision and resulting revision only.

Must not modify:
- All implementation paths - Draft does not authorize governance edits.

#### Behavior contract

Before:
- Foreign dirty paths make global scopecheck fail; filtering them without attribution would weaken the gate.

After:
- Owner either approves Option A or supplies another model that proves complete task attribution with equivalent fail-closed properties.

Preserved invariants:
- No agent-authored path list is treated as complete evidence.
- CI never uses selected subset validation.

Inputs and authority:
- Repository owner/user approves the governance trust model.
- Git diff plus controlled metadata remain the recommended authoritative inputs.

Errors and edge states:
- Ambiguous attribution or a weaker substitute fails selected validation; the approved model keeps S01-S03 executable.

### Packet S01: Selected scope fails closed in an isolated linked worktree

Status: verifying — implementation complete; independent evidence pending
Covers: REQ-001, REQ-002, AC-001, AC-002, AC-003, AC-004, AC-005
Hard dependencies: S00 because changing scope authority is a governance trust decision.
Outcome: `SCOPE_CHANGE_ID=<id> go run ./server/tools/scopecheck` validates the complete selected linked-worktree diff and rejects unsafe execution contexts.
Safe failure / rollback: Legacy dispatch remains the default. Removing selected-mode functions and their dispatch returns byte-for-byte legacy behavior; any selected-mode setup error exits nonzero.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Entry | `server/tools/scopecheck/main.go` | `main` | Always reads `.ai/scope.json` |
| Diff owner | `server/tools/scopecheck/main.go` | `changedFiles` | Uses optional `SCOPE_BASE_REF`, worktree diff, and all untracked paths |
| Matcher | `server/tools/scopecheck/main.go` | `allowed` | Supports exact/glob and directory `/**` patterns |
| Control contract | `specs/changes/<id>/control.json` | `change_id`, `status`, `repository_baseline`, `applies_to` | Existing machine-readable authority source |
| Consumer | `server/tools/verify/main.go` | scopecheck step | Inherits `SCOPE_CHANGE_ID` without code change |

Read set:
- `server/tools/scopecheck/main.go` - preserve legacy behavior and reuse matching.
- `server/tools/speccheck/main.go` - mirror only required control fields; do not import or modify the tool.
- `server/tools/verify/main.go` - confirm environment propagation and no edit requirement.

Modify set:
- `server/tools/scopecheck/main.go` - selected dispatch, control loading, lifecycle/baseline validation, linked-worktree and CI guards, Git-derived changed set.

Must not modify:
- `changedFiles` legacy semantics, `.ai/scope.json`, `server/tools/verify/**`, CI, or speccheck.

#### Behavior contract

Before:
- One `.ai/scope.json` evaluates all shared dirt; unrelated paths fail every parallel task.

After:
- With no `SCOPE_CHANGE_ID`, behavior is unchanged.
- With `SCOPE_CHANGE_ID`, the tool validates one safe ID, reads `specs/changes/<id>/control.json`, requires matching `change_id`, `Applying` or `Verifying`, a full 40-hex baseline that is an ancestor of `HEAD`, a linked worktree, and non-CI execution.
- It checks `<baseline>...HEAD`, staged/unstaged changes relative to `HEAD`, and untracked non-ignored paths against `applies_to` without first filtering allowed paths.
- Before selected validation, it enumerates all Git-known worktrees and rejects an intersection between the selected allowlist and any other worktree's complete tracked/untracked dirty set, naming the path and worktree.

Preserved invariants:
- Existing `allowed` semantics and legacy diagnostics remain.
- Git command errors cannot become success.
- Primary-worktree dirt is never silently excluded.

Inputs and authority:
- `SCOPE_CHANGE_ID` selects, but does not define, authority.
- Selected `control.json` defines ID, status, baseline, and allowlist.
- Git defines worktree identity, ancestry, and complete changed path set.

Errors and edge states:
- Unsafe ID, missing/malformed control, empty scope, wrong status, invalid/non-ancestor baseline, primary worktree, CI, Git failure, or outside path exits nonzero with a specific diagnostic.

#### Ordered edits

1. `server/tools/scopecheck/main.go` - `Scope` and selected control model
   - Change: add a minimal `SelectedControl` with `change_id`, `status`, `repository_baseline`, and `applies_to`; add safe change-ID validation permitting lowercase ASCII letters, digits, and hyphens only.
   - Constraint: do not reuse prose or evidence as authority.
   - Result: unsafe traversal and ambiguous IDs fail before file access.
2. `server/tools/scopecheck/main.go` - `main`
   - Change: dispatch to legacy behavior when `SCOPE_CHANGE_ID` is empty; otherwise call `runSelectedChange`.
   - Constraint: legacy branch retains current absent-file skip and `.ai/scope.json` exception.
   - Result: backwards-compatible default and explicit selected mode.
3. `server/tools/scopecheck/main.go` - `runSelectedChange`
   - Change: reject nonempty `CI`; load the exact control; require ID equality, status `Applying`/`Verifying`, nonempty scope, 40-hex baseline, linked-worktree proof from canonical `git rev-parse --git-dir` versus `--git-common-dir`, and `git merge-base --is-ancestor <baseline> HEAD`.
   - Constraint: `SCOPE_BASE_REF` cannot override the controlled baseline in selected mode.
   - Result: unverifiable attribution fails closed before path validation.
4. `server/tools/scopecheck/main.go` - selected changed-file discovery and overlap preflight
   - Change: parameterize Git discovery with the controlled baseline; include `<baseline>...HEAD`, `git diff --name-only HEAD`, and `git ls-files --others --exclude-standard`; propagate every Git error. Enumerate `git worktree list --porcelain`, inspect each other existing worktree with the same complete tracked/untracked discovery, and reject an intersection with selected `applies_to` before selected validation.
   - Constraint: do not filter the selected changed set using `applies_to` until after complete discovery; worktree-list, path discovery, or path-resolution failures fail closed.
   - Result: each selected-worktree outside path is named and rejected; a concurrent owned-path overlap names its worktree and stops before edits.

#### Integration trace

```text
agent-created linked worktree
  -> SCOPE_CHANGE_ID
  -> server/tools/scopecheck main selected dispatch
  -> specs/changes/<id>/control.json
  -> status/baseline/worktree/CI validation
  -> Git baseline...HEAD + local + untracked discovery
  -> existing allowed(path, applies_to)
  -> exact outside-path diagnostic or selected-scope success
  -> server/tools/verify caller observes exit status
```

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Foreign primary dirt is absent | repository root | `go test -json -run '^TestSelectedChangeIgnoresDirtyPrimaryWorktree$' ./server/tools/scopecheck -count=1` | named test pass event | temp primary repo dirt is not reported when linked worktree has only allowed edits | run selected validation in primary root | temporarily remove linked-worktree requirement; the negative assertion fails | restore hunk; pre/post `git diff --binary -- server/tools/scopecheck/main.go | git hash-object --stdin` hashes match |
| Outside selected path fails | repository root | `go test -json -run '^TestSelectedChangeRejectsOutsidePath$' ./server/tools/scopecheck -count=1` | diagnostic names fixture path | nonzero for exact outside path | allowed-only fixture passes | temporarily make `allowed` return true; named test fails | restore hunk and compare diff hash |
| Unsafe context and overlap fail closed | repository root | `go test -json -run '^(TestSelectedChangeRejectsPrimaryWorktree|TestSelectedChangeRejectsCI|TestSelectedChangeRejectsInvalidControlAndBaseline|TestSelectedChangeRejectsDirtyPathInOtherWorktree)$' ./server/tools/scopecheck -count=1` | all named pass events | each precondition and cross-worktree path conflict has a distinct error | missing Git/control/overlap must not skip | temporarily bypass each guard one at a time; corresponding test fails | restore each hunk; compare diff hash and status to pre-trigger snapshot |

#### Completion gate

- [ ] Every selected-mode guard and overlap preflight is implemented inside the modify set.
- [ ] Every selected and legacy behavior has a named fixture test.
- [ ] Every red/green mutation is restored with no residue.
- [ ] Evidence is independently replayed before AC status advances.

### Packet S02: Regression suite proves selected and legacy behavior

Status: verifying — implementation complete; independent evidence pending
Covers: REQ-001, REQ-002, REQ-003, AC-001, AC-002, AC-003, AC-004, AC-005, AC-006
Hard dependencies: S01 defines the selected-mode seams and diagnostics under test.
Outcome: Named integration tests prove isolation, complete outside-path rejection, fail-closed guards, and byte-compatible legacy decisions.
Safe failure / rollback: New tests create repositories/worktrees only under `t.TempDir`; failure cleans them through Go test cleanup. Removing the new test file leaves production unchanged.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Test target | `server/tools/scopecheck/main.go` | `main`, `changedFiles`, `allowed` | No current automated tests |
| Git fixture precedent | `server/tools/speccheck/main_test.go` | temporary repository helpers | Repository already tests Git-backed governance in temporary roots |
| Legacy consumer | `.ai/scope.example.json` | `allowed_paths` | Existing JSON contract must remain unchanged |

Read set:
- `server/tools/scopecheck/main.go` - enumerate all branch behavior.
- `server/tools/speccheck/main_test.go` - reuse repository fixture style, not implementation code.
- `.ai/scope.example.json` - legacy fixture shape.

Modify set:
- `server/tools/scopecheck/main_test.go` - new unit/CLI integration suite.

Must not modify:
- Existing gate source during test-only step, speccheck tests, or repository `.ai` files.

#### Behavior contract

Before:
- Scope behavior is validated only by manual command outcomes.

After:
- Tests create a primary repository and linked worktree under `t.TempDir`, write controlled fixtures, execute exact modes, and assert exit class plus exact path/guard diagnostics.
- Legacy fixtures cover missing scope skip, allowed success, outside failure, `SCOPE_BASE_REF`, untracked discovery, and `/**` matching.

Preserved invariants:
- Tests do not use the developer's repository Git index, worktrees, or `.ai/scope.json`.
- Existing checks are not renamed, skipped, or weakened.

Inputs and authority:
- Test-created Git repositories and control files are the only fixtures.
- Named Go test events are execution evidence.

Errors and edge states:
- Git unavailable is a setup failure, not a skip.
- Cleanup failure fails the test or is reported by `t.Cleanup`.

#### Ordered edits

1. `server/tools/scopecheck/main_test.go` - fixture helpers
   - Change: add helpers to initialize a temp Git repo, configure local identity, commit a baseline, create a linked worktree, write control/scope fixtures, and invoke validation with captured stdout/stderr.
   - Constraint: never use the real repository or global Git config.
   - Result: deterministic isolated fixtures on Windows and CI.
2. `server/tools/scopecheck/main_test.go` - selected-mode tests
   - Change: add the five named tests from S01 plus `TestSelectedChangeRejectsDirtyPathInOtherWorktree` and cases for tracked, staged, unstaged, deleted, and untracked outside paths.
   - Constraint: every expected failure asserts the exact offending path or failed precondition.
   - Result: no package-only success can substitute for named execution.
3. `server/tools/scopecheck/main_test.go` - legacy tests
   - Change: add `TestLegacyScopeMissingSkips`, `TestLegacyScopeAllowsConfiguredPaths`, `TestLegacyScopeRejectsOutsideAndUntrackedPaths`, and `TestLegacyScopeBaseRef`.
   - Constraint: no selected environment variable is set in legacy cases.
   - Result: AC-006 is protected against dispatch or behavior drift.

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| All selected cases execute | repository root | `go test -json -run '^TestSelectedChange' ./server/tools/scopecheck -count=1` | every declared selected test emits `pass` | zero skip/missing/fail | remove one test from source | inventory review detects missing name before acceptance | restore test; `rg '^func TestSelectedChange'` matches planned inventory |
| Legacy behavior preserved | repository root | `go test -json -run '^TestLegacyScope' ./server/tools/scopecheck -count=1` | four named legacy pass events | outside/untracked fixtures fail internally | route empty `SCOPE_CHANGE_ID` to selected mode | legacy missing-scope assertion fails | restore dispatch and compare diff hash |
| Repeat stability | repository root | `go test ./server/tools/scopecheck -count=20` | all scopecheck tests | twenty passes | fixture leaks or depends on timing | run with one cleanup removed; residue assertion fails | restore cleanup; temp root absent and repository status equals pre-test status |

#### Completion gate

- [ ] Every planned selected-mode and legacy test is present and passes.
- [ ] Fixture repositories/worktrees are removed by test cleanup.
- [ ] The legacy path remains byte-compatible in observable behavior.

### Packet S03: Workflow routes delegated work through isolated validation

Status: executable
Covers: REQ-004, AC-007, AC-008
Hard dependencies: S01 establishes the exact `SCOPE_CHANGE_ID` interface and failure behavior documented here.
Outcome: Agents create and validate one OS-temporary linked worktree per packet, stop on ownership overlap, and report cleanup and scope incidents without exposing control machinery to users.
Safe failure / rollback: Documentation can revert together while legacy `.ai/scope.json` remains the operational contract; no runtime or data state changes.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Global rule | `AGENTS.md` | Change procedure scope steps | Requires one narrow `.ai/scope.json` |
| User contract | `README.md` | scopecheck local gate guidance | Calls `.ai/scope.json` task-local |
| Workflow | `workflows/safe-change.md` | `Apply` flow | Creates global scope before implementation |
| Router | `skills/site/SKILL.md` | Procedure step 6 | Routes non-trivial edits through `.ai/scope.json` |
| Routed rules | `skills/site/references/change-safety.md` | scope lock steps | Assumes one worktree-local scope |
| Lifecycle | `skills/site/references/delivery-lifecycle.md` | apply scope | Requires one scope file per slice |
| Planner | `skills/site/references/integration-planning.md` | scope and baseline | Requires baseline preservation but not isolation |
| Walkthrough | `skills/site/references/user-walkthrough.md` | walkthrough preconditions | Runs scopecheck before runtime acceptance |
| Handoff | `skills/expand-implementation/references/handoff-protocol.md` | relay and postflight sections | OS-temp transport and incident log already exist |

Read set:
- Every modify path below - preserve the two-step operator contract and current handoff rules.

Modify set:
- `AGENTS.md` - selected isolated mode rule and overlap stop.
- `README.md` - local modes and CI boundary.
- `skills/expand-implementation/references/handoff-protocol.md` - linked-worktree envelope, lifecycle, cleanup, and report fields.
- `skills/site/SKILL.md` - route parallel dirty work through selected isolated mode.
- `skills/site/references/change-safety.md` - preflight and postflight commands.
- `skills/site/references/delivery-lifecycle.md` - apply lifecycle integration.
- `skills/site/references/integration-planning.md` - packet scope/proof requirements.
- `skills/site/references/user-walkthrough.md` - selected-mode preconditions before a walkthrough calls scopecheck.
- `workflows/safe-change.md` - internal apply flow.

Must not modify:
- User-facing `propose` then plain `apply` contract, provider identity rules, CI, `.ai` files, or implementation packet authority.

#### Behavior contract

Before:
- Documentation assumes `.ai/scope.json` is task-local even when many tasks share one dirty worktree.

After:
- Single-task/clean worktrees may keep legacy scope.
- Parallel or dirty-primary delegation records the initial dirty/active ownership set, rejects overlapping modify paths, creates an OS-temp linked worktree, relays the controlled artifacts/packet without repository scratch, runs `SCOPE_CHANGE_ID=<id> go run ./server/tools/scopecheck` and the verifier there, records full selected diff and incidents, then removes the worktree and prunes metadata.

Preserved invariants:
- Users still say `propose` and plain `apply`; agents own worktree mechanics.
- Pre-existing work and secrets are never copied into reports.
- Temporary files remain in OS temp and are cleaned in `finally`/postflight.

Inputs and authority:
- Approved packet modify set and selected controlled change define intended ownership.
- Git worktree/diff and exact gate output define observed scope.

Errors and edge states:
- Overlapping path, failed worktree creation, artifact relay mismatch, validation failure, or cleanup failure blocks the packet and is reported.

#### Ordered edits

1. `skills/expand-implementation/references/handoff-protocol.md` - envelope and postflight
   - Change: require change baseline, linked-worktree root, active ownership comparison, selected command, complete diff, cleanup result, and incident log; provide shell-safe PowerShell worktree creation/removal using an OS-temp directory and `try/finally`.
   - Constraint: do not prescribe provider-specific agents or repository scratch files.
   - Result: delegated implementation is reproducible and attributable.
2. `AGENTS.md`, `skills/site/SKILL.md`, routed references, and `workflows/safe-change.md` - scope routing
   - Change: distinguish legacy clean/single-task scope from selected linked-worktree mode; require overlap stop and prohibit selected CI use.
   - Constraint: retain the two-step user interface and existing speccheck/verify gates.
   - Result: agents select safe mechanics without asking users to operate them.
3. `README.md` - contributor contract
   - Change: document both modes, selected-mode environment input, linked-worktree requirement, and CI/full-diff boundary.
   - Constraint: do not claim `.ai/scope.json` provides durable cross-task attribution.
   - Result: repository guidance matches executable behavior.
4. `server/tools/scopecheck/main_test.go` - `TestScopeWorkflowDocumentsIsolationContract`
   - Change: assert every required governance document contains its exact selected-mode obligation and forbidden shortcut.
   - Constraint: compare per-file obligations; unrelated substring presence cannot satisfy another file.
   - Result: documentation wiring cannot silently regress.

#### Integration trace

```text
plain apply
  -> planner records approved packet/change and primary dirty ownership
  -> overlap preflight
  -> OS-temp linked worktree + shell-safe handoff
  -> implementation inside isolated root
  -> SCOPE_CHANGE_ID selected scopecheck
  -> targeted checks + verifier
  -> complete selected diff and independent review
  -> worktree removal/prune + incident/residue report
  -> controlled evidence remains pending until replay
```

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Docs route isolation consistently | repository root | `go test -json -run '^TestScopeWorkflowDocumentsIsolationContract$' ./server/tools/scopecheck -count=1` | named pass event and per-file table | all eight governance files satisfy their obligations | one file omits overlap/CI/cleanup rule | temporarily delete one required sentence; named test fails for that file | restore sentence; compare file diff hash |
| Real handoff is scoped | OS-temp linked worktree root | PowerShell: `$env:SCOPE_CHANGE_ID='scoped-worktree-validation'; go run ./server/tools/scopecheck` | exact selected success and changed-path count | only authorized paths reported | add `outside-scope.txt` in linked root | command fails naming `outside-scope.txt` | remove only the fixture, confirm `git status --short` equals pre-trigger selected state, then record incident |
| Full verifier consumes selected mode | OS-temp linked worktree root | PowerShell: `$env:SCOPE_CHANGE_ID='scoped-worktree-validation'; go run ./server/tools/verify` | scopecheck selected-mode line plus verifier result | selected scope runs inside normal verifier | unset ID in dirty linked root | legacy mode cannot be accepted as selected evidence | restore environment; no repository mutation |

#### Expansion stop

The owner-approved workflow must update every listed governance path together. A documentation mismatch or an unrecorded overlap remains a packet failure, not a reason to return to global shared-worktree scope.

## Traceability

| REQ/AC | Packet | Surface or consumer | Expected verification |
|---|---|---|---|
| REQ-001, AC-001, AC-002 | S01, S02 | selected scopecheck | linked-worktree integration tests and outside-path mutation |
| REQ-002, AC-003, AC-004, AC-005 | S01, S02 | trust guards | named guard tests and guard-removal mutations |
| REQ-003, AC-006 | S02 | legacy scopecheck callers | legacy fixture inventory and dispatch mutation |
| REQ-004, AC-007, AC-008 | S03 | agent workflow and handoff | per-file governance test and isolated lifecycle replay |

## Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass |
| Every AC maps to proof in at least one packet | pass |
| Every packet maps back to REQ/AC | pass |
| Baseline, observed HEAD, dirty paths, and packet ID mapping are explicit | pass |
| All paths and symbols were inspected | pass |
| Hard dependency graph has no unexplained edge or cycle | pass: S00 -> S01 -> S02/S03 |
| Scope covers every modify path and no unrelated path | pass |
| Product/authority decisions are approved or blocked | blocker: S00 governance trust model |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |

## Apply-Time Gate Sequence

After S00 approval and lifecycle transition:

1. Run `go run ./server/tools/speccheck` before editing protected governance paths.
2. Implement S01, then S02 and S03 in dependency order inside one isolated linked worktree for this change.
3. Run named tests with `go test -json`, then `go test ./server/tools/scopecheck -count=20` and `gofmt -l server/tools/scopecheck`.
4. Execute each bounded mutation separately, capture the targeted failure, restore it, and compare the affected file's binary diff hash with its pre-trigger value.
5. Run selected scopecheck and `go run ./server/tools/verify` from the isolated root.
6. Independently inspect the complete selected diff, replay the tests and outside-path failure, and record an `independent-review` receipt.
7. Confirm the primary worktree's pre-existing dirty paths were not touched; remove and prune the OS-temp worktree; record cleanup and all temporary-scope incidents.
