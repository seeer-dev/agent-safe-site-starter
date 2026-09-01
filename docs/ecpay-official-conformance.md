# ECPay AIO official conformance audit

Last audited: 2026-09-01

This document records the source-level conformance review for the starter's existing ECPay AIO v5 credit-card flow. It is not a substitute for the final public stage transaction required before production use.

## Official source snapshot

The review is pinned to:

- `ECPay/ECPay-API-Skill@ae964f75b69ec90e1c205b136364ab6587fc328c` — official master as observed on 2026-09-01.
- `guides/01-payment-aio.md` — AIO integration flow and ReturnURL/OrderResultURL separation.
- `guides/13-checkmacvalue.md` — CheckMacValue algorithm and language-specific encoding details.
- `guides/16-go-live-checklist.md` — go-live transport, callback, secret, and environment constraints.
- `guides/19-http-protocol-reference.md` — form POST and acknowledgement conventions.
- `guides/21-webhook-events-reference.md` — current AIO ReturnURL fields and processing sequence.
- `guides/23-multi-language-integration.md` — current Go AIO/CheckMacValue example.
- `test-vectors/checkmacvalue.json` — official SHA256 vectors used by the repository tests.
- `references/Payment/全方位金流API技術文件.md` — official live-document index; the audit also checked the linked ECPay Developers pages for create-order, payment-result notification, integration cautions, checksum rules, and self-test guidance.

The pinned official master commit was itself updated on 2026-09-01 to restore missing webhook-reference parameters, so the repository records the exact commit instead of saying only "latest docs".

## Audit result

| Boundary | Official rule | Starter result after this change |
| --- | --- | --- |
| AIO endpoint | Stage and production AIO v5 endpoints are distinct and finite. | Conformant; endpoint selection remains server-owned. |
| Secret ownership | Merchant credentials and HashKey/HashIV stay on the server. | Conformant; public launch fields never contain HashKey/HashIV. |
| Request amount | Create-order request uses `TotalAmount`. | Conformant; launch form still sends the durable order total as `TotalAmount`. |
| ReturnURL amount | Payment-result callback returns `TradeAmt`. | **Fixed**; previous code incorrectly required callback `TotalAmount`. |
| Simulated payment | `SimulatePaid=1` is only a ReturnURL transport test and must not change payment state or trigger fulfillment. | **Fixed**; after signature/merchant/trade/amount verification it returns `1|OK` without consuming the durable callback claim or mutating the order, so a later real callback can still capture normally. |
| CheckMacValue sorting | Sort parameter names case-insensitively and exclude `CheckMacValue` itself. | Conformant. |
| CheckMacValue wrapping | `HashKey=...&<params>&HashIV=...`. | Conformant. |
| URL encoding | Form-style encoding, lower-case, .NET compatibility replacements, `~ -> %7e`, and Go apostrophe `' -> %27`. | **Fixed**; apostrophe encoding was previously missing. |
| Digest | AIO uses SHA256 and uppercase hex. | Conformant; locked to official vectors. |
| Signature compare | Avoid ordinary timing-sensitive equality. | Conformant; `hmac.Equal` is retained as a constant-time comparison primitive. |
| Merchant correlation | Verify callback merchant identity and merchant trade identity. | Conformant. |
| Durable amount correlation | Provider-returned amount must equal the durable order attempt amount before capture. | Conformant, now using official `TradeAmt`. |
| Callback replay | Duplicate real payment notification must not create duplicate financial/order effects. | Conformant; real callbacks use a durable one-effect claim and conflicting claims fail closed. Simulated notifications intentionally do not consume that claim. |
| ReturnURL acknowledgement | Valid reconciled callback returns exact plain-text `1|OK`. | Conformant; a validated simulation is also acknowledged because its purpose is testing ReturnURL reachability, while invalid/unreconciled input fails closed. |
| Browser return | Browser redirect is presentation/navigation, not payment authority. | Conformant; storefront re-queries durable server state. |
| Callback transport | Public callback must use supported HTTP/HTTPS ports; starter requires HTTPS, therefore port 443. | **Hardened**; explicit non-443 HTTPS origins are rejected. |
| ItemName | Official limit is 400 characters and current guidance recommends keeping it within 200 to avoid provider-side truncation/CMV surprises. | **Hardened**; helper caps at 200 runes. Current product flow uses a short `Order <id>` value. |

## Correct authority flow

```mermaid
flowchart TD
    A[Durable order total] --> B[Server builds AIO form]
    B --> C[TotalAmount + CheckMacValue]
    C --> D[Browser POST to ECPay]
    D --> E[ECPay payment]
    E --> F[Server ReturnURL POST]
    F --> G[Verify CMV + MerchantID + MerchantTradeNo]
    G --> H[Parse TradeAmt]
    H --> I{TradeAmt == durable amount?}
    I -->|No| J[Fail closed]
    I -->|Yes| K{SimulatePaid == 1?}
    K -->|Yes| L[Return 1|OK only; no durable mutation]
    K -->|No| M{RtnCode == 1?}
    M -->|No| N[Durable failed callback claim]
    M -->|Yes| O[Atomic callback claim + paid transition]
    N --> P[1|OK after valid reconciliation]
    O --> P
    E --> Q[Browser OrderResultURL]
    Q --> R[Redirect storefront]
    R --> S[Re-query durable order]
    S --> T[UI reflects server payment_status]
```

A simulated notification does not consume the one-time provider callback fingerprint. That is intentional: the merchant-backend simulation feature tests ReturnURL delivery only, so a later real payment notification for the same `MerchantTradeNo` must still be able to acquire the durable claim.

## Official vectors locked in tests

The repository copies the expected digests, not the official implementation code. Tests cover:

- baseline AIO SHA256 vector;
- apostrophe (`Tom's Shop`) encoding vector;
- tilde encoding vector;
- space-as-`+` vector;
- official-shaped callback vector using `TradeAmt` and `SimulatePaid`.

This guards the language-specific Go encoding traps that are easy to miss even when the high-level algorithm is correct.

## What this audit does not prove

Source-level conformance does **not** prove provider reachability. Before production use, deployment acceptance still requires:

1. deploy API and storefront to their production-shaped public HTTPS origins;
2. configure ECPay stage merchant credentials outside the repository;
3. complete one real stage hosted checkout;
4. observe ECPay reach the ReturnURL over public HTTPS;
5. confirm exact `1|OK`, durable `payment_status=paid`, and browser re-query behavior;
6. only then switch to production merchant credentials/endpoints.

No refund, invoice, logistics, reconciliation, or additional ECPay payment-method work is implied by this conformance audit.
