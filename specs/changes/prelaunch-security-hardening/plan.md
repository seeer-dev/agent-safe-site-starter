# Prelaunch Security Hardening Plan

Change ID: prelaunch-security-hardening  
Revision: 3  
Status: Applying

Normative specification: [`spec.md`](spec.md)

## Scope lock

Revision 3 authorizes implementation. D-001 and D-002 are resolved, the deployment ledger reached a terminal clean handoff on 2026-09-22, and the owner approved application. Documentation/skill paths are now in scope for Packet S08.

Implementation scope is listed in `control.json`:

- `.env.example`
- `.env.development.example`
- `.env.production.example`
- `contracts/openapi.yaml`
- `go.mod`
- `go.sum`
- `db/migrations/sqlite/019_abuse_control.sql`
- `db/migrations/postgres/019_abuse_control.sql`
- `functions/api/**`
- `server/internal/bootstrap/**`
- `server/internal/config/**`
- `server/internal/modules/contact/**`
- `server/internal/modules/commerce/**`
- `server/internal/modules/content/**`
- `server/internal/platform/abuse/**`
- `server/internal/platform/turnstile/**`
- `server/internal/render/**`
- `admin/package.json`
- `admin/package-lock.json`
- `admin/vite.config.ts`
- `admin/scripts/security-headers.mjs`
- `admin/scripts/security-headers.test.mjs`
- `admin/src/components/MediaUploader.test.ts`
- `server/tools/render/main.go`
- `site/templates/home.html`
- `site/themes/curatory/**`
- `site/assets/site.js`
- `README.md`
- `docs/**`
- `skills/site-deployment-browser/**`
- `specs/changes/prelaunch-security-hardening/**`

## Expansion context

- Proposal baseline and observed HEAD: `847c0253197ce4625ce9ab7ad52dadcf105c58aa`.
- Pre-existing dirty paths: `README.md`, `admin/public/_redirects`, `docs/deployment-guide.html`, `docs/environment-configuration.md`, `skills/site-deployment-browser/SKILL.md`, `skills/site-deployment-browser/references/**`, and `specs/changes/deployment-dashboard-ledger/**`. Preserve them.
- Superseded Draft: `public-endpoint-rate-limit`; revision 2 resolves its topology decision with the owner-selected hybrid.
- Existing provider runbook: `docs/r2-asset-origin-runbook.md`; S08 consumes it instead of inventing a second R2 procedure.
- Existing verification facts are discovery evidence, not acceptance evidence for this new revision.

## Packet graph

`S01 -> S02`  
`S02 -> S09`  
`S03 -> S09`  
`S04 -> S09`  
`S05 -> S09`  
`S06 -> S09`  
`S07 -> S09`  
`S08 -> S09`

S03-S07 may execute independently after approval. S01 blocks only the abuse-control implementation. S08 blocks only committed-document de-identification. S09 waits for every packet.

## Packets

### Packet S01: Build the trusted edge identity and shared limiter foundation

Status: READY_AFTER_APPROVAL  
Covers: REQ-001, AC-001, AC-002  
Hard dependencies: none  
Expected anchors: `functions/api/[[path]].js`, `server/internal/platform/abuse/**`, `db/migrations/{sqlite,postgres}/019_abuse_control.sql`, bootstrap/config tests

Inspect:

- preserve the existing Pages pass-through and edge-secret behavior;
- delete any incoming `X-Edge-Client-Key`, derive an HMAC-SHA256 opaque key from Cloudflare's edge client address using domain separation and `EDGE_SECRET`, then set the trusted header;
- never forward/log the raw address and never trust `X-Forwarded-For` at the origin;
- add matching SQLite/PostgreSQL migration 019 with a bounded fixed-window bucket table and expiry index;
- implement atomic portable counter increment, fixed starter budgets, bounded expired-row cleanup, 429/`Retry-After`, and fail-closed dependency behavior;
- consume the trusted key only after edge-secret middleware succeeds.

Verification:

