# Local and Production Environment Separation Specification

Change ID: separate-local-production-environments
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner issued a plain apply instruction on 2026-08-14 authorizing implementation of revision 1.
Repository baseline: 3c3a475c1bdbd27dc13f4c19e91b4b37dc4638f0
Supersedes: none

## Outcome

Separate local developer configuration from online deployment configuration so a repository dotenv file cannot silently influence a production Go runtime or production browser build. Local development uses `.env.development.local` as its primary dotenv file. Railway and Cloudflare Pages use provider-managed process or build environment variables and do not depend on a deployed `.env.production` file.

The existing Go backend, static Cloudflare Pages delivery, Supabase Auth/PostgreSQL, R2, Resend, admin API prefix grammar, and local SQLite workflow remain unchanged.

## Scope

In scope:

- Go configuration source selection for development and production.
- Development compatibility for an existing ignored root `.env` while `.env.development.local` becomes the documented primary local file.
- Production dotenv isolation for the admin Vite build and the public minimal-cart Vite build.
- Explicit browser-safe environment variable allowlists.
- Tracked local and production example inventories containing placeholders only.
- Ignore rules for real dotenv profiles and operator documentation mapping variables to Railway, Cloudflare Pages, and browser bundles.
- Safe migration of an existing local `.env` without reading, printing, overwriting, or committing its contents.

Out of scope:

- Creating or committing a real `.env.production` file.
- Changing provider accounts, credentials, domains, CORS rules, deployment topology, or Cloudflare/Railway projects.
- Rotating existing secrets or copying secret values between providers.
- Changing database schemas, authentication semantics, API routes, public rendering behavior, or adding a backend/runtime.
- Adding a generic configuration-provider abstraction.

## Decisions and invariants

### Configuration authority

| Runtime | Authoritative source | Dotenv behavior |
|---|---|---|
| Local Go tools/API | Process environment, then `.env.development.local`, then legacy `.env` fallback | Development only; process values always win |
| Railway Go runtime | Railway process environment with `APP_ENV=production` | No repository dotenv file is read |
| Cloudflare Pages Go render/build | Cloudflare build environment with `APP_ENV=production` | No repository dotenv file is read |
| Local admin/public Vite development | Process environment plus development dotenv profile | `.env.development.local` is primary; legacy `.env` remains transitional |
| Production admin/public Vite build | Provider build process environment | Vite dotenv loading is disabled |

- `APP_ENV=production` is a process-level deployment boundary. A dotenv value MUST NOT cause a process to enter production mode or supply missing production values.
- A real `.env.production`, `.env.development.local`, `.env.local`, or other local dotenv profile MUST remain ignored by Git. Tracked example files contain names, safe defaults, and placeholders only.
- Configuration loaders MUST NOT log secret names paired with values, raw tokens, connection strings, or dotenv contents.
- Missing required production configuration continues to fail through the existing validation path; production MUST NOT fall back to development authentication or SQLite because a local dotenv file exists.

### Browser-safe allowlist

Only these values may be deliberately compiled into browser assets:

- Both admin and public theme: `AUTH_MODE`, `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY`, `AUTH_GOOGLE_ENABLED`, `AUTH_LINE_ENABLED`.
- Admin only: `ADMIN_API_BASE`.

`SUPABASE_PUBLISHABLE_KEY` is treated as a public client identifier, not a server secret. Server-only values such as `DATABASE_URL`, `DEV_AUTH_TOKEN`, `R2_SECRET_ACCESS_KEY`, `RESEND_API_KEY`, provider service-role keys, and OAuth client secrets MUST NOT be injected into either browser bundle.

Cloudflare Pages may still provide server/build-only values to the Go renderer. Their presence in the build process does not authorize emission into `dist/` or `admin/dist/`.

## Requirements

### REQ-001: Go runtime profile isolation

The Go configuration loader MUST use local dotenv files only in non-production development and MUST use process environment variables exclusively when the process declares `APP_ENV=production`.

#### AC-001: Local precedence and compatibility

- GIVEN a non-production Go process
- WHEN a value exists in the process environment, `.env.development.local`, and legacy `.env`
- THEN the process value MUST win, followed by `.env.development.local`, with legacy `.env` used only when the primary local file does not supply the value.

#### AC-002: Production ignores repository dotenv files

- GIVEN `APP_ENV=production` is set in the process environment and repository dotenv files contain conflicting sentinel values
- WHEN the Go configuration is loaded
- THEN none of those dotenv values may enter the resulting configuration, and existing production validation MUST reject missing required process configuration rather than using local fallbacks.

### REQ-002: Browser build profile isolation

The admin and public minimal-cart Vite configurations MUST distinguish development from production: development may load the documented local profile, while production MUST disable repository dotenv loading and consume only explicitly allowed process environment values.

#### AC-003: Development Vite behavior remains usable

- GIVEN local development with `.env.development.local` or a transitional legacy `.env`
- WHEN the admin or public theme Vite development server resolves its browser-safe configuration
- THEN the configured public auth values MUST remain available, and the admin MUST continue to default to the local `/api` proxy when `ADMIN_API_BASE` is unset.

#### AC-004: Production Vite builds use process environment only

- GIVEN repository dotenv profiles contain conflicting public sentinel values and the build process supplies approved public values
- WHEN production builds run for both the admin and public minimal-cart theme
- THEN emitted assets MUST reflect only the approved process values and MUST not contain the conflicting dotenv values.

### REQ-003: Secret-safe templates and browser boundary

The repository MUST clearly separate safe examples from real configuration, ignore all real dotenv profiles, document provider ownership, and restrict browser injection to the approved public allowlist.

#### AC-005: Examples, ignore rules, and provider ownership are explicit

- GIVEN a developer or deploy operator follows the repository documentation
- WHEN they configure local development, Railway, or Cloudflare Pages
- THEN they can identify the correct authoritative location for every documented variable without creating or committing a real production dotenv file, and Git ignores the resulting local files.

#### AC-006: Server secrets never enter browser artifacts

- GIVEN unique sentinel values for representative server-only variables and valid public build values
- WHEN fresh production bundles are built for the admin and public theme
- THEN neither browser output contains a server-secret sentinel, and an independent security review confirms that only the approved public keys cross the browser build boundary.

### REQ-004: Existing consumers remain reachable

The source separation MUST preserve the existing local workflow and production consumers without changing API, authentication, rendering, or deployment contracts.

#### AC-007: Go and Vite consumers use the selected profile

- GIVEN development and isolated production-mode configurations
- WHEN the Go API/config path, Go renderer/config path, admin build, and public-theme build are exercised
- THEN each consumer receives values from its selected authoritative source, the existing admin API base fallback still works, and no consumer requires browser access to server secrets.

#### AC-008: Existing local file migration is non-destructive

- GIVEN an ignored root `.env` may already contain user configuration
- WHEN implementation migrates toward `.env.development.local`
- THEN it MUST never print file contents, MUST never overwrite an existing `.env.development.local`, and MUST preserve a working fallback or a recoverable rename path.

## Amendments

### Revision 1 (2026-08-14)

- Initial proposal based on repository baseline `3c3a475`.
- Repository inspection expanded the browser-build boundary from admin-only to both Vite consumers because `admin/vite.config.ts` and `site/themes/minimal-cart/vite.config.ts` each load the repository root dotenv files.
