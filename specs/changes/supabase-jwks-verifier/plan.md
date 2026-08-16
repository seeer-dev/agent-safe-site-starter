# Supabase Local JWT Verification Implementation Blueprint

Change ID: supabase-jwks-verifier
Revision: 3
Status: Applying

Normative specification: [`spec.md`](spec.md)

## Implementation expansion context

Proposal revision: 2
Proposal status: Applying
Repository baseline: `8e8849fb4ebffb6617635dcc96b77eb2cceb853d`
Observed HEAD: `8e8849fb4ebffb6617635dcc96b77eb2cceb853d`
Pre-existing dirty paths:
- `Makefile` - preserve; unrelated tracked modification.
- `specs/changes/commerce-module-file-split/` - preserve; unrelated untracked proposal.
- `specs/changes/public-endpoint-rate-limit/` - preserve; unrelated untracked proposal.
- `specs/changes/verify-contract-checks/` - preserve; unrelated untracked proposal.
- `specs/changes/supabase-jwks-verifier/` - active untracked revision 1 proposal expanded here.
Packet ID mapping:
- Existing Packet S01 -> Packet S01.
- Packets S02-S05 are new in revision 2.

CodeGraph: not installed; exact `rg` source/caller inspection was used.

## Scope lock

Future implementation may modify only the exact paths in `control.json`. Product code, dependencies, and operator documents remain read-only while this Draft is reviewed. No database, module handler, browser, CI, route, capability, OAuth-provider, or deployment-topology edit is authorized.

Controlled lifecycle/evidence scope:
- `specs/changes/supabase-jwks-verifier/**`

### Packet S01: Confirm live Supabase compatibility and session posture

Status: ENVIRONMENT_BLOCKED
Covers: REQ-004, AC-006
Hard dependencies: none; this records live deployment facts and does not block repository-only Packets S02-S05.
Outcome: The target deployment proves asymmetric signing compatibility and explicitly accepts the bounded revocation behavior before JWKS mode becomes authoritative.
Safe failure / rollback: Keep `SUPABASE_VERIFIER_MODE=remote`; no production request uses local verification until the receipt is complete.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Current verifier | `server/internal/auth/supabase.go` | `SupabaseVerifier.Verify` | Calls `/auth/v1/user`; it does not expose the production signing algorithm or session policy. |
| Deployment config | `server/internal/config/config.go` | `Config.AuthMode`, `Config.SupabaseURL` | Repository configuration identifies the provider URL but contains no live signing-key state. |
| Production inventory | `.env.production.example` | Authentication section | Values are placeholders; production values live in Railway/Supabase. |
| Session consumer | `server/internal/bootstrap/app.go` | `GET /api/admin/me` | Real protected endpoint returns the verified principal and server-derived capabilities. |

Read set:
- `server/internal/auth/supabase.go` - remote rollback behavior.
- `server/internal/bootstrap/app.go` - real protected endpoint.
- `.env.production.example` and `docs/environment-configuration.md` - deployment ownership boundary.

Modify set after an authorized live walkthrough:
- `specs/changes/supabase-jwks-verifier/receipts/environment-compatibility-rev2.md` - non-secret observations only.
- `specs/changes/supabase-jwks-verifier/control.json` and `evidence.md` - attributable receipt/status references only after independent review.

Must not modify:
- Supabase dashboard signing/session settings - this packet observes them; it does not authorize rotation or expiry changes.
- Railway environment outside an approved validation window - mode changes require operator authority.
- Any token, private key, secret, or user record - never persist live credentials in repository evidence.

#### Behavior contract

Before:
- The repository cannot prove whether the target project exposes an asymmetric active key, what its access-token lifetime is, or whether delayed sign-out recognition is acceptable.

After:
- A receipt records only non-secret signing metadata and observable HTTP results, plus operator acceptance of the local-verification session model and explicit remote rollback.

Preserved invariants:
- No key rotation, session-policy change, production rollout, or credential disclosure occurs merely because local packets are implementable.

Inputs and authority:
- Supabase dashboard and discovery endpoint provide active algorithm/`kid`, issuer/audience, and expiry settings.
- Repository owner/operator authorizes the validation environment and accepts the access-token lifetime as the maximum local revocation delay.

Errors and edge states:
- Legacy/shared-secret signing, empty asymmetric JWKS, mismatched issuer/audience, or an unacceptable revocation window keeps production in `remote` mode.

#### Integration trace

Supabase project signing/session settings (external authority)
  -> project access token supplied ephemerally, never recorded
  -> Railway `SUPABASE_URL` + `SUPABASE_VERIFIER_MODE`
  -> `bootstrap.NewWithDB` verifier selection
  -> `auth.Authenticator.Principal`
  -> staff resolver by verified `sub`
  -> `GET /api/admin/me`
  -> status/body-only walkthrough receipt

