# Reviewer acceptance notes — revision 1

Change ID: postgres-lock-semantics-and-evidence
Reviewer: Claude Opus 5, coordinating; implementation by agy
Date: 2026-08-14
Status at time of writing: `Verifying`

This is **not** an independent-review receipt for any acceptance criterion. It
records what an adversarial acceptance pass did and did not establish, so the
reviewer who eventually closes the live-dependent criteria knows exactly what
is already settled.

The reviewer did not modify any implementation or test file in this change.
The entire diff is agy's work across three rounds; the reviewer's contribution
was defect reports. That preserves independence for a later receipt.

## What was established by mutation, not by report

Each of these was verified by breaking the guarded behaviour, observing the
targeted assertion fail, restoring, and confirming the working tree was clean.

| Target | Mutation applied | Observed failure | Restored |
|---|---|---|---|
| Live-gate rejects skipped tests | none needed — no `TEST_DATABASE_URL` locally | `test was skipped (must run and pass on live PostgreSQL)` naming all three required tests, exit 1 | n/a |
| Live-gate rejects missing tests | renamed `TestPostgresLiveStaffLockActiveOwners` | `required test missing from execution stream: …/staff TestPostgresLiveStaffLockActiveOwners`, distinct from the skip diagnostic | yes, zero residue |
| Verifier exact argv (AC-006) | `-count=10` → `-count=5` in `verify/main.go` | `--- FAIL: TestVerifyConcurrencyStepExactArgv` | yes, zero residue |

The live-gate's two diagnostics are distinguishable, which matters: a gate that
reported "missing" and "skipped" identically could not tell a deleted test from
an unconfigured environment.

## Defects found and returned to the implementer

Round 1:

1. **Fresh-schema isolation did not hold.** `SET search_path` is
   connection-scoped, but the test drove a pooled `*sql.DB`, and
   `database.Open` sets `SetMaxOpenConns` only for SQLite
   (`platform/database/database.go:44-46`). `migrate.Apply` and the assertions
   could land on different connections still pointing at `public`, where all
   migrations are already applied — so the `EXTRACT(EPOCH FROM NOW())::BIGINT`
   insert would never run while every assertion still passed. This is the same
   defect class the change supersedes, displaced one level down, and it would
   have failed non-deterministically.
2. **`gofmt` failure** in the staff integration test's import block. CI runs
   `test -z "$(gofmt -l server)"`; `verify` does not run `gofmt` at all, so
   `verify: ok` did not cover it.
3. **Staff lock test proved an outcome, not serialization** — a
   `time.Sleep(50ms)` timing assumption with no assertion that the competing
   operation ever blocked.

Round 2:

4. **Blocking classification was logged, not asserted** in both the staff and
   media tests: any non-nil error was accepted as proof of lock contention, so
   a connection fault or an unrelated early error would have been recorded as
   successful blocking evidence.

All four were fixed by the implementer. The fresh-schema remediation exceeded
what was requested: it combines a DSN-level `search_path`, `SetMaxOpenConns(1)`,
a `SHOW search_path` assertion, **and** a pre-condition that
`schema_migrations` must not exist in the fresh namespace before `Apply`. That
last assertion is what converts silent fallback into a loud failure.

## Repository-state checks

```text
gofmt -l server/                      empty
go run ./server/tools/verify          verify: ok
go run ./server/tools/postgres-live-gate   exit 1 (correctly rejects the skipped set)
go test ./server/tools/postgres-live-gate  ok
git diff -- '*_test.go' | grep '^-func Test'   no matches (no test removed)
diff                                  10 files changed, 491 insertions(+), 77 deletions(-)
.ai/scope.json vs control applies_to  identical
```

## What this pass could NOT establish

No local PostgreSQL was available: no Docker, no `psql`, no
`TEST_DATABASE_URL`. No git remote and no `gh` CLI, so no CI run exists for any
commit in this repository.

Therefore the following remain `pending` and **must not** be advanced on the
strength of this document:

- **AC-001** fresh migration timestamp — the isolation mechanism is sound by
  inspection and defended by a pre-condition assertion, but the
  `EXTRACT(EPOCH FROM NOW())::BIGINT` branch has still never executed.
- **AC-002 / AC-003** staff and media lock semantics — the tests now have
  separate connection pools, a deterministic hook barrier, bounded deadlines,
  hard blocking assertions, collected outcomes, and final invariants. None of
  that has run against a live server.
- **AC-005** observed CI run — impossible without a remote.
- **AC-007** red-before-green for the migration and lock semantics —
  the gate and verifier halves were mutation-verified above; the PostgreSQL
  halves require a live server to observe red.

`REQ-001` through `REQ-004` remain `pending` for the same reason.

## What closing this change requires

One of:

- a local PostgreSQL (container or service) exporting `TEST_DATABASE_URL`,
  which unblocks AC-001, AC-002, AC-003, and the live half of AC-007; or
- a git remote so CI runs, which additionally supplies AC-005's run identity.

AC-005 names a run identity specifically, so a local database alone cannot
close the change.

The reviewer who closes these criteria should replay the mutation table above
rather than relying on it, and must not be the person who wrote the
remediations being replayed.
