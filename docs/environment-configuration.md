# Environment configuration

How local development and online deployment get their configuration, and why
a file in this repository can never supply a production value.

## The boundary

`APP_ENV` is read from the **process environment** before any dotenv file is
opened (`server/internal/config/config.go`). That single ordering is the whole
mechanism:

| Runtime | Authoritative source | Repository dotenv files |
|---|---|---|
| Local Go API, tools, renderer | process env → `.env.development.local` → `.env` | read |
| Railway Go API | Railway process environment, `APP_ENV=production` | **ignored** |
| Cloudflare Pages Go render/build | Pages build environment, `APP_ENV=production` | **ignored** |
| Local admin / public-theme Vite | process env + local dotenv profile | read |
| Production admin / public-theme Vite build | provider build environment | **ignored** |

Consequences worth stating plainly:

- A dotenv file **cannot** put a process into production mode. Only the
  process environment can set `APP_ENV=production`.
- A dotenv file **cannot** rescue a production deployment that is missing a
  required value. Validation fails closed instead
  (`Config.Validate`), so a misconfigured deploy stops rather than silently
  falling back to SQLite or dev authentication.
- There is no `.env.production` to deploy. Do not create one.

## Local development

```bash
cp .env.development.example .env.development.local
# edit .env.development.local
go run ./server/tools/dev
```

Precedence is **process environment > `.env.development.local` > `.env`**. The
first source that supplies a name wins; nothing overwrites a value already in
the process environment.

`.env` is the legacy location. It still works — if you already have one, you
do not need to do anything. To move over, copy the values you want into
`.env.development.local`; anything you leave behind is still picked up from
`.env`. Nothing in this repository will rename, rewrite, or read out your
existing `.env`.

All real profiles (`.env`, `.env.local`, `.env.development`,
`.env.development.local`, `.env.production`, `.env.production.local`) are
git-ignored. The tracked `*.example` files hold placeholders only.

### Ephemeral local PostgreSQL gate

To run the full PostgreSQL 16 live integration test suite locally without
setting up an external PostgreSQL service or modifying local `.env` files:

```bash
go run ./server/tools/local-postgres-gate
```

Key characteristics and operational boundaries:

- **Single command & parity**: Starts an isolated PostgreSQL 16 instance on
  loopback (`127.0.0.1`) matching CI, injects a child-scoped `TEST_DATABASE_URL`
  into `postgres-live-gate`, and runs every required `TestPostgresLive*` test.
- **Process isolation**: `TEST_DATABASE_URL` exists only inside the child test
  process. No credentials or endpoints are written to `.env` files, repository
  files, or logs.
- **Automatic cleanup**: When tests complete (or on failure / interrupt), the
  child PostgreSQL process is stopped and all temporary runtime/data directories
  are removed automatically.
- **Binary caching & offline behavior**: On the first run, PostgreSQL 16
  binaries are downloaded to the user cache directory
  (`%LocalAppData%/embedded-postgres-binaries` on Windows,
  `~/.cache/embedded-postgres-binaries` on Linux). Subsequent runs execute
  offline using the cached binaries. First-run execution requires network
  connectivity to download the binary archive.
- **Port collisions**: Defaults to loopback port `5433` to avoid collisions
  with any default PostgreSQL service on port `5432`. If port `5433` is
  occupied, use the `-port` flag (e.g. `go run ./server/tools/local-postgres-gate -port 5434`).
- **Validation boundary**: Local test success verifies code against a real
  PostgreSQL 16 engine, but does not substitute for CI run receipts or production
  deployment validation.

## Deployment ownership

| Variable | Railway (Go API) | Pages (build) | Browser bundle | Secret |
|---|:--:|:--:|:--:|:--:|
| `APP_ENV=production` | ● | ● | | |
| `HTTP_ADDR` | ● | | | |
| `SITE_ORIGIN` | ● | | | |
| `PUBLIC_SITE_URL` | ● | ● | | |
| `PUBLIC_API_BASE` | ● | ● | | |
| `ADMIN_API_BASE` | | ● | ● | |
| `DB_DRIVER` | ● | ● | | |
| `DATABASE_URL` | ● | ● | | ● |
| `AUTH_MODE` | ● | ● | ● | |
| `SUPABASE_URL` | ● | ● | ● | |
| `SUPABASE_PUBLISHABLE_KEY` | ● | ● | ● | |
| `AUTH_GOOGLE_ENABLED` | | ● | ● | |
| `AUTH_LINE_ENABLED` | | ● | ● | |
| `DEV_AUTH_TOKEN` | | | | ● (never in production) |
| `R2_ACCOUNT_ID` | ● | | | ● |
| `R2_ACCESS_KEY_ID` | ● | | | ● |
| `R2_SECRET_ACCESS_KEY` | ● | | | ● |
| `R2_BUCKET` | ● | | | |
| `R2_PUBLIC_BASE_URL` | ● | ● | | |
| `RESEND_API_KEY` | ● | | | ● |
| `RESEND_FROM` | ● | | | |
| `CONTACT_NOTIFY_TO` | ● | | | |
| `CF_PAGES_PROJECT` | | ● | | |
| `CF_DEPLOY_HOOK_URL` | ● | | | ● |

## The browser boundary

Cloudflare Pages holds server-only values because the Go renderer runs in the
same build. Those values are present in the build process but must not reach
`dist/`. Only an explicit allowlist crosses:

- Both admin and public theme: `AUTH_MODE`, `SUPABASE_URL`,
  `SUPABASE_PUBLISHABLE_KEY`, `AUTH_GOOGLE_ENABLED`, `AUTH_LINE_ENABLED`
- Admin only: `ADMIN_API_BASE`

The list is `BROWSER_SAFE_KEYS` in `admin/vite.config.ts` and
`site/themes/minimal-cart/vite.config.ts`. Both builds construct their
`define` map from it, so adding a variable to the build environment does not
expose it — a name has to be added to the allowlist deliberately.

`SUPABASE_PUBLISHABLE_KEY` is a public client identifier, not a server secret.
Row-level security, not key secrecy, is what protects Supabase data.

OAuth client secrets belong in the Supabase dashboard. They are never
environment variables here and never enter a bundle.

### Checking it

Build with a sentinel in a server-only variable and confirm it is absent:

```bash
RESEND_API_KEY=SENTINEL_CHECK npm --prefix admin run build:only
grep -r SENTINEL_CHECK admin/dist/ ; # expect no matches
```

## Adding a variable

1. Add it to `.env.development.example` and/or `.env.production.example` with a
   placeholder and an ownership comment.
2. Read it in `server/internal/config/config.go` if the Go side needs it.
3. Only if a browser genuinely needs it, and only if it is not a secret, add it
   to `BROWSER_SAFE_KEYS` in the relevant `vite.config.ts`.
4. Update the ownership table above.

Step 3 is the one that needs a second opinion. Everything reachable from a
browser bundle is public.
