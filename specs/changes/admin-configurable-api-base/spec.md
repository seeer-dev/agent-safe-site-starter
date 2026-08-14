# Admin Configurable API Base Specification

Change ID: admin-configurable-api-base
Revision: 2
Status: Accepted
Decision authority: Repository owner/user
Approval basis: User approved revision 2 by plain apply instruction on 2026-08-14 to implement configurable admin API base with strict prefix grammar and loopback/HTTPS security controls.
Repository baseline: 5f5fcab3e3613d51d64a368340b36ae10e3d1168
Supersedes: none

## Outcome

Make the admin frontend API base URL build-time configurable via environment configuration (`ADMIN_API_BASE`), while preserving `'/api'` as the default fallback for local development and same-origin deployments. This removes the hardcoded frontend API path and prepares a selectable endpoint without breaking Vite's local development proxy (`/api` -> `http://localhost:8080`).

Note: This change addresses the frontend client-side configurability blocker. Cross-origin runtime deployment will separately require an exact backend CORS allowlist in `server/internal/bootstrap/app.go` once production domains are determined.

## Scope

In scope:
- Build-time environment variable injection in Vite (`admin/vite.config.ts`) and TypeScript typings (`admin/src/env.d.ts`).
- Centralized API base resolution helper (`admin/src/lib/api-config.ts`).
- Strict validation and URL normalization following the configuration grammar below.
- Safe silent fallback to `'/api'` when configuration is unset, empty, whitespace-only, or fails validation (with no token or URL logged).
- Unification of `admin/src/lib/api-client.ts` and `admin/src/lib/media-api.ts` so all admin API requests consume the shared base URL resolver.
- Documentation in `.env.example` explaining `ADMIN_API_BASE` usage and relationship with `PUBLIC_API_BASE`.
- Pure resolver unit tests (`admin/src/lib/api-config.test.ts`) and mocked-fetch integration tests (`admin/src/lib/api-client.test.ts`, `admin/src/lib/media-api.test.ts`).

Out of scope:
- Backend CORS configuration changes (`ALLOWED_ORIGINS` or `app.go` edits) - deferred until production hosting domains and topology are chosen.
- OpenAPI specification audits, swaggo, or TypeScript client/type generation.
- Admin resource list filters, pagination, or selection refactoring.
- Navigation/profile unification or component extractions.

## Decisions and Invariants

- Prefix vs Origin Semantics: `ADMIN_API_BASE` is the complete API prefix (e.g. `https://api.example.com/api` or `/api`), not a bare origin (`https://api.example.com`). Bare origins without a path ending in `/api` are rejected because Go routes in this starter are mounted under `/api`. Relative prefixes MUST strictly be `/api` (with optional trailing slashes); other root-relative paths like `/custom/api` are rejected.
- Separation from `PUBLIC_API_BASE`: `PUBLIC_API_BASE` is the public-site/backend origin (`http://localhost:8080`) where public code appends `/api` or other subpaths. `ADMIN_API_BASE` is an independent client prefix for admin requests. Leaving `ADMIN_API_BASE` unset preserves Vite's local `/api` proxy even when root `.env` defines `PUBLIC_API_BASE`.
- Security Transport Rule: Because admin requests carry `Authorization: Bearer <token>` credentials, remote absolute URLs MUST use `https://`. Unencrypted `http://` is strictly restricted to the exact canonical loopback development hosts (`localhost`, `127.0.0.1`, `[::1]`).
- Strict Validation Grammar: The resolver rejects protocol-relative URLs (`//...`), credentials/userinfo (`user:pass@host`), query strings (`?...`), fragments (`#...`), control characters, standalone literal or percent-encoded dot-segments (`/..`, `/.`, `%2e`, `%2E`), non-root-relative paths, and any relative path other than `/api` or absolute path not ending in `/api`.
- Silent Safe Failure: On any invalid or malformed value, the resolver MUST silently fall back to `/api` without logging sensitive tokens or URL parameters.
- Local-First Invariant: Running `npm run dev` in `admin/` without `ADMIN_API_BASE` MUST continue to proxy requests to `http://localhost:8080` via `/api`.

## Configuration Grammar

```text
ADMIN_API_BASE   := <Empty> | <RelativePrefix> | <AbsolutePrefix>
Empty            := "" | <WhitespaceOnly>
RelativePrefix   := "/api" ["/"]*
AbsolutePrefix   := <Scheme> "://" <Host> [<Port>] <Path>
Scheme           := "https" | ("http" ; ONLY IF Host is LoopbackHost)
LoopbackHost     := "localhost" | "127.0.0.1" | "[::1]"
Port             := ":" [0-9]+
Path             := "/api" ["/"]* | "/" [^/?#\s]+ "/api" ["/"]*

Forbidden:
  - Userinfo (e.g. "user:pass@host")
  - Query strings ("?")
  - Fragments ("#")
  - Protocol-relative URLs ("//...")
  - Standalone literal or percent-encoded dot-segments ("/../", "/./", "/%2e/", "/%2e%2e/")
  - Control characters or whitespace within the URL string
  - Non-root-relative paths (e.g. "api", "foo/api")
  - Root-relative paths other than "/api" (e.g. "/custom/api")
  - Any absolute path that does not end in "/api" (e.g. "https://api.example.com")

Normalization:
  1. Trim leading and trailing whitespace.
  2. If empty, return "/api".
  3. Validate against the grammar rules above.
  4. If validation fails, return "/api" silently.
  5. If validation passes, trim redundant trailing slashes ("/*" -> "") and return the sanitized prefix.
```

