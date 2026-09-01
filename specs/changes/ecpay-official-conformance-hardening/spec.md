# ECPay Official Conformance Hardening

Change ID: ecpay-official-conformance-hardening
Revision: 1
Status: Verifying
Decision authority: Repository owner
Approval basis: Repository owner instructed continuing the queued v1 release-readiness sequence after HTTP contract restoration. This change audits and minimally hardens the existing ECPay AIO credit flow against the current official ECPay API Skill and live ECPay Developers references without adding new payment products or commerce frameworks.
Repository baseline: 0071aebecf8b6939308e7965fb5f0a07797d8579
Supersedes: none

## Outcome

Make the existing starter-owned ECPay AIO credit flow conform to the current official ECPay callback, CheckMacValue, and go-live rules while preserving the existing server-authoritative payment model. The change fixes verified protocol mismatches and records the official-source snapshot used for review. It does not add refunds, invoice, logistics, reconciliation, additional payment methods, or a generic payment-provider abstraction.

## In scope

- Correct the AIO ReturnURL amount field to the official `TradeAmt` callback field.
- Treat `SimulatePaid=1` as a ReturnURL transport test only: verify merchant/trade/amount/signature, acknowledge it, and do not consume the durable callback claim or transition the order.
- Make the Go CheckMacValue encoder match the current official Go implementation, including apostrophe encoding and official SHA256 vectors.
- Keep ReturnURL verification timing-safe and server-authoritative, with durable amount/merchant/trade correlation and replay/conflict protection before acknowledging real payment callbacks.
- Enforce the deployment-safe HTTPS/DNS callback-origin constraints for the existing configuration boundary: no explicit port, no direct IP host, and IDNs supplied in punycode form.
- Keep the OpenAPI callback contract and mechanical contract guard aligned with the corrected protocol.
- Record the official ECPay Skill commit and live ECPay Developers pages used for the audit, and update canonical project/commerce status.

## Out of scope

- ECPay refund/cancel APIs, QueryTradeInfo/reconciliation jobs, invoices, logistics, ATM/CVS/BARCODE, recurring payments, Apple Pay, ECPG, or additional providers.
- Reworking the durable payment-attempt model or adding failed-payment retry/superseding attempts.
- Moving payment state authority into the browser.
- A real externally hosted ECPay stage transaction; that remains deployment acceptance after source-level conformance.

### REQ-001: Callback shape matches official AIO ReturnURL

The server MUST parse and verify the current official AIO ReturnURL form shape, using `TradeAmt` as the returned transaction amount and recognizing the official optional `SimulatePaid` flag.

#### AC-001: Official callback amount and simulation semantics
- GIVEN an official-shaped signed callback containing `TradeAmt`, `RtnCode`, `TradeNo`, and optionally `SimulatePaid`
- WHEN the callback is verified and reconciled
- THEN `TradeAmt` is compared to the durable TWD order amount, `RtnCode=1` with no simulated flag may capture the order, and `SimulatePaid=1` is acknowledged without consuming the durable callback claim or changing payment/order state

### REQ-002: CheckMacValue matches current official Go rules

The AIO SHA256 CheckMacValue implementation MUST match the current official ECPay Go encoding rules: case-insensitive key sorting, all transmitted fields except CheckMacValue, `HashKey`/`HashIV` wrapping, form-style URL encoding, lower-casing, the seven .NET replacements, explicit `~ -> %7e`, explicit apostrophe `' -> %27`, SHA256, and uppercase output.

#### AC-002: Official vectors pass
- GIVEN the official ECPay CheckMacValue SHA256 vectors for baseline AIO, apostrophe, tilde, spaces, and callback verification
- WHEN the starter implementation computes CheckMacValue
- THEN every copied expected digest matches exactly and tampering still fails timing-safe verification

### REQ-003: Payment callback authority remains fail-closed and idempotent

Only a correctly signed callback that matches the configured merchant, durable merchant trade number, durable amount, and non-simulated successful payment state may transition an order to paid. Browser returns remain non-authoritative. Simulated notifications MUST leave the durable claim available for a later real callback of the same MerchantTradeNo.

#### AC-003: Invalid, conflicting, or simulated callbacks cannot consume real-payment authority
- GIVEN a bad signature, wrong amount, wrong merchant identity, duplicate conflicting callback, or simulated callback
- WHEN the ReturnURL path handles it
- THEN invalid/conflicting callbacks do not capture, simulated callbacks do not mutate the payment attempt or order, and a later valid real callback can still claim and capture the same durable payment attempt exactly once

### REQ-004: Go-live constraints are reflected in code and documentation

The starter MUST keep ECPay secrets server-only, use the finite official stage/production AIO endpoints, require public HTTPS DNS origins compatible with the official callback transport constraints, and document remaining deployment-only acceptance.

#### AC-004: Configuration rejects unsupported callback origins
- GIVEN an ECPay configuration whose public origin uses an explicit port, direct IP host, non-HTTPS scheme, or unencoded Unicode hostname
- WHEN configuration is constructed
- THEN it fails closed rather than producing a ReturnURL/OrderResultURL that violates the documented ECPay deployment constraints

#### AC-005: Canonical status distinguishes conformance from stage acceptance
- GIVEN source-level official conformance is complete
- WHEN README/project/commerce status is read
- THEN official conformance is marked complete against the audited ECPay Skill snapshot while a public HTTPS stage transaction remains explicitly pending deploy/go-live acceptance
