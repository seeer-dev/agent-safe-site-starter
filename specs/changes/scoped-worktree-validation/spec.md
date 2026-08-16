# Scoped Worktree Validation Specification

Change ID: scoped-worktree-validation
Revision: 2
Status: Verifying
Decision authority: Repository owner/user
Approval basis: The repository owner authorized implementation on 2026-08-15 after the isolated linked-worktree model and review/implementation split were summarized. Revision 2 adds complete Git-worktree overlap preflight and the missing walkthrough guidance path before implementation.
Repository baseline: bc1d17f10d258c337efab975466949c92a5ec956
Supersedes: none

## Outcome

Let an implementation packet prove its own scope without treating unrelated pre-existing or parallel changes in the shared primary worktree as violations, while continuing to reject every path actually present in the selected packet's Git-derived diff that is outside its controlled allowlist.

## Scope

In scope:
- An explicit selected-change mode for `scopecheck` whose authority comes from one controlled change's `control.json`.
- A Git-derived comparison from that change's recorded repository baseline through `HEAD`, staged and unstaged edits, and untracked files in an isolated linked worktree.
- Fail-closed checks for the selected change identity, lifecycle status, baseline ancestry, linked-worktree isolation, and hosted-CI use.
- Preservation of the current `.ai/scope.json` behavior when selected-change mode is not requested.
- Agent workflow and handoff instructions that create, validate, report, and remove an OS-temporary linked worktree without repository scratch files.

Out of scope:
- Ignoring arbitrary dirty paths in the shared primary worktree.
- Inferring author identity from file timestamps, process IDs, prose reports, or an agent-supplied path list.
- Allowing selected-change mode to replace full-diff CI governance.
- Changing `speccheck`, `.ai/scope.json`, `.ai/scope.example.json`, CI, `server/tools/verify`, product code, Git history, or commit policy.
- Solving concurrent edits to the same path; overlapping ownership remains a conflict that must stop implementation.

## Design Options and Decision

### Option A: Isolated linked worktree plus selected controlled change — recommended

Create an OS-temporary linked Git worktree for one approved change. `SCOPE_CHANGE_ID=<id>` makes `scopecheck` read that change's `repository_baseline` and `applies_to`, require a linked worktree, and validate every path in the baseline-to-HEAD plus local/untracked diff. Unrelated dirt in the primary worktree is absent from this worktree; any out-of-scope path in the selected worktree remains visible and fails.

Trade-offs: strongest attribution using existing Git primitives; requires agent-managed worktree creation, artifact relay, postflight audit, and cleanup. A path already owned by another active task cannot be edited concurrently.

### Option B: OS-temporary pre/post hash snapshot

Record every path and content hash before work, then validate only state changes relative to that snapshot.

Trade-offs: handles pre-existing dirt but cannot distinguish another writer's post-snapshot change from the selected task and cannot detect a temporarily created then deleted file without a separate incident report. It remains useful for audit evidence but is not sufficient as the authority source.

### Option C: Alternate Git index or implementer-supplied patch manifest

Stage or enumerate only the purported packet changes and validate those paths.

Trade-offs: compact and scriptable, but omission of an unstaged out-of-scope change can make the gate green; overlapping edits to an already-dirty file are not attributable. It is not recommended as the sole gate.

### Approved decision

The owner approved Option A on 2026-08-15. Options B and C may supplement incident accounting but MUST NOT replace the Git-derived isolated-worktree diff.

## Invariants

- Selected mode validates the complete selected worktree diff; it never filters a shared diff down to allowed paths before checking it.
- The selected `control.json` supplies both the exact allowlist and baseline. An environment override cannot move the baseline forward and hide changes.
- Selected mode runs only for `Applying` or `Verifying` changes and only in a linked worktree. Missing, malformed, Draft, Ready, Accepted, or Superseded controls fail.
- `CI` plus selected mode fails. Hosted CI retains repository-wide protected-diff enforcement and cannot opt into a subset.
- No selected mode preserves the current `.ai/scope.json` contract, including its absent-file skip and out-of-scope diagnostics.
- Git inspection errors, a non-ancestor baseline, an ambiguous change ID, or inability to prove linked-worktree isolation fail closed.
- `.ai/scope.json` remains unchanged and is not used as an attribution ledger.
- Handoff and cleanup use OS-temp paths; no temporary worktree, patch, envelope, mutation, or test fixture remains in the repository.

