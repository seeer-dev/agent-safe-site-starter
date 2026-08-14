# Consumer reachability — Go and Vite configuration sources

Change ID: separate-local-production-environments
Revision: 1
Covers: AC-007, and supports AC-001, AC-002, AC-003, AC-004, AC-008
Observed: 2026-08-14

Every consumer was exercised through its real entry point. No fixture stood in
for a configuration source.

## Consumer 1 — Go configuration, development

Entry point: `config.Load()` (used by the API, migration, render, publish, and
dev tools).

| Surface | State | Expectation | Observation |
|---|---|---|---|
| `TestLoadDevelopmentPrecedence` | process + `.env.development.local` + `.env` all supply `PUBLIC_SITE_URL` | process wins | `from-process` |
| same | `.env.development.local` + `.env` supply `DEV_AUTH_TOKEN` | primary local file wins | `from-development-local` |
| same | only `.env` supplies `SITE_ORIGIN` | legacy fallback still works | `from-legacy-env` |
| `TestLoadDevelopmentLegacyOnly` | `.env` only, no primary file | existing setup keeps working | `from-legacy-env` |
| `TestLoadDevelopmentWithoutDotEnvFiles` | no dotenv files | documented defaults | `dev-admin`, `sqlite` |

Command: `go test ./server/internal/config -count=5` → ok.

Each test runs in its own `t.TempDir()` with `t.Chdir`. No test opens, reads, or
asserts on the developer's real dotenv files.

## Consumer 2 — Go configuration, production

| Surface | State | Expectation | Observation |
|---|---|---|---|
| `TestLoadProductionIgnoresRepositoryDotEnvFiles` | `APP_ENV=production` in process; both dotenv files hold `sentinel-must-not-load` | no dotenv value enters config | sentinels absent; `DATABASE_URL` = `postgres://from-process` |
| same | dotenv sets `AUTH_MODE=dev` | production auth mode comes from process | `supabase` |
| same | files on disk | files must not be modified | both still present |
| `TestLoadProductionFailsClosedWithoutProcessConfig` | production; Supabase values only in `.env` | validation rejects | `Validate()` returned an error |
| `TestDotEnvCannotDeclareProduction` | `.env` contains `APP_ENV=production`; process does not | boundary is process-only, dotenv still loads | `from-legacy-env` loaded |

The last case is the load-bearing one: if a dotenv file could declare
production, it would suppress its own loading and produce a confusing
half-configured state on a developer machine.

## Consumer 3 — admin Vite build

| Mode | State | Expectation | Observation |
|---|---|---|---|
| development | `.env.development.local` supplies Supabase values; `ADMIN_API_BASE` unset everywhere | local profile resolves; API base falls back | `DEVLOCAL-SENTINEL.supabase.co` and `DEVLOCAL_SENTINEL_PUBKEY` present in `dist/assets/index-_KGD-reu.js`; `ADMIN_API_BASE:""` emitted, which `resolveApiBase` maps to `/api` |
| production | conflicting dotenv sentinels present; approved values in process env | process only | approved values present, all sentinels absent (see `security-review.md`) |

Commands: `npx vite build --mode development` (run from `admin/`, with the
public names unset in the process environment) and
`npm --prefix admin run build:only`.

Supporting checks: `npm --prefix admin run typecheck` clean;
`npm --prefix admin test -- --run` → 16 files, 194 tests passed;
`npm --prefix admin run check:resource-contracts` → PASSED.

## Consumer 4 — public minimal-cart Vite build

| Mode | State | Expectation | Observation |
|---|---|---|---|
| production | conflicting dotenv sentinels present; approved values in process env | process only, and no admin API base | approved public values present; all sentinels absent; no `ADMIN_API_BASE` in the theme allowlist |

Command: `npm --prefix site/themes/minimal-cart run build:check` → passed,
including `check-dialog-a11y` and `check-category-hydration`.

## Consumer 5 — Git and template inventory

`git check-ignore` results:

| Path | Result | Expected |
|---|---|---|
| `.env` | ignored | ignored |
| `.env.local` | ignored | ignored |
| `.env.development` | ignored | ignored |
| `.env.development.local` | ignored | ignored |
| `.env.production` | ignored | ignored |
| `.env.production.local` | ignored | ignored |
| `.env.example` | trackable | trackable |
| `.env.development.example` | trackable | trackable |
| `.env.production.example` | trackable | trackable |

A pattern scan of all three example files for JWT-shaped values, `sk_live`
keys, Resend `re_` keys, and PostgreSQL URLs containing credentials returned no
matches.

## AC-008 — existing local file

No migration of the developer's `.env` was performed. The precedence chain
makes migration optional: `.env` remains a supported source, so an existing
setup keeps working with no action.

Two temporary files were created during verification
(`.env.production.local`, `.env.development.local`). Both were confirmed
absent beforehand, and both were deleted afterwards. Neither the contents nor
any value of the existing `.env` was read, printed, or written at any point.
Its size and mtime were identical before and after the whole sequence
(1058 bytes, 2026-08-13 20:00:57).

## Contract preservation

No API route, authentication semantic, database schema, rendering behavior, or
deployment contract changed. `go test ./...`, `go vet ./...`, and
`gofmt -l server` are clean.
