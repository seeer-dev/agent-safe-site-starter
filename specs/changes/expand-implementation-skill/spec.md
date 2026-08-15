# Expand implementation skill

Change ID: expand-implementation-skill
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved revision 1 with plain apply on 2026-08-15.
Repository baseline: 1f09ab78b15bab0e3e18597537b6ee0b85a3c980
Supersedes: none

## Outcome

Add a repository-local, agent-neutral skill that expands a controlled proposal into an implementation-ready blueprint. The blueprint must make repository relationships, bounded steps, evidence, and stop conditions explicit so an implementer can execute it without inventing adjacent work or reconstructing hidden dependencies.

## Scope

In scope:
- A new `expand-implementation` skill with a concise procedure and one detailed blueprint-format reference.
- Internal use of the skill while preparing non-trivial controlled proposals.
- Exact repository evidence, per-slice paths and symbols, dependency ordering, before/after contracts, forbidden work, drift stops, verification, and falsification fields.
- Workflow and repository documentation that explains the internal expansion stage without adding a new user command.
- A governance regression that confirms the site workflow retains the two-step contract and routes implementation expansion through the new skill.

Out of scope:
- Naming, selecting, assigning, scheduling, or evaluating implementation agents.
- Provider- or model-specific instructions.
- Changing product behavior, architecture, permissions, data handling, or trust boundaries while expanding a proposal.
- Authorizing product edits during `propose`.
- Adding a new controlled-change status, lifecycle command, artifact type, orchestration system, or external dependency.

## Requirements

### REQ-001: Expansion is agent-neutral
The skill MUST define implementation expansion solely from the approved outcome, controlled artifacts, repository evidence, and project constraints; it MUST NOT branch on an implementer's identity, provider, model, or assigned role.

#### AC-001: Executor identity does not change the blueprint contract
- GIVEN the same proposal and repository state
- WHEN the expansion procedure is used for any implementation executor
- THEN the required blueprint fields, stop conditions, and verification obligations remain identical and contain no agent-routing decision.

### REQ-002: Slices contain repository-grounded instructions
The skill MUST expand every non-trivial slice into exact, bounded implementation instructions backed by inspected repository paths, symbols, contracts, tests, and dependency evidence.

#### AC-002: An expanded slice is locally executable
- GIVEN a repository-backed controlled proposal
- WHEN a slice is expanded
- THEN it identifies its outcome, preconditions, read set, allowed modifications, exact symbols or anchors, before/after behavior, ordered steps, dependencies, forbidden work, rollback or safe failure, and completion evidence.

#### AC-003: Missing evidence stops invention
- GIVEN a proposal claim that cannot be confirmed in the repository or requires a new product, cost, permission, data, or trust-boundary decision
- WHEN expansion reaches that claim
- THEN the blueprint records a specific drift or decision blocker and stops that slice instead of guessing a path, symbol, contract, or behavior.

### REQ-003: Completion claims are falsifiable
The skill MUST translate acceptance into exact commands, named assertions or observable artifacts, negative cases, and bounded failure triggers that distinguish effective verification from evidence-shaped output.

#### AC-004: Every claimed behavior has executable proof
- GIVEN an expanded slice claims an observable behavior
- WHEN its completion section is reviewed
- THEN it names the command or walkthrough, expected assertion or artifact, relevant negative case, and a bounded mutation or failure trigger that would make the claimed check fail.

### REQ-004: Expansion preserves the controlled workflow
The site workflow MUST use implementation expansion as an internal proposal-preparation stage while preserving `propose` as plan-only and plain `apply` as the sole normal approval action that authorizes implementation.

#### AC-005: No third user lifecycle action is introduced
- GIVEN a user requests a non-trivial controlled change
- WHEN the proposal is prepared and later approved
- THEN the user-facing sequence remains `propose <outcome>` followed by plain `apply`, with expanded instructions stored in the existing plan rather than a new lifecycle or artifact type.

#### AC-006: Expansion transfers across different change shapes
- GIVEN two existing controlled proposals with materially different implementation surfaces
- WHEN the skill is replayed independently against both without product edits
- THEN each resulting blueprint covers every mapped requirement and acceptance scenario, identifies repository drift honestly, and remains usable without model-specific guidance.

## Amendments

None.
