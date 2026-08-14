# Local and Production Environment Separation Delivery Plan

Change ID: separate-local-production-environments
Revision: 1
Status: Draft

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Area | Observed state at proposal time | Delivery implication |
|---|---|---|
| Repository baseline | Accepted admin API-base change is committed at `3c3a475` and the worktree is clean | The new proposal has one clean comparison baseline and no overlapping active owner |
| Go configuration | `server/internal/config/config.go` always calls `loadDotEnv(".env")` before reading `APP_ENV` | Production can currently receive unset values from a developer dotenv file; source selection must precede dotenv loading |
| Go tests | `server/internal/config/config_test.go` covers validation but not dotenv source selection or precedence | Add isolated tests that never read or print the user's actual dotenv files |
| Admin Vite build | `admin/vite.config.ts` calls `loadEnv(mode, repoRoot, '')` for every mode | Production admin builds can read root dotenv profiles and require a production-only branch |
| Public-theme Vite build | `site/themes/minimal-cart/vite.config.ts` has the same root `loadEnv` behavior | The public browser bundle must receive the same production isolation and allowlist controls |
| Browser injection | Both Vite configs explicitly define public auth variables; admin additionally defines `ADMIN_API_BASE` | Preserve the explicit allowlist and prevent automatic or accidental server-secret injection |
| Examples and ignore rules | `.env.example` mixes local defaults and production/provider inventory; `.gitignore` ignores only `.env` | Split safe examples, ignore real profiles, and document provider ownership |
| Deployment | Railway runs the Go API; Cloudflare Pages runs the Go renderer/build and serves static output | Online configuration remains provider-managed process/build environment; no production dotenv file is deployed |
| Optional impact tool | CodeGraph is unavailable in the current environment | Producer/consumer impact was traced manually through both Vite configs, Go config consumers, README, Dockerfile, and Railway config |

## Producer-to-consumer trace

```text
local process / .env.development.local / legacy .env
  -> Go config source selection
  -> API, migration, render, publish, and dev tools

Railway or Cloudflare process environment
  -> APP_ENV=production boundary
  -> Go config without dotenv
  -> API runtime or static renderer

local development profile or provider build environment
  -> admin/public Vite mode selection
  -> explicit browser-safe define allowlist
  -> admin auth/API client or public auth islands
  -> emitted browser assets
```

## Scope lock

The controlled change applies strictly to:

- `.env.example`
- `.env.development.example`
- `.env.production.example`
- `.gitignore`
- `README.md`
- `docs/environment-configuration.md`
- `server/internal/config/config.go`
- `server/internal/config/config_test.go`
- `admin/vite.config.ts`
- `site/themes/minimal-cart/vite.config.ts`
- `specs/changes/separate-local-production-environments/**`

If implementation proves a dedicated test/helper path is required, update both this scope and `.ai/scope.json` with the repository evidence before editing it. No package dependency may be added without separate user approval.

## Dependency-ordered slices

### Slice 1: Go configuration source boundary

Outcome:

- Select `APP_ENV` from the process boundary before dotenv loading.
- In development, load `.env.development.local` before a legacy `.env` fallback without overriding process values.
- In production, skip all repository dotenv files.
- Add isolated temporary-directory tests for precedence, missing files, production isolation, and validation behavior.

Hard dependencies: none.

Allowed paths:

- `server/internal/config/config.go`
- `server/internal/config/config_test.go`

Acceptance evidence:

- `go test ./server/internal/config`
- Tests use only generated temporary fixtures and assert values, never the user's dotenv contents.
- Covers REQ-001, AC-001, AC-002.

Rollback or safe failure:

- Revert the loader and tests; existing `.env` remains untouched and ignored.

### Slice 2: Admin and public Vite production isolation

Outcome:

- Keep local development profile behavior for both Vite consumers.
- Disable Vite dotenv file loading for production builds.
- Read production values from the process environment and inject only the approved browser-safe allowlist.
- Preserve admin `/api` fallback and the accepted `ADMIN_API_BASE` grammar.

Hard dependencies:

- Slice 1 defines the shared production/local authority terminology used by documentation and verification.

Allowed paths:

- `admin/vite.config.ts`
- `site/themes/minimal-cart/vite.config.ts`

Acceptance evidence:

