# Commerce acceptance boundary

This document defines what “commerce is complete” means for this starter and separates source-level completion from deployment/go-live acceptance.

## Source-level completion

The commerce reference flow is source-level complete when the repository implements these authority boundaries:

- product/catalog data is server-authoritative;
- cart persistence does not persist authoritative monetary values;
- quote totals come from the server;
- shipping/payment method availability comes from server-managed configuration;
- guest and member checkout create a durable order before external payment navigation;
- create-order retries are idempotent;
- stock decrement and order creation are transactional;
- ECPay launch signing is server-owned;
- ECPay ReturnURL is the payment authority;
- request-side `TotalAmount` and callback-side `TradeAmt` are kept distinct;
- callback merchant/trade identity and `TradeAmt` reconcile against durable state;
- `SimulatePaid=1` and signed non-success/pending callbacks cannot mark paid or consume the durable real-success claim;
- only a non-simulated `RtnCode=1` may reach the durable success claim;
- real callback replay is one-effect and conflicting claimed callbacks fail closed;
- browser return never marks paid and re-queries durable server state;
- guest/member/admin order lookup paths remain functional;
- return/restock behavior remains fail-closed and verified.

The original commerce/ECPay loop reached full PR/post-merge CI on `main` under the Accepted commerce/ECPay changes. The later official-conformance hardening is independently replayed against pinned official ECPay protocol vectors and callback rules; its source audit is recorded in [`ecpay-official-conformance.md`](ecpay-official-conformance.md). During the 2026-09-01 to 2026-09-03 review window GitHub did not create pull-request Actions runs for PR #15/#16, so do not infer or report a green PR CI run for that hardening.

## ECPay source-conformance boundary

Source-level ECPay conformance is complete against pinned official `ECPay/ECPay-API-Skill@ae964f75b69ec90e1c205b136364ab6587fc328c` when all of the following hold:

- AIO v5 stage/production endpoints remain finite and server-owned;
- MerchantID / HashKey / HashIV remain server-only;
- CheckMacValue follows the official Go encoding path, including `~ -> %7e`, apostrophe `' -> %27`, the seven .NET replacements, SHA256, and uppercase output;
- the official baseline/apostrophe/tilde/space/callback SHA256 vectors match exactly;
- ReturnURL parses `TradeAmt`, not `TotalAmount`;
- invalid signature, wrong merchant, and wrong durable amount fail closed;
- `SimulatePaid=1` and signed non-success states are acknowledged only after validation and do not mutate financial state;
- later real success can still claim exactly once;
- HTTPS public origins use standard port 443 (implicit or explicit); non-standard HTTPS ports fail closed;
- browser return remains non-authoritative;
- OpenAPI models the callback as `TradeAmt` with optional `SimulatePaid`.

That is a source/protocol claim only. It does not prove ECPay can reach the deployed callback.

## Deployment acceptance

A public deployment adds requirements that source tests cannot prove:

```mermaid
flowchart LR
    A[Source/protocol complete] --> B[Public HTTPS deployment]
    B --> C[Provider credentials configured]
    C --> D[ECPay stage transaction]
    D --> E[External ReturnURL reaches API]
    E --> F[Exact 1|OK]
    F --> G[Durable order becomes paid]
    G --> H[Browser returns and re-queries paid]
    H --> I[Deployment accepted]
```

Deployment acceptance should record evidence for:

1. Cloudflare Pages or equivalent public site URL.
2. Railway or equivalent public Go API URL.
3. Production-shaped PostgreSQL migrations applied successfully.
4. Supabase Auth, R2, and Resend configured where the selected example requires them.
5. ECPay stage `MerchantID`, `HashKey`, and `HashIV` supplied through provider-managed secrets.
6. ECPay `ReturnURL` and browser return endpoints reachable over public HTTPS on the supported transport.
7. One complete ECPay stage credit-card flow.
8. ReturnURL receives the provider callback and responds with the exact required acknowledgement.
9. The durable order changes from `unpaid` to `paid` only after verified real callback handling.
10. A simulated callback demonstrably does not mark the order paid.
11. The returning browser observes payment through server re-query rather than navigation state.

## Acceptance walkthrough

Use this sequence for the sample commerce review:

```text
fresh DB
→ migrate
→ seed deterministic sample products/config
→ open storefront
→ add product to cart
→ reload and verify cart rehydration
→ obtain server quote
→ guest checkout
→ guest order lookup
→ member checkout
→ member order history
→ ECPay hosted-payment handoff
→ deployed stage callback acceptance (when public environment exists)
→ admin order lookup/status
→ return request
→ receive return
→ per-item restock
```

For a local/source-only review, the ECPay external-provider portion may remain a documented deploy-acceptance item while signing, official vectors, tamper rejection, TradeAmt reconciliation, simulation/non-success authority, durable replay handling, and browser non-authority are verified independently.

## Not required for v1 acceptance

These are post-v1 or outcome-driven extensions:

- real refund/cancel provider flow;
- AAL2/governed refund authorization;
- refund idempotency and refund reconciliation;
- electronic invoice integration;
- logistics provider integration;
- tracking lifecycle/webhooks;
- automated payment reconciliation jobs;
- failed-payment recovery operations;
- operational dashboards.

Do not weaken existing fail-closed behavior just to expose these states in UI. For example, a return reaching `received` must not be turned into a fake `refunded` state without verified refund authority and durable provider/offline confirmation.

## Current next acceptance step

The official ECPay source-conformance audit is no longer the next product task. The next release-readiness work is:

```text
fresh DB commerce walkthrough
→ production-shaped deploy configuration
→ public HTTPS endpoints
→ one ECPay stage transaction
→ record callback/payment/browser evidence
```

A real stage transaction remains required before production use because external reachability and provider behavior cannot be proven by source inspection or local protocol replay alone.
