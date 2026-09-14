# Adversarial Review Hardening — Evidence

Change ID: adversarial-checkout-hardening
Revision: 1
Status: Verifying

| Gate | Status | Proof |
|---|---|---|
| REQ-001 | passed | resolveVariant requires parent status in publicProductStatuses; live repro before fix: POST /api/orders with variant SKU of draft parent returned 201. TestDraftParentVariantNotOrderable passes; mutation (check removed) fails at 'expected not-found for draft-parent variant'. |
| REQ-002 | passed | ProductInput.UnmarshalJSON records top-level key presence (null counts as absent); UpdateProduct merges via has() so absent scalars preserve existing values while explicit zeros apply. Live repro before fix: PUT {name,description} zeroed price/stock on CUR-FRA-02, producing an orderable $0 product. TestUpdateProductPartialBodyPreservesScalars and TestProductInputPresenceFromJSON pass; mutation (preserve removed) fails at 'scalars zeroed by partial body'. |
| REQ-003 | passed | maskCustomerPII strips Timeline Note on guest/member paths; admin reads retain notes. Live repro before fix: guest lookup showed 'INTERNAL: fraud-suspect hold for review'. TestCustomerOrderHidesStaffNotes passes; mutation (strip removed) fails at 'staff note leaked to guest'. |
| REQ-004 | passed | All three fixes carry service-level regression tests in service_security_test.go, each observed failing under mutation of the protection, then passing after restore; live API re-verification done; verify chain green. |
| AC-001 | passed | TestDraftParentVariantNotOrderable: draft-parent variant order rejected; active-parent control succeeds. Mutation observed failing. |
| AC-002 | passed | TestUpdateProductPartialBodyPreservesScalars: name-only body preserves price/original/stock/featured/text; explicit stock=0 applies and derives out_of_stock. Mutation observed failing. |
| AC-003 | passed | TestCustomerOrderHidesStaffNotes: guest timeline notes empty, admin sees them. Mutation observed failing. |
| AC-004 | passed | Live on restarted dev API: CUR-FRA-02 set draft then POST /api/orders with CUR-FRA-02-A returned 400 'commerce: not found'; PUT {tag} preserved price=860 stock=26. Product restored to active/860/26 and CUR-TEX-01-B stock restored to 5 after race test. |
| AC-005 | passed | go run ./server/tools/verify after the fixes: archcheck, migration-parity (18), speccheck, scopecheck, go test ./..., commerce/staff/media -count=10, vet — all green. |

## Documented follow-ups (not fixed in this change)

- Same absent-field zeroing pattern exists in PromoInput,
  CategoryInput, ShippingMethodUpdateInput, PaymentMethodInput, and
  site-content/store-settings inputs. Lower impact: admin-only surface
  and several fail closed at validation. Sweep as a separate change.
- Product updates carry no optimistic-concurrency field (orders and
  shipping methods have expected_version); concurrent admin edits are
  last-write-wins.
- Idempotency-key replay returns 409 on payload mismatch, which reveals
  key existence; keys are client-generated so impact is minimal.
- Public endpoints (comments, coupon validate, guest order lookup,
  create-order) have no rate limit — deferred to the
  public-endpoint-rate-limit spec; the confirmed single-Railway +
  Cloudflare topology makes in-memory IP limiting meaningful.
- Order access tokens are stored in browser sessionStorage/localStorage
  (curatory_recent_orders) — exposed to any future XSS; no XSS sink
  exists today (all rendering is text interpolation, no v-html).
