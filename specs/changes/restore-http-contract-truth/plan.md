# Restore HTTP Contract Truth Delivery Plan

Change ID: restore-http-contract-truth
Revision: 2
Status: Accepted

Repository baseline: `9755c31048e3594d457748bf3d5dfb9f864a482f`

## Scope lock

- `contracts/openapi.yaml`
- `contracts/check-runtime-openapi.mjs`
- `Makefile`
- `specs/changes/restore-http-contract-truth/**`

## Delivery

Covers REQ-001, AC-001, REQ-002, AC-002, AC-003, REQ-003, AC-004.

### Slice 1 — Build a runtime route inventory

- Read route registrations from `server/internal/bootstrap/app.go` as the runtime source for path/method presence.
- Define only narrow explicit exemptions if a route is intentionally outside the public contract.
- Record the current known gaps, including prior admin omissions and the three ECPay routes.

### Slice 2 — Restore OpenAPI behavior truth

- Add missing path/method operations to `contracts/openapi.yaml`.
- Audit existing operations for success status and request/response schema drift.
- Correct the known admin product create/delete/admin-DTO mismatches.
- Represent the current ECPay prepare/ReturnURL/browser-return HTTP boundary.
- Preserve current Go runtime behavior; do not normalize responses in this slice.

### Slice 3 — Add a dependency-free parity gate

- Add `contracts/check-runtime-openapi.mjs` using Node standard-library facilities only.
- Make registered-route omission fail with a path/method diagnostic.
- Guard representative observable contract facts such as admin product create/delete status/schema and ECPay operation presence.
- Wire the script into the existing `make verify-contracts` target without adding Node work to `go run ./server/tools/verify`.

### Slice 4 — Evidence

- Run existing Go/contract checks and the restored `make verify-contracts` entry point.
- Independently mutation-test at least one route omission and one guarded success contract so AC-004 has real independent-review evidence.
- Keep generated TypeScript, response-envelope redesign, pagination, admin component splitting, and runtime API rewrites out of this change.

## Revision note

Revision 2 refreshed the repository baseline after governance-review and lifecycle-closure changes landed. Scope, REQ/AC, and delivery semantics remained unchanged from revision 1; the change is Accepted after clean 56/56 route parity and mutation-sensitive contract evidence.
