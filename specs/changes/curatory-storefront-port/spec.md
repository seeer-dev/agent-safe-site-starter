# Curatory Storefront Port Specification

Change ID: curatory-storefront-port
Revision: 2
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Owner direction 2026-09-14: storefront ports the 質選所 (CURATORY) design and dynamic effects into the Go template + Vue islands architecture; the existing starter Vue admin SPA remains the backoffice. Owner approved revision 2 via plain 'apply' on 2026-09-14, additionally granting latitude to redesign admin UX for intuitiveness (functional completeness is the bar).
Repository baseline: 10f805e18e394e7065854dfbceb327114e0d4564
Supersedes: none

## Outcome

The public storefront becomes a COMPLETE port of the 質選所 (CURATORY) design — full visual fidelity (layout, typography, color tokens, spacing, components, imagery) and full interaction fidelity (every control, state, and motion) — delivered through this repository's architecture: Go-rendered static pages plus Vue islands under a new `site/themes/curatory/` theme, backed by the single Go API. The backoffice keeps the existing starter Vue admin SPA shell and design system while porting the reference admin's complete management surface, closing the loop so every storefront behavior is staff-administrable. The reference implementation `D:\MyTools\AI設計版面\購物車系列\質選所` is a design and behavior source only; its Next.js runtime, API routes, Prisma schema, and database are NOT adopted.

## Scope

In scope:

- New theme `site/themes/curatory/` with its own templates, islands, shared stores/composables/styles, build scripts, and checks.
- Storefront pages ported as real static pages (no client-side router): home, shop (category/search/sort), product detail, news list, news detail, about, cart, checkout, order confirmation, order tracking.
- Complete style port: design tokens, typography, spacing, component styles, and page compositions matching the reference storefront.
- Complete interaction port: brand intro loader, page-load transitions, scroll reveal, image fade-in, loading skeletons, toast notifications, persistent variant-aware cart, cart drawer/page behaviors, checkout field logic, all hover/focus/disabled/empty/error states the reference exposes.
- Backend extensions needed by the port: category entities, product variants, product comments with moderation, coupon extension and public validation, order field extensions, storefront bootstrap, public product filtering, notification templates and logs, admin stats.
- Admin SPA port of the reference backoffice surface: dashboard statistics, categories, products (variants + images), orders and order detail, coupons, announcements/news, comments moderation, payment and shipping methods, notification templates and logs, store settings — using the existing admin design system and capability gating.
- Paired SQLite and PostgreSQL migrations; OpenAPI contract updates.

Out of scope:

- The 質選所 Next.js server, its API routes, its Prisma database, and its built-in admin runtime. They are reference material only.
- A second backend, SSR runtime, or Pages Function.
- A pixel-level port of the reference admin's React/shadcn visual system; the admin keeps the existing SPA design system and ports functionality and information architecture.
- Member/login storefront features beyond the existing account model.
- New payment-provider integration (existing ECPay flow stays as-is; payment methods remain selection/display).
- LINE Pay / OAuth storefront additions.

## Decisions recorded in this spec

- Guest order retrieval keeps the existing opaque `X-Order-Access-Token` model. The tracking page accepts an order id/number plus the access token issued at checkout (stored in browser session storage by the confirmation page). Order-number+email enumeration lookup is NOT introduced.
- Announcements/news reuse the `content` module's article pipeline (publish workflow, rendered pages) extended with pinned and publish-at fields rather than a parallel module.
- Store settings (store name, tagline, promo banner, thresholds, contact) live in the `sitecontent` module as a single-row settings record with the existing approve/publish governance, not a new module.
- Coupons extend the existing `promos` model (`percent|fixed|freeshipping`, `min_subtotal`, `usage_limit`, `used_count`) instead of a second promotion table.
- The storefront bootstrap endpoint is a composite owned by the `bootstrap` layer calling module services — module isolation is preserved (modules never import each other).
- Order lifecycle notifications (placed/paid/shipped/completed/cancelled) render from `notification_templates` and send through the existing mailer seam; every attempt writes a `notification_logs` row so the admin notifications view reflects real deliveries.
- The admin port keeps the starter SPA's technology and resource architecture; it ports the reference backoffice's functionality, information architecture, and workflows — not its component skins. The owner grants latitude to redesign admin UX for intuitiveness; functional completeness is the bar, not visual parity with either the reference or the current admin.
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

