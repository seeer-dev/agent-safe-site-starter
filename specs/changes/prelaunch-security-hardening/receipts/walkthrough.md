# Walkthrough Receipt — AC-006 (guest order token retention)

Date: 2026-09-22
Environment: locally rendered `dist/` served statically at 127.0.0.1:8091 (no backend needed; the TrackPage island mounts client-side).

## Steps observed in a real browser session

1. Seeded legacy `localStorage.curatory_recent_orders` with two entries containing `token` fields (`LEGACY-SECRET-AAAA/BBBB`).
2. Reloaded `/track/`. The TrackPage island mounted and rendered both recent-order chips (`TW-LEGACY1`, `TW-LEGACY2`) showing order ID + date only.
3. Read `localStorage` again: the stored list was re-saved de-identified — no `token` keys, no `SECRET` substrings.
4. Clicked the `TW-LEGACY1` chip: `#track-order` was prefilled with the order ID, `#track-token` remained empty, no `/api/orders` request was issued (fetch hook observed zero calls), and focus moved to the query-code field.

## Result

- A later/new session shows recent-order metadata but requires the query code before any protected read — matches AC-006.
- Same-session continuity for a just-created order is preserved through `sessionStorage.curatory_last_order` (unchanged code path in `OrderPage`).
- Unit-level coverage: `shared/lib/store.test.ts` 5/5 (strip, metadata-only write, dedupe/cap, malformed input).

## Live round (AC-004) — 2026-09-22, commit 849678a

- `curl -sI https://agent-safe-site-starter-admin.pages.dev/` → headers include `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' <supabase origin>; frame-ancestors 'none'; object-src 'none'; base-uri 'self'; form-action 'self'`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin` — the build-emitted `_headers` is active on the deployed admin site.
- `curl -s -o /dev/null -w %{http_code} https://agent-safe-site-starter-admin.pages.dev/login` → 200 (SPA deep link works without `_redirects`).
- `curl -s -o /dev/null https://agent-safe-site-starter-admin.pages.dev/api/admin/me` (no credentials) → 401 — admin Function proxy + origin auth intact.
- Storefront CSP check: `curl -sI https://agent-safe-site-starter.pages.dev/` shows `script-src 'self'` with `challenges.cloudflare.com` correctly absent while `TURNSTILE_SITE_KEY` is unset in the build env.
