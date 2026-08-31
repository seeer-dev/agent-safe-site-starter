# Project status and v1 boundary

Last reviewed against `main@e7af2a704405ba172108d68b1b834e9602fb1d85` (2026-08-31).

This document is the canonical high-level status for the starter. It distinguishes runtime/source completion, contract truth, deployment acceptance, and optional commerce operations so the project does not grow into a full commerce framework by accident.

## Current position

| Area | Status | Meaning |
| --- | --- | --- |
| Starter architecture | Complete for v1 | Beginner single-site boundaries, explicit composition, fail-closed architecture checks, controlled changes, and CI gates are in place. |
| AI/spec governance | Complete for v1 foundation | `architecture.yaml`, `AGENTS.md`, `speccheck`, `archcheck`, evidence rules, PR verification, and the evidence-complete lifecycle cleanup are operational. Genuinely blocked/pending evidence remains open rather than being falsely promoted. |
| Commerce sample | Runtime core flow complete | Product → cart → quote → checkout → durable order → payment → order lookup is implemented. |
| ECPay AIO credit | Source-level complete | Server-owned signing, durable ReturnURL reconciliation, replay protection, and browser-return non-authority are implemented and CI verified. |
| HTTP/OpenAPI contract truth | Complete for current runtime surface | OpenAPI 0.3.0 represents all 56 registered Go operations. Symmetric route/method parity plus guarded observable status/schema checks run in `make verify-contracts`; omission/status mutations were independently proven red. |
| Deployment readiness | Partial | Provider wiring exists, but a real public deployment and smoke acceptance are not yet recorded. |
| Full commerce operations | Intentionally incomplete | Refunds, invoices, logistics, reconciliation jobs, and production operations remain optional follow-up work. |

See [`review-status.md`](review-status.md) for the current interpretation of historical backend/admin reviews and lifecycle cleanup.

## What is complete

### Starter and governance

- Beginner-facing single-site architecture with a small default production shape.
- Static public site with Go rendering; no frontend SSR framework requirement.
- Separate Vue admin SPA and scoped Vue islands for the commerce theme.
- Explicit module/bootstrap ownership; no runtime DI container or plugin registry.
- SQLite local development and PostgreSQL production migration parity.
- Supabase Auth integration, R2 media boundary, and Resend email boundary.
- `architecture.yaml` import policy with fail-closed `archcheck`.
- Controlled specifications under `specs/changes/<change-id>/` with `speccheck` enforcement.
- Runtime/OpenAPI route and method parity for all 56 registered Go operations through `contracts/check-runtime-openapi.mjs`.
- Guarded observable contract checks for high-risk status/schema boundaries, with mutation evidence proving route omission and admin-product success-status drift fail the gate.
- Existing admin resource and public-theme OpenAPI contract checks remain part of the same `make verify-contracts` entry point.
- Repository tests, live PostgreSQL integration tests, concurrency stress tests, and `go vet` in CI.
- Commerce cohesion refactor completed across models, service behavior, persistence, order flow, and tests.
- Evidence-complete lifecycle debt closed for `commerce-boolean-adapter-and-live-evidence`, `ephemeral-postgres-local-gate`, `harden-implementation-handoffs`, and `scoped-worktree-validation`.

### Commerce sample

- Published product/catalog flow.
- Cart add/remove/quantity/variant behavior.
- Cart persistence stores identity/selection/quantity rather than browser-authoritative prices.
- Cart rehydration reloads authoritative product data and fails closed on unavailable catalog data.
- Server-authoritative quote for subtotal, discount, shipping, and total.
- Admin-managed shipping methods and payment methods.
- Promotion validation.
- Guest checkout and authenticated member checkout.
- Stable create-order idempotency.
- Transactional stock decrement and durable order creation.
- Guest opaque order access credential and authenticated member order history.
- Admin order/status management.
- Return request workflow and per-item restock ledger/workflow.

### ECPay AIO credit

The sample now reaches a real hosted-payment boundary rather than stopping at a mock checkout.

```mermaid
flowchart TD
    A[Product] --> B[Cart]
    B --> C[Server quote]
    C --> D[Create durable order]
    D --> E[Order: unpaid]
    E --> F[Server prepares ECPay AIO form]
    F --> G[Server CheckMacValue]
    G --> H[Browser POST to ECPay]
    H --> I[ECPay payment page]
    I --> J[Server-to-server ReturnURL]
    I --> K[Browser return]
    J --> L[Verify CheckMacValue / MerchantID / trade identity / amount]
    L --> M[Durable callback claim + replay guard]
    M --> N[payment_status = paid + order event]
    N --> O[Return 1|OK]
    K --> P[Storefront restores same-tab order credential]
    P --> Q[Re-query server order]
    Q --> R{payment_status == paid?}
    R -->|yes| S[Clear cart and confirm]
    R -->|not yet| T[Retry briefly / show confirming state]
```

Implemented safety properties:

- HashKey/HashIV remain server-only.
- AIO stage/production endpoint selection is finite and validated.
- Known public ECPay test credentials are rejected in production mode.
- Browser navigation cannot mark an order paid.
- `ReturnURL` verifies `CheckMacValue`, `MerchantID`, merchant trade identity, and durable amount.
- Identical callbacks are one-effect; conflicting claimed callbacks fail closed.
- Callback claim, paid transition, order version update, and payment-status event are committed atomically.
- SQLite and PostgreSQL both persist the ECPay payment attempt contract.
- ECPay-specific signing/tamper/amount/browser-authority tests are included.

