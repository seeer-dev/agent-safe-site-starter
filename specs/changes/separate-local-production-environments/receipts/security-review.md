# Security review — browser build boundary

Change ID: separate-local-production-environments
Revision: 1
Covers: AC-006
Reviewed: 2026-08-14
Reviewer: implementing agent, independent replay of build and scan

## Scope of review

Whether a server secret can reach `admin/dist/` or
`site/themes/minimal-cart/dist/` after this change, and whether a repository
dotenv file can influence a production browser build.

## Control implemented

Both `admin/vite.config.ts` and `site/themes/minimal-cart/vite.config.ts`:

1. Select `process.env` instead of `loadEnv(...)` when `mode === 'production'`,
   so no repository dotenv file is opened for a production build.
2. Build the `define` map exclusively from a literal `BROWSER_SAFE_KEYS`
   allowlist. A name absent from that list cannot be injected, regardless of
   what the build environment holds.

Allowlist contents:

- Both builds: `AUTH_MODE`, `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY`,
  `AUTH_GOOGLE_ENABLED`, `AUTH_LINE_ENABLED`
- Admin additionally: `ADMIN_API_BASE`

`SUPABASE_PUBLISHABLE_KEY` is a public client identifier. Supabase data is
protected by row-level security, not by keeping this value secret.

## Observation

A temporary `.env.production.local` was created at the repository root holding
conflicting public sentinels and four server-secret sentinels. The file did not
previously exist; it was deleted immediately after the scan. The developer's
existing `.env` was never opened, read, printed, moved, or modified — its size
and mtime were confirmed unchanged before and after (1058 bytes,
2026-08-13 20:00:57).

Fresh production builds were run for both consumers with approved public values
supplied through the process environment and the four server-secret sentinels
also present in the process environment (the Cloudflare Pages case, where the Go
renderer's secrets are legitimately in the build environment).

Result of scanning both output directories:

| Sentinel | Kind | admin/dist | theme dist | Expected |
|---|---|---|---|---|
| `SENTINEL_SERVER_SECRET_DBURL` | server secret | 0 | 0 | 0 |
| `SENTINEL_SERVER_SECRET_DEVTOKEN` | server secret | 0 | 0 | 0 |
| `SENTINEL_SERVER_SECRET_R2` | server secret | 0 | 0 | 0 |
| `SENTINEL_SERVER_SECRET_RESEND` | server secret | 0 | 0 | 0 |
| `SENTINEL_DOTENV_AUTHMODE` | dotenv conflict | 0 | 0 | 0 |
| `SENTINEL-DOTENV-SUPABASE` | dotenv conflict | 0 | 0 | 0 |
| `SENTINEL_DOTENV_PUBKEY` | dotenv conflict | 0 | 0 | 0 |
| `SENTINEL-DOTENV-APIBASE` | dotenv conflict | 0 | 0 | 0 |
| `approved-process.supabase.co` | approved public | 1 | 1 | ≥1 |
| `APPROVED_PROCESS_PUBKEY` | approved public | 1 | 1 | ≥1 |
| `api.approved-process.example` | approved public, admin only | 1 | — | ≥1 admin |

Only presence and absence were recorded. No bundle contents and no
configuration values from the developer's environment were printed.

## Findings

- No server-secret sentinel reached either bundle.
- No dotenv-supplied value reached either production bundle; the approved
  process values appeared instead, confirming the source switch is effective
  rather than merely additive.
- `ADMIN_API_BASE` remains admin-only. The public theme has no admin API base
  in its allowlist and none appeared in its output.

## Residual risk

The allowlist is a literal array in each Vite config. Adding a secret name to
it would expose that secret. This is deliberate — exposure now requires an
explicit, reviewable edit to a named list rather than happening as a side
effect of adding a build variable. `docs/environment-configuration.md`
documents the review expectation for that step.

The two allowlists are maintained separately in the two configs. They are not
mechanically compared; divergence is intended (admin has one extra name) but a
future accidental divergence would not be caught automatically.

## Conclusion

AC-006 satisfied. Only approved public keys cross the browser build boundary,
and production browser builds do not read repository dotenv files.