#### Environment verification walkthrough

1. In an approved non-production or production-safe window, record the dashboard's active signing algorithm, public `kid`, access-token lifetime, and project URL without copying any key material or token.
2. Confirm `<SUPABASE_URL>/auth/v1/.well-known/jwks.json` contains the active public `kid` and that a freshly issued access token has the expected issuer and `authenticated` audience; inspect claims only in memory.
3. Select `jwks`, deploy, and call `GET /api/admin/me` with the ephemeral token. Record only deployment identifier, UTC time, verifier mode, HTTP status, and whether expected non-secret response fields were present.
4. Present a token from another project and record HTTP 401 with no protected data.
5. In a controlled staging failure injection with no fresh cache, make discovery unavailable and record HTTP 503 with no protected data or sensitive diagnostic content; restore discovery immediately.
6. Record acknowledgement that server-side sign-out/session deletion is not checked until `exp`, and that the configured token lifetime is acceptable for this application.
7. Select `remote`, redeploy, repeat the valid protected call, then restore the approved target mode. Record only status and deployment identifiers.
8. Search the receipt and captured logs for JWT segments, bearer headers, emails, private-key fields, and provider secrets; record zero findings and destroy ephemeral token material.

#### Expansion stop

Marker: ENVIRONMENT_BLOCKED
Claim that cannot be executed: The actual deployment uses compatible asymmetric signing and accepts the local-verification revocation window and explicit rollback procedure.
Repository evidence: `config.Config` contains only placeholders and runtime environment names; no repository artifact can reveal the live Supabase signing key state, access-token lifetime, or operator risk acceptance.
Impact: REQ-004 and AC-006 remain blocked; Packets S02-S05 can still be implemented and locally verified because their contracts do not depend on a particular project value.
Smallest unblock action: Provide an approved Supabase/Railway validation environment and operator, then execute the non-secret walkthrough above without changing signing/session settings.

### Packet S02: Add explicit, validated verifier configuration

Status: executable
Covers: REQ-001, AC-001, AC-002
Hard dependencies: none; it establishes the startup contract consumed by S03 and S04.
Outcome: Configuration yields exactly `remote` or `jwks`, defaults to `remote`, and validates the URL from which issuer/JWKS endpoints are derived.
Safe failure / rollback: Empty mode remains `remote`; invalid mode or unsafe JWKS URL prevents startup validation rather than selecting another verifier.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Config model | `server/internal/config/config.go` | `type Config` | Holds auth mode, dev token, Supabase URL, and publishable key; no verifier mode exists. |
| Config producer | `server/internal/config/config.go` | `Load` | Reads process env first in production and trims the Supabase trailing slash. |
| Validation | `server/internal/config/config.go` | `Config.Validate` | Requires URL/key for `AUTH_MODE=supabase`; does not parse the URL. |
| Existing proof | `server/internal/config/config_test.go` | `TestLoadProductionFailsClosedWithoutProcessConfig` | Proves production dotenv isolation, not verifier selection or URL safety. |
| Operator inventory | `.env.development.example`, `.env.production.example`, `docs/environment-configuration.md` | Authentication variables | Lists current auth variables and browser boundary. |

Read set:
- `server/internal/config/config.go` and `config_test.go` - existing loading/validation conventions.
- `.env.development.example`, `.env.production.example`, `docs/environment-configuration.md` - server/browser ownership.

Modify set:
- `server/internal/config/config.go` - add `SupabaseVerifierMode`, load default `remote`, and validate mode plus normalized Supabase base URL for JWKS mode.
- `server/internal/config/config_test.go` - add focused load, validation, production-HTTPS, loopback-development, and unsafe-shape tests.

Must not modify:
- Browser Vite allowlists - verifier mode is server-only.
- `AUTH_MODE` values or dev verifier behavior - existing `dev|supabase` contract remains.
- Dotenv precedence or production isolation - unrelated trust boundary.

#### Behavior contract

Before:
- `AUTH_MODE=supabase` always selects the remote verifier; any non-empty Supabase URL passes validation.

After:
- `SUPABASE_VERIFIER_MODE` trims/lowercases to `remote|jwks`; empty becomes `remote`.
- JWKS mode requires an absolute host URL, no userinfo/query/fragment/extra path, HTTPS in production, and HTTPS or HTTP loopback in development.

Preserved invariants:
- `SUPABASE_PUBLISHABLE_KEY` remains required for Supabase application compatibility.
- Production reads process environment only; the server-only selector never enters browser allowlists.

Inputs and authority:
- Railway/process environment owns the mode and URL.
- `Config.Load` normalizes strings; `Config.Validate` is authoritative for accepted values.

