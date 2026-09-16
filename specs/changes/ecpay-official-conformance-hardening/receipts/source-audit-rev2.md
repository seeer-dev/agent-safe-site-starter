# ECPay official-source audit receipt — revision 2

Change ID: `ecpay-official-conformance-hardening`
Revision: `2`
Observed baseline: `0071aebecf8b6939308e7965fb5f0a07797d8579`
Audit date: `2026-09-03`

## Pinned official source

- Repository: `ECPay/ECPay-API-Skill`
- Audited master commit: `ae964f75b69ec90e1c205b136364ab6587fc328c`
- Observed commit message: `修復缺漏參數`
- Observed timestamp: `2026-09-01T07:34:46Z`

Reviewed official artifacts:

- `guides/01-payment-aio.md`
- `guides/13-checkmacvalue.md`
- `guides/15-troubleshooting.md`
- `guides/16-go-live-checklist.md`
- `guides/19-http-protocol-reference.md`
- `guides/21-webhook-events-reference.md`
- `guides/23-multi-language-integration.md`
- `test-vectors/checkmacvalue.json`
- `references/Payment/全方位金流API技術文件.md`

The reference index explicitly requires checking the linked ECPay Developers pages for concrete field rules rather than relying only on static summaries.

## Findings retained from revision 1

1. AIO ReturnURL supplies `TradeAmt`; request-side `TotalAmount` is not the callback amount field.
2. Go CheckMacValue needs the apostrophe `' -> %27` conversion in addition to `~ -> %7e` and the seven .NET replacements.
3. `SimulatePaid=1` is not real financial authority. A simulated success must not mark an order paid or trigger fulfillment.
4. `ItemName` has a provider ceiling of 400 characters and current guidance recommends staying within 200 to avoid provider-side truncation affecting the signed input.
5. Valid ReturnURL processing uses form POST, verifies CheckMacValue and business correlation, and returns the exact plain-text acknowledgement `1|OK`.

## Revision-2 transport clarification

Revision 1 interpreted the HTTPS callback-port guidance too narrowly and rejected even an explicit `:443` URL. The official troubleshooting/go-live guidance says HTTPS callbacks must use port 443 and non-443 HTTPS ports are invalid; it does not require the textual URL to omit an explicit standard port.

Revision 2 therefore defines the starter boundary as:

- `https://host` — accepted (implicit 443)
- `https://host:443` — accepted (explicit standard HTTPS port)
- `https://host:<non-443>` — rejected
- direct IP host — rejected by starter policy in favor of a public DNS/FQDN callback
- unencoded Unicode hostname — rejected; provide punycode
- non-HTTPS origin — rejected because this starter intentionally requires HTTPS

This narrows the hardening to the official transport constraint without inventing an unnecessary incompatibility.
