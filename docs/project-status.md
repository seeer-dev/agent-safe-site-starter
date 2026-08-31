# Project status and v1 boundary

Last reviewed against `main@c8249301020c6a0eceeda33b875e5d9101b91ebe` (2026-08-31).

This document is the canonical high-level status for the starter. It distinguishes source-level completion from deployment acceptance and from optional commerce operations so the project does not grow into a full commerce framework by accident.

## Current position

| Area | Status | Meaning |
| --- | --- | --- |
| Starter architecture | Complete for v1 | Beginner single-site boundaries, explicit composition, fail-closed architecture checks, controlled changes, and CI gates are in place. |
| AI/spec governance | Complete for v1 | `architecture.yaml`, `AGENTS.md`, `speccheck`, `archcheck`, evidence rules, and PR verification are operational. |
| Commerce sample | Core flow complete | Product → cart → quote → checkout → durable order → payment → order lookup is implemented. |
| ECPay AIO credit | Source-level complete | Server-owned signing, durable ReturnURL reconciliation, replay protection, and browser-return non-authority are implemented and CI verified. |
| Deployment readiness | Partial | Provider wiring exists, but a real public deployment and smoke acceptance are not yet recorded. |
| Full commerce operations | Intentionally incomplete | Refunds, invoices, logistics, reconciliation jobs, and production operations remain optional follow-up work. |

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
- Frontend/OpenAPI contract gate.
- Repository tests, live PostgreSQL integration tests, concurrency stress tests, and `go vet` in CI.
- Commerce cohesion refactor completed across models, service behavior, persistence, order flow, and tests.

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

PR #8 and the post-merge `main` CI both passed the full repository verification chain.

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

4. **One ECPay stage transaction before production**
   - Required as go-live/deployment acceptance, not as a source-code completion gate.
   - Verify hosted checkout, external ReturnURL reachability, `1|OK`, durable `paid`, and browser re-query on the deployed environment.

See [`commerce-acceptance.md`](commerce-acceptance.md) for the exact boundary.

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
    F --> H[Optional post-v1 commerce capabilities]
    H --> I[Refund]
    H --> J[Invoice]
    H --> K[Logistics]
    H --> L[Reconciliation / operations]
```

The intended decision rule is simple: **publish once the starter is deployable and the reference commerce flow is acceptance-tested; do not wait for every commerce operation to exist.**

## Maintenance rule

Update this document when a change materially alters one of these states. Do not mark a capability complete because a helper, route, component, or spec exists; completion requires the real entry point and the relevant repository verification or deployment receipt.
