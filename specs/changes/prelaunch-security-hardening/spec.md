# Prelaunch Security Hardening Specification

Change ID: prelaunch-security-hardening  
Revision: 3  
Status: Applying  
Decision authority: Repository owner/user  
Approval basis: Requested after the 2026-09-21 bounded red-team review. On 2026-09-21 the repository owner selected PostgreSQL-backed authoritative limits, trusted Pages edge identity, server-validated Turnstile on side-effecting public forms, and optional Cloudflare WAF as a coarse prefilter. The owner also selected retention of intentional public pages.dev test URLs for the open-source proof while excluding production domains, Railway origins, full provider identifiers, private recipients, credentials, database URIs, and deployment receipts. On 2026-09-22 deployment-dashboard-ledger reached a terminal clean handoff (Accepted at revision 25, commits a96f2a6 and c1c23db pushed to staging), clearing the REQ-008 scope dependency, and the owner delegated final adjudication and approved applying this change.  
Repository baseline: 847c0253197ce4625ce9ab7ad52dadcf105c58aa  
Supersedes: public-endpoint-rate-limit

## Outcome

Turn the completed no-gateway deployment walkthrough into a release-safe baseline without claiming that credit-card payment is implemented. Close the observed abuse, browser-hardening, stored-content, credential-retention, dependency, asset-origin, and repository-identification gaps; preserve the current static Cloudflare Pages plus Go-origin architecture.

## Review findings adopted by this change

- Anonymous contact, comment, quote, guest-order, and member-order traffic has no deployment-wide abuse boundary. Contact and order paths can persist data, send mail, or change stock.
- A database row can make `manual_test` publicly selectable with no independent expiry or server-only release gate.
- The admin Pages build has no generated browser-security header contract equivalent to the storefront.
- Published article `body_html` is stored and rendered as trusted HTML without an allowlist sanitizer.
- Guest order access tokens are persisted in `localStorage` as part of recent-order history.
- The admin development/test dependency graph contains high/critical advisories even though the production dependency audit is clean.
- The temporary `r2.dev` asset origin returns correct image type and immutable caching but cannot provide the required production `nosniff` header contract.
- Reusable deployment material contains no observed credentials, but installation-facing documents retain exact provider/deployment identifiers under the current ledger policy.
- Edge authentication, admin authorization, CORS denial, R2 upload/verification, order creation, inventory change, and Resend delivery worked in the bounded walkthrough. This proposal does not weaken those controls or reinterpret spam placement as send failure.

## Scope

In scope:

- deployment-wide abuse control for public mutation/expensive quote surfaces;
- a fail-closed, time-bounded server gate for the test-only manual checkout method;
- generated admin Pages security headers;
- article HTML sanitization for new writes and legacy rendered rows;
- removal and migration of persisted guest order tokens;
- admin development dependency remediation;
- production R2 custom-origin response-header and cache acceptance;
- stricter committed-document de-identification after the active deployment ledger releases those paths;
- one integrated, non-payment release-security acceptance.

Out of scope:

- credit-card capture, refunds, payment webhooks, or claiming ECPay production readiness;
- changing Cloudflare Pages from static hosting or adding new Pages application logic beyond the existing `/api/*` pass-through;
- changing Supabase authentication or staff authorization;
- sending another email, creating another order, mutating DNS/provider state, or copying a secret merely to approve this proposal;
- retroactively deleting already-created bounded test records;
- treating a process-local rate limiter as a deployment-wide control.

## Requirements

### REQ-001: Authoritative public abuse control

One deployment-wide policy MUST cover `POST /api/contact`, `POST /api/products/{slug}/comments`, `POST /api/quote`, `POST /api/orders`, and `POST /api/orders/mine`. The authoritative counter MUST live in the existing application database so every Railway replica observes the same state. SQLite support remains for local/single-process parity; production authority is PostgreSQL.

The Pages Function MUST delete any caller-supplied edge-identity header, derive an opaque HMAC-SHA256 client key from Cloudflare's edge-provided client address using a domain-separated use of `EDGE_SECRET`, and set the new `X-Edge-Client-Key` header before forwarding. The Go origin MUST use that header only after successful edge-secret authentication and MUST never trust browser-supplied `X-Forwarded-For` or `CF-Connecting-IP`. Raw client addresses MUST NOT enter the limiter table or application logs.

