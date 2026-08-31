# Restore HTTP Contract Truth

Change ID: restore-http-contract-truth
Revision: 2
Status: Applying
Decision authority: Repository owner
Approval basis: Repository owner authorized converting the 2026-08-31 reviewed contract-drift conclusion into the sole next review-ready proposal, explicitly excluding response-envelope redesign, generated TypeScript adoption, and runtime behavior changes; on 2026-08-31 the owner then instructed applying the queued changes sequentially and merging them to main.
Repository baseline: 9755c31048e3594d457748bf3d5dfb9f864a482f
Supersedes: none

## Outcome

Make `contracts/openapi.yaml` a trustworthy description of the already-implemented Go HTTP surface before any generated admin types are introduced. This slice restores contract truth and adds a drift detector; it does not redesign runtime responses merely for uniformity.

## In scope

- Audit the registered routes in `server/internal/bootstrap/app.go` against OpenAPI.
- Add missing public/admin/ECPay operations to `contracts/openapi.yaml`.
- Align declared request/response schemas and success status codes with current Go handler behavior.
- Correct admin/public DTO distinctions where the runtime already exposes different shapes.
- Add a dependency-free runtime/OpenAPI parity checker under `contracts/` and include it in `make verify-contracts`.

## Out of scope

- Changing Go route behavior, status codes, or response bodies solely to normalize the API.
- Introducing a universal `{ data, meta }` or `{ items, meta }` envelope.
- Generating or adopting admin TypeScript types in this change.
- Pagination, `ResourceListPage.vue` decomposition, rate limiting, ECPay protocol changes, or module splitting.

### REQ-001: Registered HTTP operations are represented

`contracts/openapi.yaml` MUST represent every non-internal HTTP route registered by the current bootstrap surface, including the ECPay payment routes, with the correct HTTP method and path.

#### AC-001: Route and method parity
- GIVEN the route registrations in `server/internal/bootstrap/app.go`
- WHEN the runtime/OpenAPI parity check enumerates the registered public and admin operations
- THEN every operation has a matching OpenAPI path/method or an explicit narrow exemption documented by the checker, and removing one covered operation makes the check fail

### REQ-002: Observable success contracts match runtime behavior

For covered operations, OpenAPI MUST describe the current runtime success status and response schema rather than a desired future shape.

#### AC-002: Admin product drift is corrected
- GIVEN the current admin product handlers return `201` on create, `200` plus an ID body on delete, and admin-specific product DTOs
- WHEN the OpenAPI product operations are inspected
- THEN those existing runtime status/schema contracts are represented accurately without changing the handlers to fit the old spec

#### AC-003: ECPay HTTP boundary is represented
- GIVEN the current ECPay prepare, server ReturnURL, and browser-return handlers are registered
- WHEN the OpenAPI contract is inspected
- THEN all three paths and methods are present with their current request/response boundary, including the server callback's successful plain-text acknowledgement and the browser redirect behavior

### REQ-003: Contract drift becomes mechanically visible

The repository MUST provide a dependency-free contract check that fails on registered-route omission and representative success-status/schema drift and MUST run through the existing `make verify-contracts` entry point.

#### AC-004: Mutation-sensitive parity gate
- GIVEN the restored contract and parity checker
- WHEN an independent reviewer temporarily removes one covered operation or changes a guarded success status/schema expectation
- THEN `make verify-contracts` fails with a specific contract diagnostic; after restoration it passes with no mutation residue

## Amendments

| Revision | REQ/AC | Old meaning | New meaning | Reason | Approval basis | Invalidated evidence |
| --- | --- | --- | --- | --- | --- | --- |
| 2 | all | Revision 1 was Ready against `10f805e18e394e7065854dfbceb327114e0d4564`. | Scope and behavior are unchanged; the baseline is refreshed to `9755c31048e3594d457748bf3d5dfb9f864a482f` and status moves to Applying. | Governance-review and lifecycle-closure PRs landed before implementation; using the old baseline would make their unrelated changes part of the selected diff. | Owner instructed sequential apply and merge on 2026-08-31. | None; revision 1 had no passed evidence. |
