# Minimal-cart integration controlled specification

Change ID: minimal-cart-integration
Revision: 12
Status: Verifying
Decision authority: Repository owner/user, with designated product, operations, security, and legal approvers for their named Gates
Approval basis: User approved revision 2 by plain apply on 2026-08-11, then repeatedly instructed continuation and correction through revisions 3-8. Revision 8 records the approved B4 media verification and compression decisions. Revision 9 (2026-08-13) records the user's explicit approval of the media lifecycle. Revision 10 (2026-08-13) records the user's decision that authorized staff control shipping methods and fees in the admin. The user issued plain `apply` for revision 11 on 2026-08-13, authorizing Google and LINE OAuth implementation through Supabase Auth. The user issued `apply` for revision 12 on 2026-08-14 after dashboard and manager staff-list acceptance failures were reproduced locally. The user issued plain `apply` for B01F on 2026-08-15, authorizing the static footer navigation remediation for AC-012.
Repository baseline: 7c45b616fbe3a632ffe2a39d872c98485466c991
Supersedes: none

## Outcome

The minimal-cart public site and admin will use the Go application backend and one set of authoritative commerce, content, identity, and publishing rules. Public informational/catalog content remains static-first on Cloudflare Pages; runtime quote, order, inventory, identity, and admin behavior goes through Go. Production behavior must not fall back to fabricated fixture data.

`INTEGRATION_PLAN.md` is the implementation strategy for this controlled specification. This document is normative when the two disagree.

## Scope

In scope:

- Admin authentication, capabilities, product/category/media management, orders, content publishing, member/staff identity, and relevant dashboard data.
- Catalog, product/category static routes, cart/checkout quote and order creation, account/order access, and reviewed policy content.
- SQLite/PostgreSQL schema parity, OpenAPI/client/renderer contracts, R2 upload safety, Cloudflare publishing receipts, and fixture removal.
- Empty, loading, error, forbidden, success, confirmation, recovery, keyboard, touch, and no-JavaScript behavior where those states can occur.

Out of scope:

- A second backend, browser-to-database access, request-time SSR, Pages Functions, incremental rendering without measured need, or a generic provider framework.
- Fake reviews migrated as real data, unverified online payment methods, and product revisioning for parallel published/draft product versions.
- Product behavior whose Gate below remains unresolved. The affected action remains disabled or hidden instead of receiving a guessed rule.

## Architecture invariants

- Go is the only application backend; browsers never query PostgreSQL or Supabase Database directly.
- Public content follows store -> Go render composition -> renderer -> staged `dist/` -> Cloudflare Pages.
- Renderer receives explicit input and does not open a database, call a module service, or issue HTTP requests.
- Business modules do not import one another; bootstrap/tools compose owners, and platform does not import business modules.
- `auth.Principal` is explicit at handler/service boundaries and is not hidden in `context.Context`.
- Schema changes have equivalent SQLite and PostgreSQL migrations using portable behavior.
- TWD money uses integers; browser-supplied prices, totals, status, permissions, and media keys are untrusted.
- Media association and garbage-collection claims are serialized by database state and constraints; an R2 deletion cannot race a successful product association or re-verification into a dangling reference.

## Approval Gates

| Gate | Decision required | Resolved decision | Blocks |
|---|---|---|---|
| GATE-001 Admin hosting/auth transport | Same-origin `/admin/` or explicit separate origin; token transport and any new auth dependency | RESOLVED: Admin SPA served by Vite dev proxy to Go API on :8080; Bearer token in Authorization header; same-origin in production via Go static handler. No new auth dependency beyond existing Supabase/dev verifier. | Slice 0 |
| GATE-002 Payment semantics | Which offline method can be fulfilled; provider methods and webhook semantics | RESOLVED: Offline payment only — ATM transfer and cash-on-delivery (COD). Admin manually marks orders as paid. No online payment provider integration; online/unsupported methods remain disabled. | Slice 2 |
| GATE-003 Tax and rounding | Tax-inclusive/exclusive, taxable base, invoice responsibility, integer rounding | RESOLVED: Tax-inclusive (台灣含稅內含). Product prices include 5% business tax. No separate tax line at checkout. All money in integer TWD. Invoice responsibility deferred to a future change. | Slice 2 |
| GATE-004 Shipping/inventory/returns | Taiwan/cross-border scope, stock decrement, cancellation compensation, returned-item restock | RESOLVED: Taiwan main island only. Stock decrements atomically on order creation. Cancellation before shipment restocks. Return requires admin approval; restock on return confirmation. No cross-border shipping. | Slice 2 |
| GATE-005 Admin step-up | AAL2 or equivalent recent re-auth for staff, refund, offline-payment, payment-setting actions | RESOLVED: Dev mode skips step-up. Production AAL2 is a Supabase Auth configuration concern, not Go application code. High-risk mutations (refund, payment settings) require `twcommerce.admin` capability. | Slice 2/4 |
| GATE-006 Publishing transport | Trigger-only Deploy Hook or Pages API status tracking; secret owner and last-known-good behavior | RESOLVED: Cloudflare Deploy Hook trigger only. UI shows "triggered" not "deployed." No terminal status tracking in this change. Secret owner is the deployment environment. | Slice 3 |
| GATE-007 Legal/operations copy | Named approver, effective/review dates for privacy, terms, shipping, warranty, payment, data claims | RESOLVED: Placeholder draft copy with explicit draft markers. No legal claims published until a named approver reviews. Unapproved policy pages hidden from public output. | Slice 3 and release |
| GATE-008 Customer/staff lifecycle | Guest/account scope, invite/deactivate, session refresh, metadata sync and recovery | RESOLVED: Guest checkout (no account required) plus registered accounts via Supabase Auth. Dev verifier for local development. Staff invite/deactivate via admin with `staff.update` capability. | Slice 4 |
| GATE-009 PII lifecycle | Collection minimum, retention, deletion, masking and access-log policy | RESOLVED: Minimal collection — name, email, phone, shipping address only. Permanent retention (no automatic deletion). List responses mask email/phone. Production release deferred until a formal retention/deletion policy is approved by a named authority. | Slice 4 and release |

