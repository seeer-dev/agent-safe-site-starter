# Evidence

## Delivery Status

Revision 1 was implemented and verified against baseline `e30f583c00c9ceecbbdf856016c2a01fab9c3cd2` on 2026-08-14.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | server/AGENTS.md requires before/after test inventory, prohibits unauthorized removal or weakening, and requires a bounded red-before-green failure trigger for every new regression, security, integration, branch, or governance check. |
| REQ-002 | passed | server/AGENTS.md distinguishes sequential interleaving from simultaneous execution, requires exact named-test pass events, defines observable PostgreSQL lock semantics, and rejects substring-only wiring proof. |
| AC-001 | passed | The Preserve tests and gates section requires inventory comparison and explicit approved justification for every removed, skipped, quarantined, renamed, or weakened check. |
| AC-002 | passed | The Require falsifiable evidence section requires the targeted assertion to fail under a bounded temporary trigger, restoration of the trigger, and confirmation that no temporary mutation remains. |
| AC-003 | passed | The Prove required test execution section requires exact go test -json pass events for every expected test and treats missing dependencies, missing tests, and skipped tests as failed evidence in required environments. |
| AC-004 | passed | The concurrency and wiring sections require deterministic overlap, separate controlled operations or transactions, bounded deadlines, final invariants, observable FOR UPDATE or SKIP LOCKED effects, and exact command or structured-configuration comparison. |

## Commands Replayed

```text
go run ./server/tools/speccheck   ok
go run ./server/tools/scopecheck  ok
go run ./server/tools/verify      ok
```

The global Grok and Agy rule files were also scanned for backend-specific terms; Go and PostgreSQL mechanics remain confined to `server/AGENTS.md`.
