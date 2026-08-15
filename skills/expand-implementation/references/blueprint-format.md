# Implementation blueprint format

Use this format for every non-trivial implementation packet written into the controlled change's existing `plan.md`.

## Contents

1. [Expansion context](#expansion-context)
2. [Packet header](#packet-header)
3. [Repository anchors](#repository-anchors)
4. [Behavior contract](#behavior-contract)
5. [Ordered implementation](#ordered-implementation)
6. [Integration trace](#integration-trace)
7. [Verification and falsification](#verification-and-falsification)
8. [Drift and completion](#drift-and-completion)
9. [Blueprint-wide gates](#blueprint-wide-gates)

## Expansion context

Place one context block before the first packet:

```markdown
## Implementation expansion context

Proposal revision: <revision>
Proposal status: <status>
Repository baseline: <commit recorded by the controlled change>
Observed HEAD: <commit inspected during expansion>
Pre-existing dirty paths:
- `path` - preserve; not attributable to this expansion
Packet ID mapping:
- Existing Slice N -> Packet SNN
```

Use `none` when the dirty-path or ID-mapping list is empty. Preserve existing stable slice IDs when possible. When the source plan has no IDs, assign packet IDs once and retain them across later expansion revisions.

## Packet header

```markdown
### Packet SNN: <observable outcome>

Status: executable | already-implemented | SPEC_DRIFT | DECISION_REQUIRED | ENVIRONMENT_BLOCKED
Covers: REQ-NNN, AC-NNN
Hard dependencies: SNN or none
Outcome: <one observable result>
Safe failure / rollback: <how partial work remains non-authoritative or reversible>
```

Use one outcome per packet. Split packets when they have different consumers, authorities, rollback boundaries, or proof environments.

Use `already-implemented` only for retrospective expansion when inspected repository evidence already contains the complete behavior. Distinguish historical implementation instructions from current observed evidence; do not present completed edits as future work or expected proof as newly observed proof.

When the proposal is Accepted or Superseded in the comparison base, do not write the retrospective packet into its immutable artifacts. Return it only through the review response or an isolated test harness; it is validation output, not a controlled amendment or new acceptance evidence.

`Hard dependencies` must state why ordering is required. Do not list cleanup preference or directory order as a dependency.

## Repository anchors

List inspected evidence, not anticipated filenames:

```markdown
#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Behavior owner | `path/file` | `Type.Method` | <observed behavior> |
| Contract | `path/file` | `<route/schema/type>` | <observed shape> |
| Consumer | `path/file` | `<entry/caller/component>` | <reachability evidence> |
| Existing proof | `path/file` | `TestName` | <what it actually asserts> |
```

Then define boundaries:

```markdown
Read set:
- `path/file` - why it must be understood

Modify set:
- `path/file` - exact symbol or section and permitted change

Must not modify:
- `path/file-or-region` - reason
```

Rules:

- Use exact paths already verified in the repository.
- Use symbols, routes, table names, headings, or other stable anchors. A line number alone is insufficient.
- Include the real consumer or state explicitly that no consumer exists.
- Do not silently place a newly discovered file into the modify set. Record `SPEC_DRIFT` until scope and the controlled artifacts cover it.

## Behavior contract

```markdown
#### Behavior contract

Before:
- <current observable behavior and failure behavior>

After:
- <required observable behavior and failure behavior>

Preserved invariants:
- <public API, data, authorization, compatibility, or ordering invariant>

Inputs and authority:
- <who or what supplies each consequential input>
- <where it is validated or derived authoritatively>

Errors and edge states:
- <empty, invalid, unavailable, unauthorized, conflict, retry, or partial-failure semantics>
```

Do not translate missing product semantics into implementation defaults. Use `DECISION_REQUIRED`.

## Ordered implementation

Write steps that can be checked independently:

```markdown
#### Ordered edits

1. `path/file` - `SymbolOrAnchor`
   - Change: <exact edit>
   - Constraint: <what must remain unchanged>
   - Result: <observable or inspectable state after this step>
2. `path/test_file` - `TestName`
   - Change: <exact fixture/assertion addition>
   - Constraint: <existing tests that must remain>
   - Result: <specific failure before implementation and pass after restoration>
```

Rules:

- Avoid "update as needed", "wire everything", "handle errors", and similar open-ended instructions.
- Name existing interfaces to reuse. Do not request an abstraction unless two real consumers or implementations require it.
- State creation, update, deletion, and migration behavior separately when their risks differ.
- Identify generated files and their source; edit the source unless repository instructions explicitly require generated output edits.

For an `already-implemented` retrospective packet, replace `Ordered edits` with:

```markdown
#### Observed implementation

1. `path/file` - `SymbolOrAnchor`
   - Observed behavior: <what the repository currently does>
   - Difference from baseline: <attributable historical change, or unknown>
   - Current evidence: <test, caller, artifact, or command actually observed>
   - Attribution boundary: <what cannot be attributed from the inspected state>
```

Do not invent historical steps from the current tree. Use `unknown` when the baseline diff or authorship evidence was not inspected.

## Integration trace

Include this section whenever a packet crosses a boundary:

```text
real entry point
  -> input/session/config producer
  -> validation and authorization
  -> behavior owner
  -> persistence or external adapter
  -> contract
  -> actual consumer
  -> observable result
```

For every hop, name the path and symbol. State data shape, stable identifier, authority, failure behavior, and fallback behavior where relevant.

A definition without a real invocation is not integration evidence. A test fixture, developer-only seed, browser-local value, or hand-edited state is not a production producer.

## Verification and falsification

Use one row per claimed behavior:

```markdown
#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| <claim> | `<repo-relative directory>` | `<exact argv>` | `TestName` or `<artifact>` | <exact observation> | <rejection/failure> | <bounded temporary mutation> | <restore + diff check> |
```

Requirements:

- Discover commands from repository manifests, CI, or existing tools. Record the working directory separately from the exact argument vector.
- Name selected tests. A package-level `ok` alone does not prove they executed.
- Make the failure trigger target the assertion that protects the claim; compile, setup, or unrelated timeout failures do not count.
- Record an unavailable live environment as `ENVIRONMENT_BLOCKED`; do not substitute a mock or another database for a required live semantic claim.
- Distinguish expected evidence in the blueprint from observed evidence recorded after apply.
- Include formatting, type, lint, scope, controlled-spec, and repository verifier checks only when they are relevant; do not let them replace behavior proof.

## Drift and completion

For a blocked packet, write:

```markdown
#### Expansion stop

Marker: SPEC_DRIFT | DECISION_REQUIRED | ENVIRONMENT_BLOCKED
Claim that cannot be executed: <proposal statement>
Repository evidence: <paths, symbols, commands, or absence checked>
Impact: <REQ/AC and dependent packets>
Smallest unblock action: <inspect, decide, authorize scope, or provide environment>
```

For an executable packet, write:

```markdown
#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] The bounded failure trigger fails for the claimed reason and is restored.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.
```

Do not mark these boxes during proposal expansion. They are execution gates for apply.

For an `already-implemented` retrospective packet, use this separate gate:

```markdown
#### Retrospective evidence gate

- [ ] Current repository anchors and real consumers were observed.
- [ ] Baseline differences are cited or explicitly unknown.
- [ ] Commands are reported as observed only when they were actually run.
- [ ] Expected proof, implementation reports, and current observations are distinguished.
- [ ] No immutable controlled artifact was modified.
- [ ] The replay is not presented as new acceptance evidence.
```

## Blueprint-wide gates

Before calling the proposal review-ready, verify:

```markdown
| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass/blocker |
| Every AC maps to proof in at least one packet | pass/blocker |
| Every packet maps back to REQ/AC | pass/blocker |
| Baseline, observed HEAD, dirty paths, and packet ID mapping are explicit | pass/blocker |
| All paths and symbols were inspected | pass/blocker |
| Hard dependency graph has no unexplained edge or cycle | pass/blocker |
| Scope covers every modify path and no unrelated path | pass/blocker |
| Product/authority decisions are approved or blocked | pass/blocker |
| No agent/provider/model identity changes packet semantics | pass/blocker |
| No expected result is presented as observed evidence | pass/blocker |
```

Keep the controlled proposal in Draft when a blocker changes intended behavior, authority, scope, cost, permissions, data handling, or another trust boundary. An implementation detail gap may be corrected in the plan only when the normative contract remains unchanged.
