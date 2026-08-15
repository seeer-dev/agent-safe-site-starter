# Minimal-cart integration delivery plan

Change ID: minimal-cart-integration  
Revision: 12  
Status: Verifying -- B01F implementation complete; browser acceptance pending

Normative specification: [`spec.md`](spec.md)  
Detailed implementation strategy: [`../../../INTEGRATION_PLAN.md`](../../../INTEGRATION_PLAN.md)

This plan orders implementation and evidence. When this file and `INTEGRATION_PLAN.md` disagree with `spec.md`, `spec.md` controls.

## Repository reality and baseline

| Area | Observed state at proposal time | Delivery implication |
|---|---|---|
| Backend | Go already owns application APIs; content/contact/media exist, while commerce/staff integration is not implemented | Extend Go-owned modules and bootstrap composition; do not add another backend |
| Database | SQLite and PostgreSQL currently stop at migration `001` | Every new schema slice adds and validates equivalent migrations for both drivers |
| Public site | `site/themes/minimal-cart/**` mounts public islands but production paths still consume fixture data | Replace one real consumer flow at a time; never retain fixture fallback on migrated paths |
| Admin | `admin/src/**` has only dashboard, generic resource, and states routes with fixture-backed configuration | Establish session, transport, capabilities, and truthful states before domain CRUD |
| Renderer | `server/internal/render/**` renders article input and currently replaces the output directory directly | Add explicit composed snapshots and staged atomic promotion before publishing claims |
| Contracts | `contracts/openapi.yaml` does not yet describe the proposed catalog, quote, order, publishing, and identity contracts | Update contract, server, and consumers in the same slice |
| Working tree | The repository already contains broad user-owned changes outside this proposal | Keep implementation scope narrow; do not broaden allowlists to hide unrelated baseline changes |

## Approval and execution rule

- Revision 9 is `Applying`; user approved the media lifecycle on 2026-08-13 after prior approval and continuation through revision 8.
- Product Gates in `spec.md` are resolved with explicit decisions; see spec.md Gates table.
- Slice 0 foundation and multiple later slices (1-4) have partial or automated implementation in place -- see `control.json` evidence and `evidence.md` for per-REQ/AC status. No slice is claimed complete.
- B4 automated implementation is complete with regression tests. Live R2 presign, browser PUT, Worker compression, post-upload verification, verified-key promotion, product association save/refetch, association removal/refetch, and public GET are observed in `receipts/walkthrough-live-r2-rev8.md`; revision 9 implements the approved media lifecycle, while live Cloudflare lifecycle configuration, live PostgreSQL migration 015, and public asset-origin headers remain pending.
- B4 media lifecycle follow-up: configure an `uploads/`-only 24-hour R2 lifecycle rule; add canonical verified-asset state, product-image foreign-key protection, a 7-day unassociated grace period, durable deletion jobs, dry-run/apply modes, and retry evidence. Never apply the bucket lifecycle rule to `verified/`.
- Each slice requires its hard dependencies and observed evidence before completion is claimed. A failed test, render, deploy, migration, or approval check stops promotion and preserves the last-known-good behavior.
- Overall status remains `Applying`; no slice or release is complete until all its REQ/AC evidence passes, required walkthroughs are run, and `verify` passes (or scopecheck is resolved).

## Scope lock

The controlled change may eventually affect only the following paths. Each apply turn must narrow `.ai/scope.json` to the selected slice instead of enabling this entire union:

- `.env.example`
- `.gitignore`
- `INTEGRATION_PLAN.md`
- `admin/**`
- `contracts/**`
- `db/**`
- `docs/**`
- `frontend-sample/**`
- `index.html`
- `server/internal/**`
- `server/tools/dev/**`
- `server/tools/render/**`
- `server/tools/internal/**`
- `server/tools/media-gc/**`
- `server/tools/seed/**`
- `server/tools/publish/**`
- `site/**`
- `specs/changes/minimal-cart-integration/**`
- `workflows/publish.md`

Paths outside this list require an approved amendment before editing. Governance files outside this controlled-change directory are not implementation scope for this product proposal.

## Dependency-ordered slices

| Slice | Outcome | Controlled-spec coverage | Gates before apply/completion |
|---|---|---|---|
| 0 -- Contract, auth, development foundation | Real principal, capability, transport, error, and local-development contracts | REQ-001, REQ-002, REQ-008, REQ-011, REQ-012; AC-001, AC-002, AC-003, AC-004, AC-015, AC-021, AC-023 | GATE-001 before apply; GATE-005 before privileged-action completion |
| 1 -- Catalog to static storefront | Product/category/media admin flow reaches deterministic public routes | REQ-001, REQ-003, REQ-008, REQ-010, REQ-011, REQ-012; AC-001, AC-002, AC-005, AC-006, AC-015, AC-016, AC-019, AC-020, AC-021, AC-022, AC-023 | Slice 0 evidence; media and route policies accepted |
| 2 -- Quote, order, fulfillment | Go-authoritative quote/order and auditable order lifecycle | REQ-004, REQ-005, REQ-008, REQ-009, REQ-010, REQ-011, REQ-012; AC-007, AC-008, AC-009, AC-010, AC-015, AC-016, AC-017, AC-018, AC-019, AC-020, AC-021, AC-022, AC-023 | GATE-002, GATE-003, GATE-004, GATE-005, and relevant GATE-009 decisions |
| 3 -- Site content and publishing | Approved content reaches atomic static output with truthful deployment state | REQ-006, REQ-007, REQ-008, REQ-010, REQ-011, REQ-012; AC-011, AC-012, AC-013, AC-014, AC-015, AC-016, AC-019, AC-020, AC-021, AC-022, AC-023 | GATE-006 and GATE-007 |
| 4 -- Customer/member/staff identity | Object-level customer access and canonical staff authorization | REQ-002, REQ-008, REQ-009, REQ-010, REQ-011, REQ-012; AC-004, AC-015, AC-017, AC-018, AC-019, AC-020, AC-021, AC-022, AC-023 | GATE-005, GATE-008, and GATE-009 |
| 5 -- Cleanup and release | No fabricated production fallback; contracts, drivers, builds, render, and release evidence agree | REQ-010, REQ-011, REQ-012; AC-019, AC-020, AC-021, AC-022, AC-023, AC-024 | All applicable Gates and Slices 0-4 evidence |
| 6 -- Media lifecycle | Temporary uploads and unassociated verified media retire without dangling product references | REQ-003, REQ-011, REQ-012, REQ-013; AC-006, AC-022, AC-023, AC-025, AC-026 | User-approved 24-hour/7-day policy and Slice 1 media verification |

## Slice evidence contracts

### Slice 0

- Trace: admin boot -> session/token -> Go authentication -> canonical staff capability -> structured response/UI state.
- Expected proof: authentication and authorization tests, stale-token role-change test, admin transport/typecheck/build, OpenAPI contract checks, and explicit loading/error/forbidden states.
- Safe failure: keep domain mutations hidden or disabled until the server capability and admin origin contract are approved.

### Slice 1

- Trace: admin catalog mutation -> Go validation/transaction -> dual-driver store -> render snapshot -> product/category static route.
- Expected proof: migration/store tests on both drivers, media validation, slug/path containment, deterministic render, no-JavaScript route inspection, and negative fixture-import search.
- Safe failure: preserve the previous static output; never show fixture products after an API or render failure.

### Slice 2

- Trace: cart selection -> Go quote -> idempotent order transaction -> admin fulfillment/return action -> audit receipt.
- Expected proof: tampered-money tests, stock races, idempotent retries, legal/stale transitions, object-level order access, TWD integer contract checks, and responsive/accessible checkout/admin states.
- Safe failure: leave stock and money unchanged on rejected requests; do not synthesize paid, shipped, refunded, or successful UI states.

### Slice 3

- Trace: draft content -> approval metadata -> explicit published snapshot -> staged full render -> deploy request -> terminal deployment receipt.
- Expected proof: draft non-leakage, approver/expiry checks, XSS/URL validation, atomic promotion failure tests, deploy-trigger-versus-terminal-state tests, static policy routes, and no-JavaScript inspection.
- Safe failure: preserve the last-known-good `dist/` and deployment; never promote draft content or report a trigger receipt as deployed.

### Slice 4