## Authority and delivery ownership

| Value | Durable source of truth | Authority | Delivery owner/failure behavior |
|---|---|---|---|
| Product/category description and publish state | Commerce store | Go catalog service with capability/version checks | Static renderer; failed render keeps last-known-good output |
| Price, promo, shipping, tax and payment availability | Commerce policy/store plus approved Gate decisions | Go quote/order service | Runtime API; failure stops checkout, never uses browser constants |
| Inventory and return restock | Variant/order-item rows and transaction log | Go order/inventory service | Runtime API; stale HTML is only a hint |
| Order totals and three state machines | Order rows/items/events | Go service and verified/offline payment action | Runtime API; idempotent recovery for unknown outcomes |
| Authenticated identity | Validated Supabase token or explicit dev verifier | Go auth verifier | Runtime; invalid/outage fails closed |
| Admin role/status | Active staff row after staff lifecycle; immutable trusted bootstrap role before mutations exist | Go authorization resolver/service | `/api/admin/me`; UI hiding never substitutes for enforcement |
| Site/policy content | Draft and published content copies | Go content approval/publish services | Static renderer; drafts never leak through another full render |
| Deployment status | `publish_runs` plus Cloudflare deployment result when tracking is selected | Go publish service interprets provider terminal state | Trigger acceptance is not deployment success |
| Media object | R2 bytes plus commerce metadata | Go media/catalog validation | Direct upload; unverified object never becomes public product metadata |
| PII | Owning order/member/staff store | Object/capability checks and approved retention policy | Runtime authenticated response; logs and lists minimize/redact |

## UI surface contract

