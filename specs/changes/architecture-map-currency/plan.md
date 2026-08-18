# Architecture Map Currency Delivery Plan

Change ID: architecture-map-currency
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Observation | Evidence | Delivery implication |
|---|---|---|
| Map lists 4 modules; 6 exist | `ls server/internal/modules/` returns commerce, contact, content, media, sitecontent, staff | Add `sitecontent` and `staff` |
| `commerce.owns` claims `staff_members` | `staff_members` is written only in `server/internal/modules/staff/store.go` | Move ownership to `staff` |
| `commerce.owns` omits tables it writes | `order_items`, `shipping_methods`, `restock_idempotency` appear in `commerce/store.go` | Add them |
| `media.owns` names `media_objects` only | `media_assets` and `media_gc_jobs` are also written by the media module | Add them |
| Map lists 8 deterministic tools; 14 exist | `ls -d server/tools/*/` minus the shared `internal` package | Complete the list |
| Import isolation is enforced; data isolation is not | `archcheck/main.go` walks Go imports only; `commerce/store.go:434,440` writes `media_assets` | Disclose the limit and the instance |

## Scope lock

- `architecture.yaml`
- `specs/changes/architecture-map-currency/**`

No production code, migration, workflow, or governance tool is in scope.

## Dependency-ordered slices

### Slice 1: Inventory correction

Outcome: modules and tools sections match the repository.

Acceptance evidence: mechanical comparison of the map against
`ls server/internal/modules/`, `ls -d server/tools/*/`, and the table names
written by each module. Covers REQ-001, REQ-002, AC-001, AC-002.

Rollback: revert the file; nothing depends on it at runtime.

### Slice 2: Enforcement-limit disclosure

Outcome: the map states that module isolation is import-level, and records the
`commerce` → `media_assets` write.

Hard dependencies: Slice 1 establishes the ownership vocabulary the disclosure
refers to.

Acceptance evidence: the named limitation and the cited file:line are present
and correct. Covers REQ-003, AC-003.

Rollback: remove the disclosure block only.

## Traceability matrix

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 1 | Module directory listing vs map; table-write grep per module |
| REQ-002, AC-002 | 1 | Tool directory listing vs map |
| REQ-003, AC-003 | 2 | `archcheck` source inspection; `commerce/store.go:434,440` |

## Risks and controls

- Risk: the map becomes aspirational. Control: entries describe only what
  exists; the three current `Draft` proposals are deliberately absent.
- Risk: disclosing the coupling is mistaken for approving it. Control: the
  entry is phrased as a known gap with no implied sanction, and resolving it is
  explicitly out of scope.
- Risk: ownership is guessed. Control: ownership is derived from which module
  writes the table, verified per module.
