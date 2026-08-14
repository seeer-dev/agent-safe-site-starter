# Admin Configurable API Base Delivery Plan

Change ID: admin-configurable-api-base
Revision: 2
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Area | Observed state at proposal time | Delivery implication |
|---|---|---|
| Admin API Client | `admin/src/lib/api-client.ts` hardcodes `const API_BASE = '/api'` | Refactor to consume centralized `getApiBase()` resolver |
| Admin Media Client | `admin/src/lib/media-api.ts` duplicates `const API_BASE = '/api'` | Refactor to consume centralized `getApiBase()` resolver |
| Vite Configuration | `admin/vite.config.ts` defines auth/supabase env flags but not API base | Add `ADMIN_API_BASE` to `define` in `vite.config.ts` |
| Environment Types | `admin/src/env.d.ts` lacks `ADMIN_API_BASE` | Declare `readonly ADMIN_API_BASE?: string` in `ImportMetaEnv` |
| Environment Documentation | `.env.example` mentions `PUBLIC_API_BASE` for backend/site but not `ADMIN_API_BASE` | Document `ADMIN_API_BASE` as optional admin prefix distinct from `PUBLIC_API_BASE` |
| Test Coverage | No unit tests for API base configuration or client request URL assembly | Add dedicated resolver unit tests and mocked-fetch integration tests |

## Scope lock

The controlled change applies strictly to the following paths:

- `.env.example`
- `admin/src/env.d.ts`
- `admin/src/lib/api-client.test.ts`
- `admin/src/lib/api-client.ts`
- `admin/src/lib/api-config.test.ts`
- `admin/src/lib/api-config.ts`
- `admin/src/lib/media-api.test.ts`
- `admin/src/lib/media-api.ts`
- `admin/vite.config.ts`
- `specs/changes/admin-configurable-api-base/**`

## Dependency-ordered slices

### Slice 1: Pure API Base Resolver and Security Grammar Tests
- Create `admin/src/lib/api-config.ts` implementing `getApiBase()` according to the normative grammar:
  - Default fallback to `'/api'` when unset/empty/whitespace.
  - Relative prefix MUST strictly be `'/api'` (with optional trailing slashes); reject other root-relative paths like `'/custom/api'`.
  - Absolute prefix path must end in `'/api'`.
  - Enforce HTTPS for remote URLs; allow HTTP only for exact canonical loopback hosts (`localhost`, `127.0.0.1`, `[::1]`).
  - Reject bare origins, userinfo credentials (`user:pass@host`), query, fragment, control characters, protocol-relative URLs, standalone literal or percent-encoded dot-segments (`%2e`, `%2e%2e`), and non-root paths.
  - Permit valid in-segment dots (`v1..beta`, `v1%2ebeta`, `v1.0.0`) and in-segment `@` path segments (`v1/@scope/api`).
  - Normalize trailing slashes after successful validation.
  - Silent safe fallback to `'/api'` on any validation error.
- Create `admin/src/lib/api-config.test.ts` with comprehensive unit tests for all valid and invalid cases.
- Covers: REQ-001, REQ-002, AC-001, AC-002, AC-003, AC-004.

### Slice 2: Vite Injection, Type Declarations, and Documentation
- Update `admin/vite.config.ts` to inject `import.meta.env.ADMIN_API_BASE`.
- Update `admin/src/env.d.ts` to declare `readonly ADMIN_API_BASE?: string`.
- Update `.env.example` with documentation explaining `ADMIN_API_BASE` usage and its separation from `PUBLIC_API_BASE`.
- Verify dual-mode TypeScript compilation (`npm run typecheck`) and Vite build (`npm run build`) with unset config and with custom `ADMIN_API_BASE=https://api.example.com/api`.
- Covers: REQ-003, AC-006.

### Slice 3: Client Unification and Integration Tests
- Update `admin/src/lib/api-client.ts` to import and use `getApiBase()`.
- Update `admin/src/lib/media-api.ts` to import and use `getApiBase()`.
- Create `admin/src/lib/api-client.test.ts` and `admin/src/lib/media-api.test.ts` with mocked-fetch tests verifying exact request URL construction.
- Run complete test suite (`npm run test`), typecheck (`npm run typecheck`), build (`npm run build`), and verify tools (`go run ./server/tools/speccheck`, `go run ./server/tools/verify`).
- Covers: REQ-003, AC-005.

## Traceability matrix

| REQ / AC | Slice | Target / Consumer | Verification Method | Expected Evidence |
|---|---|---|---|---|
| REQ-001 | Slice 1 | `admin/src/lib/api-config.ts` | Vitest unit tests | Automated tests pass for default and custom API prefixes |
| REQ-002 | Slice 1 | `admin/src/lib/api-config.ts` | Vitest security/grammar tests | Automated tests pass for security rejection, trailing slashes, and safe fallback |
| REQ-003 | Slices 2, 3 | `admin/src/lib/api-client.ts`, `admin/src/lib/media-api.ts`, `admin/src/env.d.ts`, `admin/vite.config.ts` | Mocked fetch tests & dual-mode build | Integration tests prove URL dispatch; dual-mode typecheck/build passes |
| AC-001 | Slice 1 | `getApiBase()` default resolution | Vitest test case | Returns `'/api'` when env is unset, empty, or whitespace |
| AC-002 | Slice 1 | `getApiBase()` custom URL resolution | Vitest test cases | Returns configured remote HTTPS prefix or loopback HTTP prefix |
| AC-003 | Slice 1 | `getApiBase()` slash normalization | Vitest test cases | Trailing slashes stripped cleanly from valid prefixes ending in `/api` |
| AC-004 | Slice 1 | `getApiBase()` security rejection | Vitest test cases | Bare origins, non-/api relative paths, non-loopback HTTP, userinfo, query, fragment, and standalone dot-segments fall back to `'/api'` |
| AC-005 | Slice 3 | `api-client.ts`, `media-api.ts` | Mocked-fetch integration tests in `api-client.test.ts` and `media-api.test.ts` | Both clients assemble correct target URLs via `getApiBase()` |
| AC-006 | Slice 2 | `env.d.ts`, `vite.config.ts` | `vue-tsc --noEmit` & `vite build` | Typecheck and build succeed with unset config and custom `ADMIN_API_BASE` |
