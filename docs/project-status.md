# Project status and v1 boundary

Last reviewed against `main@10f805e18e394e7065854dfbceb327114e0d4564` (2026-08-31).

This document is the canonical high-level status for the starter. It distinguishes runtime/source completion, contract truth, deployment acceptance, and optional commerce operations so the project does not grow into a full commerce framework by accident and so a green focused contract check is not mistaken for complete runtime/OpenAPI parity.

## Current position

| Area | Status | Meaning |
| --- | --- | --- |
| Starter architecture | Complete for v1 | Beginner single-site boundaries, explicit composition, fail-closed architecture checks, controlled changes, and CI gates are in place. |
| AI/spec governance | Complete for v1 foundation; lifecycle cleanup ongoing | `architecture.yaml`, `AGENTS.md`, `speccheck`, `archcheck`, evidence rules, and PR verification are operational. Evidence-complete historical changes are being closed; genuinely blocked/pending evidence stays open. |
| Commerce sample | Runtime core flow complete | Product → cart → quote → checkout → durable order → payment → order lookup is implemented. This does not imply every HTTP operation is correctly represented in OpenAPI. |
| ECPay AIO credit | Source-level complete | Server-owned signing, durable ReturnURL reconciliation, replay protection, and browser-return non-authority are implemented and CI verified. |
| HTTP/OpenAPI contract truth | Needs restoration before v1 deploy-ready | Known runtime/OpenAPI route, status, and schema drift remains. Restore truth before generated client types or claiming the API contract is authoritative. |
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
- Focused frontend/browser-authority/OpenAPI contract checks that guard named integration invariants; these are useful gates but are **not** a complete runtime/OpenAPI parity proof.
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

PR #8 and the post-merge `main` CI both passed the full repository verification chain for the implemented runtime behavior.

## Current contract-truth gap

`contracts/openapi.yaml` must not yet be treated as a complete authoritative projection of the Go runtime. Known verified drift includes admin product status/schema mismatches, previously identified missing admin operations, and the later ECPay payment routes.

The repair order is intentionally narrow:

```mermaid
flowchart LR
    A[Go runtime truth] --> B[OpenAPI route/method/status/schema restoration]
    B --> C[Mechanical parity gate]
    C --> D[Generated admin TypeScript types]
    D --> E[Optional envelope redesign only if product needs it]
```

Do not redesign runtime responses merely to make the spec look uniform. Restore the existing observable contract first. A resource-specific envelope such as `{ products: [...] }` is acceptable when the consumer has an explicit typed contract; guessing with `Object.values(...).find(Array.isArray)` is not.

## What remains before calling v1 deploy-ready

These are release-readiness tasks, not missing architecture foundations.

1. **HTTP/OpenAPI contract truth restoration**
   - Audit registered Go routes against `contracts/openapi.yaml`, including ECPay.
   - Align methods, status codes, request/response schemas, and admin/public DTO boundaries to current runtime behavior without changing runtime semantics merely for uniformity.
   - Strengthen the contract gate so missing operations and representative status/schema drift turn CI red.
   - Only after this is green should admin OpenAPI-generated types become the next contract step.

2. **Official ECPay conformance audit**
   - Review the current AIO implementation against the official `ECPay/ECPay-API-Skill` references.
   - Confirm current CheckMacValue encoding, required fields, ReturnURL behavior, response status/body, and go-live constraints.
   - Treat this as protocol conformance review, not as a new payment architecture project.

3. **Sample commerce acceptance review**
   - Fresh database and deterministic sample data.
   - Browse product → add cart → reload/rehydrate → quote → guest/member order → payment handoff → order lookup → admin lookup → return/restock.
   - Runtime paths that require external providers may be recorded as deploy acceptance when no public environment exists yet.

4. **Deploy readiness**
   - Railway Go API + PostgreSQL.
   - Cloudflare Pages site build/publish.
   - Supabase Auth production configuration.
   - R2 and Resend production configuration.
   - Public HTTPS ECPay ReturnURL/OrderResultURL.
   - Migration/pre-deploy behavior and smoke tests.
   - Decide public rate-limit enforcement from the real deployment topology and trusted client-IP source; do not assume an in-memory single-process limiter is globally correct.

5. **One ECPay stage transaction before production**
   - Required as go-live/deployment acceptance, not as a source-code completion gate.
   - Verify hosted checkout, external ReturnURL reachability, `1|OK`, durable `paid`, and browser re-query on the deployed environment.

See [`commerce-acceptance.md`](commerce-acceptance.md) for the commerce/deployment boundary and [`review-status.md`](review-status.md) for review findings that are historical versus current.

## Explicitly open but not automatic v1 blockers

These controlled changes remain honest about missing evidence rather than being mass-accepted for cleanliness:

- `postgres-lock-semantics-and-evidence` — remaining independent/CI evidence should be reconciled when that verification is replayed.
- `verify-contract-checks` — some independent mutation/version-floor evidence remains pending; the new contract-truth restoration work should not pretend the existing checker is complete.
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
    A[Current main] --> B[Restore HTTP/OpenAPI truth]
    B --> C[Official ECPay conformance audit]
    C --> D[Sample commerce acceptance]
    D --> E[Deploy readiness]
    E --> F{v1 release blockers?}
    F -->|No| G[Publish starter v1]
    F -->|Yes| H[Fix only blocking gaps]
    H --> E
    G --> I[Optional post-v1 capabilities]
    I --> J[Generated/admin locality refinements]
    I --> K[SQL write ownership gate]
    I --> L[Refund / invoice / logistics / reconciliation]
```

The intended decision rule is simple: **publish once the repository contract is trustworthy, the starter is deployable, and the reference commerce flow is acceptance-tested; do not wait for every commerce operation or optimization to exist.**

## Maintenance rule

Update this document when a change materially alters one of these states. Do not mark a capability complete because a helper, route, component, spec, or focused checker exists; completion requires the real entry point plus the relevant contract/runtime verification or deployment receipt.
