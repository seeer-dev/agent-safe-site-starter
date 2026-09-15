# Deployment Readiness Specification

Change ID: deployment-readiness
Revision: 1
Status: Verifying
Decision authority: Repository owner/user
Approval basis: Owner direction 2026-09-15: write the deployment README and a single-page signup/settings guide with a final checklist; fixes found while verifying the documented deployment path are part of that work.
Repository baseline: 10f805e18e394e7065854dfbceb327114e0d4564
Supersedes: none

## Outcome

The documented deployment path (Cloudflare Pages storefront + admin SPA,
Railway Go API, Supabase, R2, Resend) is accurate and the repository
artifacts it depends on actually work: the Makefile builds the theme the
renderer will use, the admin SPA ships the `_redirects` file its
history-mode router needs on Pages, the container image can boot its
sqlite default as the non-root user, and the README + standalone guide
page tell the operator where to sign up, what to collect, and where each
value goes.

## Scope

In scope:

- `Makefile` — `theme`/`site` targets follow `SITE_THEME`.
- `admin/public/_redirects` — SPA fallback for Pages.
- `Dockerfile` — writable `/app/var` for the non-root sqlite default.
- `README.md`, `docs/deployment-guide.html` — deployment documentation.

Out of scope:

- Any provider-side configuration (accounts, secrets, DNS) — operator
  work described by the docs, not repo changes.

## Requirements

### REQ-001: `make site` builds the configured theme

`SITE_THEME=curatory` MUST make `make theme` / `make site` build
`site/themes/curatory`, matching what the renderer validates. Unset MUST
keep the `minimal-cart` default so existing CI is unchanged.

#### AC-001: Theme build follows SITE_THEME
- GIVEN `SITE_THEME=curatory` in the environment
- WHEN `make theme` runs
- THEN it builds `site/themes/curatory`; unset builds `minimal-cart`.

### REQ-002: Admin SPA deep links work on Pages

The admin router uses `createWebHistory`; on Pages, refreshing a
sub-route without a fallback returns 404. `admin/public/_redirects`
MUST contain `/* /index.html 200` so it is copied verbatim into
`admin/dist` on build.

#### AC-002: _redirects lands in admin/dist
- GIVEN a clean `npm --prefix admin run build:only`
- WHEN the build completes
- THEN `admin/dist/_redirects` exists with the SPA fallback rule.

### REQ-003: Container boots its sqlite default as non-root

The image runs as `app`, but the default sqlite DSN writes
`var/site.db` under root-owned `/app`. The Dockerfile MUST create a
writable `/app/var` so `migrate` and `api` work standalone; the Postgres
production path is unaffected.

#### AC-003: Image smoke test
- GIVEN the built image
- WHEN `migrate` then `api` run in-container
- THEN migrations apply, `/healthz` and `/api/products` return 200, and
  `APP_ENV=production` without config still refuses to start.

### REQ-004: Deployment docs are complete and accurate

`docs/deployment-guide.html` MUST list every required service with its
signup/console URL, the values to collect, which platform each value is
set on, and an end-of-guide checklist. README MUST link the guide and
state the same key facts without contradicting the ownership table in
`docs/environment-configuration.md`.

#### AC-004: Docs match reality
- GIVEN the guide and README
- WHEN compared against `railway.toml`, `Dockerfile`, the Makefile,
  `.env.production.example`, and `environment-configuration.md`
- THEN every documented command, output dir, and variable name exists.