## Requirements

### REQ-001: Configurable Admin API Prefix Resolution & Default Fallback
The admin frontend MUST resolve its backend API prefix from build-time environment configuration, defaulting to `'/api'` when unconfigured, empty, or whitespace-only, preserving local development Vite proxy behavior without requiring a production domain.

#### AC-001: Default Local Fallback
- GIVEN no `ADMIN_API_BASE` environment variable (or an empty/whitespace string) is set at build time
- WHEN the admin client or media client requests an endpoint path (e.g. `'/admin/me'`, `'/media/presign'`)
- THEN the request target URL MUST use the prefix `'/api'` (e.g. `'/api/admin/me'`, `'/api/media/presign'`), routing through Vite's local development proxy to `http://localhost:8080`.

#### AC-002: Build-time Custom API Prefix Resolution
- GIVEN `ADMIN_API_BASE` is configured at build time to a valid remote HTTPS prefix or loopback HTTP prefix ending in `/api` (e.g. `'https://api.example.com/api'` or `'http://localhost:8080/api'`)
- WHEN the admin client or media client requests an endpoint path (e.g. `'/admin/me'`, `'/media/presign'`)
- THEN the request target URL MUST prepend the configured base URL without losing endpoint paths (e.g. `'https://api.example.com/api/admin/me'`, `'https://api.example.com/api/media/presign'`).

### REQ-002: Security Validation, Grammar Enforcement, and Safe Fallback
The API base resolver MUST enforce transport security and grammar validation, rejecting bare origins, non-loopback HTTP schemes, userinfo, query strings, fragments, standalone dot-segments, protocol-relative paths, and non-`/api` paths, normalizing trailing slashes only after validation and safely falling back to `'/api'` without leaking sensitive data.

#### AC-003: Trailing Slash Normalization
- GIVEN `ADMIN_API_BASE` is configured with trailing slashes on a valid prefix (e.g. `'https://api.example.com/api/'` or `'/api///'`)
- WHEN the API base URL is resolved and combined with a resource path (e.g. `'/admin/products'`)
- THEN trailing slashes MUST be trimmed so the resulting request URL is cleanly formatted without duplicate slashes (e.g. `'https://api.example.com/api/admin/products'`, `'/api/admin/products'`).

#### AC-004: Strict Security Rejection and Silent Safe Fallback
- GIVEN `ADMIN_API_BASE` is set to an invalid value (e.g. bare origin `'https://api.example.com'`, non-loopback HTTP `'http://api.example.com/api'`, protocol-relative `'//api.example.com/api'`, userinfo `'https://user:pass@api.example.com/api'`, query `'https://api.example.com/api?v=1'`, fragment `'https://api.example.com/api#top'`, literal dot-segments `'https://api.example.com/v1/../api'`, standalone percent-encoded dot-segments `'https://api.example.com/v1/%2e%2e/api'`, `'/%2e/api'`, non-`/api` relative path `'/custom/api'`, or non-root path `'api'`)
- WHEN the API base URL is resolved
- THEN the resolver MUST reject the unsafe value and silently fall back to `'/api'` without logging tokens or raw URL strings.

### REQ-003: Unified Client Consumption & Dual-Mode Type Safety
All admin API request modules (`api-client.ts` and `media-api.ts`) MUST consume the single shared configuration resolver, and TypeScript environment definitions (`env.d.ts`) MUST declare the configuration property, validating cleanly in both unset and custom build modes.

#### AC-005: Shared Client Integration Verification
- GIVEN API client calls via `api.get/post/put/patch/del` and media client calls via `presignUpload`/`verifyUpload`
- WHEN HTTP requests are dispatched in unit and integration tests
- THEN mocked fetch assertions MUST prove that both clients derive target URLs from the shared `getApiBase()` helper and preserve complete endpoint paths.

#### AC-006: Dual-Mode Build and Type Verification
- GIVEN `admin/src/env.d.ts` and `admin/vite.config.ts`
- WHEN `npm run typecheck` and `npm run build` are executed both with `ADMIN_API_BASE` unset and with `ADMIN_API_BASE=https://api.example.com/api`
- THEN TypeScript MUST compile with zero type errors and Vite MUST successfully emit the production build in both configurations.

## Amendments

### Revision 2 (2026-08-14)
- Clarified `ADMIN_API_BASE` semantics as a complete API prefix ending in `/api`, not a bare origin.
- Enforced strict relative prefix rule: only `/api` (with optional trailing slashes) is valid; other root-relative paths like `/custom/api` are rejected.
- Documented separation from `PUBLIC_API_BASE`.
- Added transport security rule requiring HTTPS for remote URLs and restricting HTTP to canonical loopback hosts (`localhost`, `127.0.0.1`, `[::1]`).
- Added rejection of standalone percent-encoded dot-segments (`%2e`, `%2E`), literal dot-segments, bare origins, userinfo, query, fragment, protocol-relative, and non-root paths before URL normalization, while permitting valid in-segment dots and in-segment `@` path segments (e.g. `/v1/@scope/api`).
- Adjusted outcome description to clarify that runtime cross-origin access still requires backend CORS configuration once domains are known.
- Split testing into dedicated pure resolver unit tests (`api-config.test.ts`) and mocked-fetch client integration tests (`api-client.test.ts`, `media-api.test.ts`).
- Added dual-mode build verification under AC-006.
