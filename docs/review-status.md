# Architecture review status overlay

Last reviewed against `main@e7af2a704405ba172108d68b1b834e9602fb1d85` on 2026-08-31.

This file is the current interpretation layer for the historical architecture assessments in `docs/backend-optimization.md` and `docs/admin-architecture-review.md`. Those reports remain useful audit material, but their original priority tables are **not** the current backlog. When a historical statement conflicts with an Accepted controlled change or `docs/project-status.md`, the newer controlled state wins.

## Review conclusions after repository verification

### 1. Governance backlog was lifecycle/context debt, not an `apply` ambiguity

Plain `apply` requires one current review-ready proposal. A `Verifying` change is not itself a review-ready proposal, so the former Verifying set did not automatically make `apply` select the wrong change. The real cost was lifecycle/context debt: stale baselines and old evidence remained visible to later agents.

The review sequence closed four evidence-complete changes that had no remaining pending/blocked REQ or AC:

| Change | Previous state | Current state | Why |
| --- | --- | --- | --- |
| `commerce-boolean-adapter-and-live-evidence` | Verifying | Accepted | All declared evidence and required receipts were already passed. |
| `ephemeral-postgres-local-gate` | Verifying | Accepted | All declared evidence and required receipts were already passed. |
| `harden-implementation-handoffs` | Verifying | Accepted | All declared evidence and required receipts were already passed. |
| `scoped-worktree-validation` | Verifying | Accepted | All declared evidence and independent-review receipts were already passed; it was closed separately to avoid overlapping ownership of the prior README review diff. |

The following must **not** be mass-promoted merely for hygiene:

| Change | Keep open because |
| --- | --- |
| `postgres-lock-semantics-and-evidence` | Some acceptance evidence still requires independent review / observed CI identity. |
| `verify-contract-checks` | Its older lifecycle still has pending independent mutation/version-floor acceptance evidence. |
| `supabase-jwks-verifier` | Live Supabase compatibility/rollback evidence is environment-blocked. |
| `minimal-cart-integration` | The umbrella change still contains deployment/provider/policy acceptance gaps even though many later slices are complete. |

`commerce-module-file-split` and `public-endpoint-rate-limit` are old Drafts from earlier baselines. Do not execute them directly as current plans. Re-propose from current repository reality if/when their outcome becomes a priority.

### 2. Historical review documents need a resolution overlay, not deletion

Several findings in the old backend review have already been resolved by later Accepted changes:

| Historical finding | Current state |
| --- | --- |
| Production admin API base/topology client configuration | Resolved by `admin-configurable-api-base` (Accepted). |
| PostgreSQL connection-pool configuration | Resolved by `postgres-connection-pool-configuration` (Accepted). |
| Request/status/client-IP observability | Resolved by `structured-request-observability` (Accepted). |
| Real PostgreSQL execution in repository CI | The current CI runs PostgreSQL migrations, live integration tests, and concurrency stress; do not treat the old “SQLite-only CI” finding as current. |
| Runtime/OpenAPI route/status/schema drift | Resolved for the current registered surface by `restore-http-contract-truth` revision 2 (Accepted). |

Keep the historical reports for reasoning and provenance, but use this overlay plus controlled changes to determine whether a finding is current.

`INTEGRATION_PLAN.md` is still referenced by the large `minimal-cart-integration` history and some source comments. Archive it only after those references are moved to current normative documents and the umbrella lifecycle is intentionally closed; do not delete it just to reduce token count.

### 3. HTTP/OpenAPI contract truth is restored for the current runtime surface

The prior review found real contract drift rather than a cosmetic response-envelope problem. That gap is now closed by Accepted change `restore-http-contract-truth` revision 2.

Delivered state:

- `contracts/openapi.yaml` is version 0.3.0 and represents all **56 operations** currently registered by `server/internal/bootstrap/app.go`.
- `contracts/check-runtime-openapi.mjs` performs symmetric runtime ↔ OpenAPI path/method parity.
- Representative observable status/schema guards cover the previously high-risk admin product, media presign, staff, and ECPay boundaries.
- `make verify-contracts` now runs the runtime/OpenAPI parity check before the existing admin-resource and public-theme contract checks.
- No Go runtime route/status/body was changed merely to fit the old specification.

