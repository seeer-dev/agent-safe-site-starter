# Architecture Map Currency Specification

Change ID: architecture-map-currency
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved updating the architecture map on 2026-08-18 after review found it lists 8 of 14 deterministic tools and 4 of 6 modules.
Repository baseline: 5cb6cd85194a2f4ad9dd66ac64f7cbc534b13a90
Supersedes: none

## Outcome

`architecture.yaml` describes the repository as it is, so an agent that follows
`AGENTS.md:3` and reads the map before a broad change is not misled into
rebuilding something that already exists or into treating a module as absent.

No runtime behavior, route, schema, or dependency changes.

## Scope

In scope:

- The module inventory and per-module table ownership.
- The deterministic tool inventory.
- Disclosure of one cross-module data coupling that the import-level rule does
  not cover.

Out of scope:

- Changing module boundaries, moving code, or resolving the disclosed coupling.
- Any change to `archcheck`, `speccheck`, or other governance tooling.
- Adding aspirational entries for work that is only proposed.

## Decisions and invariants

- The map records what exists, never what is planned. A `Draft` proposal does
  not earn an entry.
- Ownership means the module that writes the table in normal operation.
- Where a rule is enforced only at one level, the map says so rather than
  implying full enforcement.

## Requirements

### REQ-001: Module inventory reflects the repository

The `modules` section MUST list every directory under
`server/internal/modules/`, and each entry's `owns` list MUST match the tables
that module actually writes.

#### AC-001: Every module present with accurate ownership

- GIVEN the directories under `server/internal/modules/`
- WHEN `architecture.yaml` is read
- THEN every module directory MUST appear exactly once, `sitecontent` and
  `staff` MUST be present, `staff_members` MUST be owned by `staff` rather than
  `commerce`, and `commerce` MUST list the order-item, shipping-method, and
  restock-idempotency tables it writes.

### REQ-002: Deterministic tool inventory is complete

The `tools.deterministic` list MUST contain every command under
`server/tools/` that an operator or agent is expected to invoke.

#### AC-002: Tool list matches the repository

- GIVEN the directories under `server/tools/` excluding the shared `internal`
  helper package
- WHEN `architecture.yaml` is read
- THEN each MUST appear in `tools.deterministic`, including
  `local-postgres-gate`, `postgres-live-gate`, `migration-parity`, `media-gc`,
  `preview`, and `seed`.

### REQ-003: Enforcement limits are disclosed

Where an architectural rule is enforced at one level only, the map MUST record
the limit and any known instance that the enforcement does not reach.

#### AC-003: Import-level isolation limit is stated

- GIVEN `modules_must_not_import_other_modules`, which `archcheck` enforces by
  inspecting Go imports
- WHEN `architecture.yaml` is read
- THEN it MUST state that the rule is import-level only, and MUST record that
  `commerce` writes `media_assets.unassociated_since_unix`
  (`server/internal/modules/commerce/store.go:434,440`), a table owned by
  `media`, which no current gate detects.
