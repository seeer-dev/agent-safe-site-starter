# Production Content Audit Receipt (Revision 3)

Date: 2026-08-12
Change: minimal-cart-integration
Revision: 3
Auditor: Independent review (Devin)

## Scope

Audit of the production bundle (`dist/`) and browser source code for
unapproved content claims, fabricated data, and browser-authority
constants that should not ship to customers.

## Method

1. `go run ./server/tools/render` produced a fresh `dist/`.
2. Negative grep of `dist/` for fabricated data, unapproved claims,
   contact info, demo credentials, and browser pricing constants.
3. Source-level audit of `site/themes/minimal-cart/shared/lib/mock-data.ts`
   for forbidden exports.
4. Source-level audit of `site/themes/minimal-cart/shared/stores/cart.ts`
   for local monetary computation.
5. `npm run check:browser-authority` source assertion script.

## Results

### Fabricated data (0 matches in dist/)
- MN-DEMO, WELCOME10, MINIMAL20, FREESHIP, WELCOME95: 0 matches
- Demo credentials (demo@, test@, admin@, password, secret): 0 matches
- Fabricated contact info (service@monolith, 02-1234-5678): 0 matches

### Unapproved claims (0 matches in dist/)
- 加密儲存, 嚴格挑選, 經久耐用, 不追隨潮流, 真實素材, 嚴選: 0 matches
- 工作天送達, 當日出貨, 隔日送達: 0 matches
- 生物可分解, 公平勞動, 成立於 2026, 嚴選台灣: 0 matches
- 滿 3000 免運, 離島請洽客服: 0 matches
- 累積專屬購物金, 結帳更快速, 享受專屬優惠, 保存收件資料: 0 matches

### Browser authority constants (removed in revision 3)
- `FREE_SHIPPING_THRESHOLD` (was 1500): removed from mock-data.ts
- `SHIPPING_FLAT_RATE` (was 120): removed from mock-data.ts
- `SHIPPING_METHODS` (had fee/freeThreshold/icon): removed from mock-data.ts
- `PAYMENT_METHODS` (had icon/available): removed from mock-data.ts
- `cart.ts` no longer computes subtotal/discount/shipping/tax/total
- `CartDrawer.vue` no longer shows local totals or free-shipping progress
- `CheckoutDialog.vue` fetches shipping/payment methods from server API

### Source assertion
`npm run check:browser-authority` — PASSED

## Conclusion

The production bundle and browser source contain no fabricated data,
unapproved claims, or browser-authority pricing constants. All monetary
values and method availability come from server-authoritative API
responses. The shipping fee schedule (120/60/1500) is NOT approved —
checkout fails closed until a product decision is made.
