# User walkthrough

Use this reference after an applied slice to discover runtime, contract, permission, and experience gaps from a user's perspective. A walkthrough observes one fixed source state; it does not repair implementation while evidence is being collected.

## Preconditions

1. Read the controlled spec revision/status, plan, affected surface contracts, acceptance evidence, and repository status.
2. Run the narrowest mechanical gates first: targeted tests, type/lint/build checks when applicable, `scopecheck`, and the repository verifier.
3. Record failures that predate the slice and stop when the app cannot be exercised safely.
4. Freeze the source revision and runtime configuration for the walkthrough. Do not edit code until the report is complete.
5. Use test accounts and reversible test data appropriate to each persona; never expose secrets in evidence.

## Choose the inspection path

- **Static or rendered output:** start HTTP-first. Verify status, rendered content, asset links, and the store -> Go renderer -> `dist/` result before browser interaction.
- **Client-side interaction:** start browser-first after the server/build gates pass. Inspect visible behavior together with the browser-to-Go-API requests it depends on.
- **Hybrid surface:** verify which datum is static and which is runtime, then prove failure behavior without allowing one path to mask the other.

## Walk the critical journey

For each in-scope surface and persona:

1. Load the real route or mount point directly and through expected navigation.
2. Confirm orientation, attention cues, primary action, consequence, and feedback/recovery.
3. Verify relevant empty, loading, error, forbidden, and success states.
4. Confirm displayed data comes from the contracted store, render path, or API rather than a fixture or fallback.
5. Exercise valid and invalid input, validation messages, cancellation, retry, and duplicate submission behavior where relevant.
6. For a permitted write, prove both the immediate receipt and the authoritative follow-up state after refresh, refetch, or navigation.
7. For a forbidden write or view, prove denial and absence of protected data.
8. Check keyboard flow, focus, labels, status announcements, and responsive behavior for the critical task.

Keep exploration bounded to affected surfaces, their dependency paths, and critical journeys. Record adjacent findings without turning the walkthrough into an unscoped product audit.

## Evidence standard

Each observation should identify:

```text
Source revision/configuration:
Spec revision and REQ/AC ID:
Surface and route:
Persona:
State or action:
Expected behavior:
Observed behavior:
Evidence: command, assertion, request status, log excerpt, or screenshot
Data cleanup or resulting record:
```

Evidence must prove the claim without credentials, tokens, raw PII, or fabricated success. A screenshot alone does not prove persistence or authorization; pair it with the appropriate request, follow-up state, or test assertion.

Store required walkthrough evidence as a non-secret receipt under the controlled change, for example `specs/changes/<change-id>/receipts/<surface>-walkthrough.md`. Mention that relative path in the mapped evidence proof. Under strict evidence mode, declare the `walkthrough` receipt kind for every runtime REQ/AC that requires it; `speccheck` must reject passed evidence when the receipt is absent, empty, unsafe, or belongs to another revision.

The phrase "walkthrough recommended," a screenshot without authoritative follow-up, or a plan to test later is evidence of an incomplete walkthrough and must remain pending.

## Gap report

Classify every gap with one of these labels:

| Label | Meaning |
|---|---|
| `broken` | Contracted behavior exists but fails or regresses |
| `missing` | Required surface, state, action, evidence, or permission path is absent |
| `rough` | Outcome works but creates material confusion, friction, or recovery risk |
| `polish` | Non-blocking presentation improvement |
| `contract-drift` | Implementation and plan/surface contract disagree |

For each gap, include affected REQ/AC ID, surface/persona, expected versus observed behavior, evidence, user impact, whether it blocks acceptance, spec impact, and the receiving slice or decision owner.

## Close and replay

- Do not fix gaps during walkthrough mode.
- Send blocking `broken` and `missing` gaps to a new apply phase.
- For `contract-drift`, determine whether the normative spec changed. Amend and re-approve it first when required; otherwise update the plan or surface contract, identify invalidated evidence, then apply.
- Record `rough` and `polish` items without silently expanding the accepted slice.
- Re-run mechanical gates and only the affected persona/state/journey checks, plus any dependency checks invalidated by the change.

The walkthrough is complete when every in-scope REQ/AC and critical journey has evidence, all gaps are classified and routed, and no blocking gap is reported as accepted work. Only then may the controlled spec move from `Verifying` to `Accepted`.