- Trace: verified subject -> canonical customer/staff relation -> ownership/capability check -> scoped API result -> audit/outbox receipt.
- Expected proof: cross-customer denial, guessable-order denial, demotion/disable with stale token, anti-self-elevation/last-owner tests, PII minimization/redaction, and metadata-sync retry evidence.
- Safe failure: server authorization follows the canonical staff row even when identity-provider metadata lags; revoked access never reappears through fallback.

### Slice 5

- Trace: repository-wide fixture/contract/migration audit -> complete verification matrix -> inspected build/render artifacts -> release decision.
- Expected proof: negative production-import and credential searches, clean SQLite/PostgreSQL migration runs, idempotent seed, Go/admin/theme builds, render and `dist/` inspection, `speccheck`, `scopecheck`, and `verify`.
- Safe failure: any unmet criterion blocks release without deleting data, rewriting migrations, or replacing a last-known-good deployment.

### Slice 6

- Trace: temp upload or product unlink -> retention clock -> database GC claim -> durable deletion job -> R2 delete or retry receipt.
- Expected proof: dual-driver migration/backfill parity, active-only foreign-key enforcement, seven-day and 24-hour boundary tests, reference/association race protection, re-verification renewal, dry-run no-op, provider retry, count-only output, security review, and lifecycle runbook.
- Safe failure: keep referenced media and retry jobs; never apply an R2 age rule to `verified/` and never expose object keys in operational output.

## Requirement and acceptance traceability

| ID | Primary slice(s) | Surface or boundary | Required observed evidence |
|---|---|---|---|
| REQ-001 | 0, 1 | Static/runtime and module boundaries | Architecture checks plus producer/consumer trace |
| AC-001 | 0, 1 | Public static output versus runtime API | Render and runtime route inspection |
| AC-002 | 0, 1 | Go modules and database access | Architecture check and import/query review |
| REQ-002 | 0, 4 | Admin session and capabilities | Authn/authz integration suite |
| AC-003 | 0 | Admin protected routes/actions | Missing, expired, and insufficient-token tests |
| AC-004 | 0, 4 | Staff capability lifecycle | Demotion, disable, elevation, and last-owner tests |
| REQ-003 | 1 | Catalog/media/static routes | End-to-end catalog publish evidence |
| AC-005 | 1 | Admin catalog to public consumer | Created/updated/archived product route inspection |
| AC-006 | 1 | Slug, output path, media upload | Traversal/collision and upload-policy tests |
| REQ-004 | 2 | Quote and order creation | Server recomputation and transaction tests |
| AC-007 | 2 | Checkout quote/order | Tampered price/discount/shipping/tax rejection evidence |
| AC-008 | 2 | Order idempotency and inventory | Retry and concurrent stock-race tests |
| REQ-005 | 2 | Fulfillment, cancellation, return, refund | State-machine and audit integration tests |
| AC-009 | 2 | Order mutations | Illegal/stale transition no-side-effect proof |
| AC-010 | 2 | Cancel/return/restock/refund | Distinct-transition and provider-boundary evidence |
| REQ-006 | 3 | Site content admin and public pages | Draft/approval/publish integration tests |
| AC-011 | 3 | Draft and published versions | Non-leakage and approval metadata proof |
| AC-012 | 3 | Policy, footer, announcement, feature content | Static-route/no-JavaScript and reviewer evidence |
| REQ-007 | 3 | Render and Cloudflare publication | Staged render and deployment receipt tests |
| AC-013 | 3 | Publish status | Trigger versus terminal-state contract test |
| AC-014 | 3 | `dist/` and deployed site | Injected failure preserves last-known-good proof |
| REQ-008 | 0-4 | All named UI surfaces | State matrix, keyboard, touch, responsive, and recovery walkthroughs |
| AC-015 | 0-4 | Admin surfaces | Real loading/empty/error/forbidden/success evidence |
| AC-016 | 1-3 | Public surfaces | Static fallback, truthful failure, and no-fixture inspection |
| REQ-009 | 2, 4 | Customer order access and PII | Object-level auth and PII lifecycle tests |
| AC-017 | 2, 4 | Account/order/tracking | Cross-subject and guessed-identifier denial tests |
| AC-018 | 2, 4 | Lists, logs, retention/deletion | Minimization, redaction, retention, and deletion evidence |
| REQ-010 | 1-5 | Every migrated production consumer | Consumer inventory and negative fixture searches |
| AC-019 | 1-5 | Production bundles and failure paths | Import/bundle inspection and forced-failure walkthrough |
| AC-020 | 1-5 | Development-only fixtures | Explicit dev gate and production-build absence proof |
| REQ-011 | 0-5 | API, storage, renderer, UI contracts | Field ledger, contract tests, and driver parity |
| AC-021 | 0-5 | OpenAPI/server/consumers | ID/date/money/version/error-shape reconciliation |
| AC-022 | 1-5 | SQLite/PostgreSQL | Clean migration and behavior tests on both drivers |
| REQ-012 | 0-5 | Controlled delivery lifecycle | Approved revision, narrow scope, and evidence update per slice |
| AC-023 | 0-5 | Slice completion claims | Named commands, artifacts, and walkthrough receipts |
| AC-024 | Proposal and every apply | Lifecycle control | Draft produces planning artifacts only; apply requires approved revision |
| REQ-013 | 6 | R2 lifecycle, media registry, product associations, GC queue | Migration, race, retention, dry-run and retry tests |
| AC-025 | 1, 6 | Product image foreign key versus GC claim | Association/claim exclusion and dangling-reference negative proof |
| AC-026 | 6 | Temporary and verified object retirement | 24-hour/7-day boundaries, preview no-op and provider retry evidence |

## Validation sequence

Run the narrowest slice tests first, then the controlled-repository gates:

```powershell
go test ./server/internal/<changed-package>/...
go test ./...
go vet ./...
go run ./server/tools/speccheck
go run ./server/tools/scopecheck
go run ./server/tools/verify
```

When admin changes:

```powershell
Set-Location admin
npm run typecheck
npm run build
```

When public-theme or publishing output changes:

```powershell
Set-Location site/themes/minimal-cart
npm run build:check

Set-Location ../../..
go run ./server/tools/render
```

Inspect `dist/` for expected routes/content, absence of draft/fixture data, valid assets, complete manifest/page counts, and preservation of last-known-good output under injected failure.

## Revision 11 Google and LINE OAuth slice

1. **Provider-neutral session initiation** (`REQ-002`, `REQ-008`, `REQ-009`, `REQ-010`, `AC-003`, `AC-015`, `AC-017`, `AC-019`, `AC-028`): extend the existing public and admin Supabase session adapters with one typed OAuth initiation contract. Expose Google and LINE buttons only when their non-secret build flags are enabled; use `google` and `custom:line`; redirect only to the current configured application origin; preserve email/password, logout, refresh, late-response and 401 fail-closed behavior. Do not persist provider access/refresh tokens and do not add another backend or direct Supabase Database access.
2. **External configuration and independent acceptance** (`REQ-002`, `REQ-012`, `AC-003`, `AC-015`, `AC-017`, `AC-023`, `AC-024`, `AC-028`): Google uses the Supabase built-in provider and its Supabase callback URL. LINE uses an OIDC custom provider with issuer `https://access.line.me`, PKCE S256, identifier `custom:line`, and `openid profile email`; do not enable LINE until LINE email permission is approved. Independently observe disabled, redirect initiation, cancel/failure recovery, successful Supabase session, unlinked-admin forbidden, linked-admin success, public member reachability, logout, and secret-negative bundle/log evidence. Live provider success remains pending until the user finishes both external consoles.

This revision entered verification after the user’s plain `apply` on 2026-08-13. Implementation touched only the existing auth session/config/UI/test/check paths, `.env.example`, the OAuth operator runbook, and controlled evidence/receipt paths. No database migration or Go authorization change is planned.

## Revision 12 admin visibility remediation