| Surface and current/proposed location | REQ/AC | Persona and primary task | Data/read/write owner | States, consequence and recovery | Critical journey and evidence |
|---|---|---|---|---|---|
| SURF-001 Home/catalog: `templates/home.html`, Header/Hero/ProductGrid/Footer islands | REQ-003, REQ-006, REQ-008; AC-005, AC-012, AC-016 | Public visitor browses categories/products/content | Static `render.Input`; runtime only for cart/checkout | Published empty state is truthful; render failure retains previous deployment; navigation remains usable without JS | Publish product/content -> render -> inspect semantic HTML and links with JS disabled |
| SURF-002 Product/category detail: current ProductDetailDialog/CategoryFilter; proposed `/products/{slug}/`, `/categories/{slug}/` | REQ-003, REQ-008; AC-005, AC-006, AC-016 | Public visitor opens a stable product/category URL | Static catalog snapshot; quote revalidates live price/stock | Missing/archived -> real 404/absence; no fixture product; dialog and route use same snapshot | Admin publish -> static route and dialog agree; traversal/reserved slug tests pass |
| SURF-003 Cart/checkout: CartDrawer and CheckoutDialog | REQ-004, REQ-005, REQ-008; AC-007 through AC-010, AC-016 | Shopper quotes and places an order | Browser stores IDs/quantity only; Go owns quote/order | Loading, validation, corrected price, unavailable stock, retryable error, submitting, success; unknown result retries same key | Tamper totals -> server result unchanged; duplicate/race tests; no fake success |
| SURF-004 Account/order tracking: AccountDialog, TrackOrderDialog, user store | REQ-002, REQ-009, REQ-010; AC-017 through AC-020, AC-028 | Guest/customer signs in by email/password, Google, or LINE and views only authorized orders | Supabase identity + Go identity/order access; no local fake orders | Disabled providers are absent; redirect/provider/verification failure returns to a recoverable signed-out state; forbidden remains distinct from not found | Google and LINE callback -> Supabase session -> Go member route; cross-user access rejected; production bundle contains no secret or demo credentials/history |
| SURF-005 Admin session/navigation: `admin/src/router.ts`, auth store, Topbar/sidebar | REQ-002, REQ-008; AC-003, AC-004, AC-015, AC-028 | Staff authenticates by email/password, Google, or LINE and sees permitted resources | Supabase identity + canonical staff row + Go `/api/admin/me`; Go services enforce | Loading, invalid/expired session, unlinked identity, forbidden, success; 401 clears session; no role selector | OAuth callback alone grants no admin capability; linked active staff succeeds and stale/demoted identity loses capability |
| SURF-006 Admin catalog/media: `/res/:resourceKey`, ResourceListPage, products config | REQ-003, REQ-008, REQ-011; AC-005, AC-006, AC-015, AC-021, AC-022 | Authorized catalog staff creates/publishes product/category and images | Go commerce/media APIs and R2 | Empty/loading/error/forbidden/success; form dirty/invalid/submitting; version conflict reloads row; upload failure leaves no metadata | Create -> upload verify -> publish -> render; dual migration/contract/media tests |
| SURF-007 Admin orders/dashboard: generic resource page, orders config, DashboardPage | REQ-004, REQ-005, REQ-008; AC-007 through AC-010, AC-015 | Authorized operations staff fulfills, returns, refunds/restocks | Go order/inventory services and append-only events | Masked list, protected detail, confirmations, pending, 409 conflict, partial failure receipt | Order -> admin detail -> legal transition; stale/illegal actions do not mutate or append event |
| SURF-008 Admin content/publish: content resource plus proposed publish-run status | REQ-006, REQ-007, REQ-008; AC-011 through AC-015 | Content approver/publisher edits, approves and deploys | Go content/publish service; Cloudflare transport selected by Gate | Draft dirty/invalid; approval invalidated by edit; trigger requested/active/succeeded/failed; retry creates new run | Draft unchanged publicly; approved publish -> static routes; provider failure keeps previous site |
| SURF-009 Admin members/staff: members/staff resource configs | REQ-002, REQ-008, REQ-009; AC-004, AC-015, AC-017, AC-018 | Authorized owner/support manages identity with least PII | Go member/staff services; Supabase metadata is an outbox mirror | Forbidden, masked list, protected detail, conflict, sync pending/failed/succeeded; demotion effective immediately | Self-elevation/last-owner tests; metadata failure cannot restore revoked role |
| SURF-010 Policy pages/footer: Footer, FooterPageDialog, hard-coded home feature strip; proposed static policy routes | REQ-006, REQ-008, REQ-010; AC-011, AC-012, AC-016, AC-019 | Public visitor reads approved policy/operational claims | Published content copy rendered statically | Unapproved/expired content absent; links work without JS; dialog is optional enhancement | Dist search contains no unsupported promo/payment/warranty/PII claims; approval-version tests pass |

For every runtime form or mutation above, initial/dirty/invalid/submitting/succeeded/failed states, accessible labels and associated errors, focus preservation, keyboard/touch operation, confirmation for high-consequence actions, and authoritative post-mutation refresh are part of the acceptance obligation when applicable.

## Requirements

### REQ-001: Preserve repository architecture boundaries
The integration MUST keep Cloudflare Pages static, Go as the only application backend, explicit principal/service boundaries, business-module isolation, and equivalent SQLite/PostgreSQL behavior.

#### AC-001: Static and runtime paths remain separated
- GIVEN public content/catalog data and runtime checkout data
- WHEN each path is delivered
- THEN public data is rendered through Go into `dist/`, while checkout/admin runtime data goes browser -> Go API without direct database access

#### AC-002: Module and database boundaries are enforced
- GIVEN a slice adds a module or schema
- WHEN architecture and migration checks run
- THEN modules do not import one another and matching SQLite/PostgreSQL migrations and behavior tests pass

### REQ-002: Authenticate identity and authorize every admin action
The system MUST validate admin identity, derive capabilities in Go, enforce capability/object checks in services, and make role demotion or disablement fail closed.

Supabase Auth SHALL offer email/password plus explicitly enabled Google and LINE sign-in on both public and admin login surfaces. Google uses the built-in Supabase provider. LINE uses the `custom:line` OIDC provider with issuer `https://access.line.me`, PKCE enabled, and `openid profile email` scopes; the LINE channel must receive email-address permission before the provider is enabled. Provider client secrets remain in Google/LINE and Supabase configuration, never browser, Go configuration, database, logs, or committed files. Redirects are restricted to configured application origins. A social identity receives no admin capability until its Supabase user ID matches an active canonical staff row.

