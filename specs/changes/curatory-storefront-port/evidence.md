# Curatory Storefront Port Evidence

Change ID: curatory-storefront-port
Revision: 2
Status: Applying

Implementation in progress. Slices S01 (data plane) and S02 (API surface)
are implemented and smoke-tested; theme/admin/verification slices remain.

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
