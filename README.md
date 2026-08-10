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

Open the repository with a coding agent and describe the website:

> Make this into a small interior-design studio site. I need project cases, an inquiry form, admin-only publishing, and image uploads.

The agent should route itself through `skills/site/SKILL.md`, inspect the architecture, change the smallest relevant area, then verify the result.

## Run the local demo

Requires Go 1.24+ (Go 1.26 recommended).

```bash
cp .env.example .env
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

## Production shape

- **Site:** Cloudflare Pages
- **API:** Railway Go container
- **Database:** Supabase PostgreSQL
- **Auth:** Supabase Auth
- **Files:** Cloudflare R2
- **Email:** Resend

Set `DB_DRIVER=postgres`, `AUTH_MODE=supabase`, the provider environment variables, and production URLs. Railway uses `Dockerfile` + `railway.toml` and runs migrations as a pre-deploy command.

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
docs/                    short human explanations
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

## Intentional v0 limits

- Full-site render only; no incremental page graph yet.
- The sample is content/API infrastructure, not a finished admin UI.
- Supabase token verification uses the Auth server `/auth/v1/user` endpoint for clarity. A cached JWKS verifier can replace the adapter later without changing modules.
- `body_html` is treated as trusted CMS-authored HTML. Sanitize it before persistence if untrusted users can author content.
- `scopecheck` is a local/agent gate. A hosted agent runner can persist `.ai/scope.json` and enforce the same rule in PR CI later.

See `docs/architecture.md` before expanding the starter.
