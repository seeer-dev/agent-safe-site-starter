# Enforce spec governance

Change ID: enforce-spec-governance
Revision: 3
Status: Accepted
Decision authority: Repository owner/user
Approval basis: User approved the summarized skill and evidence-governance hardening on 2026-08-12 by replying '好，調整'.
Repository baseline: 7c45b616fbe3a632ffe2a39d872c98485466c991
Supersedes: none

## Outcome

Repository validation must reject protected implementation or governance changes that are not covered by one valid controlled specification. The gate must validate spec state, traceability, and acceptance evidence without relying on an agent to remember prose instructions.

## Scope

In scope: a repository-native Go `speccheck`, tests, integration with local verify and CI, the machine-readable control manifest, and site-skill/governance documentation.

Out of scope: an external Spectra/OpenSpec dependency, cryptographic proof that evidence text is truthful, GitHub branch-protection configuration, and migration of unrelated current product changes into this spec.

## Amendments

| Revision | REQ/AC | Old meaning | New meaning | Reason | Approval basis | Invalidated evidence and replay |
|---|---|---|---|---|---|---|
| 2 | REQ-005, AC-008, AC-009, AC-010 | The workflow exposed named revisions, slices, and walkthrough as normal user operations. | The user only requests propose, reviews a concise summary, then says plain apply; the agent owns controlled artifacts, slice execution, walkthroughs, and verification. | The enforcement layer must constrain agents without burdening the operator. | User clarification in the current task on 2026-08-11. | Workflow contract inspection and skill validation passed; existing gate evidence remained valid and the full verifier passed. |
| 3 | REQ-006 through REQ-009, AC-011 through AC-018 | Evidence consistency relied mainly on status/row equality and prose guidance. | Passed evidence must be semantically honest, current, receipt-backed where required, reachable through a real consumer, secure across recovery paths, and independently replayed before acceptance. | Repeated integration reviews found self-reported completion despite unreachable consumers, insecure recovery, missing walkthroughs, and unsupported claims in fresh output. | User approved the summarized hardening on 2026-08-12. | New evidence-governance tests, skill validation, targeted source/output audits, and the full verifier must be replayed. |

## Requirements

### REQ-001: Protected changes require one active controlled spec
The repository MUST reject every changed protected implementation or governance path that is not covered by exactly one controlled change whose status permits implementation.

#### AC-001: Missing controlled spec fails
- GIVEN a protected source or governance path is changed
- WHEN no controlled change covers that path
- THEN speccheck exits non-zero and identifies the uncovered path

#### AC-002: Draft spec cannot authorize implementation
- GIVEN a controlled change covers a protected path but its status is Draft or Superseded
- WHEN speccheck evaluates the changed path
- THEN the change is rejected as lacking an implementation-authorizing spec

### REQ-002: Controlled artifacts are machine validated
The repository MUST validate control metadata, spec headers, stable REQ/AC identifiers, normative scenarios, plan traceability, and evidence rows as one internally consistent contract.

#### AC-003: Artifact drift fails
- GIVEN control.json, spec.md, plan.md, or evidence.md disagree on metadata or REQ/AC coverage
- WHEN speccheck validates the change
- THEN it exits non-zero with a specific artifact violation

#### AC-004: Accepted status requires passed evidence
- GIVEN a controlled change claims Accepted or Superseded status
- WHEN any REQ/AC evidence is pending, blocked, missing, or lacks proof
- THEN speccheck rejects the claimed status

### REQ-003: Normal verification invokes speccheck
The repository SHALL run speccheck in the standard local verifier and pull-request CI with an explicit comparison baseline.

#### AC-005: Local verify runs the gate
- GIVEN a contributor runs the repository verifier
- WHEN verification steps execute
- THEN speccheck runs before scopecheck, tests, and vet

#### AC-006: CI checks the pull-request or push diff
- GIVEN CI checks a pull request or push
- WHEN speccheck runs with the event baseline
- THEN protected changed paths in that Git diff require a valid covering spec

### REQ-004: Governance paths cannot silently bypass the gate
Changes to speccheck, verify, CI, architecture rules, agent rules, or the site skill MUST themselves require controlled-spec coverage so bypass attempts are visible in the same review diff.

#### AC-007: Governance edits require coverage
- GIVEN a changed path can weaken spec or verification governance
- WHEN speccheck evaluates it
- THEN the path is treated as protected and must be covered by exactly one valid controlled change

