# Curatory Storefront Port Evidence

Change ID: curatory-storefront-port
Revision: 2
Status: Applying

Implementation in progress. Slices S01 (data plane), S02 (API surface),
S03 (theme foundation), S04 (home/shop/product pages) are implemented;
S05 (cart/checkout/order/track) is in progress; news/about/chrome
completion, admin, and final verification slices remain.

## Implementation progress

- **S01 data plane (done)**: `db/migrations/{sqlite,postgres}/018_curatory_storefront.sql`
  adds categories, product_variants, product_comments, notification_templates,
  notification_logs, store_settings; extends products (is_featured, sold_count),
  promos (min_subtotal, usage_limit, used_count, type=freeshipping), orders
  (recipient/city/district/CVS/invoice/payment_fee/coupon_code/buyer_note),
  payment_methods (fee), articles (pinned, publish_at_unix). SQLite migrate
  verified: `go run ./server/tools/migrate` → `migrations applied (sqlite)`.
- **S02 API (done, smoke-tested)**: routes wired in `bootstrap/app.go` —
  `GET /api/storefront/bootstrap`, `GET /api/categories`, product comments
  GET/POST, `POST /api/coupons/validate`, admin CRUD for categories/comments/
  notification-templates/notification-logs/store-settings/articles/stats.
  Product list accepts `?category=&featured=&q=&sort=`. Checkout resolves
  variant SKUs server-side (verified: CUP-001-B → price 930 = 880+50),
  promo percent discount, shipping, and payment fee all server-computed
  (quote total 1860-186+80+15=1769). Order persists recipient/invoice/CVS/
  coupon/buyer-note fields; `order_placed` notification logged (skipped —
  template not found, as designed). Admin stats returns real counters.
- `contracts/openapi.yaml` updated: new paths + schemas (Category, Variant,
  ProductComment, Promo, NotificationTemplate/Log, StoreSettings, AdminStats);
  Order/OrderInput/QuoteResult/Product extended. `check-openapi-contracts.mjs`
  PASSED.
- Test fixes: fake store ListAll, TransitionOrderStatus soldOut arg,
  pre-014 seed ordering, seed principal capability. `go test ./server/...`
  all green after minimal-cart theme dist build.
- **S03 theme foundation (done)**: `site/themes/curatory/` — Vite + Vue 3 +
  Tailwind v4 + tw-animate-css; self-hosted Noto Sans/Serif TC + Cormorant
  Garamond; ported design tokens (cream/ink/gold, dark mode) in
  `shared/styles/globals.css`; island bootstrap (`data-vue-island`,
  `data-prop-*`, lazy glob import, reveal directive, cart rehydrate,
  transition interceptor); `theme-init.ts` (pre-paint dark mode +
  curtain-reveal, external file because CSP forbids inline scripts);
  shared lib (types/api/cart/store/format/theme/transition/toast/utils);
  shared components (FadeImage, PriceDisplay, QuantityStepper, logos,
  SectionHeading, RichParagraphs, Skeleton, Sheet, Dialog, EmptyState,
  ProductCard); chrome islands (PageLoader, AnnouncementBar, SiteHeader
  with glass scroll/category dropdown/mobile drawer/search dialog,
  SiteFooter, CartDrawer with free-shipping progress, Toaster); Go
  renderer extended (Input/StaticPage/theme asset validation+copy/
  chromeFields/atomic staging); rendercompose fills Settings/CategoryList/
  Featured/NewsPreview/Pages/ProductCount. `npm run typecheck` +
  `npm run build` PASS; `go run ./server/tools/render` emits 7 static
  pages + product/category/news routes with island mounts verified in
  generated `dist/`.
- **S04 in progress**: ShopGrid island (sticky control bar, category chips
  as real links, client-side search/sort with `history.replaceState`,
  skeleton/error/empty states, no-JS baseline takeover), ProductDetail
  island (gallery, variant picker with price deltas + sold-out disabling,
  quantity stepper, add-to-cart/buy-now, low-stock and shipping/payment/
  returns accordions fed by bootstrap), ProductComments island (rating
  summary card with 5→1 distribution bars, comment cards with admin
  reply, submission form with star picker + pending-moderation feedback).
  `ListProductComments` now returns `{comments, summary{count,avg,dist}}`
  computed server-side. Local dev pinned to curatory via
  `.env.development.local` `SITE_THEME=curatory`.
- **Contract fix during S04**: `PublicShippingMethod` briefly exposed
  `fee`/`free_threshold`; reverted — the checked-in contract deliberately
  never includes fees publicly (`shipping_methods_test.go` asserts it);
  storefront shows the global `settings.freeShippingThreshold` instead
  and `/api/quote` remains the authoritative fee source.
