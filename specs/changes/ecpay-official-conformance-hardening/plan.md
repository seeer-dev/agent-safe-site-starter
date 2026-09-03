# ECPay Official Conformance Hardening Delivery Plan

Change ID: ecpay-official-conformance-hardening
Revision: 2
Status: Verifying
Repository baseline: `0071aebecf8b6939308e7965fb5f0a07797d8579`

## Scope lock

- `server/internal/modules/commerce/ecpay.go`
- `server/internal/modules/commerce/ecpay_payment.go`
- `server/internal/modules/commerce/ecpay_test.go`
- `server/internal/modules/commerce/ecpay_official_conformance_test.go`
- `contracts/openapi.yaml`
- `contracts/check-runtime-openapi.mjs`
- `README.md`
- `docs/project-status.md`
- `docs/commerce-acceptance.md`
- `docs/ecpay-official-conformance.md`
- `specs/changes/ecpay-official-conformance-hardening/**`

Covers REQ-001, AC-001, REQ-002, AC-002, REQ-003, AC-003, REQ-004, AC-004, AC-005.

## Slice 1 — Pin official sources and audit current behavior

- Pin the ECPay API Skill master commit used for review.
- Read current AIO, CheckMacValue, HTTP protocol, webhook, troubleshooting, go-live, and Developers reference material.
- Record verified mismatches rather than inferring conformance from the existence of an integration.

## Slice 2 — Fix protocol mismatches

- Parse callback amount from `TradeAmt` rather than request-side `TotalAmount`.
- Recognize optional `SimulatePaid`; verify and acknowledge a simulation without consuming the durable provider callback claim, so a later real callback can still capture.
- Acknowledge signed non-success observations without consuming the durable success claim.
- Add the missing apostrophe encoding rule to CheckMacValue.
- Accept implicit HTTPS 443 or explicit `:443`; reject other HTTPS ports, direct IP hosts, non-HTTPS schemes, and unencoded Unicode hostnames.
- Keep existing merchant/amount/trade correlation and durable replay/conflict protection for real callbacks.

## Slice 3 — Lock with official vectors and contract truth

- Add official SHA256 CheckMacValue vector tests for baseline, apostrophe, tilde, spaces, and callback fields.
- Convert callback tests to the official `TradeAmt` shape.
- Add simulation/pending→real-callback, origin, optional-SimulatePaid, and invalid-SimulatePaid tests.
- Update `ECPayCallbackForm` in OpenAPI and mechanically guard the corrected shape.

## Slice 4 — Provider-conformance acceptance

- Independently replay the protocol-critical Go behavior against the pinned official vectors and callback rules.
- Falsify wrong amount, signature tamper, wrong merchant identity, invalid SimulatePaid, non-standard HTTPS port, and simulation/pending authority paths.
- Preserve the existing Accepted durable one-effect/conflict boundary; this change does not modify `store_ecpay.go`.
- Update README/project/commerce status to distinguish source-level official conformance from the still-pending public stage transaction.
- Mark the change Accepted only after every REQ/AC has revision-2 evidence.

## Slice 5 — Repository delivery verification

- After the controlled change is Accepted, run the normal PR CI chain: gofmt, architecture, theme build, contract gate, migration parity, Accepted-only speccheck, PostgreSQL migrations, repository tests, live PostgreSQL, stress, and `go vet`.
- PR/main CI is a delivery regression gate, not a substitute for the provider-protocol acceptance evidence above. If GitHub Actions does not create a PR run, record that limitation explicitly; do not fabricate a green run.
- Merge only after the repository delivery risk is judged acceptable, then verify the merge-triggered `main` CI. If `main` CI fails, fix it immediately from a new clean branch.

## Revision note

Revision 2 corrects only the callback-port interpretation: explicit standard HTTPS `:443` is valid; non-443 HTTPS ports remain invalid. All strict evidence is re-observed for revision 2.