- Function tests prove spoofed identity is overwritten and edge-secret behavior is unchanged;
- migration parity and clean SQLite/PostgreSQL apply;
- concurrency tests prove one shared counter decision and bounded cleanup;
- security review proves no raw client address/key disclosure and no in-memory fallback.

Safe stop: if the authenticated edge cannot supply a trustworthy opaque key, fail the protected route and do not fall back to forwarded headers.

### Packet S02: Implement and test pre-side-effect abuse rejection

Status: READY_AFTER_S01  
Covers: REQ-001, AC-001, AC-002  
Expected anchors: `.env*.example`, `server/internal/platform/turnstile/**`, `server/internal/bootstrap/app.go`, contact/commerce handlers and tests, curatory contact/comment/checkout islands, `contracts/openapi.yaml`

Work:

- place the approved policy before JSON decoding/business service calls where practical;
- cover contact, comments, quote, guest order, and member order;
- add public `TURNSTILE_SITE_KEY` and Railway-only `TURNSTILE_SECRET_KEY` boundaries; require both in production while permitting documented Cloudflare test keys locally;
- render a managed Turnstile widget for contact, comment, and checkout submissions, include a distinct expected action, and send the resulting token only with the corresponding request;
- validate Siteverify in Go, including success, expected hostname/action, single-use/expiry behavior, timeout, and provider failure; do not add a second Worker or accept browser-only success;
- add deterministic tests with spy/fake dependencies proving no mail, row, stock, order, notification, or idempotency mutation after rejection;
- document 429 and `Retry-After` without exposing the caller key;
- bound cardinality/state and record non-PII metrics.

Verification:

- targeted middleware/handler tests;
- mocked Siteverify tests for valid, invalid, replayed, expired, wrong-host, wrong-action, timeout, and provider-failure results;
- a real bounded Turnstile validation walkthrough only after widget/secret-holder authorization;
- security-review receipt with spoofing, concurrency, exhaustion, direct-API bypass, and outage cases.

### Packet S03: Add the expiring manual-test checkout gate

Status: READY_AFTER_APPROVAL  
Covers: REQ-002, AC-003  
Expected anchors: `.env*.example`, `server/internal/config/config.go`, `server/internal/modules/commerce/payment_methods.go`, checkout validation/tests, `server/internal/bootstrap/app.go`, `contracts/openapi.yaml`

Work:

- parse `MANUAL_TEST_CHECKOUT_UNTIL` as RFC3339 UTC once at startup;
- missing/invalid/past is disabled; production rejects a deadline more than 24 hours ahead;
- inject a clock/test gate into commerce through an explicit constructor option;
- apply one predicate to public listing, quote, guest order, and member order;
- keep admin CRUD able to store the row without treating that row as public authorization;
- ensure the setting never enters browser-safe environment allowlists.

Verification:

- config boundary tests for missing, malformed, past, valid, and overlong windows;
- commerce tests using a fixed clock across listing/quote/both order routes;
- browser bundle secret scan;
- consumer-reachability receipt tracing admin row -> public list -> quote -> order.

Safe rollback: unset the variable; the test method disappears and fails validation.

### Packet S04: Generate admin Pages security headers

Status: READY_AFTER_APPROVAL  
Covers: REQ-003, AC-004  
Expected anchors: `admin/vite.config.ts`, `admin/scripts/security-headers.mjs`, `admin/scripts/security-headers.test.mjs`

Work:

- generate `dist/_headers` during production build from validated `SUPABASE_URL` and `ADMIN_API_BASE`/same-origin behavior;
- reject userinfo, non-HTTPS remote origins, paths/query/fragment where disallowed, control characters, and malformed values;
- encode the exact CSP and companion headers from REQ-003;
- keep source free of provider IDs/hosts and do not restore the deleted invalid `_redirects` catch-all.

Verification:

- Node tests for valid same-origin/cross-origin policies and malicious build inputs;
- `npm --prefix admin run build` plus inspection of `admin/dist/_headers`;
- clean-browser walkthrough for root, deep route, Supabase login, role display, and unauthorized 401;
- live header capture after action-time deployment authorization.

