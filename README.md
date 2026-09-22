# AI Site Starter — example v0

A deliberately small starter for sites that an AI agent can understand and modify safely.

## Current status

The architecture/governance foundation and the reference commerce purchase flow are now substantially complete for the intended starter scope.

**Completed for the current v1 boundary:**

- beginner single-site architecture, explicit bootstrap/module boundaries, and fail-closed architecture checks;
- controlled spec/evidence workflow, migration parity, live PostgreSQL tests, concurrency stress, and `go vet` CI;
- current Go runtime ↔ OpenAPI truth for all 56 registered HTTP operations, with a dependency-free symmetric route/method parity gate plus guarded observable status/schema checks in `make verify-contracts`;
- commerce catalog, cart rehydration, server-authoritative quote, shipping/payment configuration, promotions, guest/member checkout, idempotent order creation, stock transaction, order lookup, returns, and per-item restock;
- ECPay AIO v5 credit-card handoff with server-owned `CheckMacValue`, durable real-success ReturnURL reconciliation, replay protection, and browser-return re-query rather than browser-authoritative payment state;
- source-level ECPay conformance against pinned official `ECPay/ECPay-API-Skill@ae964f75…`, including callback `TradeAmt`, `SimulatePaid` non-authority, Go apostrophe CMV encoding, official SHA256 vectors, and standard HTTPS port-443 handling.

**Still required before calling the starter deploy-ready:**

1. run the documented sample-commerce acceptance walkthrough from a fresh database;
2. verify the production-shaped Railway / Cloudflare Pages / PostgreSQL / Supabase / R2 / Resend configuration and decide rate-limit enforcement from the real deployment topology/trusted client-IP source;
3. on a public HTTPS deployment, complete one ECPay stage transaction and record callback/payment-state/browser-return acceptance.

A real ECPay stage transaction is intentionally still required before production use. Source/protocol conformance cannot prove that the provider can reach the deployed ReturnURL.

Refunds, electronic invoices, logistics-provider integration, reconciliation jobs, and full commerce operations are **not blockers for starter v1**. They are optional outcome-driven extensions and should not turn this small starter into a full commerce framework by default.

See [`docs/project-status.md`](docs/project-status.md) for the canonical completion matrix and v1 boundary, [`docs/review-status.md`](docs/review-status.md) for the current interpretation of historical architecture reviews and controlled-change lifecycle debt, [`docs/commerce-acceptance.md`](docs/commerce-acceptance.md) for the source/deployment acceptance boundary, and [`docs/ecpay-official-conformance.md`](docs/ecpay-official-conformance.md) for the pinned official ECPay audit.

**Default path**

```text
Cloudflare Pages  <- static HTML/CSS/JS <- Go renderer <- CMS/content data
                                           |
Browser interactions -> Railway Go API ----+
                         |   |    |    |
                    SQLite/PG Auth  R2  Resend
                              |
                         Supabase Auth
```

No Nuxt, Next.js, frontend SSR runtime, DI container, plugin registry, or database access from the browser. Core site assets are plain static JavaScript/CSS; interactive Vue islands are confined to the selected commerce theme (`site/themes/minimal-cart/`), and the admin panel (`admin/`) is a separate Vue SPA.

## What a non-technical user does

Open the repository with a coding agent and ask for a proposal:

> propose: Make this into a small interior-design studio site. I need project cases, an inquiry form, admin-only publishing, and image uploads.

The agent inspects the repository and describes the proposed outcome, boundaries, important decisions, delivery slices, risks, and validation. If it is right, reply:

> apply

The agent then manages the controlled spec, implementation order, targeted tests, required UX walkthroughs, evidence, and repository verification. The user does not need to name a change ID, revision, REQ/AC, slice, artifact, or command.

## Run the local demo

Requires Go 1.24+ (Go 1.26 recommended).

```bash
cp .env.development.example .env.development.local
go mod download
go run ./server/tools/dev
```

Open `http://localhost:4173`.

Local defaults:

- SQLite database at `var/site.db`
- dev auth token `dev-admin`
- email is logged instead of sent
- R2 is disabled until configured
- the site is fully rendered into `dist/`

The dev command applies migrations, adds one sample article on an empty database, renders the site, starts the Go API on `:8080`, and serves `dist/` on `:4173`.

An existing root `.env` still works and is still read — you do not have to migrate. Local precedence is process environment, then `.env.development.local`, then `.env`.

## Production shape

