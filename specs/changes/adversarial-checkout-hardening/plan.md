# Adversarial Review Hardening Plan

Change ID: adversarial-checkout-hardening
Revision: 2
Status: Verifying

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `specs/changes/adversarial-checkout-hardening/**`
- `server/internal/modules/commerce/**`

## Slices

### S01 — Variant parent-status check

`resolveVariant` returns `ErrNotFound` when the parent product is not in
the public status set; `publicProductStatuses` constant mirrors the SQL
customer-visible set.

Covers: REQ-001 (AC-001)

### S02 — Presence-aware product update

`ProductInput.UnmarshalJSON` captures top-level key presence;
`UpdateProduct` merges scalars/strings/flag via `has()` — absent means
preserve, present means apply (explicit zero included).

Covers: REQ-002 (AC-002)

### S03 — Staff-note redaction

`maskCustomerPII` clears `Timeline[i].Note` for guest/member reads.

Covers: REQ-003 (AC-003)

### S04 — Admin entity update sweep

Shared `presentFields` + `decodeStrict` extracted in model.go; the same
presence-merge applied to UpdatePromo, UpdateCategory,
UpdateShippingMethod, UpdatePaymentMethod (now loads the existing row),
UpdateMember, and UpsertNotificationTemplate (existing-row merge;
unknown ids keep PUT-upsert semantics). Nullable usage_limit /
free_threshold use set() so explicit null clears.

Covers: REQ-004 (AC-006)

### S05 — Tests + live verification + gates

`service_security_test.go` (10 tests, each mutation-observed), live API
re-verification, verify chain.

Covers: REQ-005 (AC-004, AC-005)

## Traceability

| REQ / AC | Slice | Evidence |
|---|---|---|
| REQ-001 / AC-001 | S01 | draft-variant rejection test + mutation |
| REQ-002 / AC-002 | S02 | partial-body preservation test + mutation |
| REQ-003 / AC-003 | S03 | guest-note-strip test + mutation |
| REQ-004 / AC-006 | S04 | six partial-update tests + mutations |
| REQ-005 / AC-004 | S05 | live API re-verification |
| REQ-005 / AC-005 | S05 | verify chain |
