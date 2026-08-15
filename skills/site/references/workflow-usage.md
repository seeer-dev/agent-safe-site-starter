# Site workflow usage

Use this reference when the user says `propose`, `apply`, or `walkthrough`, or asks how a controlled specification connects to implementation and UX validation. These are lifecycle verbs inside the site skill, not separate skills or CLI commands. They do not require Spectra or OpenSpec.

## Normal user workflow

```text
user: propose <outcome>
  agent: inspect -> expand implementation packets in plan.md
         -> draft controlled artifacts -> describe the proposal

user: apply
  agent: approve the sole current proposal -> implement all slices
         -> test -> required walkthrough -> verify -> record evidence
```

The normal interface has only two user actions:

1. Ask `propose <outcome>`. The agent inspects the current repository and returns a concise proposal summary. Product code stays unchanged.
2. After reviewing the summary, say `apply`. When exactly one current review-ready proposal exists, the agent treats this as approval of its latest summarized revision and completes the approved scope.

The user does not need to provide a change ID, revision, status, REQ/AC ID, slice name, artifact path, test command, or walkthrough instruction. The agent owns those mechanics and exposes them only for an audit or to explain a genuine blocker.

## What propose returns

The agent creates or updates `specs/changes/<change-id>/control.json`, `spec.md`, `plan.md`, and `evidence.md` internally. For a non-trivial proposal, it reads `skills/expand-implementation/SKILL.md` and expands implementation instructions into the existing `plan.md` before describing:

- the intended user-visible outcome;
- important boundaries and non-goals;
- consequential product, authority, data, and runtime-ownership decisions;
- dependency-ordered vertical slices at a useful review level;
- material risks or unresolved blockers; and
- how completion and UX will be proved.

Keep the response readable. Do not dump internal IDs, state transitions, path allowlists, or commands unless the user asks. A propose-only request ends with a review-ready `Draft` and does not modify product code.

Implementation expansion is internal planning work, not a third lifecycle action. It does not select an implementer or vary instructions by agent, provider, model, or team role.

Example:

> 請 propose 商品整合，先不要實作。

## What plain apply does

Plain `apply` approves the sole latest proposal that the agent summarized when no blocker remains. The agent then:

1. records the approval basis and advances controlled-spec state;
2. locks narrow scope and preserves unrelated worktree changes;
3. implements every in-scope slice in dependency order, unless the user explicitly limited the scope;
4. runs targeted checks as each behavior becomes testable;
5. performs required runtime/UX walkthroughs when acceptance depends on user experience;
6. repairs in-contract implementation gaps and replays affected checks;
7. runs `speccheck`, scope checks, the repository verifier, and any required render inspection; and
8. updates evidence and reports the delivered outcome, validation, and remaining risks.

The agent must not ask the user to select each slice or manually advance spec status.

Example:

> apply

## Discoveries during apply

Handle ordinary discoveries without interrupting the user:

- fix defects that violate the approved outcome and add regression evidence;
- correct false repository assumptions, missing fields, consumers, UI states, or dependencies in the plan and artifacts;
- update scope, traceability, and evidence as implementation progresses; and
- run or replay every affected test and walkthrough.

Pause and describe a proposal delta only when it requires new authority: it materially changes agreed product behavior, cost, permissions, data handling/retention, destructive impact, or another trust boundary. The user's next plain `apply` can approve that latest summarized amendment. Also ask when multiple current proposals make `apply` ambiguous or a genuine implementation/verification blocker remains.

## Optional walkthrough audit

Apply already performs walkthroughs required for acceptance. A separate `walkthrough` request is optional and read-only: freeze the implementation state, walk critical journeys by persona and state, and return evidence plus classified gaps without fixing code.

Example:

> 請只做 walkthrough，不修碼。

## Internal phase boundaries

- `propose`: controlled spec and planning artifacts may change; product code must not.
- `apply`: only a `Ready`/`Applying` controlled change and approved amendments may authorize product edits.
- internal acceptance walkthrough: implementation may be repaired through the same apply run when behavior remains inside the approved contract.
- explicit walkthrough audit: evidence and gap artifacts may change; implementation must not.
- a chained request may combine phases, but the agent still preserves each internal gate and evidence trail.