- **Site:** Cloudflare Pages (static `dist/` from the Go renderer)
- **Admin:** Cloudflare Pages, second project (`admin/dist` Vue SPA)
- **API:** Railway Go container (`Dockerfile` + `railway.toml`; migrations run as a pre-deploy command)
- **Database:** Supabase PostgreSQL
- **Auth:** Supabase Auth
- **Files:** Cloudflare R2
- **Email:** Resend
- **Payments:** ECPay AIO v5 (optional; all four `ECPAY_*` vars blank disables it)

**Step-by-step guide with signup links, every value to collect, and a
progress checklist:** open [`docs/deployment-guide.html`](docs/deployment-guide.html)
in a browser. For the exact current source -> destination -> deploy sequence,
read [`skills/site-deployment-browser/references/deployment-value-refresh.md`](skills/site-deployment-browser/references/deployment-value-refresh.md)
and use its operator worksheet. The [deployment skill](skills/site-deployment-browser/SKILL.md)
is de-identified so another agent can guide a new installation; the documents
here record this installation's non-secret live/historical observations and
must not be copied as another site's settings.

**2026-09-20 live checkpoint:** the holder rotated the previously exposed
edge credential, set the storefront and admin Pages Function rows to encrypted
**Secret**, and confirmed the same new value in Railway. Nine Railway staged
changes were deployed; the new Railway and both Pages `staging` deployments
are successful. The four public proxy checks are now 200/403/200/200. Railway
lists all five R2 and three Resend variable names, storefront Pages has the
public R2 base URL, and the Resend sender domain is Verified. An authorized
image upload through the admin completed presign, browser PUT, API verify,
product association, and a loaded public R2 image on the storefront. The
contact endpoint returned `201` to one authorized labeled smoke inquiry after
a new Railway deployment. The holder found the matching message in spam, and
the matching Resend **Emails** row now shows `Delivered`. Mailbox arrival and
the provider event are proved; normal-inbox placement is not. Variable names
and domain verification alone are not enough. Use the
[current handoff checkpoint](docs/environment-configuration.md#current-provider-handoff-2026-09-20)
for this site's remaining proof; the
[operator worksheet](skills/site-deployment-browser/references/deployment-value-refresh.md#operator-worksheet)
has reusable source and destination paths. The 2026-09-16 ledger below is history.

**Temporary test URLs:** [storefront](https://agent-safe-site-starter.pages.dev/)
and [admin](https://agent-safe-site-starter-admin.pages.dev/). Their API proxy
and the earlier admin→product→cart→checkout-first-step smoke are proved after
the credential cutover. The admin deep link also loaded once on the public
site, but its local redirect-source repair has not been released. R2 image
delivery is proved on these test URLs; the labeled mail reached the holder's
spam folder and Resend shows `Delivered`. Inbox placement remains open.
No gateway payment was run. The earlier shopping smoke was **not a completed
order**: the live checkout initially had two available shipping methods but no
payment method. With the owner's authorization, a visibly labeled
`manual_test` option (sandbox, ready, zero fee, **not charged**) was added in
the active Supabase database. The Pages API and server quote now accept it;
one owner-entered order has since been created: it is pending/unpaid with one
smoke item; stock fell from 3 to 2. Its order notification was **skipped**
because production had no `order_placed` template at the time. The owner then
approved a normal customer-facing template distinguishing order creation from
payment/fulfillment; the admin now shows exactly one enabled `order_placed`
row. The first order was not retroactively mailed. The holder then submitted
a second controlled order: it is pending/unpaid for one smoke item (NT$61),
stock fell from 2 to 1, and its single `order_placed` log is `sent:mail` with
nonempty rendered content and no error. Resend independently shows the
matching order subject as `Delivered`. A holder-provided mailbox screenshot
then showed that order notification in **spam**, alongside both earlier
contact-test messages. Thus mailbox arrival is confirmed, but normal-inbox
placement is not. This public test method must be
disabled or replaced before a real-payment launch. The storefront still has
static copy claiming card/LINE Pay/COD support; those methods are not proved.

**Test-loop verdict (2026-09-20):** On the two temporary `.pages.dev` origins,
the authorized owner can sign in and publish the labeled product with its R2
image; the storefront hydrates the live catalog, adds it to cart, obtains a
server quote, and places an unpaid `manual_test` order. The database recorded
the order and one-unit stock decrement; the enabled normal template produced
one `sent:mail` log, Resend marked the matching email `Delivered`, and the
holder found it in spam. This is a proved **no-gateway test commerce loop**,
not real-payment, fulfillment, normal-inbox, custom-domain, or unreleased admin
redirect-source acceptance. See the [dated evidence and remaining release
checks](docs/environment-configuration.md#current-provider-handoff-2026-09-20).

The earlier Resend message's **Insights** flagged missing DMARC. With the
holder's authorization, a monitoring-only `_dmarc` TXT (`v=DMARC1; p=none;`)
was published in the sending zone on 2026-09-20; the Cloudflare record list
and a fresh public DNS lookup both showed it. No R2 key, Railway setting, or
Pages deployment changed. One separately authorized new labeled contact test
returned `201`; Resend now shows `Delivered` and its DMARC item under
**Doing great** with `p=none`. The holder's screenshot places that newer
contact test in **spam** too; the receiving mailbox's SPF/DKIM/DMARC verdict
is still unknown. The monitoring policy alone did not solve inbox placement
in this observation. Do not infer a single cause or change DNS again without
examining redacted receiver authentication/spam-reason evidence. See the
[mail deliverability follow-up](docs/environment-configuration.md#mail-deliverability-follow-up-dns-published).

**Observed recovery record:** the [controlled deployment evidence](specs/changes/deployment-dashboard-ledger/evidence.md)
separates the completed admin-to-cart-to-order/stock proof from the skipped
customer notification and untested gateway work, and from the remote admin
source-release check.

### Where to sign up and what to set

| Service | Sign up at | Collect | Set on |
|---|---|---|---|
| Supabase | [supabase.com/dashboard](https://supabase.com/dashboard) — project page top-right **Connect** button has all three values (keys also under Settings → API Keys) | `DATABASE_URL`, `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY` | Railway; storefront Pages build needs the DB URI and public auth values; admin Pages needs public auth values only |
| Cloudflare R2 | [dash.cloudflare.com](https://dash.cloudflare.com) → **儲存空間和資料庫（Storage & databases）** → **R2 物件儲存（R2 Object Storage）** | `R2_ACCOUNT_ID`, bucket-scoped Object Read & Write key pair, `R2_BUCKET`, `R2_PUBLIC_BASE_URL` | Railway; `R2_PUBLIC_BASE_URL` additionally goes to the storefront Pages **build** only |
| Resend | [resend.com](https://resend.com) → **Domains** / **API Keys** | verified owned domain or subdomain, `RESEND_API_KEY`, `RESEND_FROM`, `CONTACT_NOTIFY_TO` | Railway only; no Pages or browser value |
| Railway | [railway.app](https://railway.app) → Deploy from GitHub | Railway domain or `api.` subdomain | — |
| Cloudflare Pages | dash.cloudflare.com → Workers & Pages | Pages domains, deploy hook URL | — |
| ECPay (optional) | [vendor.ecpay.com.tw](https://vendor.ecpay.com.tw) (stage: [vendor-stage.ecpay.com.tw](https://vendor-stage.ecpay.com.tw)) | `MerchantID`, `HashKey`, `HashIV` | Railway |

Key details that bite if missed:

- **Supabase connection method:** this long-running Railway Go service uses
  **Connect → Direct connection string → Shared pooler → Session mode → URI**
  (the dashboard labels it `Session pooler`) as the IPv4-safe default. Enter
  that URI directly as `DATABASE_URL` on Railway and
  as an encrypted storefront Pages build secret; do not route it through a
  Pages Function. Use direct connection only after verified Railway outbound
  IPv6 or the Supabase IPv4 add-on. Transaction pooler is only for a future
  short-lived/serverless runtime, where pgx needs
  `default_query_exec_mode=simple_protocol`.
- **Supabase Data API settings** (Project Settings → Data API): nothing
  in this architecture calls the auto-generated REST API — the browser
  never queries Supabase and the Go API connects over the postgres
  protocol. Disable **Enable Data API** and **Automatically expose new
  tables**; enable **automatic RLS** as a harmless deny-by-default
  safety net (the `postgres` role bypasses RLS, so the backend is
  unaffected). Supabase Auth is a separate service and keeps working.
- **Supabase Auth account is not an admin grant:** a successful sign-in
  creates a session only. The Go API grants capabilities only when
  `public.staff_members.supabase_user_id` exactly matches the Auth user ID,
  its `status` is `active`, and its staff role is supported. It never
  falls back to matching email. Provision the first active owner only with
  current user authorization, then reload the public admin app and require
  it to display `owner`; see
  [`docs/environment-configuration.md`](docs/environment-configuration.md)
  and the deployment skill for the field-by-field procedure.
- **`SITE_THEME`** selects which theme `make site` builds and renders
  (`curatory` for this storefront; default `minimal-cart`).
- **`SITE_ORIGIN`** is the single CORS-allowed browser origin — set it to
  the exact public site origin, not a wildcard.
- **`EDGE_SECRET`** is set on Railway AND as a Pages Functions env var on
  both Pages projects. The repo-root `functions/api/[[path]].js` proxies
  same-origin `/api/*` to `API_ORIGIN` (the Railway URL) and injects the
  header — browsers never learn the origin, and direct origin hits get
  403. No `api.` subdomain or Transform Rule is needed. `/healthz` is
  exempt so Railway probes still work. **Planned, uncompromised** rotation is zero-outage: set
  `EDGE_SECRET_PREVIOUS` to the outgoing value on Railway alongside the
   new `EDGE_SECRET`, switch both Pages projects to the new value, then
   remove `EDGE_SECRET_PREVIOUS` (see `docs/environment-configuration.md`).
  For today's exposed Text value, never extend its validity with
  `EDGE_SECRET_PREVIOUS`; use the emergency cutover in the checkpoint and
  accept a short API interruption.
- **Cloudflare secret type is a release gate:** the Cloudflare Add dialog may
  default to **Text** (`文字`). For `DATABASE_URL` and `EDGE_SECRET`, select
  **Secret** / **Encrypt** before entry and verify the saved row is encrypted.
  A row with the right name but Text type is a compromised value: rotate it,
  delete the row, and recreate it as a Secret; do not deploy or reuse it.
- **Admin SPA** must not add `/* /index.html 200`: Cloudflare Pages rejects
  that catch-all as an infinite loop. With no top-level `404.html`, Pages uses
  its default SPA fallback for non-asset routes. Leave `ADMIN_API_BASE` unset
  so it defaults to same-origin `/api`, which the admin project's own Pages
  Function proxy forwards to Railway.
- **R2 has five different source fields, not one generic setup step:**
  `R2_ACCOUNT_ID` is the non-secret value under **R2 物件儲存（R2 Object Storage）
  → 概觀（Overview）→ 帳戶詳細資訊（Account details）→ 帳戶 ID（Account ID）** and is
  Railway Text. The Object Read & Write key pair comes from **管理 API 令牌
  （Manage API Tokens）→ 帳戶 API 權杖（Account API Tokens）→ 建立 Account API
  權杖（Create Account API Token）**, scoped to the one bucket,
  and both values are Railway Secrets. `R2_PUBLIC_BASE_URL` is Text on Railway
  and the storefront Pages build only; first save the narrow browser-upload
  CORS rule. The Chinese-dashboard field map, deployment order, and failure
  table are in
  [`docs/environment-configuration.md`](docs/environment-configuration.md#r2-media-deployment).
- **Resend has three different runtime roles:** first verify an owned domain or
  sending subdomain in **Domains → Add Domain** (a `*.pages.dev` hostname is
  not an owned DNS zone and cannot serve as the production sender). Copy every
  generated DNS row into the matching Cloudflare zone, verify the domain, then
  create an **API Keys → Create API Key** credential with **Sending access**
  restricted to that domain. Put only `RESEND_API_KEY` in a Railway **Secret**;
  put the matching sender such as `品牌 <orders@mail.example.com>` in Railway
  `RESEND_FROM` **Text**. `CONTACT_NOTIFY_TO` is the private management inbox
  for website contact-form notices, not the recipient for customer order
  notices; keep it server-only and protected. The complete Chinese explanation
  plus the English dashboard labels, DNS mapping, test boundary, and failure
  table are in
  [`docs/environment-configuration.md`](docs/environment-configuration.md#resend-mail-deployment).

### Where production configuration lives

Production configuration lives in the provider, not in this repository.
With `APP_ENV=production` the Go loader and both Vite builds read the
process environment only and ignore every repository dotenv file, so
there is no `.env.production` to create or deploy. `AUTH_MODE=dev` is
refused in production. `.env.production.example` is an inventory
checklist; `docs/environment-configuration.md` has the full ownership
table and the browser-safe allowlist.

### Cloudflare Pages builds

1. **Storefront (Git build):** connect the repo, build command
   `make site`, output `dist`. `make site` builds the theme bundle the
   renderer requires (the bundle is git-ignored, so a clean checkout has
   none and `go run ./server/tools/render` fails closed until built) and
   then renders. The build image needs Node as well as Go — Cloudflare's
   image has both. The renderer reads the production database during the
   build, so `DATABASE_URL` must be reachable from Pages builds.
2. **Admin (second Pages project):** build command
   `npm --prefix admin ci && npm --prefix admin run build:only`, output
   `admin/dist`. It needs `AUTH_MODE`, `SUPABASE_URL`, and
   `SUPABASE_PUBLISHABLE_KEY`; it does not need `DATABASE_URL`.
3. **API proxy (both projects):** repo-root `functions/api/[[path]].js`
   is picked up by every Pages project. Set **runtime** env vars
   `API_ORIGIN` (Railway URL) and `EDGE_SECRET` (same value as Railway)
   under each project's Settings → Variables and Secrets — these are not
   build vars and never enter a bundle.
4. **Catalog freshness and publish trigger:** the curatory storefront's
   Vue islands fetch the live, same-origin `/api/products` route after
   hydration, so an active product can immediately appear in the
   browser's shop/cart/checkout flow even when the first static HTML
   fallback still reflects the prior renderer pass. Wait for hydration
   before diagnosing a missing product.
5. **Static/SEO refresh (alternative):**
   `go run ./server/tools/publish` renders `dist/` as a pre-check
   (aborting on failure with the existing dist preserved), then POSTs
   the Pages **deploy hook** URL from `CF_DEPLOY_HOOK_URL`, which makes
   Pages rebuild from the connected Git branch — that build runs
   `make site` and renders from the production database. No dist upload
   happens. Use this from CI or manually when the static no-JavaScript/SEO
   snapshot must refresh; the deploy hook is not required for the hydrated
   no-payment commerce smoke flow.

## Repository map

```text
admin/                   separate Vue SPA for store management
server/
  cmd/api/               Go API entrypoint
  internal/
    bootstrap/           explicit wiring
    auth/                Principal + dev/Supabase verification
    modules/             vertical business modules
    platform/            DB/mail/storage adapters
    render/              CMS data -> static HTML
  tools/                 deterministic commands for AI/CI
site/
  templates/             human-readable HTML templates
  assets/                plain CSS/JS for core static pages
  themes/                interactive themes with scoped Vue islands
db/migrations/
  sqlite/
  postgres/
contracts/openapi.yaml   current HTTP contract, parity-gated against registered Go operations
skills/
  site/                  user-intent router skill
  expand-implementation/ proposal-to-blueprint expansion skill
specs/changes/           controlled specs, plans, and evidence
workflows/               concise operator workflows
architecture.yaml        machine/human architecture map
AGENTS.md                 hard agent rules
```

## Example protected calls

With the local dev token:

```bash
curl -X POST http://localhost:8080/api/admin/articles \
  -H 'Authorization: Bearer dev-admin' \
  -H 'Content-Type: application/json' \
  -d '{
    "slug":"second-post",
    "title":"Second post",
    "excerpt":"A second static page",
    "body_html":"<p>Rendered by Go.</p>",
    "published":true
  }'

go run ./server/tools/render
```

For an image upload URL, configure R2 and call `POST /api/media/presign` with the same auth header.

## Controlled changes

Protected implementation and governance paths require one controlled change under `specs/changes/<change-id>/`:

```text
control.json  machine-readable status, scope, REQ/AC IDs, and evidence state
spec.md       normative behavior and acceptance scenarios
plan.md       vertical slices and traceability
evidence.md   observed proof for every REQ/AC
```

These are agent- and reviewer-facing controls, not extra user commands. The normal user workflow remains `propose <outcome>`, review the summary, then plain `apply`. A separate `walkthrough` request is available for a read-only UX audit, but apply already runs any walkthrough needed for acceptance.

For non-trivial work, the site router uses `skills/expand-implementation/` internally to turn the proposal into repository-grounded implementation packets in the existing `plan.md`. The expansion records exact paths and symbols, ordered edits, boundaries, verification, falsification, and drift stops. It does not choose an implementation agent and does not add a user command.

`Draft` records a proposal but cannot authorize protected edits. `Ready`, `Applying`, `Verifying`, or `Accepted` may cover implementation when approval and artifacts are valid. Maintainers and agents can run the deterministic gate directly or through the full verifier:

```bash
go run ./server/tools/speccheck
go run ./server/tools/verify
```

Frontend/HTTP contract checks run separately:

```bash
make verify-contracts
```

It runs, in order:

- `contracts/check-runtime-openapi.mjs` — symmetric path/method parity for all registered Go operations plus guarded observable status/schema boundaries;
- `admin/scripts/check-resource-contracts.mjs` — admin resource/payload invariants;
- `site/themes/minimal-cart/scripts/check-openapi-contracts.mjs` — public theme/OpenAPI invariants.

The runtime/OpenAPI checker currently proves all **56 registered operations** are represented and mutation evidence under `restore-http-contract-truth` shows both route omission and guarded success-status drift turn the gate red. The ECPay callback component is additionally guarded for official `TradeAmt` and optional `SimulatePaid` fields. These scripts import only Node standard-library modules, so there is no install step for the checkers themselves. CI runs `make verify-contracts` as a required gate.

It needs **Node 20.11 or newer** — `check-resource-contracts.mjs` uses `import.meta.dirname` — and the repository reports a plain version error before running the scripts when Node is too old.

`go run ./server/tools/verify` stays Go-only and never invokes Node, so a contributor without a Node toolchain is not blocked by the repository verifier.

Pull-request CI runs the same gate against the event's base commit and accepts only an `Accepted` spec that is part of the current comparison diff. Merged Accepted/Superseded specs are immutable and cannot authorize later changes. This makes missing specs, artifact drift, stale-spec reuse, unapproved states, uncovered protected paths, duplicate spec ownership, and false `Accepted` claims fail mechanically instead of relying only on agent instructions.

### Evidence that can pass

For new non-trivial changes, `control.json` uses `strict_evidence: true`. Passed evidence is bound to the current revision with `observed_revision`; acceptance scenarios that require runtime, security, production-content, consumer-reachability, or independent review proof also name a non-secret receipt under the same change directory. `speccheck` rejects stale revisions, missing or unsafe receipt paths, and proof text that still describes an unresolved gap.

An implementation report is a handoff, not acceptance evidence. The reviewer checks the diff and independently replays the relevant commands and output inspection before restoring `passed`. If a required walkthrough, caller, session producer, failure state, approved content source, or review is still missing, the mapped REQ/AC stays `pending`.

Use these acceptance boundaries:

- An API helper, route, authenticated branch, or component definition is not integrated until a real entry point reaches it through the actual identity/session/configuration source and renders authoritative success, empty, authorization, not-found, and network-failure states.
- Protected UI fails closed. It must not replace authoritative empty or error results with fixture, browser-local, or invented data.
- Recovery requires proof of control of an approved factor. Email addresses, order IDs, names, or other identifying/contact data alone do not authorize recovery; the review also covers expiry, single use, replay, rate limiting, enumeration, rotation, audit, and safe failure.
- Production claims trace to an approved source and are checked in source, a fresh client build, and freshly rendered output. A selected keyword search or implementer summary is insufficient.
- Runtime acceptance records a structured walkthrough receipt with revision, surface, persona, state, expectation, observation, and supporting request, assertion, log, or screenshot.

## Intentional v0 limits

- Full-site render only; no incremental page graph yet.
- The sample is content/API infrastructure, not a finished admin UI.
- Supabase token verification defaults to the remote `/auth/v1/user` endpoint (`SUPABASE_VERIFIER_MODE=remote`), providing immediate revocation and legacy compatibility. Projects issuing asymmetric tokens can select local verification (`SUPABASE_VERIFIER_MODE=jwks`) to avoid per-request network hops; in JWKS mode, server-side sign-out is recognized upon token expiry, and operators can roll back to remote verification by changing the environment variable and redeploying.
- `body_html` is treated as trusted CMS-authored HTML. Sanitize it before persistence if untrusted users can author content.
- `scopecheck` operates as a local narrow-diff gate with two supported modes:
  - **Legacy mode** reads `.ai/scope.json` for single-task/clean working trees, but is not durable cross-task attribution proof.
  - **Selected linked-worktree mode** (`SCOPE_CHANGE_ID=<change-id>`) runs in an isolated linked worktree to validate the full diff against the controlled `applies_to` and `repository_baseline`, stopping on ownership overlaps across worktrees. Selected mode is local-only and forbidden in CI (CI requires full-diff verification against the PR base commit). In-repository scratch files and secrets in reports are forbidden; in `finally`, remove the temporary worktree and prune Git metadata.
- `speccheck` is the durable PR/CI-controlled-spec gate.

Read `architecture.yaml`, `AGENTS.md`, `skills/site/SKILL.md`, and `workflows/safe-change.md` before expanding the starter.
