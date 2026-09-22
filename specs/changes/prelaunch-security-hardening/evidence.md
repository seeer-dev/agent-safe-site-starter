# Prelaunch Security Hardening Evidence

Change ID: prelaunch-security-hardening  
Revision: 3  
Status: Applying

## Delivery status

Implementation complete across packets S01–S08. Local gates pass; live/provider
checks are organized below as a holder-authorized checklist because they
mutate or read provider state and are not implied by applying this change.

## Decision record

- D-001: owner selected PostgreSQL authoritative counters, trusted opaque Pages edge identity, Go-side Turnstile verification for side-effecting forms, and optional WAF prefiltering.
- D-002: owner selected retention of intentional public `pages.dev` test URLs for open-source proof, with no committed production domain, Railway origin, full provider identifiers, private recipients, credentials, database URIs, or deployment receipts.

## Discovery record

The 2026-09-21 bounded red-team review established the proposal inputs below. These observations justify scope but do not satisfy the future acceptance criteria:

- protected admin samples denied unauthenticated access and the media presign endpoint denied unauthenticated access;
- direct origin edge authentication and malicious-origin CORS behavior failed closed in the sampled checks;
- a public `manual_test` method could be exposed by database state and complete a real unpaid order with stock/mail side effects;
- public contact, comment, quote, and order routes had no common deployment-wide abuse policy;
- storefront Pages had security headers while the admin Pages root lacked the equivalent CSP/frame/nosniff contract;
- article `body_html` reached `template.HTML` under a trusted-CMS assumption without sanitization;
- `curatory_recent_orders` persisted order access tokens in localStorage;
- production dependencies audited clean, while the full admin dependency graph reported critical/high development-chain advisories through Vitest/Vite/esbuild;
- the temporary public R2 image had the expected JPEG type and immutable cache policy but no observed `nosniff` header;
- reusable deployment skill material contained no observed credentials, while installation documents retained exact non-secret provider/deployment identifiers under the active ledger policy;
- focused Go tests and the existing frontend test suite passed during discovery.

Sensitive values, private recipients, complete order IDs/tokens, database URIs, and provider credentials are intentionally omitted.

## Implementation receipts (local)

### S01/S02 — edge identity, shared limiter, Turnstile

- `functions/api/[[path]].js` strips caller-supplied `X-Edge-Client-Key` and stamps an HMAC-SHA256 key over `CF-Connecting-IP` + `EDGE_SECRET` before forwarding; raw client addresses never reach Go.
- `server/internal/platform/abuse` implements an `abuse_buckets` fixed-window limiter over the application database (`db/migrations/{sqlite,postgres}/019_abuse_control.sql`), atomic `INSERT … ON CONFLICT … RETURNING`, bounded cleanup, and `RetryAfter` reporting.
- `server/internal/bootstrap/abuse_guard.go` enforces the stamped 64-hex key when the edge credential is configured (403 otherwise), hashes the transport peer in local no-edge mode, maps store failure to 503, and returns 429 + `Retry-After` over quota before any side effect.
- Wired buckets: contact 3/10m, comment 5/10m, quote 60/m, order 5/10m shared by guest and member paths.
- `server/internal/platform/turnstile` verifies Siteverify with strict action/hostname match; production without `TURNSTILE_SECRET_KEY` fails closed; provider failure maps to 503 without provider detail.
- `TURNSTILE_SITE_KEY` is exposed through the storefront bootstrap only; the Pages CSP gains `challenges.cloudflare.com` (script/frame/connect) only when the key is configured.
- Tests: `go test ./server/internal/platform/abuse ./server/internal/platform/turnstile ./server/internal/bootstrap ./server/internal/modules/commerce ./server/internal/modules/contact ./server/internal/config` — all ok. Named coverage includes limiter windows, shared-instance consistency, invalid key/bucket rejection, store-failure 503, edge identity fail-closed, local peer hashing, Siteverify action/hostname checks, provider-outage failure, and rejection-before-persistence for contact/comment/guest/member order.
- Falsifiable mutations observed: skipping the Turnstile call in `commerce/http.go` made the rejection tests fail; removing the control-character guard made `security-headers` test 13 fail; both restored and re-verified green.