1. **Dashboard rendering** (`REQ-008`, `AC-015`, `AC-023`, `AC-024`): remove the inert nested native HTML `template` element from `DashboardPage` so server-derived KPI cards and operational panels are visible. Add a component regression test that proves the KPI and module surfaces render after the two API reads resolve.
2. **Staff-list least privilege** (`REQ-002`, `REQ-008`, `AC-003`, `AC-004`, `AC-015`, `AC-023`, `AC-024`): align `GET /api/admin/staff` with the documented `staff.read` capability. Do not grant `staff.update`; write endpoints remain protected by their existing service-layer check. Add handler coverage for authenticated read-only staff, missing-read capability, and unauthenticated requests.
3. **Acceptance** (`REQ-008`, `REQ-012`, `AC-015`, `AC-023`, `AC-024`): run focused Go/admin tests and build checks, then independently replay the linked manager session in the local admin UI. Record visible dashboard KPIs, read-only staff list, successful populated pages, and truthful zero-record empty states.

## Revision 10 shipping-authority slices

1. **Admin configuration to public discovery** (`REQ-004`, `REQ-008`, `REQ-011`, `AC-015`, `AC-021`, `AC-027`): add portable SQLite/PostgreSQL storage, versioned create/update/list admin operations behind the existing commerce-admin capability, the real admin resource surface, and a public enabled-only list. Do not seed production fees and do not touch quote/order calculation yet. Evidence: migration parity, service/HTTP authorization and validation tests, stale-version conflict test, admin typecheck/build/contracts, OpenAPI contract checks, and public empty/success/error observations.
2. **One fee rule for quote and order** (`REQ-004`, `REQ-011`, `AC-007`, `AC-021`, `AC-027`): remove the hardcoded recognized-method and injected production seam, look up the current enabled row in Go, apply `fee = 0` only when the configured fee is zero or a non-null positive threshold is met, and reuse the same calculation in quote and order. Preserve idempotent replay semantics for already-created orders. Evidence: tampering, missing/disabled method, store failure, threshold boundaries, changed-after-quote, quote/order parity, overflow, and side-effect tests.
3. **Independent end-to-end acceptance and evidence** (`REQ-008`, `REQ-012`, `AC-015`, `AC-022`, `AC-023`, `AC-024`, `AC-027`): Codex replays migrations and contracts, creates a reversible local shipping fixture through the admin path, observes admin loading/empty/error/forbidden/success/conflict states, public discovery, quote, order, disablement, and cleanup, then updates controlled evidence. Supabase is not required for these local checks; live PostgreSQL remains separately pending until credentials exist.

Slice 1 is the only delegated scope until independent review passes. Slice 2 starts only after Slice 1 producer, admin consumer, authorization, and contract evidence agree. Codex owns planning, protected governance edits, acceptance, and evidence; Grok owns bounded product implementation.

## Current disposition

- Specification completeness: approved.
- Revision approval: revision 12 is Verifying under the user's `apply` on 2026-08-14; the focused dashboard and manager staff-list repair has local automated and linked-session acceptance evidence. Prior revision 10 shipping acceptance and revision 11 OAuth external-provider verification remain recorded.
- Product Gates: all resolved with explicit decisions; see spec.md Gates table.
- Implementation evidence (per `control.json` / `evidence.md`):
  - **Passed (automated/local acceptance)**: REQ-001, REQ-002, AC-001, AC-002, AC-003, AC-004, AC-008, AC-019, AC-020, AC-027 -- revision 10 shipping authority passed independent local browser/API replay; see `receipts/shipping-walkthrough-rev10.md`.
  - **Pending (implementation or acceptance evidence incomplete)**: REQ-003, REQ-004, REQ-005, REQ-006, REQ-008, REQ-011, REQ-012, AC-005, AC-006, AC-007, AC-009, AC-010, AC-011, AC-012, AC-013, AC-014, AC-015, AC-016, AC-017, AC-018, AC-021, AC-022, AC-023, AC-024, AC-028 -- see per-REQ/AC proof in `evidence.md`; revision 10 shipping evidence is replayed, while broader non-shipping and live-environment conditions remain open. Revision 11 OAuth source/build checks pass, but live provider callbacks remain unobserved.
  - **B4 automated implementation complete** (post-upload verify, product_images association, Worker compression, admin uploader, sentinel error mapping, GIF full-decode, admin validation sentinels) with regression tests; live R2 presign/PUT/verify, association save/refetch/removal, and public GET observed. Revision 9 adds race-safe media GC and the approved 24-hour/7-day policy; pending: live Cloudflare lifecycle configuration and live PostgreSQL migration 015.
- Overall status: `Verifying`. Revision 10 shipping slices remain locally complete; revision 11 OAuth implementation and local checks are complete, while live provider acceptance remains pending.

### Completed local work (pending live verification)

