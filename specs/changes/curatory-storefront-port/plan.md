# Curatory Storefront Port Plan

Change ID: curatory-storefront-port
Revision: 2
Status: Applying

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `specs/changes/curatory-storefront-port/**`
- `site/themes/curatory/**`
- `site/assets/images/**`
- `server/internal/modules/commerce/**`
- `server/internal/modules/content/**`
- `server/internal/modules/sitecontent/**`
- `server/internal/render/**`
- `server/internal/bootstrap/**`
- `server/internal/config/**`
- `server/tools/render/**`
- `server/tools/dev/**`
- `server/tools/seed/**`
- `server/tools/internal/rendercompose/**`
- `contracts/**`
- `db/migrations/**`
- `admin/src/**`
- `docs/**`

Source of truth for design/behavior: `D:\MyTools\AI設計版面\購物車系列\質選所` (reference only — Next.js runtime, API routes, Prisma schema, and its database are never imported or executed as part of the product).

## Implementation expansion context

Proposal revision: 2
Proposal status: Draft
Repository baseline: `10f805e18e394e7065854dfbceb327114e0d4564` (main HEAD on proposal branch creation, 2026-09-14)
Pre-existing dirty paths: none.
Packet ID mapping: Slice N -> Packet S0N.

## Slices (dependency order)

### S01 — Data plane: schema + domain

Covers: REQ-004, REQ-006, REQ-008, REQ-009, REQ-011, REQ-012 (partial)

- Paired migrations (sqlite + postgres): `categories`, `product_variants`, `product_comments`, `promos` extension (`freeshipping` type, `min_subtotal`, `usage_limit`, `used_count`), `orders` extension (`recipient_name`, `recipient_phone`, `city`, `district`, `cvs_store_name`, `cvs_store_address`, `invoice_type`, `invoice_tax_id`, `payment_fee`, `coupon_code`, `buyer_note`, `completed` status), `notification_templates`, `notification_logs`, `store_settings` single-row record in `sitecontent`.
- Commerce model/service/store updates: variants, categories, featured flag, sold count, comment store surface, coupon validation + order application, order field persistence, notification emit on order lifecycle events.
- Sitecontent: settings record with approve/publish.
- Content: articles extended with `pinned`, `publish_at`.

Expected evidence: migration apply on both drivers; unit tests for pricing/validation.

### S02 — Public + admin API surface

Covers: REQ-004, REQ-005, REQ-006, REQ-007, REQ-008, REQ-009, REQ-010, REQ-011

- `GET /api/storefront/bootstrap` — bootstrap-layer composite: settings + categories + shipping + payment + announcements summary.
- Product list filters (`featured`, `sort`, `category`, `q`) and product detail including variants/images.
- `POST /api/coupons/validate`.
- `GET/POST /api/products/{slug}/comments` (public list approved; submit → pending).
- News endpoints honoring pinned/publish-at (reusing the content pipeline).
- Order create extension + guest access-token fetch (existing) + lifecycle notifications.
- Admin endpoints: stats aggregate, categories CRUD, comments moderation (approve/reject/reply), notification templates CRUD, notification logs list, settings read/update.
- `contracts/openapi.yaml` updated for every endpoint.

Expected evidence: API tests for validation, masking, coupon edge cases, stats; contract check.

### S03 — Theme scaffold `site/themes/curatory/`

Covers: REQ-001, REQ-003 (partial)

- Vite/Tailwind config, design tokens + global styles ported from the reference (colors, typography, radius, shadows, motion durations).
- `bootstrap.ts` island mounter (same pattern as minimal-cart), shared `api` client, `cart` store (variant-aware, localStorage `curatory-cart`), `format` helpers.
- Shared components: `InitialLoader`, page-intro transition, `ScrollReveal`, `FadeImage`, `Skeleton`, `Toaster`.
- `templates/` skeleton pages + renderer registration for new page kinds.
- Theme check scripts patterned on minimal-cart.

Expected evidence: theme builds; render produces curatory pages.

### S04 — Storefront pages I: home, shop, product

Covers: REQ-002, REQ-003, REQ-004, REQ-008, REQ-011