PR #8 and its post-merge `main` CI passed the full repository verification chain for the implemented runtime behavior.

## HTTP contract truth restoration

`restore-http-contract-truth` revision 2 is Accepted and merged. `contracts/openapi.yaml` now describes the current registered Go HTTP surface rather than a desired future shape.

The completed contract chain is:

```mermaid
flowchart LR
    A[Go runtime registrations] --> B[OpenAPI 0.3.0]
    B --> C[56/56 path-method parity]
    C --> D[Guarded status-schema checks]
    D --> E[Admin resource contract checks]
    E --> F[Public theme contract checks]
```

Acceptance included two CI mutations:

- removing the ECPay browser-return operation produced explicit missing-runtime / extra-OpenAPI diagnostics;
- changing admin product create `201` to `200` produced an explicit missing-`201` diagnostic;
- restoring both returned the contract gate to green with no mutation residue.

This restoration deliberately did **not** normalize all response envelopes or adopt generated TypeScript types. Resource-specific envelopes remain acceptable when consumers use explicit contracts. Generated admin types and `ResourceListPage.vue` decomposition are post-contract locality improvements, not v1 deploy blockers.

## What remains before calling v1 deploy-ready

These are release-readiness tasks, not missing architecture foundations.

1. **Official ECPay conformance audit**
   - Review the current AIO implementation against the official `ECPay/ECPay-API-Skill` references.
   - Confirm current CheckMacValue encoding, required fields, ReturnURL behavior, response status/body, and go-live constraints.
   - Treat this as protocol conformance review, not as a new payment architecture project.

2. **Sample commerce acceptance review**
   - Fresh database and deterministic sample data.
   - Browse product → add cart → reload/rehydrate → quote → guest/member order → payment handoff → order lookup → admin lookup → return/restock.
   - Runtime paths that require external providers may be recorded as deploy acceptance when no public environment exists yet.

3. **Deploy readiness**
   - Railway Go API + PostgreSQL.
   - Cloudflare Pages site build/publish.
   - Supabase Auth production configuration.
   - R2 and Resend production configuration.
   - Public HTTPS ECPay ReturnURL/OrderResultURL.
   - Migration/pre-deploy behavior and smoke tests.
   - Decide public rate-limit enforcement from the real deployment topology and trusted client-IP source; do not assume an in-memory single-process limiter is globally correct.

4. **One ECPay stage transaction before production**
   - Required as go-live/deployment acceptance, not as a source-code completion gate.
   - Verify hosted checkout, external ReturnURL reachability, `1|OK`, durable `paid`, and browser re-query on the deployed environment.

See [`commerce-acceptance.md`](commerce-acceptance.md) for the commerce/deployment boundary and [`review-status.md`](review-status.md) for review findings that are historical versus current.

## Explicitly open but not automatic v1 blockers

These controlled changes remain honest about missing evidence rather than being mass-accepted for cleanliness:

- `postgres-lock-semantics-and-evidence` — remaining independent/CI evidence should be reconciled when that verification is replayed.
- `verify-contract-checks` — this older lifecycle still has pending evidence and should be reconciled independently; it does not negate the newer Accepted `restore-http-contract-truth` parity gate.
- `supabase-jwks-verifier` — live Supabase compatibility/rollback evidence is environment-blocked. The remote verifier remains a correctness-preserving fallback unless an auth-path SLA makes JWKS mandatory.
- `minimal-cart-integration` — historical umbrella with remaining deployment/provider/policy acceptance; later accepted slices do not justify falsely marking unresolved umbrella evidence passed.

Old Drafts such as `commerce-module-file-split` and `public-endpoint-rate-limit` must be re-proposed from the current baseline before implementation; their old plans are not current authority.

## Intentionally not required for starter v1

The following are useful commerce extensions, but they are not blockers for publishing a small beginner-facing starter with a working reference purchase flow:

- ECPay refund/cancel integration.
- Refund authorization/AAL2 and refund idempotency.
- Payment reconciliation/query jobs and operational dashboards.
- ECPay electronic invoice integration.
- ECPay logistics or another shipping provider integration.
- Shipment/tracking webhook lifecycle.
- Failed-payment recovery automation and abandoned-payment policy.
- Admin refund UX.

Adding these should be driven by a concrete product outcome. They should not be added merely to make the starter resemble a full commerce platform.

## v1 release boundary

```mermaid
flowchart LR
    A[Current main] --> B[Official ECPay conformance audit]
    B --> C[Sample commerce acceptance]
    C --> D[Deploy readiness]
    D --> E{v1 release blockers?}
    E -->|No| F[Publish starter v1]
    E -->|Yes| G[Fix only blocking gaps]
    G --> D
    F --> H[Optional post-v1 capabilities]
    H --> I[Generated admin types / locality]
    H --> J[SQL write ownership gate]
    H --> K[Refund / invoice / logistics / reconciliation]
```

The intended decision rule is simple: **publish once the repository contract is trustworthy, the starter is deployable, and the reference commerce flow is acceptance-tested; do not wait for every commerce operation or optimization to exist.**

## Maintenance rule

Update this document when a change materially alters one of these states. Do not mark a capability complete because a helper, route, component, spec, or focused checker exists; completion requires the real entry point plus the relevant contract/runtime verification or deployment receipt.