- **S05 in progress**: CartPage island (item rows with stepper/remove,
  clear-cart dialog, subtotal, free-shipping progress from
  `settings.freeShippingThreshold`, coupon validate via
  `POST /api/coupons/validate` now returning `{coupon, discount,
  free_shipping}` via new `Service.PreviewDiscount`, estimated total,
  checkout CTA). CheckoutPage island — 4-step wizard (訂購資料 → 配送 →
  付款 → 確認) ported from reference checkout-view: same-recipient toggle,
  Taiwan city/district/ZIP cascade + CVS store picker dialog (static data
  ported to `shared/data/`), shipping-method radio cards (fee shown only
  after `/api/quote`), payment methods with credit-card sandbox visual
  form (never transmitted — strict JSON decode), LINE Pay/COD explainer
  cards, company-invoice fields, terms checkbox, per-step validation,
  processing overlay, idempotency key via `crypto.randomUUID()`, flat
  `OrderInput` payload, success → `rememberOrder` + navigate to
  `/order/?id=`. Quote watcher recomputes authoritative totals whenever
  items/methods/coupon change. OrderDetailCard shared component (status
  stepper pending→processing→shipped→delivered→completed, COD-pending
  badge variant, cancelled banner, items+totals, shipping/payment/invoice
  cards, timeline, CTA). OrderPage (`?id=` + token from sessionStorage/
  recent-orders/manual 查詢碼, success vs detail header) and TrackPage
  (order-id + token form, recent-orders chips, skeleton/error/empty
  states) — guest access via `GET /api/orders/{id}` +
  `X-Order-Access-Token` (no order-number+email enumeration endpoint,
  per spec decision). `npm run typecheck` + `npm run build` PASS;
  `go run ./server/tools/render` emits /cart /checkout /order /track
  with correct island mounts in generated `dist/`.
  Smoke-tested live: quote (930×2 +80 ship +15 pay fee = 1955) → order
  created (TW-…, access_token, timeline) → guest GET 400 w/o token /
  200 PII-masked w/ token.
- **S06 (done)**: home/about/news/article templates already ported in
  S03–S04 verified against reference sections (hero+stats+float card,
  value props, featured/categories/story/latest/news-preview/newsletter
  on home; story+values+stats+CTA on about; pinned-first news list).
  Added missing CSS: `.animate-float` (hero mini-card 4.2s float),
  `.prose-curatory` (article body_html tag styling). article.html
  fidelity fix: 公告/置頂 badges + calendar icon + dual footer CTAs
  (回公告列表 / 順道逛逛商店). Newsletter form + scroll-top wired via
  delegated listeners in islands/bootstrap.ts; content store orders
  articles pinned-DESC then publish_at DESC.
- **S07 in progress**: admin SPA wired for the new resources — resource
  registry registers categories/comments/notification-templates/
  notification-logs/articles; nav entries added in `profile.ts` with
  icons (Sidebar ICON_MAP extended); `FieldWidget` gained `variants`
  rendered by new `VariantsEditor` sub-table (name/sku/price_delta/
  stock/sort_order, add/remove, read-only mode, aria-labelledby);
  `FieldDef.optsSource` loads select options from an admin list endpoint
  (product category → /admin/categories); `RowAction` gained
  `reasonField`/`reasonOptional`/`reasonLabel` so comment moderation
  sends `{status, reply}` under the correct key; `ConfirmDialog` accepts
  optional reasons. Existing resources extended: products (is_featured,
  long_description, sold_count col, dynamic category select, variants
  section), promos (freeshipping type, min_subtotal, usage_limit,
  used_count), payment-methods (fee field + col), orders (completed
  transition button + recipient/CVS/invoice/coupon/fee detail fields).
  New `StoreSettingsPage` at `/settings` edits the governed
  store_settings draft (version-checked save, publish action) via
  sitecontent endpoints; backend gained `PUT /api/admin/articles` as an
  alias of the slug-keyed upsert so the generic edit form works.
  `npm run typecheck` + `npm run build` + `npm test` (195 tests) PASS.
- **S07 fix**: sqlite TEXT columns cannot scan into `json.RawMessage`
  under modernc — `sitecontent.GetStoreSettings` now scans into string
  first. Store-settings round trip verified live: GET row → PUT draft
  (version bump) → POST publish → published JSON appears in
  `/api/storefront/bootstrap`.
- **S08 seed (done)**: `server/tools/dev` gained theme-aware seeds —
  `seedProductsForTheme`/`seedPaymentMethodsForTheme` preserve the
  minimal-cart baseline while curatory gets 8 products (3 featured, 2
  with variants, 1 draft, 1 sold-out), 4 payment methods (LINE Pay +
  fees), and `seedCuratory` fills categories (5), shipping methods
  (宅配/7-11/全家 with fee+free-threshold), promos (WELCOME10 percent +
  FREESHIP freeshipping w/ min_subtotal), 3 published news articles
  (1 pinned), 4 comments (3 approved incl. admin reply, 1 pending),
  5 lifecycle notification templates, and published store settings
  (質選所 / NT$1,500 免運門檻 / announcement banner). Each block is
  independently idempotent. Fresh-DB dev run seeded all of it and
  rendered 3 articles + 7 published products + 5 categories + 7 static
  pages; island mounts verified in dist (/products/stoneware-mug/
  mounts ProductDetail + ProductComments).

## Evidence log

| REQ/AC | Status | Proof |
|---|---|---|
| REQ-001 | pending |  |
| REQ-002 | pending |  |
| REQ-003 | pending |  |
| REQ-004 | pending |  |
| REQ-005 | pending |  |
| REQ-006 | pending |  |
| REQ-007 | pending |  |
| REQ-008 | pending |  |
| REQ-009 | pending |  |
| REQ-010 | pending |  |
| REQ-011 | pending |  |
| REQ-012 | pending |  |
| AC-001 | pending |  |
| AC-002 | pending |  |
| AC-003 | pending |  |
| AC-004 | pending |  |
| AC-005 | pending |  |
| AC-006 | pending |  |
| AC-007 | pending |  |
| AC-008 | pending |  |
| AC-009 | pending |  |
| AC-010 | pending |  |
| AC-011 | pending |  |
| AC-012 | pending |  |
| AC-013 | pending |  |
| AC-014 | pending |  |
| AC-015 | pending |  |
| AC-016 | pending |  |
| AC-017 | pending |  |
| AC-018 | pending |  |
| AC-019 | pending |  |

## Gaps

- None recorded yet.