#### AC-003: Missing or insufficient identity is rejected
- GIVEN an absent, invalid, expired, wrong-project, or insufficient token
- WHEN an admin route or service action is requested
- THEN the server returns 401 or 403 without protected data and the production UI cannot switch itself to owner

#### AC-004: Staff role changes are immediately authoritative
- GIVEN an active staff member is demoted or disabled while holding an older token
- WHEN the next protected request is made
- THEN the canonical active staff row removes the old capability and metadata-sync delay cannot restore it

#### AC-028: Google and LINE sign-in preserve the Go authorization boundary
- GIVEN Google or `custom:line` is enabled and its external callback is correctly configured
- WHEN a public or admin user completes, cancels, or fails the OAuth flow
- THEN Supabase produces the session, the browser returns only to an allowed application origin, Go verifies the Supabase access token, public member access is bound to that principal, admin access additionally requires an active linked staff row, and provider secrets or provider access tokens are not persisted by application code

### REQ-003: Publish a real catalog with safe media and stable routes
The system MUST let authorized staff manage categories, products, variants, inventory hints and verified media, then render only published catalog data to stable semantic static routes.

#### AC-005: Catalog publish reaches a real public consumer
- GIVEN a valid draft category/product/variant and verified image
- WHEN authorized staff publishes and a full render succeeds
- THEN home, category and product static output contain the published snapshot while draft/archived records do not

#### AC-006: Catalog identifiers and media cannot escape policy
- GIVEN invalid, changed-after-publish, traversal/reserved slugs or an object with forged type/unsafe format/oversize dimensions
- WHEN catalog or media validation runs
- THEN the mutation is rejected and no unsafe route or product-media metadata is published

### REQ-004: Make Go authoritative for quote and order creation
The system MUST recalculate item price, promotion, shipping, tax, payment availability and totals in Go, and MUST create an order with transactional inventory and request-bound idempotency.

Authorized staff MUST manage shipping methods through the Go admin API. Each method has a stable method key, label, description, non-negative integer TWD fee, optional positive integer free-shipping threshold, enabled state, sort order, and optimistic-concurrency version. Production starts without fabricated fee rows. The public API exposes only enabled methods; quote and order creation read the current persisted method and calculate the same fee rule. A missing or disabled method is unavailable, storage failure fails closed, and browser values never override the persisted fee or threshold.

#### AC-007: Client tampering cannot control money or availability
- GIVEN a client submits altered prices, totals, promo outcome, stock, payment status or an unavailable method
- WHEN quote or order creation runs
- THEN the server ignores authoritative client claims and returns only the approved server calculation or a safe validation error

#### AC-027: Admin shipping configuration is the sole shipping authority
- GIVEN an authorized staff member creates, edits, disables, or reorders a shipping method
- WHEN the public method list, quote, or order creation reads shipping configuration
- THEN the public list reflects only enabled current rows, quote and order use the same current fee and threshold rule, stale admin updates conflict without overwriting, and no browser or production fixture fallback supplies another fee

#### AC-008: Retries and inventory races are safe
- GIVEN duplicate submissions or two requests competing for the final stock
- WHEN order transactions execute
- THEN the same key and request hash yields one order, a reused key with another payload conflicts, and stock never becomes negative or oversold

### REQ-005: Enforce auditable order, return and refund transitions
The system MUST enforce separate fulfillment, payment and return state machines, optimistic concurrency, append-only events, and explicit inventory/refund compensation.

#### AC-009: Illegal or stale transitions have no side effect
- GIVEN a stale version, forbidden actor or illegal target state
- WHEN an order mutation is requested
- THEN the server returns 403/409/validation error without changing the order or appending a success event

#### AC-010: Cancellation, return, restock and refund remain distinct
- GIVEN a cancellable or delivered order
- WHEN cancellation or return handling occurs
- THEN stock is compensated at most once under the approved rule and paid/refunded/restocked states change only through their explicit authorized actions

### REQ-006: Isolate drafts and publish only approved site content
The system MUST keep draft and published site content separate, invalidate approval after material edits, and render only the approved published copy to semantic public output.

#### AC-011: Draft and approval versions cannot leak
- GIVEN published content and a newer draft or an edited approved draft
- WHEN another resource triggers a full render or publish is attempted
- THEN the previous published copy remains public and stale/missing approval cannot authorize the changed draft

#### AC-012: Policy and operational content is truthful and static
- GIVEN a policy/feature claim without current approval or without supporting product capability
- WHEN public output is rendered
- THEN the claim is absent, while approved policy routes and footer links remain readable and navigable without JavaScript

### REQ-007: Represent deployment truth and preserve last-known-good output
The publishing flow MUST distinguish saved state, trigger acceptance and terminal deployment status, protect provider secrets, and promote only complete validated render output.

