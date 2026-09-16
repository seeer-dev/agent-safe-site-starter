# Evidence ledger

Change ID: ephemeral-postgres-local-gate
Revision: 1
Status: Accepted

| REQ/AC | Status | Proof |
|---|---|---|
| REQ-001 | passed | Codex independently ran go run ./server/tools/local-postgres-gate on 2026-08-15. It started loopback PostgreSQL 16 and invoked the existing live gate; see receipts/local-live-run-rev1.md. |
| REQ-002 | passed | Successful live run left port 5433 and postgres processes clear. Unit failure-path coverage proves gate failure stops the child and cleans its runtime path; see receipts/lifecycle-walkthrough-rev1.md. |
| REQ-003 | passed | go.mod pins github.com/fergusstrange/embedded-postgres v1.34.0 directly; code pins V16 and documentation records the cache and local-only boundaries; see receipts/independent-review-rev1.md. |
| AC-001 | passed | The independent live run reached postgres-live-gate through the one-command launcher. Child environment isolation is asserted by TestBuildChildEnv and TestBuildGateCommandProperties; see receipts/local-live-run-rev1.md. |
| AC-002 | passed | postgres-live-gate reported all required live PostgreSQL tests passed, including migration, media locks, database constraints, and staff locks; see receipts/local-live-run-rev1.md and receipts/independent-review-rev1.md. |
| AC-003 | passed | Independent review confirmed pinned V16, fixed exec arguments, loopback binding, case-insensitive child-only DSN replacement, no DSN output, and a user-cache-only binary boundary; see receipts/independent-review-rev1.md. |
| AC-004 | passed | Successful run cleanup was observed directly; deterministic startup/gate-failure unit tests assert cleanup and child stop behavior without an external database; see receipts/lifecycle-walkthrough-rev1.md. |