- `npm --prefix admin test -- --run`
- `npm --prefix admin run typecheck`
- `npm --prefix admin run build`
- `npm --prefix site/themes/minimal-cart run build:check`
- Fresh production builds with conflicting dotenv/process sentinels prove process-only public values and zero server-secret sentinel matches in `admin/dist/` and `site/themes/minimal-cart/dist/`.
- Covers REQ-002, AC-003, AC-004, AC-006.

Rollback or safe failure:

- Revert both Vite configs together; do not accept a one-consumer-only boundary.

### Slice 3: Examples, operator documentation, migration, and integration proof

Outcome:

- Make `.env.development.local` the documented primary local file.
- Keep safe compatibility guidance for existing `.env` users.
- Add a placeholder-only production inventory while stating that providers, not a deployed file, own online values.
- Ignore all real dotenv profiles while keeping tracked examples visible.
- Document which values belong to Railway, Cloudflare build-only Go rendering, and browser-safe Vite builds.
- Migrate an existing local `.env` only through a content-blind, no-overwrite, recoverable operation; otherwise retain the documented fallback.

Hard dependencies:

- Slices 1 and 2 establish the behavior described by the examples and documentation.

Allowed paths:

- `.env.example`
- `.env.development.example`
- `.env.production.example`
- `.gitignore`
- `README.md`
- `docs/environment-configuration.md`

Acceptance evidence:

- `git check-ignore` proves real profile names are ignored and example files remain trackable.
- A source review confirms examples contain placeholders only and no actual credential values.
- Isolated local and production-mode runs trace configuration through Go API/render and both Vite consumers.
- `go run ./server/tools/speccheck`
- `go run ./server/tools/scopecheck`
- `go run ./server/tools/verify`
- Security-review and consumer-reachability receipts record the observed boundary and outputs without secret values.
- Covers REQ-003, REQ-004, AC-005, AC-006, AC-007, AC-008.

Rollback or safe failure:

- Restore documentation/templates independently. Any local-file rename must be reversible and must leave the original in place when the target already exists.

## Traceability matrix

| REQ / AC | Slice | Surface or consumer | Verification | Expected evidence |
|---|---|---|---|---|
| REQ-001 | Slice 1 | Go API/tools/render config | Targeted Go tests | Development precedence and production isolation pass |
| AC-001 | Slice 1 | Development Go config | Temporary dotenv fixtures | Process > primary local > legacy fallback |
| AC-002 | Slice 1 | Production Go config | Conflicting sentinel fixture | Repository dotenv values are absent; validation fails closed when required process config is missing |
| REQ-002 | Slice 2 | Admin and public-theme Vite configs | Dual development/production checks | Both consumers select the correct source |
| AC-003 | Slice 2 | Vite development servers | Config/build assertions | Public auth values resolve; admin retains `/api` fallback |
| AC-004 | Slice 2 | Production Vite builds | Conflicting public sentinels | Only process-supplied public values appear |
| REQ-003 | Slices 2, 3 | Git, examples, browser artifacts | Ignore checks, source audit, bundle scan | Real profiles ignored; examples safe; browser allowlist enforced |
| AC-005 | Slice 3 | Developer/operator docs | `git check-ignore` and documentation review | Local, Railway, and Cloudflare ownership is unambiguous |
| AC-006 | Slices 2, 3 | `admin/dist`, theme `dist` | Fresh build sentinel scan plus security review | No server-secret sentinel in either bundle |
| REQ-004 | Slice 3 | Existing runtime/build consumers | Isolated end-to-end configuration trace | No API, auth, render, or deployment contract changes |
| AC-007 | Slices 1-3 | Go API/render, admin, public theme | Consumer-reachability receipt | Every consumer receives only its selected profile |
| AC-008 | Slice 3 | Existing ignored `.env` | No-print/no-overwrite migration check | User configuration remains recoverable |

## Risks and controls

- Risk: fixing only admin leaves the public theme exposed to the same root dotenv source. Control: Slice 2 treats both Vite consumers as one acceptance boundary.
- Risk: provider build secrets are present while Vite runs. Control: production dotenv loading is disabled, automatic exposure is restricted, and only named public keys are explicitly defined.
- Risk: changing the local filename breaks the user's configured environment. Control: primary-plus-legacy precedence and a content-blind, no-overwrite migration path.
- Risk: example files are mistaken for deployable production config. Control: `.env.production.example` is an inventory only; documentation forbids creating or deploying a real `.env.production`.
- Risk: secret verification leaks values into logs or receipts. Control: use synthetic sentinels and record only presence/absence assertions.
