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
| REQ-001 | done | SITE_THEME=curatory render emits the ported theme; minimal-cart path untouched |
| REQ-002 | done | 7 static pages + product/category/news routes in dist/; no-JS baselines present |
| REQ-003 | done | tokens/components/interactions ported from reference (see S03–S06 notes) |
| REQ-004 | done | categories + variants + featured + sold_count live (S01/S02 + seed) |
| REQ-005 | done | CheckoutPage wizard + server-authoritative quote/order (smoke-verified) |
| REQ-006 | done | TW- order number + access token + timeline + template notifications |
| REQ-007 | done | news renders governed articles, pinned-first (verified in dist/news) |
| REQ-008 | done | comments pending→approved moderation; admin reply supported |
| REQ-009 | done | percent/fixed/freeshipping + min_subtotal + usage_limit + validate endpoint |
| REQ-010 | done | admin resources cover all storefront entities (S07) |
| REQ-011 | done | /api/storefront/bootstrap verified live with seeded settings |
| REQ-012 | done | paired migrations (18 matching) + openapi.yaml updated + verify pass |
| AC-001 | pass | `go run ./server/tools/render` → dist/ with curatory pages; SITE_THEME=minimal-cart still renders original theme |
| AC-002 | pass | dist/*.html contain static nav/content/footer with islands mounted via data-vue-island (verified mounts on /, /products/stoneware-mug/) |
| AC-003 | pass | product/news/category/order deep links are pre-rendered pages; order requires token as specced |
| AC-004 | pass | section-by-section comparison against reference during S03–S06; deviations: SPA→MPA navigation, added no-JS baselines, search upgraded to live results |
| AC-005 | pass | loader/transitions/reveal/fade-in/skeleton/toast/cart-persistence ported; prefers-reduced-motion handled in globals.css and bootstrap.ts |
| AC-006 | pass | smoke: variant CUR-TEA-01-B → 930 = 880+50; order line resolves variant sku server-side |
| AC-007 | pass | ?category/featured/q/sort filters verified; shop page chips/sort/search functional |
| AC-008 | pass | quote 930×2 −186 +80 +15 = 1769; order persisted server totals (smoke) |
| AC-009 | pass | city/district cascade + CVS picker; order stores city/district/cvs/invoice/note fields (smoke order) |
| AC-010 | pass | guest GET /api/orders/{id}: 400 without token, 200 PII-masked with token (smoke) |
| AC-011 | pass | seeded-templates order → notification-logs row status=sent with rendered subject/body; order/track pages render real timeline |
| AC-012 | pass | news list pinned-first then published_at desc; only published articles render (3 seeded, 1 pinned observed first) |
| AC-013 | pass | seeded pending comment absent from public comments API; approved ones render with reply |
| AC-014 | pass | WELCOME10/FREESHIP validate + quote integration verified; used_count increments in order tx |
| AC-015 | pass | admin screens for categories/products+variants/orders/promos/articles/comments/payment/shipping/notifications/store-settings exist and are capability-gated |
| AC-016 | pass | Dashboard consumes /admin/stats (revenue, pending_comments) + orders/products/comments lists; DashboardPage test asserts 6 KPI values |
| AC-017 | pass | bootstrap returns settings/categories/methods/announcements; unpublished settings excluded (draft/publish isolation verified) |
| AC-018 | partial | migration-parity gate passes (18 paired); live PostgreSQL apply not exercised in this environment |
| AC-019 | pass | speccheck ok (32 specs); verify chain green incl. go test -count=10 on commerce/staff/media |

## Gaps

- AC-018: live PostgreSQL migration apply + behavior parity not exercised
  locally (no PG instance in this environment); schema parity is enforced
  by the migration-parity gate.
- AC-004/AC-005: comparison was performed section-by-section against the
  reference source and rendered output; no automated visual-diff tooling
  exists in this repo. A human pass over the running site vs the
  reference is recommended for pixel-level sign-off.
