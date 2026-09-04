# Fresh DB Commerce Acceptance Delivery Plan

Change ID: fresh-db-commerce-acceptance
Revision: 1
Status: Applying
Repository baseline: `fc37f411d7906380163a3c37b4b01379148ae0df`

## Scope lock

- `server/tools/dev/main.go`
- `server/tools/dev/commerce_seed.go`
- `server/tools/dev/commerce_seed_test.go`
- `server/tools/dev/commerce_seed_config_test.go`
- `server/internal/modules/commerce/payment_methods.go`
- `server/internal/modules/commerce/payment_methods_test.go`
- `README.md`
- `docs/project-status.md`
- `docs/commerce-acceptance.md`
- `specs/changes/fresh-db-commerce-acceptance/**`

Covers REQ-001, AC-001, REQ-002, AC-002, AC-003, AC-004, REQ-003, AC-005, REQ-004, AC-006.

## Slice 1 — Make sample seed checkout-capable

- Move the existing commerce seed block in `server/tools/dev/main.go` into the private `commerce_seed.go` helper used by the same dev entry point.
- Preserve the existing six products and current non-ECPay payment samples.
- Add at least one enabled shipping method when no shipping methods exist.
- Ensure an ECPay admin row exists; derive sandbox/production, readiness, and enabled state from the same validated runtime ECPay configuration used by bootstrap.
- Keep the seed idempotent and avoid overwriting operator-managed rows on subsequent starts.

## Slice 2 — Make runtime payment availability truthful

- Add a narrow service helper for runtime availability of a configured payment row.
- Non-ECPay rows continue to use the current enabled/readiness behavior.
- ECPay additionally requires `s.ecpay != nil` and environment correspondence (`sandbox`↔`stage`, `production`↔`production`).
- Reuse the same helper in public discovery and checkout validation so the two entry points cannot drift.

## Slice 3 — Fresh SQLite acceptance tests

- Apply migrations to a temp SQLite database.
- Run the real dev commerce seed helper.
- Assert seeded in-stock product, enabled shipping method, and usable non-ECPay payment discovery.
- Quote a seeded product and create a guest order; assert server-owned totals, unpaid payment state, stock change, and one-time access token.
- Run seed with no ECPay runtime and assert pending/disabled ECPay row plus public exclusion.
- Run a separate fresh DB with valid stage ECPay runtime and assert ready/enabled sandbox ECPay row plus public inclusion.
- Falsify configured-but-invalid ECPay runtime (local HTTP origins and production public test credentials) so it cannot be seeded ready.
- Unit-test database-ready ECPay rows against missing/mismatched/matching runtime configs.

## Slice 4 — Status and acceptance

- Update README/project/commerce acceptance status only after the deterministic tests are evidenced.
- Keep public HTTPS ECPay stage transaction and production-shaped deploy acceptance pending.
- Record the current GitHub Actions trigger incident separately under Issue #17; do not claim a CI run that did not occur.
