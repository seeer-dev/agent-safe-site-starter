# Backend Verification Discipline Delivery Plan

Change ID: backend-verification-discipline
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `server/AGENTS.md`
- `specs/changes/backend-verification-discipline/**`

## Slice 1: Backend-Local Rules

- Add `server/AGENTS.md` with rules for test inventory, mutation-sensitive proof, required-test event verification, deterministic concurrency, database lock behavior, exact wiring checks, and independent evidence.
- Keep language-neutral evidence honesty and model-specific failure mitigations outside this repository-local file.
- Covers: REQ-001, REQ-002, AC-001, AC-002, AC-003, AC-004.

## Verification

- Inspect inheritance against root `AGENTS.md` and confirm no conflict with architecture boundaries.
- Confirm the file contains no authorization to weaken tests, skip required execution, or self-approve evidence.
- Run `go run ./server/tools/speccheck`, `go run ./server/tools/scopecheck`, and `go run ./server/tools/verify`.