Errors and edge states:
- Unknown mode, relative URL, credentials in URL, query/fragment, non-loopback development HTTP, extra path, or production HTTP returns a stable validation error without printing credentials.

#### Ordered edits

1. `server/internal/config/config.go` - `type Config` and `Load`
   - Change: add `SupabaseVerifierMode string` sourced from `SUPABASE_VERIFIER_MODE`, normalized with the existing lowercase/trim convention and defaulted to `remote`.
   - Constraint: do not add the value to any browser-facing configuration.
   - Result: every Supabase startup has an explicit in-memory verifier mode while existing deployments remain remote.
2. `server/internal/config/config.go` - `Config.Validate` and new unexported Supabase URL validator
   - Change: accept only `remote|jwks`; for JWKS validate the exact URL rules and derive no value from token input.
   - Constraint: keep existing `AUTH_MODE`, database, and production dotenv checks unchanged.
   - Result: unsafe or ambiguous JWKS configuration fails before bootstrap wiring.
3. `server/internal/config/config_test.go` - `TestLoadSupabaseVerifierMode` and `TestValidateSupabaseJWKSConfiguration`
   - Change: cover empty/default remote, explicit remote/JWKS, invalid mode, managed HTTPS, local loopback HTTP, production HTTP rejection, and URL userinfo/query/fragment/path rejection.
   - Constraint: use `t.Setenv`/temporary directories; never read a developer `.env`.
   - Result: tests fail if mode selection becomes implicit or URL validation loosens.

#### Integration trace

Railway/local process `SUPABASE_VERIFIER_MODE` + `SUPABASE_URL`
  -> `config.Load` normalization
  -> `Config.Validate` mode and URL trust checks
  -> `bootstrap.NewWithDB` (S04)
  -> one process-wide verifier

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Mode defaults safely and rejects unknown values | `.` | `go test ./server/internal/config -run 'TestLoadSupabaseVerifierMode|TestValidateSupabaseJWKSConfiguration' -count=1` | Named config tests | Empty selects remote; only remote/JWKS pass | `auto`, `fallback`, and arbitrary values fail | Temporarily change the valid JWKS fixture to `auto`; the focused test must fail for the expected mode assertion | Restore the one-line fixture and run `git diff --exit-code -- server/internal/config/config_test.go` against the pre-trigger patch |
| JWKS URL cannot redirect trust through configuration | `.` | same focused command | URL subtests | Exact managed HTTPS and local loopback cases pass | credentials, query, fragment, path, remote HTTP fail | Temporarily allow URL userinfo in the validator; the userinfo subtest must fail | Restore validator and confirm no diff beyond the authorized implementation |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] The bounded failure trigger fails for the claimed reason and is restored.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.

### Packet S03: Implement bounded asymmetric JWKS verification

Status: executable
Covers: REQ-002, REQ-003, AC-002, AC-003, AC-004, AC-005
Hard dependencies: S02, because the validated normalized base URL is the sole issuer and network-destination authority.
Outcome: A cached local verifier validates supported Supabase-shaped tokens, coordinates rotation refresh, returns 401 for invalid tokens and 503 for unavailable verification infrastructure, and never derives capabilities.
Safe failure / rollback: The new verifier is unreachable until S04 wires explicit JWKS mode; incomplete code cannot replace the existing remote verifier.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Verifier contract | `server/internal/auth/auth.go` | `Verifier.Verify` | Returns `Principal` or classified error and stays explicit at handler boundaries. |
| HTTP mapping | `server/internal/auth/auth.go` | `WriteError` | `ErrUnauthorized` maps to 401; `ErrUnavailable` and unexpected auth errors map to redacted 503. |
| Remote verifier | `server/internal/auth/supabase.go` | `SupabaseVerifier.Verify` | Uses `/auth/v1/user` with a 5-second client timeout and established error classes. |
| Capability authority | `server/internal/auth/resolver.go` | `StaffCapabilityResolver.Resolve` | Links only by `p.UserID`; provider email/role does not grant capability. |
| Existing proof | `server/internal/auth/auth_test.go` | `TestSupabaseVerifierClassifications`, `TestWriteErrorNoSensitiveDataLeakage` | Proves remote/status mapping and diagnostic redaction, not local JWT/JWKS behavior. |
| Dependency boundary | `go.mod`, `go.sum` | direct requirements | No JWT/JWK module is declared; module Go version is 1.25. |

Read set:
- `server/internal/auth/auth.go`, `supabase.go`, `resolver.go` and their tests - interface, classification, redaction, and authority invariants.
- `go.mod`, `go.sum` - dependency and toolchain baseline.