## Requirements

### REQ-001: Selected Change Owns Scope Authority
When explicitly selected outside CI, `scopecheck` MUST read exactly one controlled change and validate the complete isolated-worktree diff against that change's `applies_to` patterns and recorded repository baseline.

#### AC-001: Primary Worktree Dirt Does Not Contaminate Selected Validation
- GIVEN unrelated dirty and untracked paths exist in the primary worktree before a task begins
- WHEN the approved task is implemented and checked in its own linked worktree with `SCOPE_CHANGE_ID`
- THEN unchanged primary-worktree paths MUST NOT appear in the selected result and the selected allowed diff MUST pass

#### AC-002: Selected Out-of-Scope Path Still Fails
- GIVEN a selected linked worktree contains one path outside the selected control's `applies_to`
- WHEN selected-change scope validation runs
- THEN it MUST exit nonzero and name that exact path even if every primary-worktree path is ignored by isolation

### REQ-002: Attribution and Baseline Fail Closed
Selected-change validation MUST reject any execution context that cannot establish the selected control, authorizing lifecycle state, immutable baseline, baseline ancestry, and linked-worktree isolation.

#### AC-003: Shared Primary Worktree Is Rejected
- GIVEN `SCOPE_CHANGE_ID` is set in the repository's primary shared worktree
- WHEN `scopecheck` runs
- THEN it MUST exit nonzero with an instruction to use an isolated linked worktree rather than silently excluding foreign dirt

#### AC-004: Invalid Selection or Baseline Is Rejected
- GIVEN the change ID is unsafe or absent on disk, the control is malformed or not `Applying`/`Verifying`, the baseline is invalid, or the baseline is not an ancestor of `HEAD`
- WHEN selected-change scope validation runs
- THEN it MUST exit nonzero with the specific failed precondition before reporting scope success

#### AC-005: Hosted CI Cannot Select a Subset
- GIVEN `CI` and `SCOPE_CHANGE_ID` are both set
- WHEN `scopecheck` runs
- THEN it MUST exit nonzero and state that selected mode is local isolated-worktree validation only

### REQ-003: Legacy Local Scope Remains Compatible
Without `SCOPE_CHANGE_ID`, `scopecheck` MUST preserve the existing `.ai/scope.json`, `SCOPE_BASE_REF`, changed-path discovery, allow-pattern, skip, and diagnostic behavior.

#### AC-006: Legacy Mode Does Not Drift
- GIVEN the same `.ai/scope.json`, environment, and Git state accepted or rejected before this change
- WHEN the revised tool runs without `SCOPE_CHANGE_ID`
- THEN it MUST produce the same allowed/outside decision and retain the absent-scope skip

### REQ-004: Handoff Makes Isolation Reproducible
Repository workflow instructions MUST give agents a provider-neutral, shell-safe lifecycle for creating the selected linked worktree, transporting the authorized packet, running scope validation and verification there, reporting baseline-relative evidence, rejecting overlap, and cleaning OS-temp artifacts.

#### AC-007: Handoff Names the Complete Isolation Lifecycle
- GIVEN an approved controlled change is delegated from a dirty primary worktree
- WHEN the handoff instructions are followed
- THEN the implementer and reviewer MUST be able to identify the change/revision, baseline, linked-worktree root, complete selected diff, exact validation command, cleanup result, and scope incidents without adding repository scratch files

#### AC-008: Concurrent Ownership Conflict Stops Work
- GIVEN a selected packet would edit a path already changed in the primary worktree or another linked worktree known to Git
- WHEN preflight enumerates every Git worktree and compares each complete tracked/untracked dirty set with the selected control's `applies_to`
- THEN implementation MUST stop and report the overlapping path and worktree rather than copying, merging, or overwriting it inside the isolated worktree

## Amendments

| Revision | REQ/AC | Old meaning | New meaning | Reason | Approval basis | Invalidated evidence |
|---|---|---|---|---|---|---|
| 2 | AC-008 | Compared an unspecified primary-worktree baseline and active handoffs. | Enumerates all Git-known worktrees and intersects their complete dirty sets with the selected allowlist. | The prior wording had no durable source for active-handoff ownership and could not be mechanically proven. | Owner authorized implementation on 2026-08-15. | None; all evidence remains pending. |
