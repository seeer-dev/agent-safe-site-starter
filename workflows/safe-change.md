# Safe change workflow

Internal agent and reviewer SOP. Users only need to request `propose <outcome>`, review the summary, then reply `apply`; they do not operate change IDs, revisions, slices, REQ/AC IDs, evidence rows, or validation commands.

## Propose

```text
user outcome
  -> read skills/site/SKILL.md and the closest routed reference
  -> inspect architecture.yaml and the owning modules
  -> discover real entry points, callers, identity/session/configuration producers, and impact
  -> create or update one Draft controlled change
  -> return outcome, boundaries, decisions, slices, risks, and validation summary
```

Product code stays read-only during proposal. Interrupt only when multiple current proposals make `apply` ambiguous or a decision materially changes product behavior, cost, permissions, data handling, or another trust boundary.

## Apply

```text
plain apply
  -> record approval and authorize the sole review-ready proposal
  -> create a narrow .ai/scope.json
  -> implement all slices in dependency order
  -> run targeted tests and required walkthroughs
  -> collect current, receipt-backed evidence
  -> independently replay the diff, commands, and generated output
  -> run scopecheck, speccheck, and the full verifier
  -> report changed behavior, validation, and remaining risk
```

Implementation discoveries that remain within the approved outcome are handled internally by updating the plan, scope, implementation, and evidence.

## Acceptance evidence

- Keep evidence `pending` while any required caller, session producer, UI failure state, walkthrough, security review, source audit, or independent replay is missing.
- Under `strict_evidence`, every passed row names the current `observed_revision`. Required receipt kinds point to non-secret files inside the same controlled-change directory, and proof text names the receipt.
- Definitions and helpers are not consumers. Trace the real runtime entry point through its producer and call site to the authoritative result.
- Protected UI fails closed; it does not replace authoritative empty, unauthorized, forbidden, not-found, or network failure with local or fixture data.
- Recovery proves control of an approved factor and reviews expiry, single use, replay, rate limiting, enumeration, atomic rotation, audit, delivery, and safe failure.
- Production claims trace to an approved source and are inspected in source, a fresh client build, and freshly rendered output.
- Runtime walkthrough receipts record revision, surface, persona, state, expectation, observation, and supporting request, assertion, log, or screenshot.
- The implementer's completion report is provisional. A reviewer inspects the actual diff, reruns the relevant checks, records remaining gaps, then restores passed status.

`go run ./server/tools/speccheck` validates controlled artifacts and evidence integrity. `go run ./server/tools/verify` runs the repository gate sequence. If public output changed, also run `go run ./server/tools/render` and inspect `dist/`.
