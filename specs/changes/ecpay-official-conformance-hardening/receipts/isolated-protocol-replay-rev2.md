# Isolated ECPay protocol replay — revision 2

Change ID: `ecpay-official-conformance-hardening`
Revision: `2`
Observed date: `2026-09-03`

This receipt records an independent local Go replay of the protocol-critical function bodies from the current branch. It intentionally does not claim full repository CI; GitHub Actions did not create pull-request runs for PR #15/#16 events during this review window.

## Environment

```text
Go standard-library-only temporary module
No provider network calls
No repository or provider secrets
```

## Official CheckMacValue vectors

Expected SHA256 digests were copied from official `ECPay/ECPay-API-Skill@test-vectors/checkmacvalue.json` at pinned commit `ae964f75b69ec90e1c205b136364ab6587fc328c`.

Observed tests:

```text
=== RUN   TestOfficialVectors
--- PASS: TestOfficialVectors
```

The replay matched all five AIO SHA256 vectors exactly:

```text
baseline    291CBA324D31FB5A4BBBFDF2CFE5D32598524753AFD4959C3BF590C5B2F57FB2
apostrophe  CF0A3D4901D99459D8641516EC57210700E8A5C9AB26B1D021301E9CB93EF78D
tilde       CEEAE01D2F9A8E74D4AC0DCE7735B046D73F35A5EC99558A31A2EE03159DA1C9
spaces      7712A5E6EDC3B57086063C88568084C66CE882A21D40E74DE5ACA3B478C6F316
callback    2AB536D86AFF8E1086744D59175040A32538C96B1C28C4135B551BD728E913B8
```

The apostrophe vector specifically proves the Go-specific `' -> %27` repair.

## Callback-origin replay

Observed tests:

```text
=== RUN   TestOrigins
--- PASS: TestOrigins
```

Accepted:

- `https://api.example.com` (implicit 443)
- `https://api.example.com:443` (explicit standard HTTPS port)
- `https://xn--fiq228c.tw` (punycode DNS host)

Rejected:

- `https://api.example.com:8443`
- direct IP callback origin
- unencoded Unicode hostname
- non-HTTPS origin

## Callback-authority replay

Observed tests:

```text
=== RUN   TestCallbackAuthority
--- PASS: TestCallbackAuthority
```

The replay exercised the current parser/authority rules:

1. signed `SimulatePaid=1`, `RtnCode=1`, correct `TradeAmt` -> exact `1|OK`, zero durable claims;
2. signed pending/non-success `RtnCode=10300066` -> exact `1|OK`, zero durable claims;
3. later signed real `RtnCode=1`, `SimulatePaid=0` for the same MerchantTradeNo -> exactly one captured claim;
4. signed callback with wrong `TradeAmt` -> amount mismatch before claim;
5. callback signed first and then `TradeAmt` tampered -> invalid signature/callback;
6. signed unknown `SimulatePaid=2` -> invalid callback.

Overall replay result:

```text
PASS
ok   ecpayreplay
```

## Contract review

The branch OpenAPI diff changes only `ECPayCallbackForm` from request-side `TotalAmount` to callback-side `TradeAmt` and adds optional `SimulatePaid` enum `['0','1']`. The runtime/OpenAPI checker adds matching component guards. No runtime routes were added or removed, so the previously accepted 56-operation route parity baseline is unchanged by this slice.

## Verification boundary

This receipt proves the provider-protocol and authority properties above. It does not claim GitHub Actions ran the full repository suite. Formal PR/main regression remains a repository delivery gate; if Actions remains unavailable for PR #16, that limitation must be recorded explicitly rather than reported as a green CI run.
