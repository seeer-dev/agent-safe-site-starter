# Integration planning

Use this reference to create or review an implementation plan that crosses architectural regions. A proposal states intended work; an integration review proves that its claims fit the current repository.

## Required review sequence

1. [Establish current repository reality](#establish-current-repository-reality).
2. [Compare plan claims with implementation reality](#compare-plan-claims-with-implementation-reality).
3. [Trace producer to consumer](#trace-producer-to-consumer).
4. [Compare schema with actual consumers](#compare-schema-with-actual-consumers).
5. [Resolve authority and source of truth](#resolve-authority-and-source-of-truth).
6. [Choose static or runtime ownership](#choose-static-or-runtime-ownership).
7. [Detect fixture and fallback dependencies](#detect-fixture-and-fallback-dependencies).
8. [Convert big-bang phases into vertical slices](#convert-big-bang-phases-into-vertical-slices).
9. [Order dependencies explicitly](#order-dependencies-explicitly).
10. [Require proof of completion](#require-proof-of-completion).
11. [Control the specification](#control-the-specification).
12. [Lock scope and preserve the baseline](#lock-scope-and-preserve-the-baseline).
13. [Apply the review completion gate](#review-completion-gate).

## Establish current repository reality

- Read `AGENTS.md`, `architecture.yaml`, manifests, migrations, contracts, nearby tests, CI, and the code on both sides of each proposed boundary.
- Record `git status --short` before attributing changes or failures to the plan.
- Discover validation commands from the repository instead of assuming them.
- Inspect actual routes, models, stores, renderers, clients, and deployment paths. Treat names in the plan as unverified until repository evidence confirms them.
- Keep a plan-only review read-only for product code. Record implementation gaps in the plan instead of fixing them during the review.

## Compare plan claims with implementation reality

Create an evidence ledger for every route, field, module, provider, runtime, and workflow the plan assumes:

| Plan claim | Repository evidence | Current reality | Gap or decision | Plan impact |
|---|---|---|---|---|
| What the plan says exists or will work | Path, symbol, test, command, or config | What exists now | Missing, incompatible, or undecided item | Revise, precede, split, or block |

Mark absent evidence as `unverified`; do not promote an assumption into a repository fact. A plan is not closed merely because its producer exists: trace at least one real consumer for every important claim.

## Trace producer to consumer

Trace both write and read paths where applicable:

```text
producer/input
  -> validation and authorization
  -> domain/service owner
  -> storage
  -> API and contract
  -> admin, renderer, or browser consumer
  -> published or runtime result
```

At each hop, record the owner, data shape, stable identifier, trust boundary, error behavior, and fallback behavior. Verify that the next hop can consume the previous hop's actual output; matching labels are not evidence of compatible contracts.

### Prove consumer reachability

A route, client helper, component, store method, or authenticated branch is only a definition. Do not call it integrated until the trace proves:

```text
real entry point
  -> reachable UI or command state
  -> actual identity/session/configuration producer
  -> client or renderer call site
  -> authoritative service/store
  -> observable success, empty, permission, and failure result
```

- Search separately for definitions and invocations. A symbol with no caller is not a consumer.
- Trace setters and injected state back to a real producer. A bearer-token branch is unreachable when no auth bootstrap can populate it.
- Exercise the path from the entry point instead of calling only the service or helper in isolation.
- Keep the mapped REQ/AC evidence pending when any required hop is absent or only reachable through test setup, developer tools, fixtures, or hand-edited browser state.

## Compare schema with actual consumers

Compare migrations and stored models against service DTOs, handlers/OpenAPI, generated or handwritten client types, admin forms, renderer/templates, browser components, fixtures, and tests.

Use a field ledger when the shape is non-trivial:

| Field or capability | Storage | API/contract | Actual consumers | Gap or decision |
|---|---|---|---|---|

Check identifiers and slugs, types, nullability, defaults, enums and state transitions, money units, timestamps, media cardinality, and deletion/lifecycle behavior. Consumer requirements reveal missing plan work; they do not automatically authorize a schema or product decision.

## Resolve authority and source of truth

`Source of truth` identifies where durable state lives. `Authority` identifies who may decide or transition that state. Build a matrix for every consequential value and treat client claims as untrusted until the authoritative owner validates or derives them.

| Value | Source of truth | Client may submit | Authoritative validation or transition |
|---|---|---|---|
| Product price | Catalog database | Product/variant ID and quantity | Server loads the current price |
| Inventory | Inventory database | Requested quantity | Server checks and reserves atomically |
| Promotion | Backend policy/database | Promo code | Server determines eligibility and discount |
| Order total/payment state | Backend and verified provider events | Intent or provider reference, not authoritative totals/status | Server recalculates totals and verifies transitions |
| Principal/permissions | Auth identity plus Go authorization policy | Access token and requested action | Server authenticates and authorizes |
| Published content | Content store and publish state | Edit or publish request | Server authorizes the transition and renders the selected revision |
| PII | Owning domain store | Required user-provided fields | Server validates, minimizes, protects, and authorizes access |

Require a security review when the trace includes auth, money, PII, permissions, or any other trust boundary.

## Choose static or runtime ownership

Choose one delivery owner for each datum consumed by the public site:

```text
static:  authoritative store -> Go renderer -> dist/ -> deploy -> browser
runtime: browser -> Go API -> authoritative service/store
```

Search for competing paths such as render/inject plus hydrate/fetch, or static output plus runtime fallback. A hybrid path is acceptable only when the plan defines why both exist, which one is authoritative, freshness and cache invalidation, failure behavior, and preview/production reconciliation. Do not let the same datum silently acquire two authorities.

## Detect fixture and fallback dependencies

- Search for `fixture`, `mock`, `demo`, `sample`, `fallback`, `default`, `stub`, hard-coded data, and browser storage in imports, loaders, environment branches, and error handlers.
- Trace whether each real consumer can reach those paths in development, preview, tests, and production.
- Do not let an API failure silently fall back to fabricated production data. Keep test/dev fixtures behind an explicit boundary.
- Treat an authoritative empty result as distinct from "not loaded." Do not use an empty array or zero value to trigger a local fallback.
- Protected consumers must show loading, empty, unauthorized, forbidden/not-found, and network failure truthfully. Do not substitute browser-local orders, identities, permissions, or PII when the authoritative request fails.
- Give each slice negative acceptance evidence that completed paths no longer depend on fixtures or legacy fallbacks. Do not require unrelated fixtures to disappear early.

For production claims, use an approval inventory rather than only a blacklist. Policy, privacy, payment, shipping, warranty, support, promotion, and brand claims must trace to a published content record or an explicitly reviewed allowlist. Audit source, a fresh client build, and freshly rendered `dist/`; record the exact corpus and command. A zero-match search over agent-selected keywords does not prove that every unsupported claim is absent.

## Convert big-bang phases into vertical slices

Reject plans organized only as all schemas -> all backend modules -> all APIs -> all admin -> all site work -> final integration. Create the smallest slice that ends in an observable user outcome and crosses only the regions needed to prove it.

For a non-trivial controlled proposal, read `../../expand-implementation/SKILL.md` and its required blueprint reference in full. Expand each vertical slice into the existing `plan.md`; do not create a parallel plan artifact or leave path, symbol, dependency, failure, or proof discovery to the implementer.

For each slice, state:

```text
Outcome:
Entry -> storage/behavior -> contract -> real consumer -> observable result
Hard dependencies:
Allowed paths:
Acceptance evidence:
Rollback or safe failure:
```

An implementation-ready packet additionally records inspected repository anchors, read/modify/must-not-modify sets, before/after behavior, ordered symbol-level edits, forbidden work, exact verification, a bounded failure trigger, restoration, and an explicit drift stop. Keep the packet blocked rather than guessing when repository evidence does not support the proposal.

Use a foundation phase only for a decision, invariant, or contract that blocks multiple slices. Do not turn it into a warehouse of unrelated migrations or abstractions.

## Order dependencies explicitly

Build a dependency graph from hard prerequisites, not directory or technology-layer order:

1. Resolve product, identity, trust, and ownership decisions that change downstream contracts.
2. Establish shared identifiers, invariants, and contracts required by the first slice.
3. Deliver the first end-to-end slice that proves the riskiest seam.
4. Add dependent slices in the order their real consumers require them.
5. Remove shared fixtures or legacy fallbacks only after their final consumer has migrated.

State why each dependency must precede another. Keep convenience, batching, and cleanup separate from hard dependencies.

## Require proof of completion

Every slice must name evidence, not only tasks. Include the user-visible outcome, exact assertions, targeted tests, contract/type alignment, authorization and failure cases, migration coverage for both supported databases when schema changes, rendered/runtime output inspection, fixture/fallback negative checks, `speccheck`, `scopecheck`, and the repository verifier as applicable.

Name the command and expected artifact or assertion. During implementation, replace expected evidence with the observed result. “Files added”, “API done”, or “UI wired” is not proof that the slice works end to end.

## Control the specification

When an implementer reports completion, keep that report provisional. A reviewer must inspect the actual diff and independently replay the narrow tests, consumer reachability, failure paths, fresh output, and required walkthrough receipts before passed status is restored.

For non-trivial work, use `spec-driven-delivery.md` to separate normative behavior from implementation strategy. Map repository-backed requirements and acceptance scenarios to stable REQ/AC IDs, then map each ID to a slice, real consumer or surface, and expected evidence. The change is not apply-ready when the controlled spec remains `Draft`, lacks an approval basis, or contains requirements that no slice and verification claim covers.

When review changes intended behavior, record a spec amendment and its impact instead of silently editing tasks. Repository reality may disprove a requirement, but only the decision authority may choose the replacement behavior.

## Lock scope and preserve the baseline

- For clean/single-task working trees, create or update `.ai/scope.json` with only the plan/review files or the current slice's expected paths (legacy mode, which does not provide cross-task attribution).
- For parallel delegated packets or dirty-primary working trees, plan for isolated OS-temporary linked worktrees: preflight active/dirty ownership across all worktrees and stop immediately on `applies_to` overlap; set `$env:SCOPE_CHANGE_ID='<change-id>'` and run `scopecheck` and `verify` inside the linked root; forbid in-repository scratch files and secrets in reports; capture the complete selected diff; and in `finally`, remove the linked worktree and prune Git metadata. Selected mode is local-only and forbidden in CI.
- Expand scope only after repository evidence proves a new path is required; record the reason in the plan.
- Keep discovered implementation fixes out of a plan-only task.
- Record pre-existing dirty files and failing checks. Do not overwrite them, include them in the slice, or report them as regressions.

Run the narrowest baseline checks before implementation when safe. After each slice, run targeted checks, `go run ./server/tools/speccheck`, `go run ./server/tools/scopecheck`, `go run ./server/tools/verify`, and, when public output changes, `go run ./server/tools/render` plus inspection of `dist/`. Compare results with the recorded baseline; never claim a check passed when unrelated worktree changes or environment failures prevented it.

## Review completion gate

An integration plan is ready only when it contains repository-backed claims, at least one real consumer trace per important flow, explicit authority and runtime ownership, fixture/fallback disposition, dependency-ordered vertical slices, per-slice acceptance evidence, controlled-spec traceability and approval, a narrow scope, baseline results, and clearly identified product decisions or blockers.
