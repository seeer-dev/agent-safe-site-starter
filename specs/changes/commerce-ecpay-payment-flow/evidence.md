# Commerce ECPay Payment Flow Evidence

Change ID: commerce-ecpay-payment-flow
Revision: 1
Status: Accepted

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | ECPay launch configuration and signing are server-owned; HashKey and HashIV are not returned to the browser, and production rejects the known public test credential tuple. |
| AC-001 | passed | The launch response contains only the hosted provider action and public signed fields; ECPay security tests cover secret non-disclosure and configuration rejection. |
| REQ-002 | passed | SQLite/PostgreSQL migration 017 adds one durable ECPay payment attempt per order, and ReturnURL verification checks CheckMacValue, MerchantID, MerchantTradeNo, and amount before paid transition. |
| AC-002 | passed | Focused commerce tests pass with live PostgreSQL available, including tampered callback and amount-mismatch rejection. |
| REQ-003 | passed | The store claims callback fingerprint and provider result transactionally; identical replay is one-effect and a conflicting claimed callback is rejected fail-closed. |
| AC-003 | passed | The payment-attempt claim, order payment_status update, version increment, and payment-status order event are committed in one SQL transaction. |
| REQ-004 | passed | Browser return verifies the signed provider form and redirects without invoking any payment-state mutation. |
| AC-004 | passed | The storefront retains the already-issued order access credential in same-tab sessionStorage and re-queries durable order payment_status after the provider returns. |
| REQ-005 | passed | Minimal-cart requests the ECPay launch only after durable CreateOrder succeeds and submits the returned signed form to the hosted endpoint. |
| AC-005 | passed | Focused config/bootstrap/commerce tests and migration parity passed with PostgreSQL; storefront production build passed in the implementation verification run. Final repository CI remains required before merge. |
