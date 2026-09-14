# Pre-Launch SEO + Reliability Hardening

Change ID: prelaunch-seo-reliability
Revision: 1
Status: Verifying
Decision authority: Repository owner/user
Approval basis: Owner direction 2026-09-14: approved the pre-launch hardening batch (SEO files + social meta + product JSON-LD + async notifications with admin retry). Owner explicitly deferred public-endpoint rate limiting pending the deployment-topology decision recorded in specs/changes/public-endpoint-rate-limit, and excluded Turnstile.
Repository baseline: 10f805e18e394e7065854dfbceb327114e0d4564
Supersedes: none

## Outcome

Ship the storefront's missing launch surfaces: crawler discovery files,
social-sharing metadata, product structured data, and a notification
pipeline that neither blocks checkout nor silently drops mail failures.

## Scope

- Go renderer emits `robots.txt` + `sitemap.xml` into `dist/` during the
  staging pass (atomic with the rest of the site).
- `doc-head` emits Open Graph + Twitter card meta on every page;
  product pages use the product image and `og:type=product`.
- Product pages embed a `Product` JSON-LD block.
- Order/lifecycle notification sends move off the request path;
  admin gains a retry action on failed/skipped notification logs.

Out of scope: rate limiting, Turnstile, sitemap index/sharding,
hreflang, RSS.

## Requirements

### REQ-001: Crawler Discovery Files
The renderer MUST emit `robots.txt` and `sitemap.xml` at the site root
during every successful staging render. The sitemap MUST list every
indexable public route (home, shop, category pages, product pages,
news list + articles, about, content pages) with absolute URLs derived
from `PublicSiteURL`. Transactional/tooling routes (cart, checkout,
track, order) MUST be excluded from the sitemap and disallowed in
robots.txt.

#### AC-001: Sitemap Covers All Indexable Routes
- GIVEN a full render with seeded catalog data
- WHEN `dist/sitemap.xml` is inspected
- THEN it contains absolute `<loc>` entries for `/`, `/shop/`,
  every `/categories/{slug}/`, every `/products/{slug}/`, `/news/`,
  every `/news/{slug}/`, `/about/`, and every `/content/{key}/`
- AND it does NOT contain `/cart/`, `/checkout/`, `/track/`, `/order/`.

#### AC-002: Robots Directs Crawlers
- GIVEN a full render
- WHEN `dist/robots.txt` is inspected
- THEN it disallows the transactional routes and references the
  sitemap absolute URL.

### REQ-002: Social Sharing Metadata
Every rendered page MUST emit `og:type`, `og:site_name`, `og:title`,
`og:description`, `og:url` (when `PublicSiteURL` is configured), and
`twitter:card`. Product pages MUST use `og:type=product` and the
product's primary image as `og:image` (absolutized against
`PublicSiteURL` when relative); other pages fall back to the site
default image.

#### AC-003: OG/Twitter Tags Present and Correct
- GIVEN rendered home and product pages in dist/
- WHEN their `<head>` is inspected
- THEN og:* and twitter:* tags are present; the product page uses
  `og:type=product` and an absolute `og:image` of its own photo.

### REQ-003: Product Structured Data
Each product page MUST embed a valid schema.org `Product` JSON-LD block
containing name, description, image, brand, and an `offers` node with
`priceCurrency=TWD`, numeric price, and availability reflecting stock.

#### AC-004: JSON-LD in Rendered HTML
- GIVEN a rendered product page in dist/
- WHEN its `<script type="application/ld+json">` is parsed
- THEN it yields valid JSON with `@type=Product`, correct name/price,
  and `InStock`/`OutOfStock` matching the product's stock.

### REQ-004: Notification Resilience
Order creation and status transitions MUST NOT block on mail delivery.
Failed or skipped notification attempts MUST be retryable from the
admin without operator access to the server.

#### AC-005: Checkout Does Not Wait on Mail
- GIVEN a notifier whose Send blocks measurably
- WHEN an order is placed
- THEN the order commit returns without waiting for the send to
  complete, and the attempt still lands in notification_logs.

#### AC-006: Admin Retry Re-sends and Records
- GIVEN a notification_logs row with status failed or skipped
- WHEN `POST /api/admin/notification-logs/{id}/retry` is called by an
  admin principal
- THEN the stored recipient/subject/body is re-sent through the
  configured sender and a NEW log row records the retry attempt
  (original row preserved).

#### AC-007: Gates Stay Green
- GIVEN the implementation
- WHEN `go run ./server/tools/verify` runs
- THEN archcheck, migration-parity, speccheck, scopecheck, go test
  (incl. -count=10 on commerce/staff/media), and vet all pass.