Modify set:
- `server/internal/auth/jwks.go` - new `JWKSVerifier`, immutable cache snapshot, coordinated refresh, strict key/token validation, and constructors.
- `server/internal/auth/jwks_test.go` - generated test keys, fake clock, counting JWKS server, resolver spy, concurrency, limits, and classification tests.
- `go.mod`, `go.sum` - pin `github.com/lestrrat-go/jwx/v3` v3.2.0 and checksums produced by Go tooling.

Must not modify:
- `server/internal/auth/auth.go` - existing verifier/principal/error contracts already suffice.
- `server/internal/auth/supabase.go` - remote verifier remains the rollback implementation.
- `server/internal/auth/resolver.go` - staff authority and linking semantics remain unchanged.
- Module handlers - all already consume the shared authenticator contract.

#### Behavior contract

Before:
- No local JWT verifier or JWKS cache exists; every Supabase verification calls `/auth/v1/user`.

After:
- `JWKSVerifier.Verify` accepts only verified `ES256`/`RS256` tokens from the configured project, returns a capability-empty user principal, and uses the bounded cache/refresh/error contract in `spec.md`.

Preserved invariants:
- `Principal` remains explicit; resolver is the only capability authority.
- Remote verifier behavior and 401/503 public bodies remain unchanged.
- Token content never chooses a URL, verifier mode, algorithm implementation, or capability.

Inputs and authority:
- Constructor receives the normalized trusted base URL from config/bootstrap.
- Test-only construction injects `http.Client` and clock without adding a production provider abstraction.
- `github.com/lestrrat-go/jwx/v3` v3.2.0 supplies audited JWK/JWS/JWT parsing and cryptographic verification; local code owns network/cache bounds and error classification.

Errors and edge states:
- Reject before fetch: oversized/malformed compact token, missing `kid`, `none`, HS256, unsupported alg.
- Unauthorized: matching-key signature failure, claim failure, or successful refresh without requested key.
- Unavailable: timeout, transport error, non-200 JWKS response, redirect, oversized body/set, malformed set, or set with no approved keys when no fresh matching key remains.
- Fresh matching keys verify without network; expired snapshots require refresh and never fall back to stale keys.

#### Ordered edits

1. `go.mod`, `go.sum` - direct JWT/JWK dependency
   - Change: run `go get github.com/lestrrat-go/jwx/v3@v3.2.0`; retain only Go-generated module changes and run `go mod tidy` after imports exist.
   - Constraint: do not use v4 because it requires Go 1.26/`GOEXPERIMENT=jsonv2`; do not add a second JWT library.
   - Result: one pinned Go-1.25-compatible JOSE implementation owns cryptographic parsing/verification.
2. `server/internal/auth/jwks.go` - `JWKSVerifier`, `jwksSnapshot`, constructors
   - Change: derive issuer/JWKS URL from the validated base; configure 5-second timeout, redirect rejection, 10-minute freshness, 1 MiB body, 32 keys, 16 KiB token, and 60-second skew constants.
   - Constraint: no URL may be read from JWT headers/claims; no background goroutine or unbounded cache.
   - Result: the verifier has one immutable key-set snapshot and a bounded synchronous fetch boundary.
3. `server/internal/auth/jwks.go` - header/key selection and `refresh`
   - Change: inspect only protected header fields needed for `kid`/`alg`, enforce exact algorithm/key-type/strength/key-use rules, parse the whole response before atomic publication, and coordinate refresh by cache generation plus mutex so concurrent attempts share it.
   - Constraint: unverified payload claims cannot influence fetch, lookup, or classification; partial sets never replace the cache.
   - Result: cold/expired cache and unknown `kid` trigger at most one fetch per observed generation and attempt.
4. `server/internal/auth/jwks.go` - `JWKSVerifier.Verify`
   - Change: verify signature and exact issuer/audience/time/subject contract, extract only verified `sub`/string `email`, and return `Principal{UserID: sub, Email: email, Role: "user"}` with no staff ID/capabilities.
   - Constraint: never accept unverified parse results; never copy JWT role, metadata, app claims, or capability-shaped values.
   - Result: invalid tokens wrap `ErrUnauthorized`; key infrastructure failures wrap `ErrUnavailable` with stable stage text and no token/claims.
5. `server/internal/auth/jwks_test.go` - focused verifier harness and tests
   - Change: generate ephemeral P-256/RSA test keys, sign Supabase-shaped tokens, count exact HTTP requests, inject time, race concurrent unknown-`kid` calls, and spy on resolver invocation through `Authenticator`.
   - Constraint: no static private key, live token, internet call, sleep-based timing, or `/auth/v1/user` fixture.
   - Result: named tests independently prove algorithms, claims, cache freshness, refresh coordination, limits, classifications, non-fallback, and capability isolation.

#### Integration trace

