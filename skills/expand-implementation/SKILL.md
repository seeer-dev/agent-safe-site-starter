---
name: expand-implementation
description: Expand an approved or review-ready proposal into a repository-grounded, implementation-ready blueprint with exact paths, symbols, ordered edits, boundaries, verification, falsification, and drift stops. Use when a non-trivial proposal needs to become executable without relying on the implementer to infer hidden relationships, invent adjacent work, or broaden scope.
---

# Expand implementation

Turn a controlled proposal into low-inference implementation packets. Keep product code read-only while expanding. Write the result into the proposal's existing `plan.md`; do not create a new lifecycle, status, or planning artifact. When an Accepted or Superseded proposal is immutable in the comparison base, leave it untouched and emit any retrospective replay only in the review response or test harness.

This skill does not select, assign, schedule, or evaluate implementers. Do not vary the blueprint by agent identity, provider, model, or team role.

## Required inputs

Read before expanding:

1. The nearest `AGENTS.md` files and repository architecture instructions.
2. The proposal's `control.json`, `spec.md`, `plan.md`, and `evidence.md`.
3. Every source, contract, migration, consumer, test, fixture, verifier, and CI path needed to verify the proposal's claims.
4. [`references/blueprint-format.md`](references/blueprint-format.md) in full.
5. [`references/handoff-protocol.md`](references/handoff-protocol.md) when delegating implementation across agents or shells.

If a required artifact is absent, contradictory, or not review-ready, record the gap. Do not compensate with an invented implementation.

## Procedure

1. Freeze repository reality.
   - Record the baseline and pre-existing dirty paths.
   - Treat every path, symbol, command, caller, and dependency named by the proposal as unverified until inspected.
   - Separate definitions from reachable consumers.
2. Close proposal coverage.
   - Map every requirement and acceptance scenario to one or more implementation packets.
   - Map every packet back to its requirements and acceptance scenarios.
   - Reject orphan tasks and uncovered acceptance IDs.
3. Inspect each implementation seam.
   - Locate exact paths and stable symbols or structural anchors.
   - Trace inputs, validation, authority, state transitions, persistence, contracts, consumers, and failure behavior when applicable.
   - Identify existing tests and repository-derived validation commands.
4. Expand dependency-ordered packets.
   - Use the mandatory format in `references/blueprint-format.md`.
   - Preserve existing slice IDs; when none exist, assign stable packet IDs and record their mapping.
   - State before/after behavior, ordered edits, allowed paths, forbidden work, safe failure, and proof.
   - Prefer a symbol or structural anchor over a line number; line numbers may supplement but never replace a stable anchor.
5. Make proof falsifiable.
   - Name each command's working directory, exact argv, selected tests or observable artifacts, expected results, negative cases, and bounded failure triggers.
   - State how the temporary failure trigger is restored and how residue is checked.
   - Replace vague verbs such as "ensure", "handle correctly", or "verify manually" with observable conditions.
6. Apply the completion gate.
   - Confirm repository evidence supports every instruction.
   - Confirm the packet set covers every in-scope requirement and acceptance scenario.
   - Confirm no packet changes product semantics, scope, cost, permissions, data handling, or another trust boundary without authority.
   - Leave blocked packets explicit; never make the plan appear complete by omitting them.

## Drift protocol

Emit one of these markers inside the affected packet and stop expanding that packet:

- `SPEC_DRIFT`: repository reality contradicts or cannot support a proposal claim, path, symbol, contract, or dependency.
- `DECISION_REQUIRED`: proceeding requires a new product, cost, permission, data, destructive, or trust-boundary decision.
- `ENVIRONMENT_BLOCKED`: exact proof requires an unavailable external environment or credential.

Include the claim, inspected evidence, impact, and smallest action that can unblock it. Continue expanding unrelated packets only when their behavior and evidence do not depend on the blocker.

During apply, do not improvise around a marker. Re-run expansion for the affected packet. Update implementation details internally when the approved contract is unchanged; return the controlled change to its review boundary when behavior or authority would change.

## Output rules

- Modify planning/control artifacts only during proposal expansion with explicit write authority; planners relaying active implementations must not silently amend controlled artifacts and must use the OS-temp transport in `references/handoff-protocol.md`.
- When delegating packets, use attributable envelopes with stable IDs, explicit modify/forbidden sets, named reviewer, and shell-safe non-secret OS-temp transport.
- Require postflight scope audits; record any temporary out-of-scope files created during implementation as incidents even after cleanup.
- Do not edit an Accepted or Superseded proposal already present in the comparison base; retrospective replay is read-only and non-authoritative.
- Preserve the repository's existing `propose` followed by plain `apply` interface.
- Keep the user-facing proposal summary concise even when `plan.md` is detailed.
- Do not add hypothetical provider seams, generic abstractions, cleanup, or adjacent features.
- Do not treat an implementation report, green package line, skipped test, source reachability, or unchecked command as proof.
- Do not mark expected evidence as observed evidence.
