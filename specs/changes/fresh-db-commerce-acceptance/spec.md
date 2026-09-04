# Fresh DB Commerce Acceptance

Change ID: fresh-db-commerce-acceptance
Revision: 1
Status: Applying
Decision authority: Repository owner
Approval basis: Repository owner instructed continuing the v1 release-readiness sequence after ECPay official conformance. This slice closes the local fresh-database commerce gap without expanding into deployment infrastructure or additional payment products.
Repository baseline: fc37f411d7906380163a3c37b4b01379148ae0df
Supersedes: none

## Outcome

A clean local database for the minimal-cart sample must contain enough deterministic commerce configuration to exercise the reference purchase flow through a durable guest order, while public payment availability must remain truthful about whether ECPay is actually configured at runtime.

## In scope

- Seed at least one enabled shipping method for the minimal-cart local sample when shipping configuration is empty.
- Seed an ECPay payment row in local sample data with readiness derived from runtime ECPay configuration: configured stage/production runtime may be ready/enabled; absent runtime configuration remains pending/disabled.
- Preserve the existing non-ECPay sample payment methods.
- Make public payment discovery and checkout validation fail closed when an ECPay database row is marked ready but runtime ECPay configuration is absent or its environment does not match the admin payment environment (`sandbox` ↔ `stage`, `production` ↔ `production`).
- Add deterministic fresh-SQLite acceptance coverage for seed → discovery → quote → durable guest order, including truthful ECPay availability.
- Update canonical project/commerce status to distinguish local fresh-DB acceptance from still-pending public deployment/stage acceptance.

## Out of scope

- Public ECPay stage transaction, Cloudflare Tunnel, Railway/Pages deployment, rate-limit topology, refund/invoice/logistics/reconciliation, or new payment providers.
- Making local test credentials a production default.
- Generic provider registries or runtime DI/plugin abstractions.
- Changing the checkout API envelope or payment-provider architecture.

### REQ-001: Fresh minimal-cart data is checkout-capable

The local development seed MUST create deterministic commerce configuration sufficient for the sample storefront to discover a sellable product, an enabled shipping method, and at least one usable payment method on a fresh database.

#### AC-001: Fresh database reaches durable guest order
- GIVEN a fresh SQLite database with migrations applied
- WHEN the minimal-cart development seed runs and the sample flow selects a seeded in-stock SKU, seeded shipping method, and usable payment method
- THEN server discovery returns those choices, server quote succeeds with authoritative totals, and guest CreateOrder persists an unpaid durable order with a one-time access token

### REQ-002: ECPay availability reflects runtime truth

An admin/database ECPay row MUST NOT make ECPay publicly available unless the service has a valid runtime ECPay configuration whose environment matches the row.

#### AC-002: Missing runtime ECPay is not publicly selectable
- GIVEN an enabled/ready ECPay database row but no runtime ECPay configuration
- WHEN public payment methods are listed or checkout validates ECPay
- THEN ECPay is excluded from public discovery and rejected as an invalid checkout payment method

#### AC-003: Runtime/admin environment mismatch fails closed
- GIVEN an enabled/ready ECPay row marked `sandbox` with production runtime, or `production` with stage runtime
- WHEN public discovery or checkout validation runs
- THEN ECPay is unavailable until the environments match

#### AC-004: Matching runtime ECPay is selectable
- GIVEN an enabled/ready ECPay row and matching runtime configuration (`sandbox` with stage or `production` with production)
- WHEN public discovery and checkout validation run
- THEN ECPay is available and may be used by the existing hosted-payment preparation flow

### REQ-003: Local seed does not pretend unconfigured ECPay is ready

The minimal-cart development seed MUST represent ECPay as pending/disabled when ECPay runtime values are absent, and as ready/enabled only when the complete runtime configuration is present.

#### AC-005: Seeded ECPay readiness follows config
- GIVEN a fresh database
- WHEN the development seed runs without ECPay configuration
- THEN the ECPay admin row exists as `pending_setup`, disabled, and `sandbox`
- AND WHEN the seed runs on a fresh database with complete stage ECPay configuration
- THEN the ECPay row is `ready`, enabled, and `sandbox`

### REQ-004: Acceptance boundary remains honest

Local fresh-DB acceptance MUST NOT be reported as public provider/deployment acceptance.

#### AC-006: Canonical status keeps stage transaction pending
- GIVEN the local fresh-DB gate passes
- WHEN README/project/commerce status is read
- THEN local sample acceptance is marked complete while public HTTPS deployment and one real ECPay stage transaction remain pending release-readiness work
