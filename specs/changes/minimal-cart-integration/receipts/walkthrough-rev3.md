# Runtime Walkthrough Receipt (Revision 3)

Date: 2026-08-12
Change: minimal-cart-integration
Revision: 3
Auditor: Independent review (Devin)

## Scope

Runtime verification of the five defects fixed in revision 3:
1. P0 shipping validation (server-side)
2. Browser authority (no local pricing constants)
3. Authority boundary (no unapproved 120/60/1500 fees)
4. Governance (strict_evidence, observed_revision binding)
5. Fixture residue (admin/dashboard.html gitignored)

## Environment

- OS: Windows (MSYS_NT-10.0-19045)
- Go server: `go run ./server/cmd/api` on port 18080
- Database: SQLite (`file:var/test-walkthrough.db`)
- Migrations: `go run ./server/tools/migrate` (sqlite, 6 migrations)
- Seed: `go run ./server/tools/seed` (articles + site content)
- Test product: inserted via SQL (SKU-APP-01, price 500, stock 10, active)

## Test results

### 1. GET /api/shipping-methods — server-authoritative shipping methods

```
Status: 200
Response: {"shipping_methods":[
  {"id":"home_delivery","label":"宅配","available":false,"description":"運費待確認"},
  {"id":"store_711","label":"7-11 超商取貨","available":false,"description":"運費待確認"},
  {"id":"store_familymart","label":"全家超商取貨","available":false,"description":"運費待確認"}
]}
```

All methods marked `available: false` — no fee schedule approved (GATE-004
scope only). Browser must use this endpoint, not hardcoded constants.

### 2. POST /api/quote — empty shipping method (P0 regression)

```
Request: {"items":[{"sku":"SKU-APP-01","quantity":1}],"shipping_method":"","promo_code":""}
Status: 400
Response: {"error":"invalid or unknown shipping method"}
```

Quote rejects empty shipping method with ErrInvalidShippingMethod. No
silent default to home_delivery.

### 3. POST /api/quote — unknown shipping method (P0 regression)

```
Request: {"items":[{"sku":"SKU-APP-01","quantity":1}],"shipping_method":"totally_invalid","promo_code":""}
Status: 400
Response: {"error":"invalid or unknown shipping method"}
```

Quote rejects unknown shipping method with ErrInvalidShippingMethod.

### 4. POST /api/quote — recognized method without approved fee schedule

```
Request: {"items":[{"sku":"SKU-APP-01","quantity":1}],"shipping_method":"home_delivery","promo_code":""}
Status: 503
Response: {"error":"shipping fee schedule not yet approved"}
```

Quote rejects recognized method with ErrShippingFeeScheduleNotApproved
when no fee schedule is approved (production default). Checkout fails
closed.

### 5. POST /api/orders — unknown shipping method (P0 regression)

```
Request: {...,"shipping_method":"totally_invalid",...}
Status: 400
Response: {"error":"invalid or unknown shipping method"}
```

CreateOrder rejects unknown shipping method. No order created.

### 6. POST /api/orders — recognized method without approved fee schedule

```
Request: {...,"shipping_method":"home_delivery",...}
Status: 503
Response: {"error":"shipping fee schedule not yet approved"}
```

CreateOrder rejects recognized method without approved fee schedule. No
order created, no stock decremented.

### 7. Browser authority source check

```
npm run check:browser-authority
→ Browser authority check PASSED: no pricing/shipping/payment constants in browser code.
```

Verified:
- `mock-data.ts` does NOT export FREE_SHIPPING_THRESHOLD, SHIPPING_FLAT_RATE,
  SHIPPING_METHODS, or PAYMENT_METHODS
- `cart.ts` does NOT compute subtotal, discount, shipping, tax, or total
- `CheckoutDialog.vue` fetches shipping/payment methods from server API
- `CartDrawer.vue` does not show local monetary totals or free-shipping progress

### 8. dist/ negative search

Ripgrep of `dist/` for fabricated data, unapproved claims, contact info,
demo credentials, and browser pricing constants: **0 matches**.

### 9. Governance verification

```
go run ./server/tools/speccheck
→ speccheck: ok (2 controlled spec(s), 319 protected changed file(s))
```

- `strict_evidence: true` enabled
- All passed evidence bound to `observed_revision: 3`
- AC-023 stale text fixed (no longer claims REQ-010/AC-012/AC-019 are pending)
- AC-024 fixed (no longer claims all gates resolved)
- `evidence.md` regenerated to match `control.json`
- `spec.md` approval_basis and revision updated to match `control.json`
- `production-content-audit` receipt provided for AC-019

### 10. Fixture residue

`admin/dashboard.html` is an obsolete design mockup:
- Not built (Vite uses `admin/index.html` as entry point)
- Not served (not referenced by any route or config)
- Not tracked in git (was untracked, now gitignored)
- Added to `.gitignore` to prevent accidental commit

## Automated verification summary

| Check | Result |
|-------|--------|
| `gofmt -l server/` | 0 files |
| `go test ./server/... -count=1` | PASS (all packages) |
| `go vet ./server/...` | clean |
| `go run ./server/tools/verify` | ok |
| `go run ./server/tools/speccheck` | ok |
| `admin npm run build` | PASS |
| `site npm run build:check` (typecheck + build) | PASS |
| `site npm run check:browser-authority` | PASS |
| `go run ./server/tools/render` | dist/ produced |
| dist/ negative search | 0 matches |

## Conclusion

All five defects are fixed and verified. The shipping fee schedule
(120/60/1500) is NOT approved — checkout fails closed until a product
decision is made. The browser is no longer the authority for pricing,
shipping, or payment availability. Governance is strict and evidence is
bound to the observed revision.