### REQ-003: Complete visual and interaction fidelity

The port SHALL reproduce the reference storefront's complete visual design — layout, typography, color tokens, spacing, component styling, imagery treatment — and its complete interaction set: brand intro loader, page-load transitions, scroll reveal, image fade-in, skeleton placeholders during island fetch, toast notifications, persistent variant-aware cart, cart drawer/page behaviors, checkout field logic, and every hover/focus/disabled/empty/error state the reference exposes. Reduced-motion preferences MUST be respected.

#### AC-004: Visual fidelity verified page by page
- GIVEN the rendered site and the reference
- WHEN a per-page walkthrough compares layout, typography, color, spacing, and component styling
- THEN each ported page matches the reference design, with any intentional deviations listed and approved in evidence.

#### AC-005: Interaction and motion inventory complete
- GIVEN the rendered site
- WHEN a walkthrough exercises first visit, navigation, image loads, island fetches, add-to-cart, checkout field flows, and form errors
- THEN the intro loader, page transitions, reveals, fades, skeletons, toasts, cart persistence, and all reference control states are observed; reduced-motion preferences are respected.

### REQ-004: Catalog depth

Products SHALL support multiple verified images, variants (name, price delta, stock, sku), a featured flag, sold count, and category entities (slug, name, image, sort order, active flag) managed by staff — replacing the free-text category field for new data while remaining backward compatible with existing seeded rows.

#### AC-006: Variant selection drives price and stock
- GIVEN a product with two variants of different price deltas and stocks
- WHEN a visitor selects a variant on the product page
- THEN displayed price, stock, and add-to-cart reflect the selected variant, and quote/order pricing uses the server-authoritative variant price.

#### AC-007: Catalog browsing matches the reference
- GIVEN seeded products across categories
- WHEN the shop page is filtered by category, searched by keyword, and sorted by newest/price
- THEN results, ordering, and featured merchandising on home match the server data.

### REQ-005: Checkout parity

Checkout SHALL collect: recipient name/phone, city/district/address for home delivery, CVS store selection for CVS shipping methods, invoice type with tax id when company invoice is chosen, an optional coupon code, buyer note, and a different billing/contact identity where the reference collects one. Totals MUST come from the server quote.

#### AC-008: Checkout validation and totals are server-authoritative
- GIVEN a cart and shipping/payment selections
- WHEN the quote and order are submitted
- THEN server-computed subtotal, discount, shipping fee (including free-shipping thresholds and CVS/home fees), payment fee, and total are persisted; tampered client totals are ignored.

#### AC-009: Taiwan address and CVS flows
- GIVEN the checkout page
- WHEN home delivery is chosen, city/district selection constrains the address; when CVS is chosen, a store must be selected before submission
- THEN the order stores the structured address or CVS store name/address, invoice fields, and buyer note.

### REQ-006: Order surface

Orders SHALL expose a public order number, an opaque guest access token returned only at creation, a status timeline, and an order-confirmation page. Order lifecycle events (placed/paid/shipped/completed/cancelled) SHALL emit notifications rendered from templates through the configured mailer (log sender locally), each attempt recorded in notification logs.

#### AC-010: Guest order access stays capability-bound
- GIVEN a created order
- WHEN it is fetched without the access token or with a wrong token
- THEN it fails closed; with the token it returns the masked-PII order. No order-number+email enumeration endpoint exists.

