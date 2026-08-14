# Independent Auth Walkthrough Receipt (Revision 9)

Date: 2026-08-13

## Scope and boundary

This receipt records Codex's independent acceptance pass for the revision 9
Supabase Auth frontend integration. Grok 4.6 implemented the code through
Herdr; Codex inspected the implementation, reproduced defects, returned three
focused remediation rounds, reran the checks, and performed the local browser
walkthrough.

The local `.env` used `AUTH_MODE=dev` and had a configured development token,
but `SUPABASE_URL` and `SUPABASE_PUBLISHABLE_KEY` were empty. No Supabase
credential or real user password was available. This receipt therefore does
not claim a live provider sign-in, refresh, sign-up email, staff-row match, or
member-order success against Supabase. It covers deterministic automated
integration evidence plus local unavailable/invalid-session browser states.

## Implemented path

- Public and admin builds use `@supabase/supabase-js` as the session producer.
- Browser code never queries Supabase Database/PostgreSQL; bearer tokens go
  only to the Go API, which remains the authentication and authorization
  boundary.
- The public user store receives identity and bearer state only from the
  Supabase session synchronizer; legacy custom user persistence and arbitrary
  user/token setters are absent.
- Member checkout uses `POST /api/orders/mine`; account history uses
  `GET /api/orders/mine`; listed member order detail uses
  `GET /api/orders/mine/{id}` with the current bearer token and does not ask
  for the guest `X-Order-Access-Token`.
- Member list/detail decoding rejects malformed envelopes, IDs, statuses,
  amounts, quantities, and timestamps rather than fabricating empty, pending,
  zero-value, or current-time results.
- Admin uses one session adapter across initialize/sign-in/logout. `/admin/me`
  results are generation/token-bound so stale refresh and post-logout
  responses cannot restore identity. A 401 clears local identity and calls
  the provider session sign-out.
- Rendered CSP `connect-src` includes only a validated Supabase origin and
  fails closed for invalid configured origins.

## Browser observations

### Public site (`http://localhost:4173/`)

- The static site rendered normally and the account entry opened.
- With Supabase unset, the account view showed `會員登入尚未開放` and truthfully
  stated that guest checkout remains available.
- The account close control exposed the accessible name `關閉帳號視窗`.
- Escape closed the dialog.
- No browser console warnings or errors were observed during the covered path.

### Admin (`http://127.0.0.1:5173/`)

- Dev mode rendered a development-token form and explicitly stated that the
  token remains in memory instead of custom browser storage.
- Submitting the fabricated local token `invalid-local-token` caused the Go
  API to reject authentication, cleared the session, and displayed the
  generic `role=alert` message `無法驗證身分，請重新登入。`.
- No backend response body, secret, or credential detail appeared in the UI.

## Mechanical verification

- `admin: npm test`: 10 files, 140 tests passed.
- `admin: npm run build`: passed.
- `admin: npm run check:resource-contracts`: passed.
- `theme: npm run build:check`: passed.
- `theme: npm run check:auth-session`: passed, including the member-order
  envelope and bearer-detail flow assertion.
- `theme: npm run check:browser-authority`: passed.
- `theme: npm run check:openapi-contracts`: passed.
- `theme: npm run check:checkout-idempotency`: passed.
- `go test ./server/internal/render ./server/tools/dev ./server/tools/render -count=1`: passed.
- `go vet ./server/...`: passed.
- Second `go test ./server/... -count=1`: passed after the independently
  recorded intermittent restock ordering failure and 10 focused passes.
- `go run ./server/tools/render`: passed; rendered 1 article, 5 products,
  4 categories, and 1 content page.
- `npm audit --omit=dev`: zero vulnerabilities in both admin and theme.
- `go run ./server/tools/verify`: `archcheck` and `speccheck` passed, then
  `scopecheck` stopped on the repository's pre-existing dirty paths outside
  this revision's narrow `.ai/scope.json`; no full verify pass is claimed.
- Production-bundle search found no `DEV_AUTH_TOKEN` or `admin_token` marker.

One full non-race Go run exposed a pre-existing commerce restock concurrency
ordering failure unrelated to the frontend Auth diff. The exact failing test
then passed 10 consecutive focused repetitions, and a second full
`go test ./server/... -count=1` run passed. A later revision 9 slice reproduced
the failure 5 times in 100 runs, fixed the post-early-check idempotency ordering,
and independently passed 500 focused repetitions plus race coverage. See
`restock-idempotency-validation-rev9.md`. This paragraph remains as acceptance
history; the repository-level risk is now resolved locally.

## Result

Revision 9 member Auth consumer wiring and the covered local fail-closed states
are accepted. `REQ-009` and `AC-017` remain **pending** because secure guest
token recovery is still intentionally absent and a configured live Supabase
success-path walkthrough has not been observed.
