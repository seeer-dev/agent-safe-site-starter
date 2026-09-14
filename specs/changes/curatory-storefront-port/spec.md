# Curatory Storefront Port Specification

Change ID: curatory-storefront-port
Revision: 1
Status: Draft
Decision authority: Repository owner/user
Approval basis: Owner direction 2026-09-14: storefront ports the 質選所 (CURATORY) design and dynamic effects into the Go template + Vue islands architecture; the existing starter Vue admin SPA remains the backoffice. Pending review of this proposal.
Repository baseline: 10f805e18e394e7065854dfbceb327114e0d4564
Supersedes: none

## Outcome

The public storefront becomes a full port of the 質選所 (CURATORY) design — its pages, interactions, and motion — delivered through this repository's architecture: Go-rendered static pages plus Vue islands under a new `site/themes/curatory/` theme, backed by the single Go API and the existing Vue admin SPA. The reference implementation `D:\MyTools\AI設計版面\購物車系列\質選所` is a design and behavior source only; its Next.js runtime, API routes, Prisma schema, and database are NOT adopted.

## Scope

In scope:

- New theme `site/themes/curatory/` with its own templates, islands, shared stores/composables/styles, build scripts, and checks.
- Storefront pages ported as real static pages (no client-side router): home, shop (category/search/sort), product detail, news list, news detail, about, cart, checkout, order confirmation, order tracking.
- Ported dynamic effects: brand intro loader, page-load transitions, scroll reveal, image fade-in, loading skeletons, toast notifications, persistent variant-aware cart.
- Backend extensions needed by the port: category entities, product variants, product comments with moderation, coupon extension and public validation, order field extensions, storefront bootstrap, public product filtering.
- Admin SPA resources for the new/extended entities.
- Paired SQLite and PostgreSQL migrations; OpenAPI contract updates.

Out of scope:

- The 質選所 Next.js server, its API routes, its Prisma database, and its built-in admin. They are reference material only.
- A second backend, SSR runtime, or Pages Function.
- Member/login storefront features beyond the existing account model.
- Payment-provider integration (payment methods stay display/selection only; no gateway).
- LINE Pay / OAuth storefront additions.
- Full notification-template administration; order confirmation mail uses the existing mailer seam.

## Decisions recorded in this spec

- Guest order retrieval keeps the existing opaque `X-Order-Access-Token` model. The tracking page accepts an order id/number plus the access token issued at checkout (stored in browser session storage by the confirmation page). Order-number+email enumeration lookup is NOT introduced.
- Announcements/news reuse the `content` module's article pipeline (publish workflow, rendered pages) extended with pinned and publish-at fields rather than a parallel module.
- Store settings (store name, tagline, promo banner, thresholds, contact) live in the `sitecontent` module as a single-row settings record with the existing approve/publish governance, not a new module.
- Coupons extend the existing `promos` model (`percent|fixed|freeshipping`, `min_subtotal`, `usage_limit`, `used_count`) instead of a second promotion table.
- Browser code never queries any database directly; all storefront data flows through the Go API or the renderer.

## Requirements

### REQ-001: Curatory theme

The repository SHALL contain a `site/themes/curatory/` theme selectable via `SITE_THEME=curatory` that renders the ported storefront. The existing `minimal-cart` theme MUST remain selectable and unchanged in behavior.

#### AC-001: Theme selection renders the port
- GIVEN `SITE_THEME=curatory`
- WHEN `go run ./server/tools/render` runs
- THEN `dist/` contains the curatory pages with the ported design, and `minimal-cart` output remains available when `SITE_THEME=minimal-cart`.

### REQ-002: Page parity with real URLs

Each storefront view SHALL be a real server-rendered page with a stable URL: `/`, `/shop/` (with category/search/sort parameters), `/products/{slug}/`, `/news/`, `/news/{slug-or-id}/`, `/about/`, `/cart/`, `/checkout/`, `/order/`, `/track/`. Every page MUST render a meaningful no-JS baseline (navigation, available content, footer) before any island mounts.

#### AC-002: All routes render a no-JS baseline
- GIVEN a fresh render
- WHEN any storefront URL is fetched without executing JavaScript
- THEN it returns 200 HTML containing real navigation, the page's authoritative static content or an honest empty state, and footer content — never a blank shell or fixture data.

#### AC-003: Deep links work
- GIVEN a product slug, a news item, or an order confirmation URL
- WHEN the URL is loaded directly
- THEN the correct page renders without relying on prior client-side state (except the order confirmation page, which requires its access token).

### REQ-003: Motion and feedback parity

The port SHALL reproduce the reference's dynamic effects adapted to multi-page navigation: brand intro loader on first visit, page-intro transition on load, scroll-reveal animations, image fade-in, skeleton placeholders during island fetch, toast notifications, and cart persistence across pages.

#### AC-004: Motion inventory present
- GIVEN the rendered site
- WHEN a walkthrough exercises first visit, navigation, image loads, island fetches, add-to-cart, and form errors
- THEN the intro loader, reveal/fade/skeleton/toast behaviors are observed; reduced-motion preferences are respected.

### REQ-004: Catalog depth

Products SHALL support multiple verified images, variants (name, price delta, stock, sku), a featured flag, sold count, and category entities (slug, name, image, sort order, active flag) managed by staff — replacing the free-text category field for new data while remaining backward compatible with existing seeded rows.

#### AC-005: Variant selection drives price and stock
- GIVEN a product with two variants of different price deltas and stocks
- WHEN a visitor selects a variant on the product page
- THEN displayed price, stock, and add-to-cart reflect the selected variant, and quote/order pricing uses the server-authoritative variant price.