Starter limits are fixed at 3 contact submissions per 10 minutes, 5 comment submissions per 10 minutes, 60 quotes per minute, and 5 combined guest/member order submissions per 10 minutes for one client key. A future tuning change requires a controlled revision; unrestricted environment overrides are not allowed. Expired buckets MUST be removed through a bounded cleanup path.

Contact, comment, guest-order, and member-order requests MUST also carry a Cloudflare Turnstile token. The Vue/browser integration supplies the public site key and token; the Go backend validates the token directly with Cloudflare Siteverify before business handling, including expected hostname and action checks. `TURNSTILE_SECRET_KEY` remains Railway-only; the public `TURNSTILE_SITE_KEY` may enter the storefront build. Tokens are single-use and short-lived. No second verification Worker is introduced because Go remains the sole application backend and browser-only validation would be bypassable.

Cloudflare WAF rate limiting MAY be enabled on a later custom domain as a coarse first layer, but it MUST NOT be the authoritative counter because Cloudflare documents per-data-center counters and fail-open behavior. Over quota MUST return HTTP 429 with `Retry-After`. Missing trusted edge identity, unavailable PostgreSQL enforcement, or unavailable Turnstile verification MUST fail closed before side effects with an observable non-2xx response.

This requirement adopts and supersedes the intent of `public-endpoint-rate-limit`. The earlier Draft remains historical until this change reaches an approved executable revision.

#### AC-001: Replica and identity bypasses are closed

- GIVEN traffic reaches any production API replica
- WHEN one caller exceeds the approved policy or supplies spoofed forwarding headers
- THEN excess requests are rejected consistently across replicas and untrusted headers cannot create a fresh identity bucket;
- AND a valid single-use Turnstile token is required for the four side-effecting browser forms while quote remains limiter-only.

#### AC-002: Rejection is side-effect free

- GIVEN the authoritative control reports over quota
- WHEN the target is contact, comment, quote, guest order, or member order
- THEN the response is 429 with `Retry-After` before mail, persistence, inventory, order, notification, or idempotency side effects;
- AND invalid/expired/replayed Turnstile tokens are rejected before those effects;
- AND limiter or verifier dependency failure is fail-closed and observable without exposing the client key or raw address.

### REQ-002: Test-only checkout expires fail closed

`manual_test` MUST remain an explicit no-charge test method and MUST require an independent server-only UTC expiry setting named `MANUAL_TEST_CHECKOUT_UNTIL`. Missing, malformed, past, or unreasonably distant values MUST make the method unavailable even when its database row is enabled and ready. Production MUST cap a single activation window at 24 hours. The same decision MUST govern public listing, quote, guest order, and member order so no alternate endpoint can bypass it.

#### AC-003: Database state alone cannot expose test checkout

- GIVEN an enabled and ready `manual_test` database row
- WHEN the server-only window is absent, invalid, expired, or more than 24 hours ahead in production
- THEN the method is omitted from public payment methods and rejected by quote/order validation;
- AND when a valid future window is configured, the method is available only until that exact instant;
- AND no browser build contains the value.

### REQ-003: Admin Pages emits browser-security headers

