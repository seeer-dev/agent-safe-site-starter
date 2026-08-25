# Beginner Starter Architecture Hardening Delivery Plan

Change ID: sitecore-lessons-for-beginner-starter
Revision: 1
Status: Verifying

Normative specification: [`spec.md`](spec.md)

Repository baseline: `ade97a8d58dfffe1d2b61633d552dfffcb7ba3f6`

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| `architecture.yaml` documents module/platform ownership | `architecture.yaml` | It is already the natural human/agent architecture source |
| `archcheck` separately hard-codes module and platform paths | `server/tools/archcheck/main.go` | Documentation and enforcement can drift |
| the starter is intentionally one static-first site with one Go backend | `README.md`, `AGENTS.md`, `skills/site/SKILL.md` | SiteCore multi-site/runtime machinery would be counterproductive here |
| cross-module isolation is enforced only at the Go import layer | `archcheck`, `architecture.yaml` | Typed bootstrap wiring should remain the normal synchronous collaboration path |
| `commerce` owns many concerns and has grown large | `architecture.yaml`, `server/internal/modules/commerce/` | Internal cohesion should be improved before inventing more top-level architecture |

## Scope lock

- `architecture.yaml`
- `AGENTS.md`
- `.github/workflows/ci.yml`
- `server/tools/archcheck/**`
- `skills/site/SKILL.md`
- `skills/site/references/architecture-boundaries.md`
- `skills/site/references/change-safety.md`
- `specs/changes/sitecore-lessons-for-beginner-starter/**`

## Dependency-ordered slices

### Slice 1: Manifest-driven import enforcement

Outcome: add a deliberately small `enforcement.import_rules` section to `architecture.yaml`; refactor `archcheck` to read it and fail closed; add regression tests proving configured roots and rule flags drive behavior; run those tests in the existing CI architecture gate before scanning the real repository.

Covers REQ-001, AC-001, AC-002.

Verification:

```text
go test ./server/tools/archcheck -count=1
go run ./server/tools/archcheck
```

Negative proof: remove a required import-rule field in a test fixture and require parser failure; configure an alternate module root and require a violation there.

### Slice 2: Beginner architecture boundary reference

Outcome: add one concise architecture reference for the site router and update repository/skill guidance so agents preserve the single-site product shape, use consumer-owned typed ports, split large modules internally before proliferating modules, and preserve existing UI contracts.

Covers REQ-002, REQ-003, REQ-004, AC-003, AC-004, AC-005.

Verification: source inspection plus repository search proving no new user-facing lifecycle verb is introduced and the site skill still exposes `propose` followed by plain `apply`.

### Slice 3: Repository verification and acceptance

Outcome: run the controlled-spec gate, archcheck tests, full verifier, and frontend contract gate where available; reconcile evidence to the current revision before changing status to `Accepted`.

Hard dependencies: Slices 1 and 2.

Verification:

```text
go run ./server/tools/speccheck
go test ./server/tools/archcheck -count=1
go run ./server/tools/verify
make verify-contracts
```

If the execution environment cannot run repository verification, leave the change in `Verifying` with the affected evidence pending rather than claiming acceptance.

## Traceability matrix

| REQ / AC | Slice | Proof |
|---|---|---|
| REQ-001, AC-001 | 1 | alternate-root regression test proves architecture data drives detection |
| REQ-001, AC-002 | 1 | missing/unsupported/unsafe policy regression tests fail closed |
| REQ-002, AC-003 | 2 | agent guidance explicitly preserves beginner single-site product shape |
| REQ-003, AC-004 | 2 | typed-port, cohesion-first, and UI-first rules are directly stated |
| REQ-004, AC-005 | 2 | site skill keeps propose/review/apply and hides architecture machinery |

## Risks and controls

- Risk: parsing all YAML would add a dependency or another framework. Control: `archcheck` reads only its deliberately flat enforcement subsection with the standard library and rejects unsupported keys.
- Risk: architecture policy becomes an excuse for generic platform metadata. Control: the enforcement section contains only rules the checker actually consumes.
- Risk: SiteCore concepts leak into the beginner starter. Control: the new reference explicitly names those concepts as out of scope unless the starter's product purpose changes.
- Risk: large-commerce cleanup becomes a risky rewrite. Control: this change records the cohesion rule but moves no commerce business code; a later controlled change can split cohesive files/packages with behavior-preserving tests.