#### AC-013: Trigger acceptance is not reported as deployment success
- GIVEN a content/catalog publish request triggers Cloudflare
- WHEN the trigger is accepted but no terminal status is known
- THEN the admin shows requested/active only and shows success solely when the selected status contract proves it

#### AC-014: Failed render or deployment preserves the prior site
- GIVEN rendering, asset validation or Cloudflare deployment fails
- WHEN the publish attempt ends
- THEN incomplete staging output is not promoted, the previous site remains available, and the admin receives a safe recoverable failure receipt

### REQ-008: Provide complete and accessible surface states
Every in-scope UI surface SHALL orient the user, expose the next permitted action and consequence, distinguish relevant states, and provide accessible feedback and recovery tied to authoritative data.

#### AC-015: Admin surfaces handle real states and consequences
- GIVEN an authorized or unauthorized staff user encounters empty, loading, validation, conflict, provider or server outcomes
- WHEN they use an in-scope admin surface by keyboard or touch
- THEN the state and permitted action are clear, protected data remains hidden, and success refreshes from the authoritative response

#### AC-016: Public surfaces degrade without fabrication
- GIVEN JavaScript is unavailable or a runtime quote/order call fails
- WHEN a visitor browses or attempts checkout
- THEN static catalog/content/navigation remains usable, interactive checkout stops with recovery guidance, and no fixture price/order/content appears

### REQ-009: Protect customer identity and PII
The system MUST minimize PII, restrict order/member/staff access by identity and capability, redact logs/lists, and block release until an approved retention/deletion policy exists.

#### AC-017: Object-level access does not rely on guessable identifiers
- GIVEN a customer or staff user requests another person's order or protected detail using a known order number/ID
- WHEN authorization runs
- THEN access is denied without leaking ownership or PII and allowed users receive only their permitted view

#### AC-018: PII exposure and lifecycle are controlled
- GIVEN PII enters order/member/staff flows
- WHEN it is stored, listed, logged, exported or deleted
- THEN only approved fields and retention behavior occur, lists/logs are minimized or masked, and secrets/tokens never appear

### REQ-010: Remove fabricated production behavior by real consumer
Completed production paths MUST NOT import or fall back to mock products, prices, accounts, orders, payment success, roles, content or unsupported operational/legal claims.

#### AC-019: Migrated production paths contain no fixture fallback
- GIVEN a slice has a real producer-to-consumer path
- WHEN production imports, bundles, error handlers and rendered output are inspected
- THEN that path contains no fabricated data or credential and failure is represented as empty/error/retry instead

#### AC-020: Development fixtures remain explicitly bounded
- GIVEN a test or local demonstration still requires fixture data
- WHEN development/test and production builds run
- THEN the fixture is enabled only by an explicit non-production boundary and cannot enter the production bundle or runtime fallback

### REQ-011: Keep storage, API and consumers contract-aligned
The system MUST keep migration models, service DTOs, OpenAPI, admin/theme types, renderer inputs and actual consumers aligned for identifiers, nullability, enums, money, timestamps, booleans, versions and errors.

#### AC-021: Contract fields match real consumers
- GIVEN a catalog, order, content, identity or publish contract changes
- WHEN contract/type tests and consumer builds run
- THEN every required field has one canonical name/type/unit and obsolete aliases or operation IDs are absent

#### AC-022: Both database drivers prove equivalent behavior
- GIVEN a slice changes persistent behavior
- WHEN clean migration, constraint and integration tests run for SQLite and PostgreSQL
- THEN both drivers enforce the same required invariants or an explicit approved exception blocks completion

### REQ-012: Deliver only through controlled, evidenced slices
The change MUST remain governed by this revision/status, map every requirement and acceptance scenario to dependency-ordered slices and current proof, and SHALL NOT authorize implementation while the controlled specification is Draft or blocked.

#### AC-023: Slice completion requires named observed evidence
- GIVEN an implementation slice claims completion
- WHEN its gate is evaluated
- THEN targeted tests, contract/type alignment, permission/failure cases, fixture-negative checks, `speccheck`, scopecheck and verifier results are recorded against all covered REQ/AC IDs

#### AC-024: Propose phase cannot silently become apply
- GIVEN this controlled specification is Draft or a required Product Gate is unresolved
- WHEN planning artifacts are reviewed
- THEN only spec/plan/evidence artifacts change and no product implementation path is treated as authorized

### REQ-013: Retire media without dangling references
The system MUST expire temporary uploads after 24 hours and garbage-collect verified media only after 7 continuous days without a product association, while preserving referenced media and retrying provider deletion failures safely.