#### AC-011: Confirmation, tracking, and notifications reflect real state
- GIVEN a completed checkout and subsequent staff status changes
- WHEN the confirmation/tracking pages load and status changes occur
- THEN they render items, totals, timeline, and status from the API; each lifecycle event produces a notification attempt and a log row; failure states render honestly without fixture data.

### REQ-007: News from governed content

News list and detail pages SHALL render published announcements/articles through the existing publish workflow, honoring pinned ordering and publish-at timing; drafts, stale, and future-dated items never render.

#### AC-012: News visibility follows publish state
- GIVEN draft, published, pinned, and future-dated items
- WHEN news pages render
- THEN only currently published items appear, pinned first; removing publish state removes the item after re-render.

### REQ-008: Product comments with moderation

Visitors SHALL be able to submit a product comment (nickname, content, optional 1–5 rating). New comments are pending, are not publicly rendered, and become visible only after staff approval; rejected comments never render. Staff may also record an admin reply shown with the comment.

#### AC-013: Comments fail closed until approved
- GIVEN a submitted comment
- WHEN the public product page is rendered before approval
- THEN the comment is absent; after approval it appears with rating and admin reply. Approval state is the only path to visibility.

### REQ-009: Coupons

Coupons SHALL support percent, fixed-amount, and free-shipping types with minimum subtotal, usage limit, validity window, and a public validate endpoint that reports validity, discount, free-shipping effect, and a user-facing message. Quote and order totals MUST apply the same validated result; exhausted, expired, inactive, or below-minimum codes fail closed.

#### AC-014: Coupon lifecycle is enforced
- GIVEN active, expired, exhausted, inactive, and below-minimum codes
- WHEN validated and applied to quote/order
- THEN valid codes produce the exact computed effect and all invalid cases are rejected with a message; `used_count` increments only on accepted orders.

### REQ-010: Admin port closes the loop

The admin SPA SHALL port the reference backoffice's complete management surface through capability-gated resources rendered in the existing admin design system: dashboard statistics, categories, products (variants and image association), orders and order detail, coupons, announcements/news, comments moderation, payment and shipping methods, notification templates, notification logs, and store settings.

#### AC-015: Every storefront-governing entity is administrable
- GIVEN staff sessions with and without the relevant capabilities
- WHEN each ported admin screen is opened
- THEN permitted staff see and edit real data for all listed entities; denied or unauthenticated access fails closed.

#### AC-016: Dashboard shows real statistics
- GIVEN seeded orders and products
- WHEN the admin dashboard loads
- THEN status counts, revenue figures, low-stock list, and recent activity come from a real stats endpoint — no fixture data.

### REQ-011: Storefront bootstrap

A single public endpoint SHALL return store settings (names, tagline, promo banner, thresholds, contact), active categories, enabled shipping and payment methods, and current announcements summary for shared chrome and the intro loader. It MUST contain no PII and no admin-only data.

#### AC-017: Bootstrap feeds shared chrome
- GIVEN the bootstrap endpoint
- WHEN any storefront page loads
- THEN header/footer/promo banner render from it and unpublished or disabled items never leak.

### REQ-012: Data plane and contract parity

Every schema change SHALL ship paired SQLite and PostgreSQL migrations with matching results; `contracts/openapi.yaml` SHALL document every new/changed endpoint; `go run ./server/tools/verify` and contract checks MUST pass.

#### AC-018: Both drivers reach the same schema and behavior
- GIVEN the migrations
- WHEN applied to SQLite and PostgreSQL
- THEN the same tables/fields exist and the affected API flows behave identically.

#### AC-019: Gates pass
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

- 質選所's SPA hash-router becomes real page navigation; route-transition overlays adapt to per-page intro/load animation (cross-document transitions where the platform supports them).
- CVS convenience-store list ships as static client data in the theme (ported from the reference data file), not a database table.
- The reference admin's React/shadcn skins are not ported; functionality and information architecture are.
