# Fresh DB Commerce Acceptance

Change ID: fresh-db-commerce-acceptance
Revision: 1
Status: Applying
Decision authority: Repository owner
Approval basis: Repository owner instructed continuing the v1 release-readiness sequence after ECPay official conformance. This slice closes the local fresh-database commerce gap without expanding into deployment infrastructure or additional payment products.
Repository baseline: fc37f411d7906380163a3c37b4b01379148ae0df
Supersedes: none

## Outcome

A clean local database for the minimal-cart sample must contain enough deterministic commerce configuration to exercise the reference purchase flow through a durable guest order and, when valid ECPay stage runtime configuration is supplied, through server preparation of the hosted ECPay form. ECPay discovery, checkout validation, and payment preparation must all remain truthful about runtime provider readiness.

## In scope

- Seed at least one enabled shipping method for the minimal-cart local sample when shipping configuration is empty.
- Seed an ECPay payment row in local sample data with readiness derived from the same validated runtime ECPay configuration used by bootstrap: valid stage/production runtime may be ready/enabled; absent runtime configuration remains pending/disabled; configured-but-invalid runtime fails closed.
- Preserve the existing non-ECPay sample payment methods.
- Make public payment discovery, checkout validation, and hosted ECPay payment preparation fail closed when an ECPay database row is marked ready but runtime ECPay configuration is absent or its environment does not match the admin payment environment (`sandbox` ↔ `stage`, `production` ↔ `production`).
- Add deterministic fresh-SQLite acceptance coverage for seed → discovery → quote → durable guest order and, with valid stage configuration, hosted-payment form preparation, including truthful ECPay availability.
- Update canonical project/commerce status to distinguish local fresh-DB acceptance from still-pending public deployment/stage acceptance.

## Out of scope

- Public ECPay stage transaction, Cloudflare Tunnel, Railway/Pages deployment, rate-limit topology, refund/invoice/logistics/reconciliation, or new payment providers.
- Making local test credentials a production default.
- Generic provider registries or runtime DI/plugin abstractions.
- Changing the checkout API envelope or payment-provider architecture.

### REQ-001: Fresh minimal-cart data is checkout-capable

The local development seed MUST create deterministic commerce configuration sufficient for the sample storefront to discover a sellable product, an enabled shipping method, and at least one usable payment method on a fresh database.

#### AC-001: Fresh database reaches durable guest order and local hosted-payment boundary
- GIVEN a fresh SQLite database with migrations applied
- WHEN the minimal-cart development seed runs and the sample flow selects a seeded in-stock SKU, seeded shipping method, and usable payment method
- THEN server discovery returns those choices, server quote succeeds with authoritative totals, and guest CreateOrder persists an unpaid durable order with a one-time access token
- AND GIVEN valid stage ECPay runtime configuration on a separate fresh database
- WHEN ECPay is selected and the guest order is created
- THEN `PrepareECPayPayment` produces the stage hosted-payment form with authoritative amount, server-owned callback URLs, Credit mode, and CheckMacValue without requiring a provider network call

### REQ-002: ECPay availability reflects runtime truth

An admin/database ECPay row MUST NOT make ECPay usable unless the service has a valid runtime ECPay configuration whose environment matches the row. This rule applies consistently to public discovery, checkout validation, and hosted-payment preparation.

#### AC-002: Missing runtime ECPay is not publicly selectable
- GIVEN an enabled/ready ECPay database row but no runtime ECPay configuration
- WHEN public payment methods are listed or checkout validates ECPay
- THEN ECPay is excluded from public discovery and rejected as an invalid checkout payment method

#### AC-003: Runtime/admin environment mismatch fails closed at every payment entry point
- GIVEN an enabled/ready ECPay row marked `sandbox` with production runtime, or `production` with stage runtime
- WHEN public discovery, checkout validation, or `PrepareECPayPayment` runs
- THEN ECPay is unavailable until the environments match
- AND hosted-payment preparation does not create a provider payment attempt while the mismatch exists

#### AC-004: Matching runtime ECPay reaches hosted-payment preparation
- GIVEN an enabled/ready ECPay row and matching runtime configuration (`sandbox` with stage or `production` with production)
- WHEN public discovery and checkout validation run
- THEN ECPay is available
- AND the existing hosted-payment preparation flow may create its durable attempt and signed AIO form

### REQ-003: Local seed does not pretend unconfigured or invalid ECPay is ready

The minimal-cart development seed MUST represent ECPay as pending/disabled when ECPay runtime values are absent, as ready/enabled only when the complete runtime configuration passes the real ECPay configuration constructor, and MUST fail closed when ECPay values are present but the runtime configuration is invalid.

#### AC-005: Seeded ECPay readiness follows real runtime config
- GIVEN a fresh database
- WHEN the development seed runs without ECPay configuration
- THEN the ECPay admin row exists as `pending_setup`, disabled, and `sandbox`
- AND WHEN the seed runs on a fresh database with complete valid stage ECPay configuration
- THEN the ECPay row is `ready`, enabled, and `sandbox`
- AND WHEN ECPay fields are present but public origins are invalid or production uses known public test credentials
- THEN seeding fails closed rather than recording ECPay as ready

### REQ-004: Acceptance boundary remains honest

Local fresh-DB acceptance MUST NOT be reported as public provider/deployment acceptance.

#### AC-006: Canonical status keeps stage transaction pending
- GIVEN the local fresh-DB gate passes
- WHEN README/project/commerce status is read
- THEN local sample acceptance is marked complete while public HTTPS deployment and one real ECPay stage transaction remain pending release-readiness work