#### AC-025: Product references always protect verified media
- GIVEN a verified object is associated with any product, or association races with a GC claim
- WHEN garbage collection evaluates or claims that object
- THEN the product association wins or fails atomically, and GC never leaves a committed product row pointing to deleted media

#### AC-026: Unassociated media expires predictably and retryably
- GIVEN an `uploads/` object older than 24 hours, a stale verification older than 24 hours, or active verified media continuously unassociated for at least 7 days
- WHEN lifecycle cleanup or the media GC job runs
- THEN only eligible objects are selected, dry-run performs no deletion, successful deletion clears its job, failed deletion remains queued for retry, and re-verification renews protection

## Amendment control

Revision 1 is the initial controlled specification. Any change to observable behavior, authority, static/runtime ownership, failure semantics, scope or an acceptance scenario must record affected REQ/AC IDs, old/new meaning, reason, decision authority, affected slices and invalidated evidence; the spec returns to Draft before implementation continues.

| Revision | REQ/AC | Old meaning | New meaning | Reason | Approval basis | Invalidated evidence and replay |
|---|---|---|---|---|---|---|
| 12 | REQ-002, REQ-008, REQ-012, AC-003, AC-004, AC-015, AC-023, AC-024 | The dashboard's real data rendered inside an inert native `<template>` and therefore was invisible; staff navigation used `staff.read` while `GET /api/admin/staff` incorrectly required `staff.update` | Dashboard KPI/panel DOM must render after authoritative read responses; any principal with `staff.read` may list staff while every staff mutation remains `staff.update`-protected | Local linked-manager walkthrough reproduced invisible dashboard data and a staff-list 403 despite the manager's `staff.read` grant | User issued `apply` on 2026-08-14 | AC-015, AC-023 and AC-024 require focused source, test, build and linked-session browser replay at revision 12; no write capability is broadened |
| 11 | REQ-002, REQ-008, REQ-009, REQ-010, REQ-012, AC-003, AC-015, AC-017, AC-019, AC-023, AC-024, AC-028 | Supabase Auth session producers exposed email/password only; live provider success was pending without a selected social identity contract | Public and admin surfaces prepare Google and LINE sign-in through Supabase; Google is built-in, LINE is `custom:line` OIDC with PKCE and required email permission; OAuth never bypasses Go member/staff authorization | The user requested Google and LINE be prepared together after configuring Supabase | User asked `google 跟 line 一起準備？` on 2026-08-13, then issued plain `apply` on 2026-08-13 | Existing email/password and Go authorization evidence remains current; new AC-028 and the already-pending auth consumer, UI-state, bundle/security, live-provider and controlled-delivery evidence require replay for both providers |
| 10 | REQ-004, REQ-008, REQ-011, REQ-012, AC-007, AC-015, AC-021, AC-022, AC-023, AC-024, AC-027 | Recognized shipping method keys failed closed because no authority could approve or persist fees; public methods were hardcoded unavailable descriptors | Authorized staff own shipping methods and integer TWD fees through versioned Go-backed admin configuration; production seeds no fee rows; enabled rows drive the public list; quote and order read the same current persisted fee and optional free-shipping threshold | The user clarified that delivery fees belong under backend control and requested full-flow validation | User stated `宅配費應該是要後台控制` and then instructed Codex to split the next stage for Grok implementation on 2026-08-13 | AC-027 starts pending; the previous shipping failure-mode evidence for REQ-004/AC-007 and affected admin/API/database/delivery evidence return to pending until the three revision-10 slices and independent walkthrough are replayed |
| 9 | REQ-003, REQ-011, REQ-012, REQ-013, AC-006, AC-022, AC-023, AC-025, AC-026 | Verified-object deletion had no approved retention window and the schema could not serialize deletion against product association or re-verification | Temporary uploads expire after 24 hours; verified media must remain continuously unassociated for 7 days; database-backed asset state and deletion jobs make GC race-safe and retryable; referenced media is never auto-deleted | User approved the proposed lifecycle after reviewing the GC meaning and trade-offs | User replied `可以` on 2026-08-13 to the 24-hour/7-day lifecycle proposal | New AC-025/026 and REQ-013 start pending; impacted database, media, security and verification evidence must be replayed at revision 9 |
| 1 | none | none | Initial Draft | New controlled specification | Pending user and named Gate approvals | No implementation evidence exists |
| 2 | GATE-001 through GATE-009 | Gates pending with safe-state-only behavior | Gates resolved with safe defaults (NOT all explicit decisions) | User issued `apply` and answered GATE-002/003/004/009; remaining Gates resolved with safe defaults | User `apply` on 2026-08-11 plus Gate answer selections | AC-024 replayed: spec is now Applying, not Draft; implementation is authorized. NOTE: the original revision-2 text claimed 'All Gates resolved with explicit decisions' — this was incorrect. The shipping fee schedule was never resolved (no approval artifact authorizes 120/60/1500). Corrected in revision 4. |
| 3 | REQ-004, REQ-010, AC-007, AC-010, AC-016, AC-019, AC-021, AC-023, AC-024 | Shipping methods silently defaulted to home_delivery with unapproved 120/60/1500 fees; browser hardcoded pricing/shipping/payment constants; governance lacked strict_evidence and observed_revision binding | Quote/CreateOrder reject empty/unknown methods (ErrInvalidShippingMethod) and recognized methods without approved fee schedule (ErrShippingFeeScheduleNotApproved); browser fetches shipping/payment methods from server API; cart.ts no longer computes monetary values; strict_evidence enabled, passed evidence bound to observed_revision 3 | Independent review found five defects: P0 shipping validation, browser authority, authority boundary, governance, fixture residue | User instructed to continue existing change and fix all defects on 2026-08-12 | AC-007/010/016/019/021/023/024 and REQ-004/010/012 invalidated and replayed at revision 3; production-content-audit receipt added for AC-019 |
| 4 | REQ-004, REQ-010, REQ-011, AC-007, AC-010, AC-016, AC-019, AC-021, AC-022, AC-023, AC-024 | Cart persisted full Product object (price/stock/name trusted from localStorage); CartDrawer rendered item.product.price * quantity (pre-quote monetary claim); payment method was UI-only (not sent to quote/order APIs, not validated server-side, not persisted); checkout auto-selected unavailable shipping method and allowed progression; revision-2 amendment text claimed 'All Gates resolved' (contradicted by unresolved shipping fee schedule); AC-023 walkthrough receipt was API-only and mislabeled as independent; admin/dashboard.html evidence did not state the mockup is excluded from VCS/build/serve | cart.ts persist() serializes only identifiers (productId, sku, quantity, selections); restore() rehydrates from catalog API, fails closed if API unavailable; CartDrawer shows no monetary values; payment method wired end-to-end (QuoteInput, OrderInput, Order, DB migration 007, OpenAPI, server validation against enabled+ready, persisted on order); checkout keeps selectedShipping empty when no available method, blocks progression at shipping step; revision-2 amendment text corrected; AC-023 downgraded to pending (implementer walkthrough, not independent); admin/dashboard.html evidence states precisely that obsolete untracked mockup is excluded from VCS/build/serve | Independent acceptance rejected revision 3 with seven findings | User instructed to continue existing change and fix all findings on 2026-08-12 | AC-007/010/016/019/021/022/023/024 and REQ-004/010/011/012 invalidated and replayed at revision 4; implementer walkthrough receipt added for AC-023 (downgraded to pending) |
| 5 | REQ-005, REQ-011, AC-008, AC-021, AC-023, AC-024 | samePayload only compared SKU/quantity/customer_name/email; createOrder validated mutable state before idempotency key lookup; CheckoutDialog generated fresh idempotency key on every invocation (Date.now+Math.random); CheckoutDialog fabricated order ID on malformed 2xx (TW-Date.now); OpenAPI QuoteResult/Order required fields did not match Go/TS guarantees; empty idempotency key bypassed protection; checkoutFingerprint computed/watch declared before referenced reactive state (TDZ risk); idempotencyKey unconditionally reset in ui.checkoutOpen watcher; computeRequestFingerprint used \n-separated field=value encoding (delimiter collision) | samePayload expanded to all client-controlled fields via persisted SHA-256 request fingerprint (migration 008); early idempotency lookup before mutable-state validation; stable crypto.randomUUID key; fail-closed order ID validation; OpenAPI required fields aligned; empty key rejected with ErrMissingIdempotencyKey (400); declaration order fixed; open-watcher key reset removed; fingerprint encoding changed to encoding/json canonical struct (length-delimited); checkoutFingerprint uses JSON.stringify over typed object; SKU not trimmed in fingerprint; CheckoutDialog fail-closed on all required Go-emitted order fields | Independent review found multiple defects in the payment-method and idempotency path | User instructed to continue existing change and fix all defects on 2026-08-12 | AC-008/021/023/024 and REQ-005/011/012 invalidated and replayed at revision 5; 15+ regression tests added; 3 collision regression tests added |
| 6 | REQ-002, REQ-007, REQ-008, AC-006, AC-015, AC-023, AC-024 | content.Publish used principal.Role == admin instead of auth.Can(principal, content.publish); admin ConfirmDialog used v-html for body/meta; ResourceListPage injected untrusted row values into HTML strings; render.go site-content Body used template.HTML(b.Body) violating INTEGRATION_PLAN.md:388; media.Presign only checked UserID non-empty (any Supabase user could upload) and used image/ prefix (allowed SVG) | content.Publish uses auth.Can(principal, content.publish); ConfirmDialog v-html removed, structured typed props with Vue text interpolation; contentPageData.Body changed to string, html/template escapes it; media.Presign uses auth.Can(principal, media.upload) capability check, raster MIME allowlist (jpeg/png/gif/webp/avif, SVG rejected), extension-to-MIME cross-check, canonical extension in key | Independent review confirmed four high-confidence defects | User instructed to continue existing change and fix all defects on 2026-08-12 | AC-006/015/023/024 and REQ-002/007/008/012 invalidated and replayed at revision 6; 13 media regression tests added; LIMITATION: presign cannot verify magic bytes or byte size — post-upload verification required (documented in evidence, not claimed as done) |
| 7 | REQ-003, REQ-004, REQ-005, REQ-006, REQ-007, REQ-008, REQ-010, REQ-011, AC-005, AC-006, AC-007, AC-009, AC-010, AC-011, AC-012, AC-014, AC-015, AC-016, AC-019, AC-021, AC-023, AC-024 | commerce createOrder validated items/customer_name before idempotency lookup (same-key different-payload replay with empty items returned validation error instead of ErrIdempotencyConflict); mulOverflow/addOverflow used int64 intermediates that overflow on 64-bit; HTTP switch had duplicate cases identical to default; OpenAPI OrderInput lacked maxLength constraints; REQ-003/AC-006 claimed passed without post-upload media validation; REQ-006/AC-011 claimed passed without approval workflow/version history/expiry and with published snapshot reading draft key/placement/sort_order; REQ-007/AC-014 claimed passed with inconsistent fail-open and no path containment for unvalidated Category/Key; REQ-005/AC-009/AC-010 claimed passed without transactional cancel restock, auto-restock on return approval (violates INTEGRATION_PLAN received-then-explicit-restock), no real refund/payment transition, timeline not append-only order_events with reason; REQ-008/AC-015 claimed passed with admin form type mismatches (number/switch/datetime as strings), missing required slug field, placeholder image, payment method form missing service fields, promo starts_at/expires_at vs API starts_unix/expires_unix, bulk category actions with no endpoint mapping, published content edit keeps status=published so publish action hidden; REQ-011/AC-021 claimed passed with /category/ vs spec /categories/{slug}/, raw Product vs {product}, raw Order vs {order}+422, images JSON string vs TS string[], admin field type/name mismatch; AC-005 claimed passed but ListPublishedProducts only queries status=active so low_stock/out_of_stock products vanish from public/render, GetProductBySlug has no status filter so draft products exposed on public detail; AC-016 claimed passed but static route/consumer snapshot inconsistent (plural route contract, low/out omission, draft detail exposure); REQ-004/AC-007 claimed passed but /api/promos/active publicly enumerates all active promo codes (violates INTEGRATION_PLAN.md:351/1070), calculateDiscount returns 0,nil for unknown code (no rejection), QuoteResult echoes submitted promo_code regardless of validity; REQ-010/AC-012/AC-019 claimed passed but server/tools/dev/main.go:147 seeds '全館滿 1500 免運' (published), Hero.vue:137 displays '滿 NT$1,500 免運', service.go:84-97 defines testFeeSchedule with 1500/120/60 in production source | commerce createOrder validation ordering corrected (idempotency_key blank check is only pre-lookup check; items/customer_name checks moved after FindOrderByIdempotencyKey); overflow helpers corrected to division-based bounds checking; HTTP switch duplicate cases removed; OpenAPI OrderInput maxLength constraints added; REQ-003/AC-006, REQ-006/AC-011, REQ-007/AC-014 downgraded to pending (B4/B5/B6 blockers); REQ-005/AC-009/AC-010 downgraded to pending (B7 cancel/return/refund/payment blockers); REQ-008/AC-015 downgraded to pending (B8 admin form blockers); REQ-011/AC-021 downgraded to pending (B9 contract/consumer misalignment blockers); AC-005/AC-016 downgraded to pending (B10 published catalog/snapshot blockers); REQ-004/AC-007/REQ-010/AC-012/AC-019 downgraded to pending (B11 promo enumeration + invalid promo silent accept + 1500/120/60 source residue); REQ-006/AC-011 proof augmented with legacy article BodyHTML template.HTML + no CSP/_headers; REQ-007/AC-014 proof augmented with theme missing dist/islands CSS/JS silent skip | Second independent acceptance found additional defects in cancel/return/refund/payment flow, admin form contracts, public catalog behavior, contract/consumer alignment, promo enumeration, and source residue | User instructed to continue existing change and fix all findings on 2026-08-12 | AC-005/006/007/009/010/011/012/014/015/016/019/021/023/024 and REQ-003/004/005/006/007/008/010/011/012 invalidated and replayed at revision 7; all honestly downgraded to pending — no product code fabricated; release blockers B3 through B11 documented |
