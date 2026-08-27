# Evidence

| Requirement | Proof |
|---|---|
| REQ-001 | `server/internal/modules/commerce/ecpay.go`, runtime wiring in `server/internal/bootstrap/app.go` |
| REQ-002 | `db/migrations/*/017_ecpay_payment_attempts.sql`, `store_ecpay.go`, `ecpay_payment.go` |
| REQ-003 | callback fingerprint CAS in `store_ecpay.go` |
| REQ-004 | `Handler.ECPayBrowserReturn` delegates to verification-only `Service.ECPayBrowserReturn` |
| REQ-005 | `shared/lib/api.ts` + `CheckoutDialog.vue` hosted form handoff |
