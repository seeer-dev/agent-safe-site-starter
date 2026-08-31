# Commerce acceptance boundary

This document defines what “commerce is complete” means for this starter and separates source-level completion from deployment/go-live acceptance.

## Source-level completion

The commerce reference flow is source-level complete when all of the following are true and repository CI is green:

- product/catalog data is server-authoritative;
- cart persistence does not persist authoritative monetary values;
- quote totals come from the server;
- shipping/payment method availability comes from server-managed configuration;
- guest and member checkout create a durable order before external payment navigation;
- create-order retries are idempotent;
- stock decrement and order creation are transactional;
- ECPay launch signing is server-owned;
- ECPay ReturnURL is the payment authority;
- callback identity and amount reconcile against durable state;
- callback replay is one-effect and conflicting callbacks fail closed;
- browser return never marks paid and re-queries durable server state;
- guest/member/admin order lookup paths remain functional;
- return/restock behavior remains fail-closed and verified.

At `main@c8249301020c6a0eceeda33b875e5d9101b91ebe`, these implementation conditions are present and the full PR/post-merge CI chain passed.

## Deployment acceptance

A public deployment adds requirements that source tests cannot prove:

```mermaid
flowchart LR
    A[Source-level complete] --> B[Public HTTPS deployment]
    B --> C[Provider credentials configured]
    C --> D[ECPay stage transaction]
    D --> E[External ReturnURL reaches API]
    E --> F[200 + 1|OK]
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
6. ECPay `ReturnURL` and browser return endpoints reachable over public HTTPS.
7. One complete ECPay stage credit-card flow.
8. ReturnURL receives the provider callback and responds exactly as required by the current official ECPay contract.
9. The durable order changes from `unpaid` to `paid` only after verified callback handling.
10. The returning browser observes payment through server re-query rather than navigation state.

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

For a local-only review, the ECPay external-provider portion may remain a documented deploy-acceptance item while signing, tamper rejection, amount reconciliation, replay handling, and browser non-authority stay covered by repository tests.

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

## Official ECPay review

Before declaring a deployment production-ready, review the implementation against the current official `ECPay/ECPay-API-Skill` payment, CheckMacValue, HTTP protocol, callback, and go-live references. A real stage transaction remains a deployment acceptance requirement even when static conformance is clean, because external reachability and provider behavior cannot be proven by source inspection alone.