Protected handler in bootstrap/commerce/content/media/sitecontent/staff
  -> `auth.Authenticator.Principal` parses bearer token
  -> `JWKSVerifier.Verify` checks bounded header
  -> trusted derived `/.well-known/jwks.json` via cache only when required
  -> JWX signature + issuer/audience/time/sub validation
  -> capability-empty `Principal{UserID, Email, Role:user}`
  -> `StaffCapabilityResolver.Resolve` lookup by verified `sub`
  -> existing handler capability check
  -> unchanged success/401/403/503 response contract

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Warm cache is local and capabilities remain server-derived | `.` | `go test ./server/internal/auth -run 'TestJWKSVerifierWarmCacheAvoidsNetwork|TestJWKSVerifierVerifiedClaimsDoNotGrantCapabilities' -count=1` | Named tests plus fetch/resolver counters | Zero fetch on warm second call; resolver sees verified sub; JWT capability claims ignored | Token role/metadata cannot grant capability | Temporarily copy JWT role into `Principal.Role`; capability-isolation assertion must fail | Restore line; `git diff --exit-code -- server/internal/auth/jwks.go` against pre-trigger patch |
| Invalid token classes return unauthorized without resolver/fallback | `.` | `go test ./server/internal/auth -run 'TestJWKSVerifierRejectsInvalidTokens|TestJWKSVerifierRejectsSymmetricAndUnsupportedAlgorithms' -count=1` | Table subtests and zero-call spies | Every listed token maps through `errors.Is(..., ErrUnauthorized)` | wrong issuer/aud, expired/future, empty sub, bad signature, HS256/none/missing kid | Temporarily change expected audience to `anon`; valid-token control must fail for audience mismatch | Restore constant and rerun focused command |
| Unknown kid refresh is single and coordinated | `.` | `go test -race ./server/internal/auth -run 'TestJWKSVerifierUnknownKIDRefreshesOnce|TestJWKSVerifierConcurrentUnknownKIDRefreshIsCoordinated' -count=1` | Exact fetch count and race detector | One refresh for an attempt/generation; concurrent callers share publication | Successful refresh still missing kid returns unauthorized | Temporarily remove the generation recheck inside the refresh lock; concurrent fetch-count assertion must fail | Restore recheck; rerun with `-race`; confirm no temporary files |
| JWKS failures are bounded and unavailable | `.` | `go test ./server/internal/auth -run 'TestJWKSVerifierUnavailableWithoutFreshKey|TestJWKSVerifierRejectsOversizedOrInvalidJWKS|TestJWKSVerifierDoesNotUseExpiredKeyAfterRefreshFailure' -count=1` | Error class, limits, cache snapshot, response/log checks | `ErrUnavailable`, no partial/stale publication, no protected data | timeout/non-200/redirect/oversize/malformed/no-approved-key | Temporarily return `ErrUnauthorized` for fetch failure; classification assertion must fail | Restore branch; rerun tests and `git diff --check` |
| Dependency graph is reproducible | `.` | `go mod verify` | Module checksum verification | All modules verified | Tampered module cache fails verification | No repository mutation; rely on checksum verifier's negative semantics rather than corrupting shared cache | `git diff --check -- go.mod go.sum`; no cache mutation performed |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] The bounded failure trigger fails for the claimed reason and is restored.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.

### Packet S04: Wire one verifier at startup with no request fallback

Status: executable
Covers: REQ-001, REQ-002, AC-001, AC-003, AC-004
Hard dependencies: S02 supplies validated mode/URL; S03 supplies the JWKS verifier.
Outcome: Bootstrap selects exactly one process-wide verifier and every existing protected consumer reaches it through the unchanged explicit authenticator.
Safe failure / rollback: Constructor/config errors abort startup; `remote` restores the previous verifier on redeploy without code changes.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Wiring owner | `server/internal/bootstrap/app.go` | `NewWithDB` | Switches only on `AuthMode`; Supabase always constructs `NewSupabaseVerifier`. |
| Real entry | `server/internal/bootstrap/app.go` | `GET /api/admin/me` | Calls the shared `authenticator.Principal`. |
| Protected consumers | `server/internal/modules/{commerce,content,media,sitecontent,staff}` HTTP files | calls to `Authenticator.Principal` | Seven inspected production files share the authenticator wired by bootstrap. |
| Adapter proof | `server/internal/bootstrap/staff_lookup_test.go` | staff lookup adapter tests | Proves resolver adapter errors, not verifier-mode wiring. |

Read set:
- `server/internal/bootstrap/app.go`, `staff_lookup.go`, `staff_lookup_test.go` - construction and resolver wiring.
- Seven production files returned by `rg -l '\.Principal\(r\)' server/internal --glob '*.go'` - consumer reachability only.

