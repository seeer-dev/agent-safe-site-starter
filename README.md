# AI Site Starter — example v0

A deliberately small starter for sites that an AI agent can understand and modify safely.

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

No Nuxt, Next.js, frontend SSR runtime, DI container, plugin registry, or database access from the browser.

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

- **Site:** Cloudflare Pages
- **API:** Railway Go container
- **Database:** Supabase PostgreSQL
- **Auth:** Supabase Auth
- **Files:** Cloudflare R2
- **Email:** Resend

Set `DB_DRIVER=postgres`, `AUTH_MODE=supabase`, the provider environment variables, and production URLs. Railway uses `Dockerfile` + `railway.toml` and runs migrations as a pre-deploy command.

Production configuration lives in the provider, not in this repository. With `APP_ENV=production` the Go loader and both Vite builds read the process environment only and ignore every repository dotenv file, so there is no `.env.production` to create or deploy. `.env.production.example` is an inventory checklist; `docs/environment-configuration.md` has the full ownership table and the browser-safe allowlist.

For Pages, use either:

1. **Git build:** build command `go run ./server/tools/render`, output `dist`; this is the preferred CMS/deploy-hook path when Pages can reach the production database.
2. **Direct upload:** `CF_PAGES_PROJECT=... go run ./server/tools/publish`; useful for AI/CI-driven publishing.

Cloudflare Pages' build image supports Go and custom build commands, so the renderer does not require a JavaScript site framework.

## Repository map

```text
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
  assets/                plain CSS/JS
db/migrations/
  sqlite/
  postgres/
contracts/openapi.yaml   HTTP contract
skills/site/             one user-intent router skill
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

`Draft` records a proposal but cannot authorize protected edits. `Ready`, `Applying`, `Verifying`, or `Accepted` may cover implementation when approval and artifacts are valid. Maintainers and agents can run the deterministic gate directly or through the full verifier:

```bash
go run ./server/tools/speccheck
go run ./server/tools/verify
```

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
- Supabase token verification uses the Auth server `/auth/v1/user` endpoint for clarity. A cached JWKS verifier can replace the adapter later without changing modules.
- `body_html` is treated as trusted CMS-authored HTML. Sanitize it before persistence if untrusted users can author content.
- `scopecheck` remains a local narrow-diff gate because `.ai/scope.json` is task-local; `speccheck` is the durable PR/CI-controlled-spec gate.

Read `architecture.yaml`, `AGENTS.md`, `skills/site/SKILL.md`, and `workflows/safe-change.md` before expanding the starter.
