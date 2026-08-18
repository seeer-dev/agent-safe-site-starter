# PostgreSQL Connection Pool Configuration Specification

Change ID: postgres-connection-pool-configuration
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved the pre-launch operability batch on 2026-08-18. PostgreSQL now executes in tests and is the production default, while the pool is left at database/sql defaults of unlimited open connections and two idle.
Repository baseline: 70c7098865909617e6b3774f3b86ca2a08cb4edd
Supersedes: none

## Outcome

A PostgreSQL deployment bounds its own connection use instead of inheriting
`database/sql`'s unlimited default, and an operator can size that bound without
editing code.

No query, schema, route, or authentication behavior changes.

## Scope

In scope:

- Pool limits for the PostgreSQL driver and the settings that carry them.
- The operator-facing documentation of those settings.

Out of scope:

- Choosing values for a specific deployment. Sizing depends on the pooler mode
  in front of PostgreSQL, which is an owner decision this change does not make.
- pgx execution-mode tuning for transaction poolers.
- Any change to SQLite behavior.

## Decisions and invariants

- The defaults are conservative and safe for a single small instance. They are
  a floor, not a recommendation for any particular deployment.
- SQLite keeps `SetMaxOpenConns(1)`. Its serialization is load-bearing: several
  lock guards are documented as no-ops on SQLite precisely because of it.
- An unparseable or non-positive setting falls back to the default rather than
  producing an unbounded pool.

## Requirements

### REQ-001: PostgreSQL pools are bounded and configurable

Opening a PostgreSQL database MUST apply a maximum open connection count, an
idle count, a connection lifetime, and an idle timeout, each overridable
through configuration.

#### AC-001: Limits are applied and overridable

- GIVEN a PostgreSQL handle opened through the platform layer
- WHEN configuration supplies pool values, and separately when it supplies none
- THEN the supplied values MUST be applied, the documented defaults MUST be
  applied when nothing is supplied, and a zero, negative, or unparseable value
  MUST fall back to the default rather than leaving the pool unbounded.

### REQ-002: SQLite serialization is preserved and the settings are documented

The SQLite single-connection invariant MUST remain, and every new setting MUST
appear in the operator documentation with its ownership.

#### AC-002: SQLite remains single-connection

- GIVEN a SQLite handle opened through the platform layer
- WHEN any pool configuration is present
- THEN the handle MUST still be limited to one open connection, because the
  lock guards documented as SQLite no-ops depend on that serialization.

#### AC-003: Settings are documented for operators

- GIVEN an operator configuring Railway or Cloudflare Pages
- WHEN they read the environment documentation
- THEN each new setting MUST appear in the ownership table and in the
  production inventory, and the documentation MUST state that sizing depends on
  the pooler mode and is not decided here.