Modify set:
- `server/internal/bootstrap/app.go` - extract unexported `newAuthVerifier(config.Config)` and call it from `NewWithDB`.
- `server/internal/bootstrap/app_test.go` - new constructor-selection/error tests.

Must not modify:
- Module HTTP files - the shared authenticator already reaches them.
- `auth.Authenticator` or `Principal` transport - explicit boundary remains.
- Database/staff adapter construction - capability resolution remains identical in both Supabase verifier modes.

#### Behavior contract

Before:
- Supabase mode always constructs the remote verifier.

After:
- Bootstrap constructs remote for empty/`remote`, JWKS for `jwks`, and returns a startup error for invalid constructor input. The selected verifier is immutable for process lifetime.

Preserved invariants:
- Both Supabase modes install the same `StaffCapabilityResolver`.
- All protected handlers receive the same authenticator; no handler branches on verifier mode.

Inputs and authority:
- Only validated `config.Config` supplies mode/base URL/publishable key.
- Request tokens cannot alter the selected verifier or invoke the remote verifier from JWKS mode.

Errors and edge states:
- JWKS constructor/configuration errors abort `NewWithDB` and close the database through the existing `New` error path.

#### Ordered edits

1. `server/internal/bootstrap/app.go` - `NewWithDB` verifier construction
   - Change: extract `newAuthVerifier(cfg)` returning `(auth.Verifier, error)`; select existing `NewSupabaseVerifier` for remote, `NewSupabaseJWKSVerifier` for JWKS, and existing dev verifier otherwise.
   - Constraint: selection occurs once; no composite/fallback verifier and no token inspection in bootstrap.
   - Result: mode-to-implementation mapping is directly testable and all consumers share it.
2. `server/internal/bootstrap/app_test.go` - `TestNewAuthVerifierSelectsExplicitMode`
   - Change: assert concrete remote/JWKS/dev selection and constructor error propagation using non-secret local URLs.
   - Constraint: do not open a database, start a live server, or contact Supabase.
   - Result: swapping or adding an implicit fallback fails a narrow bootstrap test.
3. `server/internal/bootstrap/app_test.go` - `TestSupabaseVerifierModesRetainResolverBoundary`
   - Change: inspect the construction path and exercise an authenticator with a resolver spy for both verifier modes using injected/local verifier dependencies.
   - Constraint: assert staff lookup uses verified `sub`; do not duplicate module handler suites.
   - Result: both modes retain server-side capability authority.

#### Integration trace

validated `config.Config`
  -> `bootstrap.newAuthVerifier`
  -> exactly one `auth.Verifier`
  -> `auth.NewAuthenticatorWithResolver`
  -> bootstrap + commerce + content + media + sitecontent + staff handlers
  -> `Authenticator.Principal`
  -> selected verifier only
  -> staff resolver
  -> unchanged protected response

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Startup mapping is explicit and has no fallback | `.` | `go test ./server/internal/bootstrap -run 'TestNewAuthVerifierSelectsExplicitMode' -count=1` | Concrete selection/error assertions | remote/JWKS/dev map once | invalid constructor input returns error | Temporarily map JWKS to `NewSupabaseVerifier`; concrete-type assertion must fail | Restore branch and rerun focused test |
| Resolver remains authoritative | `.` | `go test ./server/internal/bootstrap -run 'TestSupabaseVerifierModesRetainResolverBoundary' -count=1` | Resolver spy | Both Supabase modes resolve by verified sub | JWT role/capability data cannot bypass resolver | Temporarily construct JWKS authenticator without resolver in test seam; assertion must fail | Restore construction and confirm only authorized diff remains |
| Every production consumer uses shared authenticator | `.` | `rg -l '\.Principal\(r\)' server/internal --glob '*.go'` | Seven known production files | Same inventory; none construct/select verifiers | Direct module verifier construction is absent | `rg -n 'NewSupabase(JWKS)?Verifier' server/internal/modules` must return no matches | No mutation; record both inventory commands in consumer-reachability receipt |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] The bounded failure trigger fails for the claimed reason and is restored.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.

### Packet S05: Document rollout and independently replay the security contract

Status: executable
Covers: REQ-001, REQ-002, REQ-003, REQ-004, AC-001, AC-002, AC-003, AC-004, AC-005, AC-006
Hard dependencies: S02-S04, because documentation and review must describe and inspect the implemented configuration, verifier, cache, and wiring. S01 is not a hard dependency for repository-only review; its live receipt remains separately blocked.
Outcome: Operators can deliberately select/rollback verifier mode, and repository evidence independently proves the local trust boundary without claiming live compatibility.
Safe failure / rollback: Documentation keeps `remote` as default and labels JWKS activation blocked until S01; failed review leaves all evidence pending.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Starter guidance | `README.md` | Intentional v0 limits | Calls cached JWKS a future optimization. |
| OAuth runbook | `docs/supabase-oauth.md` | After the callback | Describes access-token flow but not verifier selection/revocation behavior. |
| Environment ownership | `docs/environment-configuration.md` | Deployment ownership/browser boundary | Verifier mode is absent; browser allowlist is explicit. |
| Inventories | `.env.development.example`, `.env.production.example` | Authentication sections | No server-only verifier selector. |
| Repository gates | `Makefile`, `.github/workflows/ci.yml` | `verify`, CI Go tests | Establish exact Go/gate commands; no workflow edit is required. |