### S03 — manual_test activation window

- `MANUAL_TEST_CHECKOUT_UNTIL` parses once at startup: empty/malformed/past disables the method; production caps the window at 24h, non-production at 30d.
- One predicate governs public listing, quote, guest order, and member order; the value never enters the browser bundle.
- `manual_test_window_test.go` covers disabled-without-window, expired, inside-window availability, and shared predicate across list/quote/order.

### S04 — admin `_headers`

- `admin/scripts/security-headers.mjs` emits `admin/dist/_headers` during every production `vite build` (closeBundle plugin in `vite.config.ts`); CSP derives only from validated `SUPABASE_URL`/`ADMIN_API_BASE` (same-origin `/api` or an explicit `…/api` cross-origin prefix).
- Rejected inputs fail the build: non-loopback `http`, userinfo, query/fragment, non-`/api` paths, control characters, malformed URLs. Verified: `SUPABASE_URL='javascript:alert(1)' npx vite build` exits 1.
- Emitted policy: `default-src 'self'`, `script-src 'self'`, `connect-src 'self' + exact origins`, `frame-ancestors 'none'`, `object-src 'none'`, `base-uri 'self'`, `form-action 'self'`, plus `nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `X-Frame-Options: DENY`. No wildcards; no provider IDs hardcoded.
- `node --test admin/scripts/security-headers.test.mjs`: 15/15 pass (mutation evidence above). `admin/dist/_redirects` remains absent.

### S05 — article HTML sanitization

- `server/internal/modules/content/sanitize.go` defines the single allowlist policy (bluemonday v1.0.27): semantic block/inline text, headings, lists, block quotes, tables, figures, and `http`/`https`/`mailto` links with `rel=nofollow`; scripts, styles, forms, frames, objects, embeds, base, event attributes, unsafe schemes, and media surfaces are dropped. Output is bounded via truncate-then-resanitize so emitted markup is never cut mid-tag.
- `Publish` sanitizes before `Store.Upsert`; the renderer sanitizes again before `template.HTML` for legacy rows.
- Named tests: `TestSanitizeBodyHTMLRemovesActiveContent` (16 payload cases), `TestSanitizeBodyHTMLPreservesSemanticMarkup`, `TestSanitizeBodyHTMLBoundOutput`, `TestPublishSanitizesBeforePersist`, `TestRenderArticleSanitizesLegacyRow` — all pass under `go test -json` with per-test `pass` events.
- Falsifiable mutations observed: removing the render-boundary sanitize failed `TestRenderArticleSanitizesLegacyRow`; removing the write-boundary sanitize failed `TestPublishSanitizesBeforePersist`; both restored and re-verified green.

### S06 — session-bound order credentials

- `shared/lib/store.ts`: `RecentOrder` is `{orderId, date}` only; `loadRecentOrders` strips legacy `token` fields and re-saves the de-identified list; `rememberOrder` stores metadata only; `findRecentToken` removed.
- `curatory_last_order` stays in `sessionStorage` for same-session continuity; `OrderPage`/`TrackPage` no longer perform automatic token lookup — recent chips prefill only the order ID and focus the query-code field.
- Vitest (5.0.1) tests in `shared/lib/store.test.ts`: 5/5 pass, including legacy-token strip/re-save, metadata-only writes, dedupe/cap, and malformed payloads. Mutation keeping `token` in the mapped record failed 2 tests; restored green.
- Bundle scan of `dist/assets/islands/chunks/store-*.js` confirms the recent-orders writer emits `{orderId, date}` objects only.
- Browser walkthrough (local static serve of rendered `dist/`): seeded legacy `curatory_recent_orders` with tokens → after `/track/` load the stored list contains no tokens, chips render orderId+date only; clicking a chip prefills `#track-order`, leaves `#track-token` empty, issues no API call, and moves focus to the query-code field.