- `home.html` + islands: Header, PromoBanner, Hero, FeaturedProducts, ProductGrid, Footer — full visual parity.
- `shop.html` with cat/q/sort params.
- `product.html` + islands: image gallery, variant selector, add-to-cart, comments list + submit form.
- Server-rendered baselines for SEO/no-JS; islands enhance only after authoritative load.

Expected evidence: rendered HTML inspection + visual-fidelity walkthrough receipt (AC-004).

### S05 — Storefront pages II: cart, checkout, order, track

Covers: REQ-002, REQ-003, REQ-005, REQ-006, REQ-009

- `cart.html` + cart island (line items, qty, coupon validate).
- `checkout.html` + checkout island: Taiwan city/district cascade, CVS store picker (static ported store data), invoice type/tax id, recipient fields, coupon, note; server quote display; complete field logic from the reference.
- `order.html` confirmation (access token from session storage), `track.html` (token-based retrieval).
- Ported effects on these flows: skeletons, toasts, transitions.

Expected evidence: end-to-end walkthrough receipt (cart → checkout → order → track), quote/order parity test.

### S06 — Storefront pages III: news + about + chrome completion

Covers: REQ-002, REQ-007, REQ-011

- `news.html`, `news-detail.html` (published-only, pinned ordering), `about.html`.
- Footer pages/dialog parity where the reference exposes policy content.

Expected evidence: render inspection; publish-state AC replay.

### S07 — Admin SPA port (closed loop)

Covers: REQ-010

- `admin/src/config/resources/`: categories, announcements/news, coupons, comments (moderation actions), notification templates, notification logs, settings; product form variant editor.
- Dashboard stats view wired to the stats endpoint.
- Route/nav registration mirroring the reference backoffice information architecture; capability gating wired to `/api/admin/me`.
- Existing admin design system reused; no pixel port of the reference React/shadcn skins.

Expected evidence: admin walkthrough receipt per resource (permitted + denied), stats consumer-reachability.

### S08 — Content seed + end-to-end verification

Covers: REQ-001–REQ-012 closure

- Dev seed extended: categories, variants, featured, announcements, coupons, settings, notification templates matching reference demo data.
- Full render + storefront walkthrough suite; targeted tests; `speccheck`, `verify`; `dist/` inspection.

Expected evidence: all AC receipts current at final revision.

## Traceability

| REQ/AC | Slice | Surface or consumer | Verification |
|---|---|---|---|
| REQ-001 / AC-001 | S03, S08 | render tool, dist/ | render + theme-switch render |
| REQ-002 / AC-002, AC-003 | S03–S06 | static pages | HTML inspection, walkthrough |
| REQ-003 / AC-004, AC-005 | S03–S06 | shared components, all pages | walkthrough receipts |
| REQ-004 / AC-006, AC-007 | S01, S02, S04 | variants, categories, filters | API tests + UI walkthrough |
| REQ-005 / AC-008, AC-009 | S01, S02, S05 | quote/order/checkout | API tests + walkthrough |
| REQ-006 / AC-010, AC-011 | S01, S02, S05 | orders, notifications | security-review + walkthrough |
| REQ-007 / AC-012 | S01, S02, S06 | news pages | render inspection |
| REQ-008 / AC-013 | S01, S02, S04, S07 | comments | security-review + moderation walkthrough |
| REQ-009 / AC-014 | S01, S02, S05, S07 | coupons | API edge-case tests |
| REQ-010 / AC-015, AC-016 | S02, S07 | admin SPA | admin walkthrough + consumer-reachability |
| REQ-011 / AC-017 | S01, S02, S03 | bootstrap endpoint | consumer-reachability receipt |
| REQ-012 / AC-018, AC-019 | S01, S08 | migrations, gates | migration-parity, speccheck, verify |

## Stop conditions

- Any discovery requiring new trust-boundary decisions (e.g., guest order lookup semantics, comment spam controls beyond moderation) returns to the owner before implementation continues.
- A slice that cannot reach its ACs without out-of-scope edits stops and reports instead of improvising.