### REQ-005: Controlled delivery exposes a two-step operator contract
The site workflow MUST let a user request `propose <outcome>`, review a concise proposal, then issue plain `apply`, while the agent owns change IDs, revisions, statuses, REQ/AC traceability, slice selection, required walkthroughs, and verification.

#### AC-008: Propose hides control machinery
- GIVEN an ordinary non-trivial site outcome with no existing proposal
- WHEN the user requests propose
- THEN the agent inspects repository reality, creates or updates Draft control artifacts, and returns a concise proposal summary without asking the user to operate SDD metadata or commands

#### AC-009: Plain apply completes the approved scope
- GIVEN exactly one current review-ready proposal has been summarized and no blocker remains
- WHEN the user issues plain apply
- THEN the agent records approval, implements every in-scope slice in dependency order, runs required validation and walkthroughs, and reports the result without requiring a revision or slice name

#### AC-010: Ordinary implementation discoveries remain internal
- GIVEN an implementation discovery can be resolved within the approved outcome and authority
- WHEN the agent updates the plan, scope, implementation, or evidence
- THEN it handles and verifies the discovery internally, interrupting only for ambiguous proposals, a material product/cost/trust decision, or a genuine blocker

### REQ-006: Passed evidence is current and semantically honest
The repository MUST reject passed evidence that contains an unresolved-gap statement, belongs to an older controlled revision under strict evidence mode, or lacks a required receipt.

#### AC-011: Contradictory passed proof fails
- GIVEN an evidence row is marked passed while its proof says required work was not captured, not validated, unavailable, or still recommended before release
- WHEN speccheck validates the control
- THEN it rejects the row and identifies the contradictory phrase

#### AC-012: Strict evidence is revision-bound and receipt-backed
- GIVEN a strict controlled change requires a named receipt kind for a REQ/AC ID
- WHEN passed evidence belongs to another revision, omits the receipt, references an unsafe path, or points to a missing artifact
- THEN speccheck rejects the evidence

### REQ-007: Integration completion requires a reachable authoritative consumer
The site workflow MUST treat a client, route, or component definition as incomplete until a real entry point reaches it through the actual identity/session source and renders authoritative success and failure states without fabricated fallback.

#### AC-013: Definitions are not consumers
- GIVEN an API helper or authenticated branch exists but has no reachable caller or session producer
- WHEN integration evidence is evaluated
- THEN the affected requirement remains pending rather than being described as wired

#### AC-014: Protected consumers fail closed
- GIVEN an authoritative API returns empty, unauthorized, forbidden, not found, or network failure
- WHEN a protected UI consumer handles the result
- THEN it shows the contracted state and does not silently substitute browser-local or fixture data

### REQ-008: Recovery preserves the original trust boundary
The site workflow MUST block a recovery design that relies only on identifiers or contact data, weakens the original authentication factor, or lacks expiry, single-use, replay, rate-limit, enumeration, and atomic-rotation analysis.

#### AC-015: Recovery requires possession proof and threat-model evidence
- GIVEN access to an order, account, secret, or other protected object can be recovered
- WHEN the recovery contract is proposed or implemented
- THEN a security review proves control of an approved factor and covers expiry, single use, replay, rate limiting, enumeration, rotation, audit, and safe failure before the related evidence can pass

### REQ-009: Production and walkthrough evidence is independently reproducible
The site workflow MUST audit approved claims against source and fresh generated output, require structured walkthrough receipts for runtime acceptance, and keep implementer self-reports provisional until an independent replay confirms them.

#### AC-016: Production-claim audit checks source and fresh output
- GIVEN policy, privacy, payment, shipping, warranty, support, promotion, or brand claims can reach production
- WHEN acceptance evidence is collected
- THEN every claim traces to an approved source and the audit inspects source, a fresh client build, and freshly rendered production output rather than a self-selected keyword subset

#### AC-017: Required walkthroughs produce receipts
- GIVEN an acceptance scenario depends on runtime UI behavior
- WHEN the scenario is marked passed
- THEN a non-secret structured walkthrough receipt identifies the fixed revision, surface, persona, state, expectation, observation, and supporting request, assertion, log, or screenshot

#### AC-018: Implementer completion is replayed independently
- GIVEN an implementer reports a slice complete
- WHEN acceptance status is decided
- THEN a reviewer independently checks the diff, reruns the relevant commands and output inspections, and records remaining gaps before restoring passed status