Every production admin build MUST emit a root/deep-route `_headers` asset with a least-privilege CSP derived from validated build inputs. The policy MUST deny framing and object embedding, restrict base/form targets, include `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, and `X-Frame-Options: DENY`, and allow only the exact Supabase/Auth/API origins required by the existing admin. Invalid non-empty origins MUST fail the build rather than broaden policy. Provider IDs or installation-specific hosts MUST NOT be hardcoded in source.

#### AC-004: Headers protect the deployed admin without breaking auth

- GIVEN a fresh admin build and deployment
- WHEN the root and a representative deep link are requested
- THEN both return the required headers with no wildcard script/connect/frame source;
- AND a normal Supabase login still resolves the server-granted role;
- AND an unauthenticated protected API request remains 401.

### REQ-004: Published article HTML is allowlist-sanitized

The content module MUST sanitize article HTML before persistence with a maintained sanitizer library and MUST sanitize again at the renderer boundary until legacy rows have been rewritten or audited. The allowlist MAY preserve semantic text, headings, lists, block quotes, and safe links. It MUST remove scripts, styles, forms, frames, embedded objects, executable/event attributes, unsafe URL schemes, and unapproved media. Rendering MUST cast only the sanitizer result to trusted template HTML.

#### AC-005: Active markup is removed and safe formatting survives

- GIVEN a publisher submits mixed safe formatting and active payloads
- WHEN the article is saved and rendered, including from a pre-existing unsanitized row
- THEN active elements, event attributes, unsafe schemes, and form/embed surfaces are absent;
- AND allowed paragraphs, headings, lists, emphasis, block quotes, and standard HTTP(S)/mailto links remain usable;
- AND sanitizer errors never fall back to raw HTML.

### REQ-005: Guest order credentials are session-bound

The storefront MUST NOT persist plaintext order access tokens in `localStorage`, IndexedDB, URLs, analytics, or logs. A token returned immediately after checkout MAY remain in `sessionStorage` for same-tab/session continuity. Persistent recent-order history MAY retain only non-sensitive metadata such as order ID and date. Existing token-bearing recent-order entries MUST be migrated by deleting the token before any re-save. After a new browser session, lookup MUST require the user to enter the query code again; the `X-Order-Access-Token` API contract remains unchanged.

#### AC-006: Old tokens are removed and not reintroduced

- GIVEN legacy `curatory_recent_orders` entries contain tokens
- WHEN the updated storefront loads them
- THEN returned and re-saved entries contain no token;
- AND a just-created order remains viewable in the same session;
- AND a later session shows recent order metadata but requests the query code before reading protected order data.

### REQ-006: Development dependency graph has no high/critical advisories

The admin test stack MUST move off the vulnerable Vitest/Vite/esbuild chain using a compatible fixed Vitest release verified at apply time. Unused optional DOM implementations SHOULD be absent. The full admin dependency audit MUST contain no high or critical advisories, the production-only audit MUST remain clean, and the existing admin tests, typecheck, resource-contract check, and build MUST pass.

#### AC-007: Upgrade is functional and auditable

- GIVEN a clean admin dependency install from the lockfile
- WHEN full and production-only audits plus the repository's admin verification commands run
- THEN high and critical advisories are zero and all existing checks pass without suppressing advisories.

### REQ-007: Production R2 uses a hardened custom origin

Production MUST use an R2 custom domain rather than `r2.dev`. A representative verified image MUST return the stored `Content-Type`, `Cache-Control: public, max-age=31536000, immutable`, and `X-Content-Type-Options: nosniff`; a repeated request MUST demonstrate the intended Cloudflare cache behavior. If R2 does not emit `nosniff`, the operator MUST add an approved response-header transform on the custom domain. No receipt may contain a credential, presigned URL, private recipient, or full object key.

#### AC-008: Asset origin passes live header and cache checks

- GIVEN a released custom asset domain and a non-sensitive verified test image
- WHEN bounded HEAD/GET checks run twice
- THEN the required content type, immutable cache policy, `nosniff`, and expected cache transition are observed;
- AND `R2_PUBLIC_BASE_URL` and the storefront CSP use the custom origin, not `r2.dev`.

### REQ-008: Committed deployment material is de-identified

Reusable skills and committed installation documents MUST contain no credential, database URI, private recipient, personal identifier, Railway origin, production-domain value, or full provider account/project/service/environment/deployment identifier. Because this is an open-source starter, intentionally public `*.pages.dev` storefront/admin test URLs MAY remain as reproducible deployment proof. All other service origins, full UUIDs, and deployment receipts MUST move to a local/private operator ledger or use clearly synthetic/redacted examples. Reusable instructions MUST still derive every target per installation rather than treating the retained test URL as a template value. The active `deployment-dashboard-ledger` change owns the affected paths, so this requirement MUST NOT edit them until that change reaches a terminal clean handoff and this control's scope is revised.

#### AC-009: Fresh-tree content audit finds no installation identity leak

- GIVEN a clean checkout after the ledger handoff
- WHEN repository text and generated documentation are scanned
- THEN no secret or prohibited installation identifier is present;
- AND operator instructions still state where each value is obtained and entered using placeholders and bilingual UI labels;
- AND each retained `pages.dev` test URL is documented as intentional public open-source proof rather than a reusable or production value;
- AND no production domain is committed by this installation.

### REQ-009: Security release acceptance is evidence based

The release gate MUST replay the affected unit/integration tests, full repository verification, fresh builds/renders, dependency audit, security review, and bounded live read-only checks. No unresolved Critical finding may ship; unresolved High findings MUST block release unless this specification is revised with explicit owner acceptance. The `manual_test` window MUST be expired or unset at release. Live side-effecting checks require current action-time authorization and are not implied by approving or applying this change.

#### AC-010: The non-payment loop is release-ready

- GIVEN the implementation has passed local gates and provider configuration is complete
- WHEN the final acceptance runs
- THEN edge authentication, unauthenticated admin denial, CORS denial, article sanitization, order-token retention, public abuse rejection, admin headers, R2 headers/cache, and de-identification all have recorded evidence;
- AND the storefront/admin/backend/R2/Resend non-payment path remains functional;
- AND no claim is made that credit-card payment or inbox placement is complete.

## Resolved owner decisions

### Decision D-001: Hybrid abuse control

Selected on 2026-09-21: the application database is the authoritative cross-replica limiter; Pages supplies an opaque trusted client key over the authenticated edge hop; Go performs Turnstile Siteverify for side-effecting public forms; Cloudflare WAF is optional coarse shielding only. Process-local and Cloudflare-only counters are rejected as the authority.

### Decision D-002: Open-source test URL retention

Selected on 2026-09-21: intentional public `pages.dev` storefront/admin test URLs may remain so another operator can observe a working deployment. Production domains, Railway origins, full provider UUIDs, private recipients, credentials, database URIs, and deployment receipts do not remain in committed material.

## Remaining dependency before Review

Resolved at revision 3: `deployment-dashboard-ledger` reached a terminal clean handoff on 2026-09-22 (Accepted at revision 25), so this change now owns the overlapping README/docs/skill paths. No further owner product decision is required.

## Design decisions already fixed

- `manual_test` uses a server-only absolute UTC deadline, defaults off, and is capped to 24 hours in production.
- Article HTML uses a mature allowlist sanitizer; no handwritten parser and no broad user-generated-content policy.
- New writes and legacy render reads are both sanitized before trusted rendering.
- Guest tokens are session-only; persistent order history loses automatic cross-session lookup by design.
- Admin headers are generated from validated build inputs and are verified on root plus deep routes.
- `r2.dev` remains test-only; the release gate requires a custom R2 domain and live headers.
- Approval never authorizes sending email, placing an order, changing DNS, or exposing/copying a secret.

## Amendments

Revision 2 records owner choices D-001 and D-002. It selects the PostgreSQL/edge/Turnstile hybrid, fixes starter rate budgets and fail-closed behavior, rejects a browser-only or second-Worker Turnstile path to preserve the Go-only backend, and allows only intentional public `pages.dev` test URLs as installation-specific open-source proof.

Revision 3 opens implementation: the deployment ledger handoff completed on 2026-09-22, so `control.json` now covers `README.md`, `docs/**`, `skills/site-deployment-browser/**`, and `site/assets/site.js` (the base-theme contact form, which must carry a Turnstile token). Status moves to Applying under the owner's delegated adjudication and approval.

## Rollback

- Abuse control: revert routing/middleware/provider policy together; never leave a contract that claims 429 without enforcement.
- Manual checkout gate: unset `MANUAL_TEST_CHECKOUT_UNTIL` to fail closed immediately.
- Admin headers: roll back to the last known-good Pages deployment; do not weaken to wildcard sources as a recovery shortcut.
- Article sanitizer: preserve raw database backups outside the application path if editorial recovery is required; production rendering still sanitizes.
- Order token change: reverting persistence requires a new security decision and is not an automatic rollback.
- Dependencies: restore the prior lockfile only if the security gate is also reopened and documented.
- R2: revert DNS/origin routing only to a known-safe custom-domain configuration; `r2.dev` is not a production rollback target.