Read set:
- All S02-S04 modify paths and tests - implementation under review.
- `README.md`, `docs/environment-configuration.md`, `docs/supabase-oauth.md`, env examples - operator contract.
- `server/tools/speccheck`, `server/tools/verify`, `.github/workflows/ci.yml` - command discovery only.

Modify set:
- `.env.development.example`, `.env.production.example` - add server-only `SUPABASE_VERIFIER_MODE=remote` inventory and JWKS opt-in warning.
- `README.md` - replace the future-optimization statement with the explicit mode/rollback/session semantics.
- `docs/environment-configuration.md` - add Railway-only ownership, accepted values, URL constraints, and browser exclusion.
- `docs/supabase-oauth.md` - document remote/JWKS verification and sign-out latency.
- `specs/changes/supabase-jwks-verifier/receipts/security-review-rev2.md` - independent source/diff/test review.
- `specs/changes/supabase-jwks-verifier/receipts/consumer-reachability-rev2.md` - verifier-to-handler inventory.
- `specs/changes/supabase-jwks-verifier/control.json`, `evidence.md` - update only from independently observed results; keep AC-006 blocked until S01.

Must not modify:
- Vite configs/browser safe-key allowlists - selector is server-only.
- CI, Makefile, governance tools, architecture, or skills - existing gates suffice.
- `spec.md`/`plan.md` during apply unless repository drift triggers controlled re-expansion.

#### Behavior contract

Before:
- Operator docs promise only remote verification and do not expose the local verification/revocation trade-off.

After:
- Docs identify `remote` as default/rollback, `jwks` as asymmetric-only opt-in, and local sign-out recognition as bounded by access-token expiry.
- Independent receipts distinguish local observed evidence from the environment-blocked live walkthrough.

Preserved invariants:
- The new variable is server-only and never compiled into admin/site assets.
- Draft/blocked evidence is never promoted based on implementation reports or mocked provider behavior.

Inputs and authority:
- Source/tests provide repository evidence; official Supabase documentation provides provider semantics; S01 alone provides live project facts.

Errors and edge states:
- Any mismatch between docs, config validation, implementation constants, tests, or official provider contract leaves mapped evidence pending and triggers re-expansion.

#### Ordered edits

1. `.env.development.example`, `.env.production.example`, `docs/environment-configuration.md` - authentication inventory
   - Change: document `SUPABASE_VERIFIER_MODE=remote`, Railway/server-only ownership, accepted modes, JWKS URL rules, and environment validation prerequisite.
   - Constraint: do not add it to browser allowlists or mark it secret.
   - Result: operator configuration is explicit without exposing new browser state.
2. `README.md`, `docs/supabase-oauth.md` - Supabase verification sections
   - Change: document remote default/rollback, asymmetric-only JWKS mode, no request fallback, and sign-out/session termination remaining valid until JWT expiry.
   - Constraint: do not claim the current deployment is compatible before S01.
   - Result: rollout and security semantics match code and provider documentation.
3. `specs/changes/supabase-jwks-verifier/receipts/security-review-rev2.md` - independent review
   - Change: inspect dependency diff, algorithm/key filters, URL derivation/redirect handling, claim validation, cache concurrency/limits, error/log redaction, session semantics, and all bounded mutation results.
   - Constraint: no secret/token/provider body in receipt; implementer report is not evidence.
   - Result: AC-002/003/004/005 gain attributable review evidence or remain pending with exact findings.
4. `specs/changes/supabase-jwks-verifier/receipts/consumer-reachability-rev2.md` - caller trace
   - Change: record bootstrap selection through the shared authenticator to all seven inspected production consumer files and prove no module constructs a verifier.
   - Constraint: source definitions alone are insufficient; include selected test execution and real `GET /api/admin/me` entry.
   - Result: AC-001/003 gain attributable reachability evidence.
5. `control.json`, `evidence.md` - evidence reconciliation
   - Change: bind passed local evidence to revision 2 only after independent replay; retain AC-006 `blocked` until the S01 receipt exists.
   - Constraint: do not mark expected commands, mocks, or documentation as live Supabase evidence.
   - Result: the controlled record accurately separates implemented local behavior from environment proof.

