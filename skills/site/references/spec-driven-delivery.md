# Spec-driven delivery

Use this reference to make user-visible behavior and cross-boundary contracts controllable before and during implementation. The controlled specification is normative; the implementation plan is a strategy for satisfying it.

## Contents

1. [User-facing contract](#user-facing-contract)
2. [When a controlled spec is required](#when-a-controlled-spec-is-required)
3. [Artifact layout](#artifact-layout)
4. [Spec control header](#spec-control-header)
5. [Deterministic enforcement](#deterministic-enforcement)
6. [Normative content](#normative-content)
7. [Traceability](#traceability)
8. [State transitions](#state-transitions)
9. [Amendment control](#amendment-control)
10. [Completion gate](#completion-gate)

## User-facing contract

The controlled artifacts are an enforcement layer, not the user interface. Keep the ordinary interaction to two moves:

1. `propose <outcome>`: inspect repository reality, create or update the controlled artifacts as a review-ready Draft, and summarize the outcome, boundaries, important decisions, slices, risks, and proof strategy. Keep product code read-only.
2. Plain `apply`: when exactly one current review-ready proposal exists and no blocker remains, record this as approval of the latest summarized revision, implement every in-scope slice in dependency order, run required tests and walkthroughs, update evidence, and finish the state transitions.

The agent chooses the change ID and maintains revisions, statuses, REQ/AC IDs, scope, slice order, commands, and evidence. Do not require the user to name or edit them. Expose those details only when the user asks for an audit, multiple proposals make `apply` ambiguous, a material product/cost/trust decision needs authority, or a genuine blocker must be explained.

## When a controlled spec is required

Create a controlled spec for non-trivial user-visible behavior, multi-region integration, schema or API contract changes, static/runtime ownership changes, auth, money, PII, permissions, or any change whose acceptance could drift during implementation.

An isolated low-risk fix may use a compact `## Controlled specification` section in its plan or issue. It must still include status, scope, requirements, acceptance evidence, and change control.

## Artifact layout

Default new changes to:

```text
specs/changes/<change-id>/
  control.json  # machine-readable status, scope, IDs, and evidence state
  spec.md       # normative outcome and requirements
  plan.md       # vertical slices, scope, dependencies, expected evidence
  evidence.md   # observed test and walkthrough evidence, plus gaps
```

If the user already named a plan such as `INTEGRATION_PLAN.md`, keep it as the plan and create or link a companion controlled spec. Markdown and Git are sufficient; do not add Spectra, OpenSpec, or another CLI solely for this workflow.

## Spec control header

Every controlled spec records:

```text
Change ID:
Revision:
Status: Draft | Ready | Applying | Verifying | Accepted | Superseded
Decision authority:
Approval basis:
Repository baseline:
Supersedes:
```

Do not fabricate approval. A propose-only request normally leaves a review-ready `Draft`. `Ready` requires explicit user acceptance or an earlier instruction that clearly delegates the relevant product decisions. When exactly one current proposal has been summarized and no blocker remains, a later plain `apply` supplies the approval basis for that latest revision; record the instruction, issue, or review reference internally.

## Deterministic enforcement

Keep `control.json` consistent with the spec header, `applies_to` paths, REQ/AC lists, and every evidence row. Run:

```text
go run ./server/tools/speccheck
```

The gate fails when a protected implementation or governance path has zero or multiple authorizing specs from the current comparison diff; a changed spec directory lacks `control.json`; scope, artifacts, or IDs drift; Draft/Superseded state attempts to authorize implementation; or Accepted/Superseded evidence is not fully passed with proof. CI requires `Accepted`. A spec already Accepted or Superseded in the comparison base is immutable and cannot authorize a later diff; create a new change and link `Supersedes`.

Do not use an environment bypass. CI supplies `SPEC_BASE_REF` only to select the trusted comparison diff. Governance paths that can weaken the validator are themselves protected; use repository review/branch protection so an agent cannot approve a gate-weakening diff alone.

### Strict evidence mode

New non-trivial controlled changes should set `strict_evidence: true` in `control.json`. In strict mode:

- every passed evidence record includes `observed_revision` equal to the current control revision;
- implementation or artifact changes that invalidate an observation return the mapped evidence to pending until replayed;
- `evidence_requirements` maps a REQ/AC ID to required receipt kinds;
- the evidence record lists receipts as `{ "kind": "<kind>", "path": "receipts/<file>.md" }`;
- receipt paths stay inside the controlled-change directory, exist, are non-empty, and appear in the proof text.

Supported receipt kinds are `walkthrough`, `consumer-reachability`, `security-review`, `production-content-audit`, and `independent-review`. Use only the kinds the acceptance claim genuinely needs.

Regardless of strict mode, a passed proof must not contain an unresolved-gap statement such as required work being not captured, not validated, unavailable, or still recommended before release. A different pending AC is not an excuse to describe the current ID as complete when its own normative outcome remains blocked.

## Normative content

The spec contains outcome, in-scope and out-of-scope behavior, invariants, authority/source-of-truth decisions, static/runtime ownership, failure and permission behavior, and requirements with stable IDs.

```text
### REQ-001: Short behavior name
The system MUST/SHALL ...

#### AC-001: Observable scenario
- GIVEN ...
- WHEN ...
- THEN ...
```

- Give every normative requirement and acceptance scenario a unique stable ID.
- Do not renumber or reuse IDs after review. Mark removed requirements and explain the disposition.
- Express observable behavior rather than filenames or implementation mechanics.
- Link UI surfaces, API consumers, security boundaries, and negative/failure cases to the relevant IDs.
- Keep unresolved decisions as blockers, not requirements with guessed semantics.

## Traceability

The plan or evidence artifact maintains:

| REQ/AC | Slice | Surface or consumer | Verification | Observed evidence | Status |
|---|---|---|---|---|---|

Every slice task cites the IDs it implements. Every acceptance ID names a test, command, rendered/runtime inspection, or walkthrough observation. A checked task without current evidence does not satisfy a requirement.

Definitions are not consumer evidence. For client/runtime integration, trace a reachable entry point and actual identity/session/configuration producer through the call site to the authoritative result. Record empty, permission, and failure behavior without fixture or browser-local substitution.

## State transitions

| Transition | Gate |
|---|---|
| `Draft -> Ready` | Repository reality checked, blockers resolved, traceability planned, and approval basis recorded |
| `Ready -> Applying` | Current slice selected, baseline recorded, and scope locked |
| `Applying -> Verifying` | All in-scope slices have observed mechanical evidence |
| `Verifying -> Accepted` | Required walkthroughs pass, no blocking gaps remain, and verifier evidence is current |
| `Accepted -> Superseded` | A later controlled change explicitly replaces all or part of the spec |

An active normative change returns the spec to `Draft`. Before the change enters the comparison base, a user clarification may reopen it as a new revision with an amendment record and replayed evidence. Once an Accepted change is in the comparison base, do not edit it in place to describe new behavior; create a new change and link `Supersedes`.

## Amendment control

Implementation discoveries do not authorize silent spec changes. Before changing normative behavior:

1. Record the affected REQ/AC IDs, repository evidence, reason, and user impact.
2. Add an amendment entry with old meaning, new meaning, approval basis, affected slices, and invalidated evidence.
3. Return the spec to `Draft`, review producer/consumer and security impact, then regain `Ready`.
4. Rescope implementation and replay only the evidence invalidated by the amendment.

Record amendments as `Revision | REQ/AC | Old meaning | New meaning | Reason | Approval basis | Invalidated evidence and replay`.

Implementation detail may change in the plan without a spec revision only when observable behavior, authority, scope, failure behavior, and acceptance remain unchanged.

Even without a normative amendment, an implementation edit invalidates any observation that depended on the changed behavior or artifact. Reset those evidence rows to pending before editing or immediately when the dependency is discovered, then record the new revision-bound observation after replay.

## Completion gate

Accept the change only when the spec status is `Accepted`, all in-scope REQ/AC IDs have current observed evidence, every amendment has an approval basis and replay result, `speccheck` and scope checks pass, and no implementation or walkthrough gap contradicts the normative spec.
