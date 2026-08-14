# Delivery lifecycle

Use this reference to keep planning, implementation, and experience discovery connected without silently changing the contract between phases. Internally, `propose`, `apply`, and `walkthrough` remain lifecycle phases. Externally, the normal workflow is only `propose <outcome>`, review the agent's summary, then plain `apply`; required walkthrough work happens inside apply.

## Durable artifacts

Keep phase handoffs in agent-managed repository artifacts. The user does not need to name or edit them:

- **Controlled spec:** normative outcome, scope, invariants, stable requirement/acceptance IDs, revision, status, and approval basis. See `spec-driven-delivery.md`.
- **Plan/proposal:** repository evidence, decisions, dependency-ordered slices, scope, baseline, expected evidence, and open blockers. It describes how to satisfy the spec but cannot redefine it.
- **Surface contract:** affected routes or mount points, personas, tasks, data, states, interactions, journeys, and UX acceptance. It may be a section of the plan or a companion document.
- **Evidence/walkthrough report:** observed evidence mapped to requirement/acceptance IDs and classified gaps against a fixed source state.

Prefer Markdown that the repository can review and version. Do not add Spectra, OpenSpec, or schema tooling merely to name these phases.

## Propose phase

Read `spec-driven-delivery.md`, plus `integration-planning.md` for cross-region work and `ui-surfaces.md` for UI work.

1. Inspect current repository reality, consumers, baseline commands, and existing dirty files.
2. Resolve or expose product, authority, runtime ownership, and dependency decisions in the controlled spec.
3. Define stable requirement and acceptance IDs, then the smallest vertical slices that satisfy them.
4. Give every slice explicit allowed paths and proof of completion.
5. Check identifiers, terminology, data shapes, dependencies, and acceptance claims for internal consistency.
6. Keep product code read-only. Only requested planning and contract artifacts may change.

A proposal is not implementation-ready when it contains placeholders, path-only tasks, vague verbs such as "wire up" without an outcome, unverified repository claims, unnamed consumers, or acceptance criteria that cannot be observed.

The phase ends with a review-ready `Draft`, or `Ready` only when approval was already explicit. The next eligible slice must be unambiguous without inventing a product decision.

Return a concise user-facing summary of the outcome, non-goals/boundaries, consequential decisions, delivery slices, risks or blockers, and how acceptance will be proved. Keep internal IDs and status mechanics out of the summary unless they help resolve a blocker or the user asks for them.

## Apply phase

Before each slice:

1. Re-read the controlled spec, its status/revision, the plan, current slice, affected surface contracts, allowed paths, and acceptance evidence. Do not edit product code unless the spec is `Ready` or `Applying`; plain `apply` may approve the sole latest summarized proposal under the gate in `spec-driven-delivery.md`.
2. Confirm the repository baseline and preserve unrelated worktree changes.
3. Reject or repair an unclear contract before coding; do not guess through a path-only or contradictory task.
4. Create a narrow `.ai/scope.json` for only this slice.

During the slice:

- Implement one complete producer-to-consumer path in dependency order.
- Run targeted checks as behavior becomes testable; do not defer integration proof to the final slice.
- Replace expected evidence with observed commands, assertions, or inspected output.
- Mark completion only after the user-visible outcome and required failure/permission cases pass.
- Re-read the implementation contract before starting the next slice.
- Keep implementer self-reports provisional. Before restoring passed evidence, independently inspect the diff and replay the mapped commands, real consumer path, failure behavior, fresh generated output, and required receipts.

Plain `apply` covers every in-scope slice by default. Select and execute them in dependency order without asking the user to name each slice; an explicit user scope limit still wins.

## Discovery loop during apply

| Discovery | Contract action | Implementation action |
|---|---|---|
| Defect inside the agreed contract | No contract change | Fix and add regression evidence |
| Missing field, state, surface, consumer, or false repository claim | Determine whether the spec or only the plan is wrong; amend the spec first when normative behavior changes | Rescope, then implement or replay affected work |
| New product, authority, or trust-boundary decision | Record an explicit blocker; run security review when required | Stop that path until decided |
| Required work outside the slice | Add a dependency or receiving follow-up with rationale | Do not hide it as cleanup |
| Non-blocking roughness or polish | Record a gap | Keep it out of the current slice |

Never make implementation and artifact semantics diverge silently. A normative change follows the amendment and approval gates in `spec-driven-delivery.md`; identify completed slices whose evidence is invalidated and replay only those checks.

## Walkthrough transition

Enter walkthrough internally after the selected slice passes its mechanical and contract checks whenever runtime UX is part of acceptance. Freeze the source state, map observations to REQ/AC IDs, record gaps, repair in-contract defects through apply, and replay affected checks. Contract drift updates the controlled spec, plan, or surface contract as appropriate before code changes. A user can still request a separate `walkthrough` for a read-only UX audit.

An implementation report from another agent is a handoff, not acceptance evidence. The receiving reviewer reads repository state rather than relying on the report, records any contradiction as a gap, and keeps affected evidence pending until the independent replay passes. Do not let the same report both assert completion and serve as the sole proof that completion is true.

## Resume and combined requests

- On resume, read the controlled spec revision/status, durable artifacts, and repository status instead of relying on chat history.
- If one request chains phases, preserve the internal propose, apply, and walkthrough gates while reporting one concise outcome.
- Resolve equally eligible slices using dependency order and the smallest safe sequence. Request direction only when multiple current proposals are ambiguous or a choice materially changes product behavior, cost, permissions, data handling, or another trust boundary.
