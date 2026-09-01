# ECPay Official Conformance Hardening Delivery Plan

Change ID: ecpay-official-conformance-hardening
Revision: 1
Status: Applying
Repository baseline: `0071aebecf8b6939308e7965fb5f0a07797d8579`

## Scope lock

- `server/internal/modules/commerce/ecpay.go`
- `server/internal/modules/commerce/ecpay_payment.go`
- `server/internal/modules/commerce/ecpay_test.go`
- `server/internal/modules/commerce/ecpay_official_conformance_test.go`
- `contracts/openapi.yaml`
- `contracts/check-runtime-openapi.mjs`
- `README.md`
- `docs/project-status.md`
- `docs/commerce-acceptance.md`
- `docs/ecpay-official-conformance.md`
- `specs/changes/ecpay-official-conformance-hardening/**`

Covers REQ-001, AC-001, REQ-002, AC-002, REQ-003, AC-003, REQ-004, AC-004, AC-005.

## Slice 1 — Pin official sources and audit current behavior

- Pin the ECPay API Skill master commit used for review.
- Read current AIO, CheckMacValue, HTTP protocol, webhook, go-live, and live Developers callback/checksum pages.
- Record verified mismatches rather than inferring conformance from the existence of an integration.

## Slice 2 — Fix protocol mismatches

- Parse callback amount from `TradeAmt` rather than request-side `TotalAmount`.
- Recognize optional `SimulatePaid`; verify and acknowledge a simulation without consuming the durable provider callback claim, so a later real callback can still capture.
- Add the missing apostrophe encoding rule to CheckMacValue.
- Reject ECPay public origins that specify a port, use a direct IP, use a non-HTTPS scheme, or provide an unencoded Unicode hostname.
- Keep existing merchant/amount/trade correlation and replay/conflict protection for real callbacks.

## Slice 3 — Lock with official vectors and contract truth

- Add official SHA256 CheckMacValue vector tests for baseline, apostrophe, tilde, spaces, and callback fields.
- Convert callback tests to the official `TradeAmt` shape.
- Add simulation→real-callback, origin, optional-SimulatePaid, and invalid-SimulatePaid tests.
- Update `ECPayCallbackForm` in OpenAPI and mechanically guard the corrected shape.

## Slice 4 — Acceptance and status

- Run the repository contract gate and full CI chain.
- Record an independent-review receipt with the pinned official-source snapshot and observed vector/behavior evidence.
- Mark the change Accepted only after every REQ/AC is evidenced.
- Update README/project/commerce status to distinguish source-level official conformance from the still-pending public stage transaction.