### Packet S05: Sanitize article HTML at write and render boundaries

Status: READY_AFTER_APPROVAL  
Covers: REQ-004, AC-005  
Expected anchors: `go.mod`, `go.sum`, `server/internal/modules/content/service.go`, content tests, `server/internal/render/render.go`, render tests

Work:

- add `github.com/microcosm-cc/bluemonday` at an apply-time verified stable version;
- define one narrow policy in the content package and expose a small sanitizer function usable by the renderer without crossing module boundaries;
- allow semantic block/inline text and safe links; apply `AllowStandardURLs`, require safe link attributes, and omit style/script/form/frame/object/embed/base/media surfaces;
- sanitize before `Store.Upsert` and sanitize every legacy `Article.BodyHTML` immediately before `template.HTML`;
- preserve length checks on submitted raw input and bound sanitized output.

Verification:

- table tests for script/style/iframe/form/object/embed/base, event attributes, CSS/URL tricks, malformed markup, and nested payloads;
- preservation tests for headings, paragraphs, lists, emphasis, block quotes, and HTTP(S)/mailto links;
- render test that feeds an unsanitized legacy row and proves the emitted page is safe;
- full Go tests and fresh render inspection.

Safe rollback: rendering keeps sanitization even if write-time rollout is paused; raw fallback is forbidden.

### Packet S06: Remove persistent guest order tokens

Status: READY_AFTER_APPROVAL  
Covers: REQ-005, AC-006  
Expected anchors: `site/themes/curatory/shared/lib/store.ts`, `CheckoutPage.vue`, `OrderPage.vue`, `TrackPage.vue`, curatory package/test files

Work:

- change `RecentOrder` to order ID/date only and make the loader strip legacy `token` fields before returning/re-saving;
- keep only the just-created token in `sessionStorage` under the existing session key;
- remove `findRecentToken` and automatic token lookup from `OrderPage`/`TrackPage`;
- make recent-order actions prefill only the order ID and visibly request the query code;
- add focused storage/UI tests; use the same fixed Vitest line as S07 if a new curatory test dependency is needed.

Verification:

- unit tests with legacy localStorage fixtures and a new-order session fixture;
- bundle/static scan proving the recent-order writer has no token field;
- browser walkthrough: same-session confirmation works, new session requires a code, invalid code remains denied.

### Packet S07: Upgrade the admin test dependency chain

Status: READY_AFTER_APPROVAL  
Covers: REQ-006, AC-007  
Expected anchors: `admin/package.json`, `admin/package-lock.json`, test configuration only as required by the new major

Work:

- upgrade Vitest to a current compatible release with advisories fixed (minimum known target from discovery: 5.0.1; re-check at apply time);
- keep Vite on a compatible fixed line and remove unused optional `happy-dom` if it remains in the resolved graph;
- adapt tests/config only for documented breaking changes, without weakening assertions or adding audit ignores.

Verification:

- `npm --prefix admin ci`;
- `npm --prefix admin audit --audit-level=high`;
- `npm --prefix admin audit --omit=dev --audit-level=high`;
- `npm --prefix admin test`;
- `npm --prefix admin run typecheck`;
- `npm --prefix admin run check:resource-contracts`;
- `npm --prefix admin run build`.

### Packet S08: Harden the R2 release boundary and de-identify committed guidance

Status: READY (scope handoff cleared 2026-09-22)  
Covers: REQ-007, REQ-008, AC-008, AC-009  
Hard dependency: `deployment-dashboard-ledger` reached a terminal clean handoff; action-time authorization for provider changes  
Modify set in revision 3: `README.md`, `docs/**`, `skills/site-deployment-browser/**`

With the dependency cleared and `control.json` now covering the documentation paths:

