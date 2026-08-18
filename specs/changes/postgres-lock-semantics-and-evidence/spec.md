# PostgreSQL Lock Semantics and Execution Evidence Specification

Change ID: postgres-lock-semantics-and-evidence
Revision: 1
Status: Verifying
Decision authority: Repository owner/user
Approval basis: Repository owner approved opening and delegating this next phase on 2026-08-14 after reviewing the exact fresh-migration, lock-semantics, and named-test evidence outcomes.
Repository baseline: f8053f8ea3ac640586384e24667933e5f556b689
Supersedes: postgres-execution-gate

## Outcome

Replace the under-evidenced PostgreSQL branch claim from `postgres-execution-gate` with tests and CI evidence that are sensitive to the behavior they name. Migration timestamp insertion must execute in a fresh namespace, staff owner locking must preserve the last-owner invariant under deterministic overlap, media row locks and `SKIP LOCKED` must produce observable competing-transaction behavior, and CI must prove every required live test executed and passed rather than merely returning a package-level success.

This change supersedes only the prior change's AC-002 evidence and lock-coverage claims. Its migration-parity, database-governance, SQLite, and explicit cancellation-quarantine outcomes remain historical accepted behavior and are not reimplemented here.

## Scope

In scope:
- Fresh-schema or fresh-database execution of the real PostgreSQL migration sequence, including a newly inserted `schema_migrations.applied_unix` value.
- Deterministically overlapped PostgreSQL staff-owner operations proving the `FOR UPDATE` guard prevents zero active owners.
- Observable media `FOR UPDATE` blocking and `FOR UPDATE SKIP LOCKED` behavior across separate transactions or connections.
- A standard-library live-test gate that consumes `go test -json`, requires an exact package/test inventory, rejects missing or skipped tests, and propagates test-process failure.
- CI wiring for the live-test gate and an exact verifier-wiring regression test.
- Minimal, unexported test synchronization seams only if database-observable coordination cannot make overlap deterministic; such seams must have no non-test runtime behavior when unset.

Out of scope:
- Database schema or migration content changes.
- Changes to staff role policy, last-owner product semantics, media retention policy, GC eligibility, or order-cancellation loser semantics.
- PostgreSQL performance tuning, auth caching, observability baseline work, or SQLite removal.
- Editing the immutable accepted artifacts under `specs/changes/postgres-execution-gate/`.

## Decisions and Invariants

- Fresh Means Unapplied: the migration test must use a namespace with no `schema_migrations` rows and no pre-existing application tables. Querying timestamps inserted by the preceding CI migration step is not evidence for the migration test.
- Lock Semantics, Not Reachability: removing the relevant `FOR UPDATE` or `SKIP LOCKED` behavior must make the mapped semantic test fail. Sequential success is insufficient.
- Deterministic Overlap: tests that claim simultaneous behavior use two independently controlled operations and an explicit synchronization point, bounded wait, collected outcomes, and final invariant. Repetition alone does not create overlap.
- Staff Invariant: two concurrent attempts to demote or disable different active owners may not both succeed; exactly one active owner remains, and the losing operation reports the canonical last-owner failure.
- Media Lock Invariant: a competing operation observes bounded blocking while a target row is locked. A `SKIP LOCKED` claimant does not block on or return the locked row, may claim another eligible row, and cannot duplicate a claim.
- Exact Execution Inventory: the gate owns an explicit package/test list. A missing, skipped, duplicate-only, or package-level `ok` result is failure. Local absence of PostgreSQL may remain an explicit skip for direct developer runs, but the required CI gate cannot accept it.
- Honest Acceptance: repository-side reachability is not enough for AC-005. The change remains `Verifying` until a real PostgreSQL CI run is recorded with all exact tests observed as passed.
- Reviewer Independence: Agy may implement; Claude may adversarially review. If Claude changes implementation or tests, those edits require a separate Codex replay before independent-review evidence can pass.

## Requirements

### REQ-001: Fresh PostgreSQL Migration Execution
The live migration test MUST apply the repository's PostgreSQL migrations in an isolated, initially empty namespace and prove that the PostgreSQL timestamp insertion branch creates current positive `applied_unix` records.

#### AC-001: Fresh Migration Timestamp
- GIVEN a live PostgreSQL server whose shared CI database may already be migrated
- WHEN the migration integration test creates an isolated empty namespace and calls the real migration applicator
- THEN all repository PostgreSQL migrations MUST be newly applied there, `schema_migrations` MUST contain the exact current migration set, and newly created positive timestamps MUST be observed without reading the shared pre-applied records.

### REQ-002: Observable PostgreSQL Lock Semantics
The staff and media PostgreSQL tests MUST observe the concurrency effects for which their locking clauses exist.

#### AC-002: Staff Cross-Row Last-Owner Serialization
- GIVEN two active owners in live PostgreSQL and two independently controlled operations targeting different owners
- WHEN both demotion or disable attempts overlap across separate transactions/connections
- THEN the operations MUST serialize through the owner-row lock, exactly one MUST succeed, the other MUST return the canonical last-owner error, and one active owner MUST remain.

#### AC-003: Media Blocking and Skip-Locked Claims
- GIVEN live media rows and one transaction holding the relevant target lock
- WHEN a competing reservation/completion and a GC claimant execute through separate connections
- THEN the locking operation MUST exhibit bounded blocking or a lock-timeout observation, `SKIP LOCKED` MUST omit the held row without blocking, an unlocked eligible row MAY be claimed, and no object key may be claimed twice.

### REQ-003: Exact Live-Test Execution Gate
CI MUST mechanically prove the complete required live PostgreSQL test inventory executed and passed.

#### AC-004: Missing and Skipped Test Rejection
- GIVEN `go test -json` output for the configured live PostgreSQL packages
- WHEN a required package/test is absent, skipped, duplicated without a pass, or the test process exits non-zero
- THEN the live-test gate MUST exit non-zero with a diagnostic naming the unsatisfied package/test; a complete exact pass set MUST exit zero.

#### AC-005: Observed Live CI Run
- GIVEN the implementation commit running in repository CI with its ephemeral PostgreSQL service
- WHEN the live PostgreSQL gate completes
- THEN evidence MUST record the run identity and show AC-001 through AC-004's exact tests as executed and passed with no skip; without that observation the controlled change MUST remain `Verifying`.

### REQ-004: Mutation-Sensitive and Exact Wiring Evidence
The test and verifier wiring MUST fail when the claimed protection is removed or miswired.

#### AC-006: Exact Verifier Command Assertion
- GIVEN the repository verifier command inventory
- WHEN its unit test inspects the concurrency stress step
- THEN it MUST compare the complete expected argument vector, including the intended packages and repeat count, rather than accepting independent substring presence.

#### AC-007: Red-Before-Green Lock and Migration Evidence
- GIVEN each new migration or lock semantic test
- WHEN the mapped PostgreSQL timestamp, `FOR UPDATE`, or `SKIP LOCKED` behavior is temporarily disabled within authorized scope
- THEN the targeted semantic assertion MUST be observed failing; after restoration the same test MUST pass, and the temporary mutation MUST not remain in the diff.

## Amendments

None.