#### AC-006: Catalog browsing matches the reference
- GIVEN seeded products across categories
- WHEN the shop page is filtered by category, searched by keyword, and sorted by newest/price
- THEN results, ordering, and featured merchandising on home match the server data.

### REQ-005: Checkout parity

Checkout SHALL collect: recipient name/phone, city/district/address for home delivery, CVS store selection for CVS shipping methods, invoice type with tax id when company invoice is chosen, an optional coupon code, buyer note, and a different billing/contact identity where the reference collects one. Totals MUST come from the server quote.

#### AC-007: Checkout validation and totals are server-authoritative
- GIVEN a cart and shipping/payment selections
- WHEN the quote and order are submitted
- THEN server-computed subtotal, discount, shipping fee (including free-shipping thresholds and CVS/home fees), payment fee, and total are persisted; tampered client totals are ignored.

#### AC-008: Taiwan address and CVS flows
- GIVEN the checkout page
- WHEN home delivery is chosen, city/district selection constrains the address; when CVS is chosen, a store must be selected before submission
- THEN the order stores the structured address or CVS store name/address, invoice fields, and buyer note.

### REQ-006: Order surface

Orders SHALL expose a public order number, an opaque guest access token returned only at creation, a status timeline, and an order-confirmation page. A confirmation email SHALL be emitted through the configured mailer (log sender locally).

#### AC-009: Guest order access stays capability-bound
- GIVEN a created order
- WHEN it is fetched without the access token or with a wrong token
- THEN it fails closed; with the token it returns the masked-PII order. No order-number+email enumeration endpoint exists.

#### AC-010: Confirmation and tracking pages render real state
- GIVEN a completed checkout
- WHEN the confirmation and tracking pages load
- THEN they render items, totals, timeline, and status from the API; failure states render honestly without fixture data.

### REQ-007: News from governed content

News list and detail pages SHALL render published announcements/articles through the existing publish workflow, honoring pinned ordering and publish-at timing; drafts, stale, and future-dated items never render.

#### AC-011: News visibility follows publish state
- GIVEN draft, published, pinned, and future-dated items
- WHEN news pages render
- THEN only currently published items appear, pinned first; removing publish state removes the item after re-render.

### REQ-008: Product comments with moderation

Visitors SHALL be able to submit a product comment (nickname, content, optional 1–5 rating). New comments are pending, are not publicly rendered, and become visible only after staff approval; rejected comments never render. Staff may also record an admin reply shown with the comment.

#### AC-012: Comments fail closed until approved
- GIVEN a submitted comment
- WHEN the public product page is rendered before approval
- THEN the comment is absent; after approval it appears with rating and admin reply. Approval state is the only path to visibility.

### REQ-009: Coupons

Coupons SHALL support percent, fixed-amount, and free-shipping types with minimum subtotal, usage limit, validity window, and a public validate endpoint that reports validity, discount, free-shipping effect, and a user-facing message. Quote and order totals MUST apply the same validated result; exhausted, expired, inactive, or below-minimum codes fail closed.

#### AC-013: Coupon lifecycle is enforced
- GIVEN active, expired, exhausted, inactive, and below-minimum codes
- WHEN validated and applied to quote/order
- THEN valid codes produce the exact computed effect and all invalid cases are rejected with a message; `used_count` increments only on accepted orders.

### REQ-010: Admin coverage

The admin SPA SHALL manage the new/extended entities through capability-gated resources: categories, product variants within the product form, announcements/news, coupons, comments moderation, and store settings — without replacing authoritative empty/error states with local fixtures.

#### AC-014: New resources are reachable and gated
- GIVEN a staff session with and without the relevant capabilities
- WHEN each new resource screen is opened
- THEN permitted staff see real lists/forms; denied or unauthenticated access fails closed.

### REQ-011: Storefront bootstrap

A single public endpoint SHALL return store settings (names, tagline, promo banner, thresholds, contact), active categories, enabled shipping and payment methods, and current announcements summary for shared chrome and the intro loader. It MUST contain no PII and no admin-only data.

#### AC-015: Bootstrap feeds shared chrome
- GIVEN the bootstrap endpoint
- WHEN any storefront page loads
- THEN header/footer/promo banner render from it and unpublished or disabled items never leak.

### REQ-012: Data plane and contract parity

Every schema change SHALL ship paired SQLite and PostgreSQL migrations with matching results; `contracts/openapi.yaml` SHALL document every new/changed endpoint; `go run ./server/tools/verify` and contract checks MUST pass.

#### AC-016: Both drivers reach the same schema and behavior
- GIVEN the migrations
- WHEN applied to SQLite and PostgreSQL
- THEN the same tables/fields exist and the affected API flows behave identically.

#### AC-017: Gates pass
- GIVEN the final diff
- WHEN `go run ./server/tools/speccheck` and `go run ./server/tools/verify` run
- THEN both pass against this change's scope and evidence.

## Invariants

- Static-first publish path unchanged: CMS/data → Go renderer → `dist/` → Cloudflare Pages.
- Browser never touches PostgreSQL/Supabase directly; islands fetch Go API or consume render-time data.
- Server remains the pricing/stock/coupon authority; client totals are never trusted.
- Module boundaries unchanged: modules do not import each other; platform never imports modules; `auth.Principal` stays explicit.
- Protected UI fails closed: no fixture or browser-local substitution for authoritative states.

## Open risks / noted deferrals

- 質選所's SPA hash-router becomes real page navigation; transition overlays adapt to per-page intro animation rather than in-app route swapping.
- CVS convenience-store list ships as static client data in the theme (ported from the reference data file), not a database table.
- Notification template administration and full email lifecycle are deferred; only order-confirmation mail via the existing mailer seam is in scope.