- preserve the existing bilingual source -> destination -> type instructions but replace full provider IDs, Railway origins, production domains, private recipients, and deployment receipts with placeholders or a private/local ledger;
- retain intentional public `pages.dev` storefront/admin test URLs only as open-source proof, never as reusable template values;
- scan source plus generated HTML for secrets and prohibited identifiers;
- configure/verify an R2 custom domain and `nosniff` response transform only with current authorization;
- update `R2_PUBLIC_BASE_URL`, rebuild/redeploy, and capture non-secret header/cache results without object keys or presigned URLs.

Safe stop: if the active ledger is still dirty/applying, do not touch its paths. If the custom domain is not ready, keep `r2.dev` test-only and do not declare production release-ready.

### Packet S09: Integrated non-payment release acceptance

Status: BLOCKED_BY_S02_TO_S08  
Covers: REQ-009, AC-010  
Hard dependencies: all prior packets, clean scope ownership, action-time approval for any live mutation

Local evidence:

- targeted Go/content/commerce/bootstrap tests;
- admin and curatory dependency installs, tests, typechecks, builds, and audits;
- `go run ./server/tools/render` followed by `dist/` inspection;
- `go run ./server/tools/speccheck prelaunch-security-hardening`;
- `$env:SCOPE_CHANGE_ID='prelaunch-security-hardening'; go run ./server/tools/scopecheck`;
- `go run ./server/tools/verify`;
- fresh-tree credential/identifier/bundle scan;
- security-review, consumer-reachability, production-content-audit, and walkthrough receipts required by `control.json`.

Bounded live evidence after separate authorization:

- Railway health 200 and direct uncredentialed API 403;
- storefront/admin same-origin public API success, protected admin 401 when unauthenticated, and malicious-origin CORS denial;
- admin CSP/headers on root and deep link;
- over-quota requests return 429 before side effects using a test-safe policy/window;
- custom R2 content/cache/nosniff proof;
- Resend/R2/order flow evidence may reuse still-current non-sensitive receipts; no blind resend or new order is required;
- `MANUAL_TEST_CHECKOUT_UNTIL` is expired/unset before release.

Release claim: storefront + admin + Go + Supabase + R2 + Resend non-payment loop is hardened and operational. Credit-card payment and inbox placement remain excluded.

## Traceability

| Requirement / acceptance | Packet | Primary proof |
|---|---|---|
| REQ-001, AC-001, AC-002 | S01, S02 | topology decision, side-effect spies, replica/provider proof, security review |
| REQ-002, AC-003 | S03 | fixed-clock config/service tests and consumer reachability |
| REQ-003, AC-004 | S04 | header generator tests, fresh build inspection, live root/deep-link walkthrough |
| REQ-004, AC-005 | S05 | sanitizer adversarial/preservation tests and legacy render test |
| REQ-005, AC-006 | S06 | legacy storage migration tests and two-session walkthrough |
| REQ-006, AC-007 | S07 | clean install, full/prod audit, tests/typecheck/build |
| REQ-007, AC-008 | S08 | custom-domain header/cache live receipt |
| REQ-008, AC-009 | S08 | clean-tree source/generated content audit |
| REQ-009, AC-010 | S09 | full verify plus security/walkthrough receipts |

## Blueprint-wide gates

| Check | Revision 2 result |
|---|---|
| Every REQ maps to at least one packet | pass |
| Every AC maps to a planned proof | pass |
| Product paths stay read-only during proposal | pass |
| Existing dirty work is preserved | pass |
| Rate limiting is deployment-wide, not process-local | pass: PostgreSQL is authoritative; Pages/Turnstile/WAF are layered controls |
| Turnstile cannot be bypassed through direct API calls | pass by design: Go performs Siteverify before side effects |
| Identifier-retention policy is explicit | pass: public `pages.dev` tests only; no production/Railway/provider/private values |
| Documentation scope cleared by ledger handoff | pass: ledger Accepted 2026-09-22; README/docs/skill paths in scope |
| Credit-card payment stays out of scope | pass |
| Provider mutations require action-time authorization | pass |
| Expected evidence is not reported as observed | pass |
