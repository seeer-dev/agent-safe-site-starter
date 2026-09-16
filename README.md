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
in a browser.

### Where to sign up and what to set

| Service | Sign up at | Collect | Set on |
|---|---|---|---|
| Supabase | [supabase.com/dashboard](https://supabase.com/dashboard) — project page top-right **Connect** button has all three values (keys also under Settings → API Keys) | `DATABASE_URL`, `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY` | Railway (+ Pages build for DB/auth) |
| Cloudflare R2 | [dash.cloudflare.com](https://dash.cloudflare.com) → R2 | `R2_ACCOUNT_ID`, API token pair, `R2_BUCKET`, `R2_PUBLIC_BASE_URL` | Railway (+ `R2_PUBLIC_BASE_URL` on Pages) |
| Resend | [resend.com](https://resend.com) | `RESEND_API_KEY`, verified domain for `RESEND_FROM` | Railway |
| Railway | [railway.app](https://railway.app) → Deploy from GitHub | Railway domain or `api.` subdomain | — |
| Cloudflare Pages | dash.cloudflare.com → Workers & Pages | Pages domains, deploy hook URL | — |
| ECPay (optional) | [vendor.ecpay.com.tw](https://vendor.ecpay.com.tw) (stage: [vendor-stage.ecpay.com.tw](https://vendor-stage.ecpay.com.tw)) | `MerchantID`, `HashKey`, `HashIV` | Railway |

Key details that bite if missed:

- **Supabase transaction pooler** (port 6543): append
  `&default_query_exec_mode=simple_protocol` to `DATABASE_URL` — pgx
  prepared statements need it. Direct connection (5432) does not.
- **Supabase Data API settings** (Project Settings → Data API): nothing
  in this architecture calls the auto-generated REST API — the browser
  never queries Supabase and the Go API connects over the postgres
  protocol. Disable **Enable Data API** and **Automatically expose new
  tables**; enable **automatic RLS** as a harmless deny-by-default
  safety net (the `postgres` role bypasses RLS, so the backend is
  unaffected). Supabase Auth is a separate service and keeps working.
- **`SITE_THEME`** selects which theme `make site` builds and renders
  (`curatory` for this storefront; default `minimal-cart`).
- **`SITE_ORIGIN`** is the single CORS-allowed browser origin — set it to
  the exact public site origin, not a wildcard.
- **`EDGE_SECRET`** is set on Railway AND as a Pages Functions env var on
  both Pages projects. The repo-root `functions/api/[[path]].js` proxies
  same-origin `/api/*` to `API_ORIGIN` (the Railway URL) and injects the
  header — browsers never learn the origin, and direct origin hits get
  403. No `api.` subdomain or Transform Rule is needed. `/healthz` is
  exempt so Railway probes still work. Rotation is zero-outage: set
  `EDGE_SECRET_PREVIOUS` to the outgoing value on Railway alongside the
  new `EDGE_SECRET`, switch both Pages projects to the new value, then
  remove `EDGE_SECRET_PREVIOUS` (see `docs/environment-configuration.md`).
- **Admin SPA** needs `admin/public/_redirects` (`/* /index.html 200`,
  already shipped). Leave `ADMIN_API_BASE` unset — it defaults to
  same-origin `/api`, which the admin project's own Pages Function proxy
  forwards to Railway.

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
   `admin/dist`.
3. **API proxy (both projects):** repo-root `functions/api/[[path]].js`
   is picked up by every Pages project. Set **runtime** env vars
   `API_ORIGIN` (Railway URL) and `EDGE_SECRET` (same value as Railway)
   under each project's Settings → Environment variables — these are not
   build vars and never enter a bundle.
4. **Publish trigger (alternative):**
   `go run ./server/tools/publish` renders `dist/` as a pre-check
   (aborting on failure with the existing dist preserved), then POSTs
   the Pages **deploy hook** URL from `CF_DEPLOY_HOOK_URL`, which makes
   Pages rebuild from the connected Git branch — that build runs
   `make site` and renders from the production database. No dist upload
   happens. Admin publish actions only flip content state in the
   database; run `tools/publish` from CI or manually to take them live.

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
