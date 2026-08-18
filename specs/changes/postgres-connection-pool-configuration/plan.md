# PostgreSQL Connection Pool Configuration Delivery Plan

Change ID: postgres-connection-pool-configuration
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| Only SQLite is bounded | `platform/database/database.go` sets `SetMaxOpenConns(1)` under a SQLite-only branch | Add a PostgreSQL branch |
| PostgreSQL defaults are unbounded | `database/sql` defaults: unlimited open, two idle | A traffic spike can exhaust the provider's connection allowance |
| PostgreSQL now really runs | `server/tools/local-postgres-gate` and the live gate execute against PostgreSQL 16 | The setting is testable rather than theoretical |
| SQLite serialization is load-bearing | `staff/store.go` documents `FOR UPDATE` as a SQLite no-op relying on single-connection serialization | Must not be widened |

## Scope lock

- `server/internal/config/config.go`
- `server/internal/config/config_test.go`
- `server/internal/platform/database/database.go`
- `server/internal/platform/database/database_test.go`
- `server/internal/bootstrap/app.go`
- `.env.development.example`, `.env.production.example`
- `docs/environment-configuration.md`
- `specs/changes/postgres-connection-pool-configuration/**`

## Dependency-ordered slices

### Slice 1: Configurable pool settings

Outcome: configuration carries four pool values with documented defaults and
fallback on invalid input; the platform layer applies them for PostgreSQL only.

Acceptance evidence: table tests over supplied, absent, zero, negative, and
unparseable values; a SQLite case asserting the single-connection invariant
survives. Covers REQ-001, AC-001, AC-002.

Rollback: revert both files; the previous unbounded default returns.

### Slice 2: Operator documentation

Outcome: the ownership table and the production inventory list every new
setting, and state that sizing depends on the pooler mode.

Hard dependencies: Slice 1 fixes the names and defaults being documented.

Acceptance evidence: each name appears in the ownership table, the production
inventory, and the development example. Covers REQ-002, AC-003.

Rollback: revert the documentation only.

## Traceability matrix

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 1 | Pool-setting table test including invalid input |
| REQ-002, AC-002 | 1 | SQLite single-connection assertion |
| REQ-002, AC-003 | 2 | Name presence across the three documents |

## Risks and controls

- Risk: a default is read as a sizing recommendation. Control: the
  documentation states the defaults are a conservative floor and that sizing
  depends on the pooler mode, which this change does not decide.
- Risk: widening SQLite breaks lock guards that rely on serialization.
  Control: an explicit assertion, and the PostgreSQL branch is separate.
- Risk: an invalid value silently produces an unbounded pool. Control: fallback
  to the default is asserted for zero, negative, and unparseable input.
