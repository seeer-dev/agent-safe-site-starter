# Commerce ECPay Payment Flow Delivery Plan

Change ID: commerce-ecpay-payment-flow
Revision: 1
Status: Accepted

Repository baseline: `8968f4943b6697a70d981ed3e5338d4584518b6f`

## Scope lock

- `.env.example`
- `.env.development.example`
- `.env.production.example`
- `db/migrations/sqlite/017_ecpay_payment_attempts.sql`
- `db/migrations/postgres/017_ecpay_payment_attempts.sql`
- `server/internal/config/config.go`
- `server/internal/bootstrap/app.go`
- `server/internal/modules/commerce/service.go`
- `server/internal/modules/commerce/store.go`
- `server/internal/modules/commerce/ecpay.go`
- `server/internal/modules/commerce/ecpay_payment.go`
- `server/internal/modules/commerce/store_ecpay.go`
- `server/internal/modules/commerce/ecpay_http.go`
- `server/internal/modules/commerce/ecpay_test.go`
- `server/internal/modules/commerce/ecpay_security_test.go`
- `site/themes/minimal-cart/shared/lib/api.ts`
- `site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue`
- `specs/changes/commerce-ecpay-payment-flow/**`

## Delivery

Covers REQ-001, AC-001, REQ-002, AC-002, REQ-003, AC-003, REQ-004, AC-004, REQ-005, AC-005.

- add explicit all-or-none ECPay stage/production runtime configuration and fail closed on known public test credentials in production;
- derive hosted checkout, provider ReturnURL, and browser-return URLs from the configured public origins;
- add parity SQLite/PostgreSQL durable payment-attempt storage without widening existing order scan contracts;
- sign ECPay AIO launch fields on the server and expose no signing secrets to the browser;
- verify ReturnURL CheckMacValue, MerchantID, merchant trade identity, and amount against durable state before any paid transition;
- claim callback fingerprint and provider result transactionally, update order payment_status, and append the payment-status order event within the same transaction;
- acknowledge an identical callback replay with one effect and reject a conflicting claimed result;
- verify browser return without mutating payment state;
- retain the already-issued order access credential in same-tab sessionStorage before hosted navigation, then re-query durable payment state after return;
- launch ECPay only after CreateOrder succeeds with an enabled ready ECPay payment method;
- require migration parity, ECPay-specific/backend tests, controlled specification validation, storefront production build, and the repository CI chain before merge.
