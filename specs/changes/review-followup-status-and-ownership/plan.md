# Architecture Review Follow-up Delivery Plan

Change ID: review-followup-status-and-ownership
Revision: 1
Status: Accepted

Repository baseline: `10f805e18e394e7065854dfbceb327114e0d4564`

## Scope lock

- `README.md`
- `architecture.yaml`
- `docs/README.md`
- `docs/project-status.md`
- `docs/review-status.md`
- `specs/changes/review-followup-status-and-ownership/**`

## Delivery

Covers REQ-001, AC-001, REQ-002, AC-002, REQ-003, AC-003.

1. Refresh the root/canonical project-status wording and add HTTP contract truth restoration as a separate v1 blocker before provider/deployment acceptance.
2. Add `ecpay_payment_attempts` to commerce ownership and replace the stale line-number reference for the known commerce-to-media SQL write with the current `store_catalog.go:markMediaAssociationsTx` location.
3. Add a current review overlay that records which historical findings were resolved by Accepted changes, which controlled changes still have pending or blocked evidence, and which old Drafts must be re-proposed rather than applied directly.
4. Route the root README and documentation index through the canonical status/review overlay while retaining the historical review documents for audit context.
5. Do not change runtime behavior, API envelopes, generated client types, rate-limit topology, JWKS policy, or module boundaries in this change.