### S07 — admin dependency chain

- `vitest` 2.1.9 → 5.0.1, `vite` 6.0.5 → 7.3.6, `@vitejs/plugin-vue` 5.2.1 → 6.0.9; `overrides` pins transitive `happy-dom` to ≥20.8.9 (installed 20.14.5). Resolved `esbuild` is 0.28.2.
- `npm audit`: 0 vulnerabilities (was 1 high + 2 critical + 3 moderate).
- `npm test`: 22 files / 254 tests pass; `vue-tsc --noEmit` and `vite build` pass. One test fix was required: `MediaUploader.test.ts` Worker mock now uses a `function` implementation so `new Worker()` is constructible under Vitest 5.
- Pre-existing failure noted (not caused by this change, out of scope): `admin/scripts/check-resource-contracts.mjs` rejects the tuple-format `opts` already committed on `origin/staging`; tracked as a follow-up because the script is outside `applies_to`.

### S08 — de-identification

- `docs/deployment-guide.html` and `docs/environment-configuration.md`: Railway origin, workspace/project/service/deployment UUIDs, and the installation bucket name replaced with `<railway-*>`, `<*-pages-deployment-id>`, `<r2-bucket-name>` placeholders (29 substitutions). `pages.dev` test URLs retained and already labelled as intentional public proof.
- Residual scan over `docs/`, `README.md`, `skills/`, env examples, contracts, functions, server, site sources, admin sources: no UUIDs, Railway origins, Resend IDs, `r2.dev` public hostnames, or private recipients remain (hits are generic placeholders and test fixtures).

## Holder-authorized live checklist — round 1 executed

Live round executed against `849678a` (staging → Railway + both Pages):

