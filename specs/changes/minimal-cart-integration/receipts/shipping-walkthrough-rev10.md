# Revision 10 shipping-authority acceptance

Date: 2026-08-13 (Asia/Taipei)

Reviewer: Codex, independent of the delegated Grok Build implementation

Scope: revision 10 slices 1-3; `REQ-004`, `REQ-008`, `REQ-011`, `REQ-012`,
`AC-007`, `AC-015`, `AC-021`, `AC-022`, `AC-023`, `AC-024`, and `AC-027`.

## Environment and isolation

- Ran the local Go API, static visitor site, and admin Vite app against a
  dedicated SQLite database under `.tmp/`.
- Applied the full SQLite migration chain through migration 016.
- Used the existing development verifier only for the local authorized-admin
  path. No Supabase configuration was needed or substituted.
- Did not print or persist the development token, order access token, or other
  credentials in this receipt.

## Independent walkthrough

1. Public shipping discovery returned zero rows before configuration.
2. The admin UI showed its real empty state and created one method through the
   Go admin API. The public API then returned one enabled method without fee or
   threshold fields.
3. Quote used subtotal 420, shipping 80, total 500 from the persisted row.
4. An admin update kept the method key immutable, changed the fee to 90, and
   advanced the version from 1 to 2. Replaying a stale version returned 409 and
   did not overwrite the row.
5. Order creation after that quote used the current fee 90, producing total
   510. This confirms the quote snapshot did not become browser authority.
6. Disabling the row advanced it to version 3. Public discovery returned zero
   rows and a new quote for that method returned 400.
7. Replaying the already-created order with the same idempotency key returned
   the same order and original shipping 90. Stock changed from 2 to 1 only
   once, and the replay response did not disclose an access token.
8. The admin UI independently showed the same row at fee 90, threshold 1000,
   disabled, version 3. The visitor checkout showed no available shipping
   method and disabled progression. Browser console warning/error logs were
   empty for the observed states.

The runtime browser walkthrough directly covered empty, success, disabled, and
visitor-unavailable UI states. Stale conflict was replayed over the real HTTP
boundary. Deterministic service/HTTP/frontend tests cover loading, store-error,
unauthorized/forbidden, validation, and conflict recovery states that were not
artificially injected into the browser session; `AC-015` remains pending for
its broader non-shipping acceptance scope.

## Automated replay

- `go test -race ./server/... -count=1`: passed, including commerce,
  migrations, auth, render, and tools.
- `go vet ./server/...`: passed.
- Shipping-focused commerce tests cover migration apply, SQLite boolean
  binding, deterministic/admin/public lists, auth, validation, duplicate and
  stale updates, disabled/missing methods, store failures, zero/no-threshold/
  boundary fees, changed-after-quote behavior, quote/order/member parity,
  persisted-fee overflow, HTTP 400/503 mapping, side effects, and idempotency.
- Admin `npm test -- --run`: 148 tests passed; `npm run typecheck`, `npm run
  build`, and `npm run check:resource-contracts` passed.
- Theme `npm run build:check`, `check:browser-authority`,
  `check:openapi-contracts`, `check:checkout-idempotency`, and
  `check:auth-session`: passed.
- `go run ./server/tools/migrate`: passed against the original SQLite database.
- `go run ./server/tools/render`: rendered 0 articles, 5 products, 4
  categories, and 1 content page after restoring the original data source.
- `git diff --check`: passed. Relevant Go shipping files are `gofmt` clean.
- `go run ./server/tools/speccheck`: passed after the controlled evidence edit
  (`2 controlled spec(s), 385 protected changed file(s)`).
- `go run ./server/tools/verify`: `archcheck` and `speccheck` passed, then
  `scopecheck` stopped on the pre-existing repository-wide dirty tree outside
  this slice's narrow `.ai/scope.json`; no full verifier pass is claimed.

## Cleanup and limits

- Removed the exact temporary shipping fixture from the pre-existing local
  database.
- Stopped both local servers, verified ports 8080 and 5174 were closed, removed
  the isolated SQLite database and sidecars, and removed the empty `.tmp/`
  directory.
- Re-rendered `dist/` from the original database. A negative scan found no
  acceptance method, test email, walkthrough description, or isolated order
  key in `dist/`.
- Live PostgreSQL migration 016 and concurrency behavior remain unverified
  because no PostgreSQL credentials are configured; this remains tracked by
  `AC-022` and does not invalidate the local `AC-027` authority result.
- The overall controlled change remains `Applying` because unrelated Supabase,
  R2/Cloudflare, PostgreSQL, retention, and other acceptance items remain open.