- B7 restock cascade + all-of capability gate: `restock_idempotency.order_id` foreign key now `ON DELETE CASCADE` to `orders` (SQLite + PostgreSQL migration 014). Admin restock `RowAction` uses `allCaps: ['orders.returns', 'inventory.adjust']` matching the backend's dual-capability requirement. Regression tests: `TestRestockIdempotencyCascadeOnOrderDelete`, `ResourceTable` allCaps/combined cap tests (110 admin tests). PENDING LIVE: live PostgreSQL migration 014 apply.
- B9 public asset-origin headers: Go renderer generates `dist/_headers` with strict CSP + safety headers (X-Content-Type-Options nosniff, Referrer-Policy, X-Frame-Options DENY, CSP with R2/API origin allowlist). Empty origins omitted from CSP; non-empty invalid origins fail-closed (render aborts, LKG dist preserved). R2 `CopyObject` sets `Cache-Control: public, max-age=31536000, immutable` (content-addressed SHA256 key) + `MetadataDirectiveReplace`. Hardcoded external images (Unsplash) removed from production theme. `dev` and `render` tools wire `R2PublicBaseURL` via tested `rendererConfig` helper. Cloudflare runbook (`docs/r2-asset-origin-runbook.md`) documents Pages `_headers` does not cover R2 origin; operator must live HEAD/GET verify `nosniff` on R2 responses, and only if missing add a Response Header Transform Rule or Managed Transform. 12 renderer header tests + 5 R2 CopyObject input builder tests + 4 dev/render wiring tests. PENDING LIVE: live R2 CopyObject verification, live R2 custom domain `nosniff` HEAD/GET verification, live R2 CDN cache behavior verification, live Pages `_headers` deployment.
- B11 promo enumeration + shipping-fee authority: `calculateDiscount` now returns `ErrInvalidPromoCode` for unknown/non-active codes and `ErrPromoValidationUnavailable` for store failures. Quote/CreateOrder query one submitted code via `GetActivePromoByCode`; unknown, disabled, expired, or not-yet-active codes return `ErrInvalidPromoCode` (400), lookup failure returns stable `ErrPromoValidationUnavailable` (503). Public `/api/promos/active` removed from bootstrap and OpenAPI; browser local enumeration (`fetchActivePromos`/`loadPromos`/local promo-list matching) removed. The cart records a non-empty normalized code only as pending and labels it for server validation. The test-only 1500/120/60 `FeeSchedule` moved from production `service.go` into `service_test.go`; the dev site's published free-shipping seed and Hero shipping promises were removed; the browser-source comment no longer repeats those numeric values. Regression tests: `TestQuoteRejectsUnknownPromoCode`, `TestQuoteRejectsExpiredPromoCode`, `TestQuoteAcceptsActivePromoCode`, `TestQuoteFailsClosedWhenPromoValidationUnavailable`, `TestCreateOrderRejectsUnknownPromoWithoutSideEffects`, HTTP 400/503 tests. site theme `check:browser-authority` and `check:openapi-contracts` pass. PENDING LIVE: live PostgreSQL migration apply, independent admin UI walkthrough, member auth consumer reachability.
- Dev-render composition fix: `server/tools/dev` was calling `RenderAll(articles)` only, producing legacy `AI Site Starter` / `Static Publish Starter` output instead of the integrated minimal-cart site. Fixed: `dev` now defaults `SiteTheme` to `minimal-cart` and uses the shared `server/tools/internal/rendercompose.Compose` to build the full public input (articles + published products/categories/site content), then calls `RenderAllFull` -- identical to `server/tools/render`. `rendercompose.Compose` sorts categories deterministically for stable dist. End-to-end regression: `TestComposeAndRenderProducesMinimalCartOutput` seeds a real SQLite DB, composes, renders with the real minimal-cart theme, and asserts `dist/index.html` contains `質物選物`, does NOT contain `Static Publish Starter`, and product/category/content route files exist. 3 rendercompose tests + 3 dev rendererConfig tests pass. Independent local walkthrough re-run COMPLETE (receipts/walkthrough-rev5.md): clean dev restart produces dist/index.html with 質物選物, server-rendered product names, #shop-static, ProductGrid island; Static Publish Starter absent; draft 寬版工作褲 absent.
- B8 admin nullish cell display fix: `ResourceTable.cellContent` rendered literal `undefined`/`null` in number/text/mono cells when API omitted optional fields (e.g. `approved_version`/`published_version` with json omitempty on legacy published rows). Fixed: `absentDash` helper renders null/undefined as em dash (`--`), preserves numeric `0` as `0` (does not conflate absent with zero). 5 focused tests: undefined, null, numeric zero, real version numbers, mixed row. B8 media/accessibility walkthrough COMPLETE for covered local admin scope (receipts/walkthrough-rev6.md): desktop keyboard operation confirmed by independent desktop browser observation (edit dialog focused SKU; Shift+Tab wrapped first field to Cancel; Tab wrapped Cancel back to SKU while disabled Save was skipped; changing a field then Escape triggered the unsaved-change native confirm); narrow/mobile walkthrough COMPLETE at 390x844 (no horizontal overflow clientWidth=scrollWidth=390; modal rect width 366.59 within viewport; .fgrid one column 328.59px; modalback overflow-y:auto scrollHeight=1297 reached footer scrollTop=453 footer y=745; media file input aria-labelledby=label-field-product_images visible text 商品圖片; clicking label[for=field-name] focused #field-name; Shift+Tab from #field-sku wrapped to enabled Cancel; Tab from Cancel wrapped to #field-sku skipping disabled Save; Escape closed clean dialog and restored focus to triggering 編輯 button; focus indication visible via 3px box-shadow and changed border color; button measurements 選擇圖片 78x33 and Cancel/Save 54x35.5 both exceed 24x24; no new responsive/accessibility defect reproduced); MediaUploader focus management (error->retry, success->idle->select, cancel->idle->select, disabled/loading safety) verified via 5 jsdom tests; readable error/empty/success/threshold-rejection states verified; provider/network failure behavior exercised via HTTP (presign 503, verify 503, cross-user key 403, missing auth 401). Two code defects found and fixed with focused regression tests: (1) touch-target size below WCAG 2.5.8 AA 24x24px minimum for MediaUploader small buttons and alt-text input — fixed with `min-height: 24px` + `min-width: 24px` (for `.btn-move`) + increased padding as inline style + scoped CSS, 1 regression test (mounts 2 images, asserts `.btn-move` minHeight+minWidth, `.btn-remove` minHeight, `.media-alt-input` minHeight); (2) form labels not associated with inputs via for/id, and read-only form labels pointed to nonexistent controls — fixed by adding `id` prop to Input/Textarea/Select, `for` attribute on labels, `labelledby` prop on MediaUploader with `aria-labelledby`, omitting `for`/`id` when field is read-only, 4 regression tests (for/id association, read-only label omission, aria-labelledby present, aria-labelledby absent). PENDING: B4 actual media upload (file selection now succeeds; pre-R2 small-file direct and large-image Worker-compression paths are observed in receipts/walkthrough-rev7.md; live R2 PUT, verify, and product_images association remain unobserved because R2 is not configured).
- B4 pre-R2 browser walkthrough COMPLETE for covered local scope (receipts/walkthrough-rev7.md): Chrome file selection now works; a 113,744-byte PNG took the direct path; a 1,277,444-byte 3840x1200 JPEG took the Worker-compression path and progressed from `壓縮中…` to `上傳中…`; both reached the truthful `R2 is not configured` presign failure with Retry focused. No image association or product save occurred. PENDING LIVE: R2 PUT, post-upload verify, `product_images` association, save/refetch persistence, and public delivery.
- No-JS home product baseline + dev same-origin proxy: Independent public/no-JS walkthrough found that `dist/index.html` contained only `data-vue-island` mounts with ZERO product names/cards in raw HTML; no-JS users saw no products and JS users saw `無法載入商品，請稍後再試。` due to cross-origin dev fetch. Fixed: (1) Go renderer now passes `Products`/`Categories`/`CategoryLabels` to `homeData` and the minimal-cart `home.html` template emits a server-rendered `#shop-static` section with product cards (name, price, link, stock/availability semantics) and category navigation links. (2) `ProductGrid` island now hides `#shop-static` only on successful fetch; during loading and error the static baseline remains visible. Skeleton is shown only during filter changes, not initial load. (3) Dev site server (`devSiteHandler`) now proxies `/api/*` to the API server via `httputil.ReverseProxy` with validated host:port resolution (`resolveDevAPIAddr`: `:8080` -> `localhost:8080`), 502 on proxy error (no static fallthrough), and explicit `staticDir` parameter (no CWD dependency). Dev render uses empty `PublicAPIBase` so `data-api-base` is empty and browser fetches same-origin. Production CORS is not weakened. Regression tests: `TestRenderHomeContainsServerRenderedProducts` (raw HTML asserts product names, links, category nav, prices, stock semantics), `TestRenderHomeNoJSBaselineSurvivesIslandMount` (both `#shop-static` and island mount present), `TestRenderHomeEmptyProductsShowsEmptyMessage` (honest empty state), `TestResolveDevAPIAddr` (leading-colon + explicit host:port), `TestDevSiteHandlerProxiesAPIAndServesStatic` (both address forms, proxy + static), `TestDevSiteHandlerProxyErrorReturns502` (listener-allocated-then-closed port, 502 not static fallthrough). rendercompose e2e test also asserts `Test Product` name and `#shop-static` in raw HTML. Independent local walkthrough re-run COMPLETE (receipts/walkthrough-rev5.md): raw home HTML includes #shop-static, 素面亞麻襯衫/陶製小碟/線裝筆記本, product/category links, ProductGrid island; draft 寬版工作褲 absent; Static Publish Starter absent; same-origin GET /api/products -> 200 with 5 published products; ProductGrid hydrated 5/5 with correct low-stock/out-of-stock states and no load error; /products/linen-shirt/ -> 200; /categories/apparel/ -> 200 with active product and draft absent; /content/footer.about/ -> 200 after UI approve/publish + clean dev restart; prior expired/unapproved footer snapshot returned 404 before re-approval.

- Revision 9 Supabase Auth frontend integration is complete for local scope: public sign-in/sign-up/sign-out session production, admin session verification, refresh/logout race safety, authenticated member checkout/history/detail routing through Go, fail-closed response parsing, validated CSP origin wiring, and local unavailable/invalid-session browser states. Independent evidence: `receipts/walkthrough-auth-rev9.md`. Live provider success remains external acceptance work.
- Revision 9 full visitor/admin remediation replay is complete for local scope: shared public Dialog runtime accessibility, category-scoped hydration, real admin navigation links, minimal-cart titles, and exact sample-article fixture removal all passed independent browser replay. Evidence: `receipts/walkthrough-full-rev9.md`. Live R2 `nosniff`, Supabase success, and PostgreSQL validation remain external acceptance work.

### Next safe local work (no new product decisions or external environment required)

- Define and approve verified-media deletion/retention semantics before adding cleanup code. The live R2 upload and reversible product association walkthrough is complete (`receipts/walkthrough-live-r2-rev8.md`). B5 content approval/publish gate walkthrough is complete (`receipts/walkthrough-rev5.md`).
- Security review finding: automatic verified-object deletion is not safe with the current schema alone. `product_images.object_key` has no foreign key to `media_objects`, `media_objects.object_key` is intentionally non-unique, and no tombstone/lease coordinates association with deletion. A cleanup slice must define the retention window and add race-safe reference checks before it can delete R2 bytes.
- Static/no-JavaScript public route walkthrough for product/category/content pages COMPLETE (receipts/walkthrough-rev5.md).
- B8 admin media/accessibility walkthrough COMPLETE for covered local admin scope (receipts/walkthrough-rev6.md): desktop keyboard and narrow/mobile (390x844) confirmed by independent browser observation; MediaUploader focus/states/provider-failure verified via jsdom tests and HTTP; two code defects (touch-target size, form label association) found and fixed with focused regression tests. B4 actual media upload remains pending (file selection now succeeds; pre-R2 small-file direct and large-image Worker-compression paths are observed in receipts/walkthrough-rev7.md; live R2 PUT, verify, and product_images association remain unobserved because R2 is not configured).

### Work requiring external environment or acceptance evidence

- Live R2 upload/post-upload verification and product association save/refetch/removal are complete for the covered browser path; R2 response-header/CDN verification remains.
- Live PostgreSQL migration 013 apply and integration run (concurrent idempotency, order transition, event sequencing, last-owner FOR UPDATE).
- Live Cloudflare Deploy Hook receipt (GATE-006).
- Formal retention/deletion policy approval by named authority (GATE-009).
- Secure token recovery mechanism (account takeover risk assessment).
- Live Supabase customer/staff success-path walkthrough with configured `SUPABASE_URL` and `SUPABASE_PUBLISHABLE_KEY`. Revision 9 local wiring, fail-closed provider-unavailable state, invalid-session cleanup, and browser-to-Go member create/list/detail reachability are implemented and covered by `receipts/walkthrough-auth-rev9.md`.
- Independent acceptance walkthrough by a non-implementer reviewer.

## Implementation expansion context

Proposal revision: 12
Proposal status: Verifying
Repository baseline: `7c45b616fbe3a632ffe2a39d872c98485466c991`
Observed HEAD: `52e1f3fc1bb3e103c4a3988f0a479d70bd7cecab`
Pre-existing dirty paths:
- `Makefile` - preserve; pre-existing working tree modification, not attributable to B01F planning
- `specs/changes/minimal-cart-integration/control.json` - preserve; pre-existing revision 12 evidence update, not attributable to B01F planning
- `specs/changes/minimal-cart-integration/evidence.md` - preserve; pre-existing revision 12 evidence update, not attributable to B01F planning
- `specs/changes/minimal-cart-integration/receipts/b5-approval-walkthrough-rev12.md` - preserve; pre-existing revision 12 receipt, not attributable to B01F planning
- `specs/changes/commerce-module-file-split/` - preserve; pre-existing untracked proposal directory, not attributable to B01F planning
- `specs/changes/public-endpoint-rate-limit/` - preserve; pre-existing untracked proposal directory, not attributable to B01F planning
- `specs/changes/supabase-jwks-verifier/` - preserve; pre-existing untracked proposal directory, not attributable to B01F planning
- `specs/changes/verify-contract-checks/` - preserve; pre-existing untracked proposal directory, not attributable to B01F planning
Packet ID mapping:
- Batch 1 -> Packet B01E (evidence + walkthrough)
- Batch 2 -> Packet B01F (static footer policy navigation)

### Packet B01E: B5 approval gate evidence sync and independent walkthrough

Status: executable
Covers: REQ-006, AC-011, AC-012
Hard dependencies: none
Outcome: AC-011 and AC-012 evidence is updated from revision 7 "IN PROGRESS" to revision 12 observed state, and an independent local walkthrough receipt records the approve→publish→render flow with the real Go API and admin UI.
Safe failure / rollback: All edits are to `specs/changes/minimal-cart-integration/control.json`, `evidence.md`, and a new receipt file. No product code changes. If the walkthrough finds a gap, the receipt records it as a blocker and the AC remains pending.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Behavior owner | `server/internal/modules/sitecontent/service.go` | `Service.Approve` | Requires `content.approve` cap, rejects empty UserID, rejects past/zero expiry, atomic conditional UPDATE on draft_version match |
| Behavior owner | `server/internal/modules/sitecontent/service.go` | `Service.Publish` | Requires `content.publish` cap, pre-validates key, atomic conditional UPDATE checking draft_version + approved_version = draft_version + approved_expiry_unix > now |
| Behavior owner | `server/internal/modules/sitecontent/service.go` | `Service.Update` | Requires `content.update` cap, increments draft_version (invalidating approval), never touches published_* fields |
| Behavior owner | `server/internal/modules/sitecontent/store.go` | `SQLStore.Publish` | Conditional UPDATE: copies draft→published_*, freezes approval metadata into published_version/published_approver_user_id/published_approved_unix/published_approval_expiry_unix, checks approved_expiry_unix > now |
| Behavior owner | `server/internal/modules/sitecontent/store.go` | `SQLStore.ListPublished` | Filters `published_approval_expiry_unix > now` — expired snapshots absent from public render |
| Contract | `server/internal/modules/sitecontent/http.go` | `Handler.Approve` | POST /admin/site-content/{id}/approve, 403/404/409/400 error mapping |
| Contract | `server/internal/modules/sitecontent/http.go` | `Handler.Publish` | POST /admin/site-content/{id}/publish, 403/404/409(stale)/409(approval)/400 error mapping |
| Consumer | `admin/src/config/resources/content.ts` | `rowActions[approve]` | cap: content.approve, expect: draft_version, expiryInput: true, showWhen: approve_always |
| Consumer | `admin/src/config/resources/content.ts` | `rowActions[publish]` | cap: content.publish, expect: draft_version, showWhen: publish_ready |
| Consumer | `admin/src/config/resources/content.ts` | `rowMap` | formatUnix for approved_at, approved_expiry_at, published_approved_at, published_approval_expiry_at |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestApproveRecordsApproverIdentityAndVersion` | Asserts approver_user_id, approved_version, approved_unix, approved_expiry_unix persisted |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestEditInvalidatesApproval` | Edit increments draft_version → Publish returns ErrStaleVersion (old version) or ErrApprovalConflict (new version, stale approval) |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestExpiredApprovalRejectsPublish` | Expired approval → ErrApprovalConflict |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestPublishedSnapshotExpiryFilter` | ListPublished filters published_approval_expiry_unix > now |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestPublishedSnapshotExpiryEditDoesNotChangeFrozenMetadata` | Editing draft after publish does not change frozen published approval metadata |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestUnapprovedPublishedRowFailClosed` | Published row with default 0 approval metadata → absent from ListPublished |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestMigration012SQLiteFullApply` | Migration 012 applies cleanly, all 9 governance columns present |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestMigration012PostgresParity` | PostgreSQL 012 migration matches SQLite schema intent |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestUpdatePublishedContentKeepsPublishedCopyLive` | AC-011 core: editing draft does not take published copy offline |
| Existing proof | `server/internal/modules/sitecontent/service_test.go` | `TestPublishSwitchesToNewVersion` | Re-approve + publish switches public copy to new version |
| Existing proof | `server/internal/modules/sitecontent/http_test.go` | `TestApproveHTTPSuccess`, `TestPublishHTTPSuccess`, `TestApproveHTTPStaleVersion409`, `TestPublishHTTPStaleVersion409`, `TestPublishHTTPWithoutApproval409` | HTTP layer wiring verified |
| Migration | `db/migrations/sqlite/012_site_content_approval.sql` | 9 governance columns | draft_version, approved_version, approver_user_id, approved_unix, approved_expiry_unix, published_version, published_approver_user_id, published_approved_unix, published_approval_expiry_unix |
| Migration | `db/migrations/postgres/012_site_content_approval.sql` | matching schema | PostgreSQL parity |

Read set:
- `server/internal/modules/sitecontent/service.go` - understand Approve/Publish/Update gate logic
- `server/internal/modules/sitecontent/store.go` - understand conditional UPDATE and ListPublished expiry filter
- `server/internal/modules/sitecontent/http.go` - understand HTTP error mapping
- `server/internal/modules/sitecontent/service_test.go` - inventory all 60 top-level tests
- `admin/src/config/resources/content.ts` - understand admin UI approve/publish wiring
- `specs/changes/minimal-cart-integration/control.json` - current AC-011/012 evidence (stuck at revision 7)
- `specs/changes/minimal-cart-integration/spec.md` - AC-011/012 normative text

Modify set:
- `specs/changes/minimal-cart-integration/control.json` - update AC-011 and AC-012 observed_revision to 12, update proof to reflect completed B5 implementation, update status from pending to passed (if walkthrough confirms)
- `specs/changes/minimal-cart-integration/evidence.md` - update AC-011 and AC-012 proof text
- `specs/changes/minimal-cart-integration/receipts/b5-approval-walkthrough-rev12.md` - new independent walkthrough receipt

Must not modify:
- Any Go source file - B5 implementation is already complete
- Any admin source file - admin UI is already complete
- Any migration file - migrations 009/012 are already applied
- Any file outside `specs/changes/minimal-cart-integration/**` - scope boundary

#### Behavior contract

Before:
- AC-011 `observed_revision` = 7, proof says "B5 IN PROGRESS (revision 7): formal approval/version/expiry gate is partially implemented"
- AC-012 `observed_revision` = 7, proof says "B5 IN PROGRESS (revision 7): formal current-approval metadata is partially implemented"
- Both ACs are `pending`
- No independent walkthrough receipt for the B5 approve→publish→render flow at revision 12

After:
- AC-011 `observed_revision` = 12, proof reflects the fully implemented Approve/Publish gate with all 60 top-level tests, HTTP wiring, admin UI wiring, and migration 012
- AC-012 `observed_revision` = 12, proof reflects the snapshot-scoped governance (published_approval_expiry_unix filter), unapproved-row fail-closed behavior, and all regression tests
- Both ACs are `passed` (if walkthrough confirms no gaps) or remain `pending` with a named blocker (if walkthrough finds a gap)
- New receipt `receipts/b5-approval-walkthrough-rev12.md` records the independent local walkthrough: create draft → approve with expiry → publish → verify ListPublished shows approved content → edit draft → verify published copy unchanged → re-approve → re-publish → verify new version live → verify expired approval blocks publish

Preserved invariants:
- No product code changes
- No migration changes
- No admin UI changes
- Existing 60 top-level tests remain unchanged and passing
- control.json revision remains 12
- control.json status remains Verifying (only AC-level status changes)

#### Ordered edits

1. `specs/changes/minimal-cart-integration/receipts/b5-approval-walkthrough-rev12.md` - new receipt
   - Change: Create a new walkthrough receipt recording the independent local walkthrough of the B5 approve→publish→render flow. The walkthrough must:
     - Start the Go dev server (`go run ./server/tools/dev`)
     - Create a draft site content block via the admin UI or API
     - Approve it with a future expiry via the admin UI or API
     - Publish it via the admin UI or API
     - Verify `ListPublished` returns the published content
     - Verify `go run ./server/tools/render` produces dist/ with the published content
     - Edit the draft and verify the published copy remains unchanged (AC-011 core)
     - Verify the draft edit invalidated the approval (publish fails with 409)
     - Re-approve and re-publish, verify the new version is live
     - Test expired approval (use a past expiry or manipulate the row) and verify publish fails
     - Record source revision, exact commands, observed HTTP responses, and dist/ inspection results
   - Constraint: No credentials, tokens, or raw PII in the receipt. Record exact commands and observed outputs.
   - Result: A complete independent walkthrough receipt proving the B5 gate works end-to-end at revision 12.

2. `specs/changes/minimal-cart-integration/control.json` - AC-011 and AC-012 evidence update
   - Change: Update AC-011 `observed_revision` from 7 to 12. Update AC-011 `proof` to reflect the completed B5 implementation: Approve/Publish gate with capability separation, optimistic concurrency, approval expiry validation, snapshot-scoped governance, and 60 passing tests. Update AC-011 `status` from `pending` to `passed` (if walkthrough confirms). Add `receipts` entry pointing to `receipts/b5-approval-walkthrough-rev12.md` with kind `walkthrough`. Same for AC-012.
   - Constraint: `revision` remains 12, `status` remains `Verifying`. No other AC is touched.
   - Result: AC-011 and AC-012 reflect revision 12 observed state instead of revision 7 stale "IN PROGRESS" text.

3. `specs/changes/minimal-cart-integration/evidence.md` - AC-011 and AC-012 proof text update
   - Change: Update the AC-011 and AC-012 rows in the evidence table to match the updated control.json proof text.
   - Constraint: No other evidence row is touched.
   - Result: evidence.md and control.json are consistent.

#### Integration trace

```text
B5 approval gate walkthrough
  -> go run ./server/tools/dev (starts Go API + dev site)
  -> POST /admin/site-content (create draft, content.create cap)
  -> POST /admin/site-content/{id}/approve (content.approve cap, expiry_unix, expected_draft_version)
  -> Service.Approve -> Store.Approve (conditional UPDATE, draft_version match)
  -> POST /admin/site-content/{id}/publish (content.publish cap, expected_draft_version)
  -> Service.Publish -> Store.Publish (conditional UPDATE, draft_version + approved_version + expiry)
  -> GET /api/site-content (ListPublished, published_approval_expiry_unix > now filter)
  -> go run ./server/tools/render (dist/ inspection)
  -> PUT /admin/site-content/{id} (edit draft, draft_version increments, approval invalidated)
  -> verify ListPublished still shows old published copy (AC-011 core invariant)
  -> POST /admin/site-content/{id}/publish (fails with 409, approval stale)
  -> re-approve + re-publish (new version live)
  -> expired approval test (publish fails with 409)
```

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| 60 sitecontent tests pass | `D:\Projects\AI-go-starter` | `go test ./server/internal/modules/sitecontent/ -v -count=1` | All 60 Test* functions | All PASS | N/A (existing tests) | N/A | N/A |
| Admin typecheck + build | `D:\Projects\AI-go-starter\admin` | `npm run typecheck && npm run build` | exit 0 | Both PASS | N/A | N/A | N/A |
| OpenAPI contracts | `D:\Projects\AI-go-starter\site\themes\minimal-cart` | `npm run check:openapi-contracts` | "PASSED" | PASS | N/A | N/A | N/A |
| Approve→Publish→Render walkthrough | `D:\Projects\AI-go-starter` | `go run ./server/tools/dev` then manual API/UI walkthrough | `receipts/b5-approval-walkthrough-rev12.md` | All steps observed, no gaps | If publish without approval returns 409 | N/A (walkthrough is observation, not mutation) | N/A |
| Draft edit keeps published copy live | `D:\Projects\AI-go-starter` | walkthrough step: edit draft after publish, verify ListPublished | `TestUpdatePublishedContentKeepsPublishedCopyLive` + walkthrough observation | Published copy unchanged | If published copy changed after draft edit | N/A | N/A |
| Expired approval blocks publish | `D:\Projects\AI-go-starter` | walkthrough step: attempt publish with expired approval | `TestExpiredApprovalRejectsPublish` + walkthrough observation | 409 Conflict | If publish succeeded with expired approval | N/A | N/A |
| Evidence updated to revision 12 | `D:\Projects\AI-go-starter` | `git diff specs/changes/minimal-cart-integration/control.json` | AC-011/012 observed_revision = 12 | Updated | N/A | N/A | N/A |
| Receipt exists and is non-empty | `D:\Projects\AI-go-starter` | `Test-Path receipts/b5-approval-walkthrough-rev12.md` | file exists, non-empty | True | N/A | N/A | N/A |
| speccheck passes | `D:\Projects\AI-go-starter` | `go run ./server/tools/speccheck` | "ok" | PASS (if no other dirty spec dirs conflict) | N/A | N/A | N/A |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] The bounded failure trigger fails for the claimed reason and is restored.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.

### Packet B01F: Discoverable static footer policy navigation without JavaScript

Status: implemented; browser walkthrough evidence pending
Covers: REQ-006, AC-012
Hard dependencies: none
Outcome: Public static HTML templates server-render semantic `<nav id="footer-static">` with `<a href="/content/<key>/">` links for all currently published and unexpired footer and policy content, providing accessible and discoverable policy navigation when JavaScript is disabled, while hiding the fallback on successful Footer island mount to prevent duplicate footers.
Safe failure / rollback: If sitecontent lookup fails, `rendercompose.Compose` returns an error and `renderToStaging` halts promotion, preserving the last-known-good `dist/`; if no footer/policy blocks exist, the templates render an honest empty state without broken links.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Authoritative data source | `server/internal/modules/sitecontent/store.go` | `SQLStore.ListPublished` | Queries `status = 'published'`, `published_updated_unix > 0`, and `published_approval_expiry_unix > now` sorted by `published_sort_order ASC, published_updated_unix DESC` |
| Composition owner | `server/tools/internal/rendercompose/compose.go` | `Compose` | Calls `scStore.ListPublished(ctx)` and extracts all published `footer` and `policy` blocks into `Input.ContentBlocks` |
| Render engine & data contracts | `server/internal/render/render.go` | `RenderAllFull`, `homeData`, `articleData`, `productPageData`, `categoryPageData`, `contentPageData` | Passes page data to html/template; currently omits `FooterContent` from template data structs |
| Public template (home) | `site/themes/minimal-cart/templates/home.html` | `<div data-vue-island="Footer"></div>` | Lacks static server-rendered `#footer-static` nav baseline |
| Public template (product) | `site/themes/minimal-cart/templates/product.html` | `<div data-vue-island="Footer"></div>` | Lacks static server-rendered `#footer-static` nav baseline |
| Public template (category) | `site/themes/minimal-cart/templates/category.html` | `<div data-vue-island="Footer"></div>` | Lacks static server-rendered `#footer-static` nav baseline |
| Public template (content) | `site/themes/minimal-cart/templates/content.html` | `<div data-vue-island="Footer"></div>` | Lacks static server-rendered `#footer-static` nav baseline |
| Public template (article) | `site/themes/minimal-cart/templates/article.html` | `<div data-vue-island="Footer"></div>` | Lacks static server-rendered `#footer-static` nav baseline |
| Progressive enhancement owner | `site/themes/minimal-cart/islands/Footer/Footer.vue` | `<script setup>`, `onMounted` | Needs `onMounted` hook to hide `#footer-static` on successful mount (matching `ProductGrid.vue` pattern) |
| Existing render test | `server/internal/render/render_test.go` | `TestRenderStagingSuccessPromotesToDist` | Proves atomic staging and template execution |
| Existing compose e2e test | `server/tools/internal/rendercompose/compose_test.go` | `TestComposeAndRenderProducesMinimalCartOutput` | Asserts `#shop-static`, product routes, category routes, and content routes in `dist/` |

Read set:
- `server/internal/modules/sitecontent/store.go` - understand published snapshot filtering (`published_approval_expiry_unix > now`)
- `server/tools/internal/rendercompose/compose.go` - observe that `Compose` already queries published content and populates `Input.ContentBlocks` with `footer` and `policy` blocks
- `server/internal/render/render.go` - understand template data shapes and staging render flow
- `site/themes/minimal-cart/templates/*.html` - understand existing static HTML template structures and island mount points
- `site/themes/minimal-cart/islands/Footer/Footer.vue` - understand Footer island lifecycle and mount behavior
- `site/themes/minimal-cart/islands/ProductGrid/ProductGrid.vue` - reference existing `hideStaticSection` / `onMounted` progressive-enhancement pattern
- `specs/changes/minimal-cart-integration/spec.md` - AC-012 normative requirements

Modify set:
- `server/internal/render/render.go` - add `FooterContent []SiteContentData` to `homeData`, `articleData`, `productPageData`, `categoryPageData`, `contentPageData`; pass `contentBlocks` from `RenderAllFull` to all sub-renderers (`renderHomeAndArticles`, `renderProductsTo`, `renderCategoriesTo`, `renderSiteContentTo`)
- `site/themes/minimal-cart/templates/home.html` - add server-rendered `<nav id="footer-static">` with `{{range .FooterContent}}` above `<div data-vue-island="Footer"></div>`
- `site/themes/minimal-cart/templates/product.html` - add server-rendered `<nav id="footer-static">` with `{{range .FooterContent}}` above `<div data-vue-island="Footer"></div>`
- `site/themes/minimal-cart/templates/category.html` - add server-rendered `<nav id="footer-static">` with `{{range .FooterContent}}` above `<div data-vue-island="Footer"></div>`
- `site/themes/minimal-cart/templates/content.html` - add server-rendered `<nav id="footer-static">` with `{{range .FooterContent}}` above `<div data-vue-island="Footer"></div>`
- `site/themes/minimal-cart/templates/article.html` - add server-rendered `<nav id="footer-static">` with `{{range .FooterContent}}` above `<div data-vue-island="Footer"></div>`
- `site/themes/minimal-cart/islands/Footer/Footer.vue` - add `onMounted(() => { document.getElementById('footer-static')?.setAttribute('hidden', '') })` to hide static fallback once Vue island mounts
- `server/internal/render/render_test.go` - add `TestRenderStaticFooterContentLinks` and `TestRenderStaticFooterEmptyContent`
- `server/tools/internal/rendercompose/compose_test.go` - update `TestComposeAndRenderProducesMinimalCartOutput` to assert static footer anchor tags in `dist/index.html`

Must not modify:
- `server/tools/internal/rendercompose/compose.go` - `Compose` already extracts published `footer` and `policy` blocks into `Input.ContentBlocks`
- `site/themes/minimal-cart/islands/Footer/Footer.vue` `NAV_LINKS` / dialog keys - preserve existing dialog behavior intact without speculative key mapping
- `server/internal/modules/sitecontent/**` - storage and approval gating are already complete and tested
- `admin/**` - admin content management is already complete
- `db/migrations/**` - no schema changes needed
- `contracts/**` - OpenAPI contracts already reflect site content endpoints
- Any file outside `specs/changes/minimal-cart-integration/**` during planning phase

#### Behavior contract

Before:
- Public static HTML pages contain `<div data-vue-island="Footer"></div>` without server-rendered footer markup or policy links.
- When JavaScript is disabled, the footer area is completely empty; no policy navigation is discoverable or clickable.
- `Footer.vue` mounts when JS runs, but does not coordinate with any static baseline.

After:
- Public static HTML templates (`home.html`, `product.html`, `category.html`, `content.html`, `article.html`) emit a focused semantic navigation fallback `<nav id="footer-static" class="border-t border-border/60 bg-muted/20 px-4 py-8 sm:px-6 lg:px-8" aria-label="網站資訊與政策">` with a list of `<a href="{{if $.PublicSiteURL}}{{$.PublicSiteURL}}{{end}}/content/{{.Key}}/">{{.Title}}</a>` for all currently published, unexpired `footer` and `policy` content blocks.
- If no footer/policy content blocks are published, the template gracefully renders an honest placeholder (`<p class="text-xs text-muted-foreground">政策與網站資訊將於正式上線前公告。</p>`) without broken links.
- When JavaScript is disabled or fails to execute, `#footer-static` remains visible, accessible, and navigable; visitors can click any link to open the static policy page (`/content/<key>/index.html`), which renders completely without client JS.
- When JavaScript executes successfully, `Footer.vue` mounts and its `onMounted` hook adds the `hidden` attribute to `#footer-static`, preventing dual/duplicate footers while providing interactive dialogs and newsletter features.

Preserved invariants:
- Only published, unexpired content rows (`status = 'published'` AND `published_approval_expiry_unix > now`) are included in `FooterContent`; draft or expired content is never rendered.
- Content route containment is enforced by `safeJoin` and `validateRouteSegment`.
- Template strings (`Title`, `Body`) are safely escaped by `html/template`.
- `Footer.vue` interactive dialog behavior and modal triggers remain unchanged.
- Atomic staging promotion via `renderToStaging` remains intact (render failure preserves last-known-good `dist/`).

Inputs and authority:
- Data source: `scStore.ListPublished(ctx)` via `rendercompose.Compose(ctx, db, dialect, r2PublicBaseURL)`.
- Server authority: Go renderer evaluates published status and expiry timestamps at render time.

Errors and edge states:
- Empty footer/policy blocks: honest empty placeholder rendered, no template error or broken `<a>` tags.
- JS disabled: semantic `<a href="/content/<key>/">` navigates directly to static policy routes.
- JS enabled: `#footer-static` is hidden by `onMounted` in `Footer.vue`, showing only the interactive island without visual duplication.
- Island mount failure: `#footer-static` stays visible as a safe degradation fallback.

#### Ordered edits

1. `server/internal/render/render.go` - `homeData`, `articleData`, `productPageData`, `categoryPageData`, `contentPageData`, `RenderAllFull`, `renderHomeAndArticles`, `renderProductsTo`, `renderCategoriesTo`, `renderSiteContentTo`
   - Change: Add `FooterContent []SiteContentData` field to all five page data structs (`homeData`, `articleData`, `productPageData`, `categoryPageData`, `contentPageData`). In `RenderAllFull`, pass `contentBlocks` directly to `renderHomeAndArticles`, `renderProductsTo`, `renderCategoriesTo`, and `renderSiteContentTo` as `FooterContent`.
   - Constraint: Do not alter `RenderAllFull` or `RenderAll` public function signatures.
   - Result: All rendered page templates receive the authoritative list of published footer/policy content blocks.

2. `site/themes/minimal-cart/templates/home.html`, `product.html`, `category.html`, `content.html`, `article.html` - static footer navigation markup
   - Change: Immediately above `<div data-vue-island="Footer"></div>`, add:
     ```html
     <nav id="footer-static" class="border-t border-border/60 bg-muted/20 px-4 py-8 sm:px-6 lg:px-8" aria-label="網站資訊與政策">
       <div class="mx-auto max-w-7xl">
         <h2 class="text-xs font-medium uppercase tracking-wider text-muted-foreground">網站資訊與政策</h2>
         {{if gt (len .FooterContent) 0}}
         <ul class="mt-4 flex flex-wrap gap-x-6 gap-y-2">
           {{range .FooterContent}}
           <li>
             <a href="{{if $.PublicSiteURL}}{{$.PublicSiteURL}}{{end}}/content/{{.Key}}/" class="text-sm text-foreground/80 transition-colors hover:text-foreground">{{.Title}}</a>
           </li>
           {{end}}
         </ul>
         {{else}}
         <p class="mt-4 text-xs text-muted-foreground">政策與網站資訊將於正式上線前公告。</p>
         {{end}}
       </div>
     </nav>
     ```
   - Constraint: Keep static fallback focused on accessible navigation only; do not duplicate newsletter forms, brand logo, or contact claims.
   - Result: All static HTML pages contain semantic footer navigation readable without JavaScript.

3. `site/themes/minimal-cart/islands/Footer/Footer.vue` - hide static fallback on island mount
   - Change: Import `onMounted` from `vue` (if not already imported) and add:
     ```typescript
     onMounted(() => {
       document.getElementById('footer-static')?.setAttribute('hidden', '')
     })
     ```
   - Constraint: Preserve existing `NAV_LINKS` array and `ui.openFooterPage(link.key)` dialog behavior unchanged.
   - Result: Island mount hides `#footer-static` to prevent dual visible footers when JavaScript is active.

4. `server/internal/render/render_test.go` - `TestRenderStaticFooterContentLinks`, `TestRenderStaticFooterEmptyContent`
   - Change: Add `TestRenderStaticFooterContentLinks` to assert that `RenderAllFull` outputs HTML containing `<nav id="footer-static"` and `<a href=".../content/footer.about/">About</a>` across home, product, category, content, and article pages. Add `TestRenderStaticFooterEmptyContent` to assert that when `contentBlocks` is empty, all pages render the honest empty notice `<p class="mt-4 text-xs text-muted-foreground">政策與網站資訊將於正式上線前公告。</p>` without broken `<a>` tags.
   - Constraint: Must not weaken existing render tests.
   - Result: Automated regression proof covering static footer link rendering and empty states across all 5 template families.

5. `server/tools/internal/rendercompose/compose_test.go` - `TestComposeAndRenderProducesMinimalCartOutput`
   - Change: Add assertions verifying that `dist/index.html` contains `<nav id="footer-static"` and `<a href="http://localhost:4173/content/footer.about/">About</a>`.
   - Constraint: Must not break existing e2e render composition assertions.
   - Result: End-to-end regression proof from SQLite DB seed through `Compose` to rendered `dist/index.html`.

#### Integration trace

```text
db published site_content rows (status='published', published_approval_expiry_unix > now, placement IN ('footer', 'policy'))
  -> scStore.ListPublished(ctx) (store.go)
  -> rendercompose.Compose (compose.go: filters placement == "footer" || "policy" into Input.ContentBlocks)
  -> Renderer.RenderAllFull (render.go: passes ContentBlocks as FooterContent to homeData/productPageData/categoryPageData/contentPageData/articleData)
  -> template execution (home.html, product.html, category.html, content.html, article.html)
  -> dist/*.html generated containing <nav id="footer-static"> with <a href="/content/<key>/">
  -> JS-disabled visitor loads dist/index.html -> views #footer-static -> clicks /content/footer.about/
  -> browser loads dist/content/footer.about/index.html -> renders policy title and body without client JS
  -> (progressive enhancement) JS-enabled visitor loads page -> islands.js mounts Footer.vue -> Footer.vue onMounted adds hidden to #footer-static -> only interactive Footer island is visible
```

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Static footer links rendered in home HTML | `D:\Projects\AI-go-starter` | `go test ./server/internal/render/ -run TestRenderStaticFooterContentLinks -v -count=1` | `TestRenderStaticFooterContentLinks` | PASS: `dist/index.html` contains `<nav id="footer-static"` and `<a href=".../content/footer.about/">` | If policy link href missing or malformed | Temporarily remove `FooterContent` from `homeData` in `render.go` -> assert test fails | Restore `render.go` + `git diff --check` |
| Static footer links across all 5 template families | `D:\Projects\AI-go-starter` | `go test ./server/internal/render/ -run TestRenderStaticFooterContentLinks -v -count=1` | `TestRenderStaticFooterContentLinks` | PASS: home, product, category, content, and article pages all contain `#footer-static` and link | If any template omits `#footer-static` | Temporarily remove `#footer-static` from `product.html` -> assert test fails | Restore `product.html` + `git diff --check` |
| Graceful empty state when no footer/policy content exists | `D:\Projects\AI-go-starter` | `go test ./server/internal/render/ -run TestRenderStaticFooterEmptyContent -v -count=1` | `TestRenderStaticFooterEmptyContent` | PASS: all 5 templates render honest empty notice, no broken `<a>` tags | If empty slice causes panic or empty broken hrefs | Temporarily remove empty check in `home.html` -> assert test fails | Restore `home.html` + `git diff --check` |
| End-to-end compose and render produces footer links in dist/ | `D:\Projects\AI-go-starter` | `go test ./server/tools/internal/rendercompose/ -run TestComposeAndRenderProducesMinimalCartOutput -v -count=1` | `TestComposeAndRenderProducesMinimalCartOutput` | PASS: seeded `footer.about` appears as `<a href="http://localhost:4173/content/footer.about/">` in `dist/index.html` | If `ContentBlocks` omitted from template data | Temporarily omit `FooterContent` in `RenderAllFull` -> assert test fails | Restore `render.go` + `git diff --check` |
| Theme build and islands typecheck | `D:\Projects\AI-go-starter\site\themes\minimal-cart` | `npm run build:check` | `dist/islands.js` and `dist/islands-*.css` | exit 0, build succeeds | If Footer.vue syntax or type error | N/A | N/A |
| Fresh render inspection | `D:\Projects\AI-go-starter` | `go run ./server/tools/render` then inspect `dist/index.html` | `dist/index.html`, `dist/content/footer.about/index.html` | `dist/index.html` contains `<nav id="footer-static"` with `<a href=".../content/footer.about/">` | If static nav missing in render output | N/A | N/A |
| JS-disabled browser navigation walkthrough | `D:\Projects\AI-go-starter` | `go run ./server/tools/dev` then navigate with JS disabled | Browser inspection of `http://localhost:4173/` -> click footer link -> `http://localhost:4173/content/footer.about/` loads and displays title/body | 200 OK, full policy text readable without JavaScript | If footer link missing or navigation broken | N/A | N/A |
| JS-enabled progressive enhancement no-duplicate walkthrough | `D:\Projects\AI-go-starter` | `go run ./server/tools/dev` then inspect DOM with JS enabled | Browser inspection of `http://localhost:4173/` -> `#footer-static` has `hidden` attribute; only interactive Footer island is visible | `#footer-static[hidden]` present; no visual duplication | If onMounted fails to hide `#footer-static` | N/A | N/A |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] The bounded failure trigger fails for the claimed reason and is restored.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.

#### Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass (REQ-006 -> B01E, B01F) |
| Every AC maps to proof in at least one packet | pass (AC-011 -> B01E, AC-012 -> B01E, B01F) |
| Every packet maps back to REQ/AC | pass (B01E -> REQ-006, AC-011, AC-012; B01F -> REQ-006, AC-012) |
| Baseline, observed HEAD, dirty paths, and packet ID mapping are explicit | pass |
| All paths and symbols were inspected | pass (render.go, compose.go, store.go, templates/*.html, Footer.vue, ProductGrid.vue, render_test.go, compose_test.go all read) |
| Hard dependency graph has no unexplained edge or cycle | pass (no hard dependencies) |
| Scope covers every modify path and no unrelated path | pass (only specs/changes/minimal-cart-integration/** during planning; named modify set for apply) |
| Product/authority decisions are approved or blocked | pass (progressive enhancement + static nav aligns with spec AC-012) |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |
