# ECPay official-source audit receipt — revision 1

Change ID: `ecpay-official-conformance-hardening`
Revision: `1`
Observed baseline: `0071aebecf8b6939308e7965fb5f0a07797d8579`
Audit date: `2026-09-01`

## Pinned official source

- Repository: `ECPay/ECPay-API-Skill`
- Audited master commit: `ae964f75b69ec90e1c205b136364ab6587fc328c`
- Observed commit message: `修復缺漏參數`
- Observed timestamp: `2026-09-01T07:34:46Z`

Reviewed official artifacts:

- `guides/01-payment-aio.md`
- `guides/13-checkmacvalue.md`
- `guides/16-go-live-checklist.md`
- `guides/19-http-protocol-reference.md`
- `guides/21-webhook-events-reference.md`
- `guides/23-multi-language-integration.md`
- `test-vectors/checkmacvalue.json`
- `references/Payment/全方位金流API技術文件.md`

The linked live ECPay Developers documentation for AIO order creation, payment-result notification, integration cautions, checksum mechanism, and self-test guidance was also reviewed because the official reference index explicitly says not to rely only on static summaries for concrete field rules.

## Independent findings

1. Existing callback verification required `TotalAmount`, but official AIO ReturnURL supplies `TradeAmt`.
2. Existing Go CheckMacValue URL conversion repaired `~` but did not perform the official Go-specific apostrophe `' -> %27` conversion.
3. Existing capture logic treated every signed `RtnCode=1` callback as paid, but official ReturnURL guidance requires `SimulatePaid=0` before treating a success as a real payment.
4. Existing HTTPS-origin validation accepted explicit non-443 ports, while the official callback/go-live constraints support standard HTTP/HTTPS ports only; the starter is HTTPS-only, so its ECPay callback origin must be 443.
5. Existing `ItemName` helper used the provider hard maximum as its truncation point; current official guidance recommends keeping ItemName within 200 characters to avoid provider-side truncation changing the signed input.

These findings were derived from the official ECPay sources above, not from the prior starter implementation or its tests.
