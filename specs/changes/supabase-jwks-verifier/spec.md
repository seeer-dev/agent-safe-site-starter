# Supabase Local JWT Verification Specification

Change ID: supabase-jwks-verifier
Revision: 3
Status: Applying
Decision authority: Repository owner/user
Approval basis: Repository owner authorized implementation by Agy on 2026-08-16. Revision 3 rebases the controlled baseline onto the committed scoped-worktree proposal, preserves fail-closed verifier, claim, cache, and rollback defaults, and leaves live Supabase compatibility evidence environment-blocked.
Repository baseline: 8e8849fb4ebffb6617635dcc96b77eb2cceb853d
Supersedes: none

## Outcome

Allow a production Supabase project that actually issues asymmetric access tokens to select local JWT verification against a bounded JWKS cache, removing the per-protected-request `/auth/v1/user` round trip. Preserve the existing remote verifier as the default and as an explicit operator rollback mode. Never choose or fall back between verifiers from an untrusted request or token.

## Scope

An eventual approved apply may add one JWKS verifier inside `server/internal/auth`, add explicit server configuration and bootstrap wiring, pin one audited JWT/JWK dependency, update server-owned environment documentation, and record focused security and deployment evidence. It may not change handlers, capabilities, staff linking, browser authentication, database state, routes, or deployment topology.

Out of scope:

- Caching staff capabilities, staff rows, or complete `auth.Principal` values.
- Reading capabilities, staff identity, or application roles from JWT claims.
- Local verification of legacy/shared-secret (`HS256`) tokens.
- Request-time fallback from JWKS verification to `/auth/v1/user`, or automatic mode detection from token headers.
- Immediate revocation checks for server-side sign-out, session deletion, user deletion, or user bans while JWKS mode is selected.
- Database, migration, CORS, API route, browser bundle, OAuth-provider, or Supabase dashboard changes.

## Decisions and invariants

- `SUPABASE_VERIFIER_MODE=remote|jwks` is a server-only selector. Empty defaults to `remote`, preserving the current behavior. Any other value fails configuration validation.
- `remote` continues to use the existing `SupabaseVerifier` and `/auth/v1/user`. `jwks` uses only the derived JWKS endpoint. Neither mode falls back to the other during a request.
- JWKS mode accepts only asymmetric `ES256` with P-256 keys and `RS256` with RSA keys of at least 2048 bits. It rejects `HS256`, `none`, unsupported algorithms, missing `kid`, and mismatched `kid`/`alg`/key type before trusting claims.
- For JWKS mode, configuration validates `SUPABASE_URL` as an absolute URL with a host and no userinfo, query, or fragment. Production requires HTTPS; development additionally permits HTTP loopback for local Supabase and tests. The normalized URL has no trailing slash or extra path.
- The issuer is derived only as `<normalized SUPABASE_URL>/auth/v1`; token-supplied `iss`, `jku`, or `x5u` never controls a network destination. The only JWKS destination is `<issuer>/.well-known/jwks.json`, and redirects are rejected.
- Every locally verified token must have the exact derived issuer, audience containing `authenticated`, a present unexpired `exp`, an `nbf` that is not in the future when present, and a non-empty string `sub`. A fixed 60-second clock-skew allowance applies to time validation.
- Only verified `sub` and a string `email` claim are copied into the initial principal. The initial role is always `user`, and staff ID/capabilities are empty. `StaffCapabilityResolver` remains the sole staff capability authority and still links only by verified `sub` through `supabase_user_id`.
- A fresh cache entry is no older than 10 minutes. Fetches use a 5-second HTTP timeout, a 1 MiB response limit, a 32-key maximum, and strict JWK parsing. Stale keys are not used after refresh failure.
- A cold or expired cache performs one fetch. An unknown `kid` in a fresh cache performs at most one generation-coordinated refresh for that verification attempt; concurrent attempts observing the same generation share that refresh.
- Malformed JWTs, missing or invalid claims, disallowed algorithms, known-key signature failures, and an unknown `kid` after a successful refresh return `auth.ErrUnauthorized` and do not call the staff resolver. If no fresh matching key exists because JWKS retrieval or parsing failed, the result wraps `auth.ErrUnavailable`, maps to HTTP 503 through `auth.WriteError`, and logs no token or claim data.
- Local JWT verification validates token cryptography and claims, not current Supabase session/user state. A server-side sign-out or session termination can remain usable until the access token expires. Selecting JWKS mode therefore requires a recorded deployment input confirming the live signing algorithm/issuer/audience and accepting the configured access-token lifetime; operators can restore immediate remote checks by selecting `remote` and redeploying.

## Requirements

### REQ-001: Explicit Verifier Selection

The application MUST select one verifier at startup from a validated server-only mode, MUST default to the existing remote verifier, and MUST NOT perform request-time fallback or token-directed verifier selection.

#### AC-001: Remote Compatibility and Explicit JWKS Selection