Acceptance was mutation-sensitive:

- a temporary ECPay browser-return route omission made CI fail with explicit missing-runtime / extra-OpenAPI diagnostics;
- a temporary admin product create `201` → `200` drift made CI fail because the guarded `201` disappeared;
- restoration returned 56/56 parity and both existing contract checks to green with no mutation residue.

The follow-up contract/locality sequence is now optional/post-foundation work rather than a deploy-readiness blocker:

```mermaid
flowchart LR
    A[OpenAPI truth restored] --> B[Generated admin TypeScript types]
    B --> C[Explicit typed response access]
    C --> D[ResourceListPage locality split]
    D --> E{Concrete pagination/meta need?}
    E -->|No| F[Keep resource-specific envelopes]
    E -->|Yes| G[Normalize envelope deliberately]
```

A universal response envelope is still optional. The unacceptable behavior remains a consumer guessing with `Object.values(data).find(Array.isArray)` rather than consuming an explicit typed contract.

### 4. Admin locality should improve before another broad commerce split

`admin/src/pages/ResourceListPage.vue` remains a large cross-resource component with many `Record<string, any>` boundaries. Now that OpenAPI truth is restored, generated types and explicit response access are the logical prerequisites before splitting the component into cohesive data/form/action/restock seams.

Commerce still has large internal files, but it has already gone through several cohesion slices. If further splitting is needed, re-inventory the **current** files and preserve the existing `commerce` top-level module. Do not apply the stale `commerce-module-file-split` Draft verbatim and do not create new top-level modules just to reduce file size.

### 5. SQL ownership deserves a mechanical guard, after ownership metadata is current

`archcheck` currently enforces Go import boundaries, not SQL table-write ownership. The review already refreshed `architecture.yaml` so commerce owns `ecpay_payment_attempts` and the known commerce-to-media write points at `store_catalog.go:markMediaAssociationsTx` rather than stale line numbers.

A later low-cost guard should scan module SQL writes (`INSERT INTO`, `UPDATE`, `DELETE FROM`) against `architecture.yaml` ownership plus explicit exceptions. Start with writes only; cross-module reads do not need the same correctness barrier. Do not add a heuristic scanner against stale ownership metadata.

### 6. Deploy-readiness interpretation

- **ECPay official conformance:** now the next v1 release-readiness check. Compare the merged AIO implementation against the official `ECPay/ECPay-API-Skill` references before treating provider protocol review as closed.
- **Fresh-DB commerce acceptance:** run the real sample path from deterministic data through order/admin/return-restock behavior. Provider-dependent portions may remain deploy acceptance until a public environment exists.
- **Rate limiting:** a real pre-production decision, but the existing Draft depends on replica topology, trusted client-IP source, enforcement owner, and operating cost. Do not silently install an in-memory limiter and call the problem solved.
- **JWKS:** desirable for auth hot-path latency and external dependency reduction, but the current remote verifier can remain a correctness-preserving v1 fallback. Live JWKS acceptance should not block v1 unless the product establishes an SLA that requires it.
- **ECPay stage transaction:** deployment/go-live acceptance, not a source-code completion gate. Official protocol conformance review should happen before the stage transaction.

## Revised priority order

1. **Deploy readiness** — official ECPay audit, fresh-DB commerce acceptance, provider/env wiring, rate-limit topology decision, one stage transaction.
2. **Admin contract/locality** — generated OpenAPI types, explicit response access, then split `ResourceListPage.vue` along cohesive seams.
3. **Data-ownership enforcement** — add SQL-write ownership scanning after the refreshed ownership map proves stable.
4. **Residual cohesion/performance** — current-baseline commerce splitting, pagination/N+1/index work only when concrete scale or maintenance pressure justifies it.

The architectural decision remains unchanged: beginner single-site, static public output, one Go backend, explicit module ownership, no runtime DI/plugin registry, and no speculative platformization.