#### Integration trace

implemented config/verifier/bootstrap diff
  -> targeted auth/config/bootstrap tests
  -> protected-consumer source inventory
  -> independent security review and bounded mutation replay
  -> server-only operator documentation
  -> revision-bound local receipts/evidence
  -> separate S01 live walkthrough before JWKS production activation

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Operator docs match server-only mode contract | `.` | `rg -n 'SUPABASE_VERIFIER_MODE|remote|jwks|sign.out|expiry' README.md docs/environment-configuration.md docs/supabase-oauth.md .env.development.example .env.production.example` | Authentication sections | Mode/rollback/session caveat present; variable absent from browser ownership | No claim of automatic fallback/live compatibility | Temporarily remove the production inventory line; selected documentation assertion/checklist must report the omission | Restore line and run `git diff --check` |
| Targeted behavior suites pass | `.` | `go test ./server/internal/auth ./server/internal/config ./server/internal/bootstrap -count=1` | Named S02-S04 tests in output/receipt | All selected tests execute and pass | Invalid/availability/selection subtests remain present | Replay the packet-specific bounded mutations one at a time | Restore each mutation; rerun command and record clean scoped diff |
| Architecture and controlled scope remain intact | `.` | `go run ./server/tools/archcheck` then `go run ./server/tools/speccheck` then `go run ./server/tools/scopecheck` | Three deterministic gates | PASS | Module imports, uncontrolled paths, or stale evidence fail | No gate mutation; inspect actual implementation diff | `git diff --check`; compare changed paths to `control.json` and task-local scope |
| Repository verifier passes without substituting for behavior proof | `.` | `go run ./server/tools/verify` | Full verifier plus separate named-test receipts | PASS after targeted proof | Any unrelated/pre-existing failure is reported, not hidden | No verifier mutation | Re-run `git status --short`; preserve all pre-existing dirty paths |
| Live compatibility remains honestly blocked | `.` | inspect `control.json`, `evidence.md`, and receipt inventory | AC-006 status/receipt | AC-006 stays blocked until S01 receipt exists | Mock/local JWKS test cannot pass AC-006 | Temporarily attempt to mark AC-006 passed without walkthrough receipt; `speccheck` must fail | Restore blocked record and confirm `speccheck` returns to prior result |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] The bounded failure trigger fails for the claimed reason and is restored.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.

## Traceability matrix

| REQ / AC | Packet | Behavior owner / consumer | Expected proof |
|---|---|---|---|
| REQ-001 | S02, S04, S05 | config -> bootstrap -> all protected consumers | mode validation, constructor selection, consumer receipt |
| AC-001 | S02, S04, S05 | `Config.Validate`, `newAuthVerifier` | default/explicit/invalid mode tests; no module verifier construction |
| REQ-002 | S03, S04, S05 | `JWKSVerifier.Verify` -> resolver -> handlers | local/no-network and capability-isolation tests |
| AC-002 | S02, S03, S05 | URL authority and JWT/JWK validation | algorithm/key/issuer/audience/time/sub tables plus security review |
| AC-003 | S03, S04, S05 | warm cache -> verifier -> resolver -> real handlers | exact fetch count, resolver spy, consumer reachability |
| AC-004 | S03, S04, S05 | invalid token -> `ErrUnauthorized` -> `WriteError` | 401/no-resolver/no-fallback tables and security review |
| REQ-003 | S03, S05 | cache snapshot/refresh -> error mapping | race/fetch-count/limit/unavailable tests |
| AC-005 | S03, S05 | bounded fetch/cache/error boundary | refresh coordination, stale rejection, redaction and security receipt |
| REQ-004 | S01, S05 | deployment config -> protected endpoint -> receipt | environment walkthrough and rollback observation |
| AC-006 | S01, S05 | actual Supabase/Railway deployment | non-secret walkthrough plus security review; remains blocked meanwhile |

## Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass |
| Every AC maps to proof in at least one packet | pass; AC-006 proof is environment-blocked and explicit |
| Every packet maps back to REQ/AC | pass |
| Baseline, observed HEAD, dirty paths, and packet mapping are explicit | pass |
| All repository paths and symbols named by executable packets were inspected | pass |
| Hard dependency graph has no unexplained edge or cycle | pass |
| Scope covers every future modify path and no unrelated path | pass |
| Product/authority decisions are approved or blocked | pass; no `DECISION_REQUIRED` remains, live values are `ENVIRONMENT_BLOCKED` |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |

The proposal remains Draft because product edits are not yet authorized and S01 requires a real deployment environment. Plain apply may execute S02-S05 only after review/approval; it must not improvise S01 evidence or activate JWKS in production without the stated inputs.
