# Revision 11 OAuth local validation

Date: 2026-08-14  
Scope: Google and LINE OAuth initiation through Supabase Auth on the public
account dialog and admin sign-in gate.

## Implemented boundary

- Provider input is typed and allowlisted to `google` and `custom:line`.
- Both browser surfaces hide Google and LINE until the corresponding non-secret
  `AUTH_GOOGLE_ENABLED` or `AUTH_LINE_ENABLED` flag is explicitly true and
  Supabase browser configuration is complete.
- Initiation returns only to `window.location.origin`; paths, queries, and
  fragments are rejected.
- LINE requests `openid profile email`; the operator runbook requires LINE
  email-address permission and PKCE S256 before enabling it.
- Application code does not query Supabase Database, use a service-role key, or
  retain provider access/refresh tokens. Go remains the canonical member/staff
  authorization boundary.

## Independent local checks

- `npm test` in `admin/`: passed, 12 files and 152 tests.
- `npm run typecheck`, `npm run check:resource-contracts`, and `npm run build`
  in `admin/`: passed.
- `npm run check:auth-session`, `npm run check:browser-authority`,
  `npm run check:openapi-contracts`, `npm run check:checkout-idempotency`,
  `npm run typecheck`, and `npm run build:check` in
  `site/themes/minimal-cart/`: passed.
- `go run ./server/tools/render`: passed; regenerated 44 files (565030 bytes)
  in `dist/`.
- `go run ./server/tools/archcheck` and `go run ./server/tools/speccheck`:
  passed.
- Generated bundles contained no `DEV_AUTH_TOKEN`, service-role, Google client
  secret, LINE channel secret, or Supabase service-role identifier. OAuth source
  contains no provider-token persistence.

## Not yet accepted

- Configure Google in Google Cloud and Supabase, then configure LINE Custom
  OIDC in Supabase after LINE email-address permission is approved. Follow
  [`docs/supabase-oauth.md`](../../../docs/supabase-oauth.md).
- Replay each provider's cancel/failure and successful callback at an allowed
  origin. Verify public member reachability, unlinked admin denial, linked
  active-staff admin success, logout, and no secret leakage in the deployed
  bundle/logs.
- `go test ./...` is not fully green: the out-of-scope existing commerce test
  `TestConcurrentOrderCancellationRestocksOnce` failed with `invalid order
  status transition`. It is not attributed to this browser-only OAuth slice.
- `go run ./server/tools/scopecheck` remains blocked by the repository's broad
  pre-existing dirty worktree, which contains files outside this revision's
  narrow scope.