- GIVEN `AUTH_MODE=supabase`
- WHEN `SUPABASE_VERIFIER_MODE` is empty or `remote`
- THEN bootstrap MUST use the existing `/auth/v1/user` verifier unchanged.
- AND WHEN the mode is `jwks`
- THEN bootstrap MUST use only the local JWKS verifier.
- AND any other value MUST fail configuration validation before serving requests.

#### AC-002: Fixed Trust Inputs and Claims

- GIVEN JWKS mode and a bearer token
- WHEN authentication validates it
- THEN the verifier MUST accept only `ES256`/P-256 or `RS256`/RSA-2048-or-stronger keys from the single URL derived from validated `SUPABASE_URL`.
- AND it MUST verify exact issuer, audience containing `authenticated`, required `exp`, optional `nbf`, and non-empty string `sub` with the fixed skew.
- AND it MUST copy only verified `sub` and string `email` into a capability-empty user principal.

### REQ-002: Local Verification Without Capability Drift

JWKS mode MUST verify supported access tokens locally after key acquisition, MUST never call `/auth/v1/user`, and MUST continue to derive every staff role and capability from the existing server-side resolver.

#### AC-003: Warm-Cache Verification Has No Auth-User Round Trip

- GIVEN a valid asymmetric Supabase-shaped token and a fresh cached matching key
- WHEN any protected handler calls `Authenticator.Principal`
- THEN verification MUST complete without a JWKS fetch or `/auth/v1/user` request.
- AND `StaffCapabilityResolver` MUST receive the verified `sub`; JWT role, metadata, or capability-shaped claims MUST not grant access.

#### AC-004: Invalid Tokens Fail Closed as Unauthorized

- GIVEN a malformed token, missing `kid`, invalid signature, disallowed algorithm, mismatched key type, wrong issuer/audience, missing or empty `sub`, missing or expired `exp`, or future `nbf`
- WHEN JWKS verification runs
- THEN it MUST return `ErrUnauthorized`, produce HTTP 401 through the existing shared mapping, perform no remote-verifier fallback, and MUST NOT invoke staff capability lookup.

### REQ-003: Bounded Key Refresh and Failure Separation

JWKS retrieval and cache refresh MUST be bounded, coordinated, limited to the configured discovery URL, and preserve the difference between an invalid token and unavailable verification infrastructure.

#### AC-005: Rotation, Cache, and Availability Semantics

- GIVEN an empty/expired cache or an unknown `kid`
- WHEN verification needs keys
- THEN it MUST perform at most one coordinated fetch for the observed cache generation, reject redirects, enforce the timeout/body/key limits, and atomically publish only a fully validated asymmetric key set.
- AND a successful refresh without the requested approved key MUST return `ErrUnauthorized`.
- AND a fetch, HTTP, size, parse, or no-approved-key failure that leaves no fresh matching key MUST wrap `ErrUnavailable`, map to HTTP 503, retain no partial cache, and expose no token, email, provider body, or URL detail publicly.

### REQ-004: Deployment Compatibility and Reversible Rollout

JWKS mode MUST remain opt-in until the actual Supabase project proves compatible asymmetric signing and the operator accepts the local-verification session semantics. Rollback MUST be an explicit configuration change and redeploy, never a per-request fallback.

#### AC-006: Live Project Input and Security Acceptance

- GIVEN the target Supabase project and an approved staging or production-safe validation window
- WHEN JWKS mode is enabled
- THEN a non-secret receipt MUST record the observed active asymmetric algorithm, issuer, audience, discovery-key match, configured access-token lifetime, and operator acceptance that server-side sign-out is not observed until token expiry.
- AND a protected endpoint MUST succeed with a valid project access token, reject a project-mismatched token, and return 503 without protected data when discovery is unavailable and no fresh key exists.
- AND the receipt MUST demonstrate that selecting `remote` restores the audited `/auth/v1/user` path after redeploy without changing application code.
- AND no bearer token, JWT payload, email, private key, or provider secret may be written to the receipt or logs.

## Official provider basis

- [Supabase JWT verification](https://supabase.com/docs/guides/auth/jwts) defines `/.well-known/jwks.json`, asymmetric-only discovery, and the 10-minute edge cache/20-minute rotation guidance.
- [Supabase JWT signing keys](https://supabase.com/docs/guides/auth/signing-keys) documents `ES256`, `RS256`, legacy `HS256`, rotation states, and cache/revocation behavior.
- [Supabase JWT claims](https://supabase.com/docs/guides/auth/jwt-fields) defines issuer, audience, time, subject, and email claim shapes.
- [Supabase sessions](https://supabase.com/docs/guides/auth/sessions) and [server-side auth guidance](https://supabase.com/docs/guides/auth/server-side/advanced-guide) state that local claim verification does not immediately observe server-side session termination.

## Amendments

- Revision 2 replaces the revision 1 decision-only draft with repository-derived fail-closed defaults and an implementation blueprint. Live project compatibility remains environment-blocked rather than an unresolved product decision.
