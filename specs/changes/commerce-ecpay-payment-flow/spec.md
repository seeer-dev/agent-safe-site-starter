# Commerce ECPay Payment Flow Specification

Change ID: commerce-ecpay-payment-flow
Revision: 1
Status: Accepted
Decision authority: Repository owner
Approval basis: Owner approved implementing the starter-owned ECPay payment flow on 2026-08-27, preserving server-authoritative payment truth and fail-closed callback handling.
Repository baseline: 8968f4943b6697a70d981ed3e5338d4584518b6f
Supersedes: none

### REQ-001: Server-owned ECPay launch

The server MUST derive ECPay endpoints and callback URLs from finite runtime configuration, keep HashKey and HashIV server-only, reject known public test credentials in production, and sign the hosted checkout form.

#### AC-001: Hosted launch exposes no signing secrets

- GIVEN ECPay is explicitly configured for stage or production
- WHEN a valid unpaid order requests an ECPay launch
- THEN the server returns only the provider action URL and public signed fields, and never returns HashKey or HashIV

### REQ-002: Durable provider payment truth

Each order MUST have at most one starter-owned ECPay payment attempt. The provider ReturnURL MUST verify CheckMacValue, MerchantID, MerchantTradeNo, and TotalAmount against durable state before payment can become paid.

#### AC-002: Only a verified matching ReturnURL can mark paid

- GIVEN a durable ECPay payment attempt exists for an unpaid order
- WHEN a ReturnURL callback is received
- THEN payment_status becomes paid only if the signed callback verifies and its merchant trade identity and amount match the durable attempt

### REQ-003: Callback replay and conflict safety

The implementation MUST make identical verified callbacks one-effect and MUST fail closed when a later signed callback conflicts with an already claimed durable result.

#### AC-003: Claim and paid transition are atomic

- GIVEN an unclaimed durable payment attempt
- WHEN its first verified terminal callback is claimed
- THEN the callback fingerprint, provider result, order paid transition, and payment-status order event are committed within one SQL transaction; an identical replay produces no second transition

### REQ-004: Browser return is not payment authority

The ECPay browser return MUST verify the signed provider form and redirect to the storefront only. Browser navigation MUST NOT transition payment state.

#### AC-004: Storefront re-queries durable truth after return

- GIVEN the browser navigated to ECPay using an already-issued order access credential retained in same-tab sessionStorage
- WHEN ECPay returns the browser to the storefront
- THEN the storefront re-queries the order payment_status and never infers paid from the browser return itself

### REQ-005: Storefront hosted-payment handoff

After a durable order is created with an enabled and ready ECPay payment method, the minimal-cart storefront MUST request a signed launch form from the server and POST it to the hosted ECPay endpoint.

#### AC-005: Full starter flow remains verifiable

- GIVEN the starter checkout creates a durable order before external payment navigation
- WHEN the ECPay flow is enabled
- THEN SQLite/PostgreSQL migration parity, focused backend tests, controlled specification validation, and the minimal-cart production build must pass before merge
