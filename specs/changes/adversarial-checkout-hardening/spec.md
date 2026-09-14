# Adversarial Checkout Hardening Specification

Change ID: adversarial-checkout-hardening
Revision: 1
Status: Verifying
Decision authority: Repository owner/user
Approval basis: Owner direction 2026-09-15: run the PostgreSQL live gate and perform an adversarial security review; the review live-exploited three defects (draft-parent variant ordering, partial-PUT scalar zeroing, staff-note leakage) and fixing them is the review's implied remediation step.
Repository baseline: 10f805e18e394e7065854dfbceb327114e0d4564
Supersedes: none

## Outcome

The commerce module closes three defects that adversarial probing of the
running local stack exploited end-to-end before any fix: a variant SKU
of a draft product could be ordered (a free-order primitive when paired
with defect 2), a partial product PUT silently zeroed omitted scalars
(price fell to 0 on a name-only update), and staff timeline notes were
readable by any holder of a guest order token.

## Scope

In scope:

- `server/internal/modules/commerce/` — resolveVariant parent-status
  guard, presence-aware ProductInput merge on update, timeline note
  redaction in maskCustomerPII, and the security regression test file.

Out of scope:

- The same absent-field zeroing pattern in promo/category/shipping/
  payment/sitecontent inputs (admin-only, several fail closed) —
  recorded as a follow-up.
- Product optimistic concurrency (orders/shipping methods have
  expected_version; products do not).
- Public-endpoint rate limiting — remains with the deferred
  public-endpoint-rate-limit spec.

## Requirements

### REQ-001: Variant SKUs must not bypass parent publish status

Checkout resolves a SKU against `product_variants` when no published
product matches. `resolveVariant` loaded the parent row but never
checked its status, so variants of `draft` products remained purchasable
by anyone who knows the SKU. The variant path MUST require the parent
product to be in the customer-visible status set (active / low_stock /
out_of_stock) — the same set `ListPublishedProducts` exposes — and a
non-public parent's variant MUST fail with the same not-found error an
unknown SKU produces.

#### AC-001: Draft-parent variant rejected
- GIVEN a draft product whose variants carry purchasable SKUs
- WHEN CreateOrder resolves such a SKU
- THEN the order is rejected with not-found; a control order on a
  variant whose parent is active still succeeds.

### REQ-002: Partial product update must preserve absent fields

`UpdateProduct` applied every `ProductInput` scalar directly, so a JSON
body containing only `{"name": "x"}` zeroed `price`, `original_price`,
`stock`, `is_featured`, `material`, `origin`, `tag`, and descriptions.
`ProductInput` MUST record which top-level JSON keys were present. On
update, absent fields MUST keep their existing values; explicitly
present fields MUST apply, including explicit zeros. Programmatic
construction keeps legacy all-present semantics.

#### AC-002: Partial body preserves scalars
- GIVEN an existing product with non-zero price, stock, featured flag,
  and populated text fields
- WHEN an update supplies only `{"name": "..."}`
- THEN all absent scalars and text fields keep their previous values,
  and a body containing `{"stock": 0}` still applies explicitly.

### REQ-003: Staff order notes must not reach customer-facing reads

Order status transitions accept a staff `note` appended to the order
timeline. `maskCustomerPII` redacted contact fields but left timeline
notes intact, so any holder of the guest access token could read
internal operational remarks (e.g. fraud-review holds). The storefront
order/track views never render note text. `maskCustomerPII` MUST strip
`Note` from timeline events; admin order reads MUST keep notes.

#### AC-003: Guest timeline hides staff notes
- GIVEN an order whose timeline carries a staff note
- WHEN GetOrderForGuest returns the order
- THEN every timeline entry has an empty note while the admin read
  retains the original note text.

### REQ-004: Regression coverage and gates

Every fixed defect SHALL carry a service-level regression test whose
failure trigger was observed via mutation of the protection, and the
fixes SHALL be re-verified against the running dev API. The standard
verification chain MUST pass under this change's scope.

#### AC-004: Live re-verification
- GIVEN the fixed code on the restarted dev API
- WHEN a variant SKU under a draft parent is ordered and a partial PUT
  updates a product
- THEN the order is rejected and the PUT preserves price and stock.

#### AC-005: Verification chain
- GIVEN the final diff
- WHEN `go run ./server/tools/verify` runs
- THEN it passes under this change's scope and evidence.