- `GET https://agent-safe-site-starter.pages.dev/api/products` → 200 (edge proxy + Go origin healthy).
- `GET /api/storefront/bootstrap` → 200 and carries the new `turnstile_site_key` field → confirms the new build is serving.
- Storefront `_headers` live: CSP `default-src 'self'`, `script-src 'self'`, `img-src 'self' data: <r2 origin>`, `connect-src 'self' <site> <supabase>`; `challenges.cloudflare.com` correctly absent while `TURNSTILE_SITE_KEY` is unset in the build env.
- Admin `_headers` live: CSP `default-src 'self'`, `connect-src 'self' <supabase>`, `frame-ancestors 'none'`, `object-src 'none'`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`.
- Admin deep link `GET /login` → 200; unauthenticated `GET /api/admin/me` → 401.
- `manual_test` absent from `payment_methods` and bootstrap payload while `MANUAL_TEST_CHECKOUT_UNTIL` unset (fail-closed ✓).
- Unguarded `POST /api/orders/fake-id/payments/ecpay` → 400 (handler reached; DB reachable).
- Guarded POSTs (`/api/contact`, `/api/quote`) → uniform 503 `service_unavailable`; 65 sequential quote POSTs never returned 429.

**Finding:** the guard is reaching `limiter.Allow` (identity resolves — missing/malformed key would be 403) but the shared store errors on every call → `abuse_buckets` is missing or unwritable in production Postgres, i.e. migration 019 did not apply during Railway `preDeployCommand`. This is the designed fail-closed posture (public writes are refused, zero side effects), but it currently takes contact/comment/quote/order endpoints offline. Holder action required: check the Railway deploy log's `migrate` step output, or re-run `railway run migrate` / apply `db/migrations/postgres/019_abuse_control.sql` via Supabase SQL editor, then re-run the 429 checks.

**Still pending (holder/provider actions):**

- Post-migration replay: 4+ contact POSTs → 429 + `Retry-After`; Turnstile absent-token → 403 (requires `TURNSTILE_SECRET_KEY` on Railway + `TURNSTILE_SITE_KEY` in the Pages build env); direct-origin hit without edge credential → 403.
- R2 custom domain (REQ-007/AC-008): create the custom domain + optional transform for `nosniff`, point `R2_PUBLIC_BASE_URL` at it, run bounded HEAD/GET twice for Content-Type/immutable/nosniff/cache transition, then update the storefront CSP origin.
- `MANUAL_TEST_CHECKOUT_UNTIL` remains unset for release; final acceptance replay (REQ-009/AC-010) after the above.

## Acceptance evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Shared-DB limiter + HMAC edge identity + Turnstile Siteverify wired; named tests pass; mutations observed. |
| REQ-002 | passed | MANUAL_TEST_CHECKOUT_UNTIL parsed at startup; single predicate gates list/quote/orders; bounded windows tested. |
| REQ-003 | passed | dist/_headers emitted each prod build; strict origin validation fails closed; 15/15 node tests; no wildcards. |
| REQ-004 | passed | bluemonday allowlist at write+render boundaries; 21 named tests pass; dual mutation evidence. |
| REQ-005 | passed | RecentOrder metadata-only; legacy tokens stripped+resaved; sessionStorage keeps fresh token; vitest 5/5 + walkthrough. |
| REQ-006 | passed | vitest 5.0.1/vite 7.3.6/plugin-vue 6.0.9; happy-dom >=20.8.9 override; audit 0 vulns; 254 tests pass. |
| REQ-007 | pending | R2 custom domain + nosniff transform are holder/provider actions; live checklist item 2. |
| REQ-008 | passed | 29 placeholder substitutions in docs; residual scan clean; pages.dev retained as documented proof. |
| REQ-009 | pending | Local gates green (verify ok); live round 1 executed — admin CSP live, fail-closed posture confirmed; 429 replay blocked on prod migration 019 (holder action). |
| AC-001 | passed | Shared-instance limiter test + stamped-identity/peer-hash guard tests + Turnstile-required tests pass; replica-consistent via shared DB. Live multi-replica spot-check listed. Security review receipt: receipts/security-review.md. |
| AC-002 | passed | Rejection-before-persistence tests for contact/comment/order; 429+Retry-After; 503 on store/verifier failure. Security review receipt: receipts/security-review.md. |
| AC-003 | passed | Absent/malformed/expired/over-cap windows disable manual_test in tests; inside-window availability verified; value never bundled. Security review receipt: receipts/security-review.md. Consumer reachability receipt: receipts/consumer-reachability.md. |
| AC-004 | passed | Build emits _headers (inspected); live round: admin pages.dev serves CSP/nosniff/frame-deny/referrer headers, deep link 200, unauth /api/admin/me 401. Security review receipt: receipts/security-review.md. Walkthrough receipt: receipts/walkthrough.md. |
| AC-005 | passed | Write + render sanitize tests incl. legacy-row render proof; no raw fallback path exists. Security review receipt: receipts/security-review.md. |
| AC-006 | passed | Vitest strip/re-save tests + browser walkthrough: later session shows metadata only and requires the query code. Security review receipt: receipts/security-review.md. Walkthrough receipt: receipts/walkthrough.md. |
| AC-007 | passed | npm audit 0 high/critical; tests/typecheck/build pass; no advisories suppressed. Security review receipt: receipts/security-review.md. |
| AC-008 | pending | Holder must create R2 custom domain and run bounded header/cache checks (checklist item 2). |
| AC-009 | passed | Fresh-tree scans find no installation identifiers; instructions keep per-installation placeholders; pages.dev URLs documented as proof. Security review receipt: receipts/security-review.md. Production content audit receipt: receipts/production-content-audit.md. |
| AC-010 | pending | Local loop verified; final live acceptance awaits holder authorization. |
