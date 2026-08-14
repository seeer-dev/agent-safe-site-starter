# Authentication Error Separation Specification

Change ID: auth-error-separation
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner issued plain apply instruction on 2026-08-14 to implement revision 1.
Repository baseline: 374ffc755f0098ac22059e9987f4bf5ff7c9cd8e
Supersedes: none

## Outcome

Keep valid-but-unlinked users, rejected credentials, and authentication dependency failures as distinct states from the staff store through every protected HTTP consumer. A staff database or Supabase Auth availability failure must never be reported as a missing staff link, lack of permission, or expired login. Public responses remain generic and fail closed, while the server retains a safe diagnostic signal and the admin client preserves a still-valid session for retryable failures.

## Scope

In scope:
- Typed separation between a missing `staff_members` link and an unexpected staff lookup failure without importing the staff module into `auth`.
- Explicit authentication error classification for credential rejection versus dependency unavailability.
- A shared protected-request error mapping used by `/api/admin/me` and every existing module handler that calls `auth.Principal`.
- Minimal server-side diagnostic output for authentication dependency failures that identifies the failing stage without exposing credentials or user PII.
- Regression coverage for resolver behavior, bootstrap error translation, representative protected endpoints, and admin session behavior on HTTP 503.

Out of scope:
- Authentication caching, local JWT/JWKS verification, request coalescing, or changes to the number of Supabase/staff lookups.
- The broader observability baseline: request IDs, structured access logs, global panic recovery, metrics, traces, and standardized 5xx logging remain a separate controlled change.
- Staff role/capability policy, staff linking workflows, email-based linking, OAuth provider setup, or session persistence design.
- Database schema, migration, CORS, deployment topology, and API route changes.

## Decisions and Invariants

- Missing Link Is Expected: only the staff module's canonical not-found result is translated by the bootstrap adapter into the auth package's unlinked-user result. The resolver returns a valid `role=user` principal with no staff ID or capabilities for this state.
- Infrastructure Failure Is Not Identity Failure: staff database errors, Supabase transport failures, upstream throttling/server failures, and malformed successful provider responses are classified as authentication unavailability. They fail closed and return no principal.
- Credential Rejection Remains 401: missing or malformed bearer headers, empty tokens, and an explicit provider credential rejection remain a generic HTTP 401 response.
- Authorization Remains Separate: valid unlinked users and disabled staff keep their current principal semantics. Capability checks continue to return the existing forbidden behavior; this change does not grant or infer capabilities.
- Explicit Boundary Remains: handlers continue to obtain `auth.Principal` explicitly. The principal is not stored in `context.Context`, and business modules do not import one another.
- Consistent HTTP Mapping: every existing `Principal(r)` consumer uses one shared classification contract. An implementation that fixes only `/api/admin/me` or only one module is incomplete.
- Safe Diagnostics: an availability failure emits a stable server-side event with the dependency stage and wrapped cause. The event and public body must not contain bearer tokens, authorization headers, email addresses, provider response bodies, or database connection strings.
- Retryable Admin State: HTTP 503 from `/api/admin/me` places the admin store in its generic failed state, clears server-derived capabilities, and does not clear the access token, invalidate the provider session, or invoke sign-out. HTTP 401 keeps the existing invalidation behavior.

## Requirements

### REQ-001: Staff Lookup Result Separation
The authentication resolver MUST distinguish a canonical missing staff link from every unexpected lookup failure and MUST preserve current unlinked, disabled, and development-principal behavior.

#### AC-001: Valid Unlinked User
- GIVEN a verified Supabase principal whose `supabase_user_id` has no staff row
- WHEN the bootstrap staff adapter and capability resolver execute
- THEN the result MUST contain the verified user ID, `role=user`, an empty staff ID, and no capabilities without returning an error or attempting email fallback.

#### AC-002: Staff Lookup Infrastructure Failure
- GIVEN a verified principal and a staff lookup that fails for any reason other than the canonical staff not-found result
- WHEN capability resolution executes
- THEN it MUST return no principal, preserve the original cause through wrapping, and classify the result as authentication unavailable rather than unlinked, unauthorized, or forbidden.

### REQ-002: Authentication Provider Error Separation
The authentication layer MUST preserve the difference between rejected credentials and a failure of the configured identity provider.

#### AC-003: Credential Rejection
- GIVEN a missing or malformed bearer header, an empty bearer token, or an explicit invalid-credential response from Supabase Auth
- WHEN a protected endpoint authenticates the request
- THEN it MUST return HTTP 401 with a stable generic body and MUST NOT emit provider, token, or database details to the client.

#### AC-004: Authentication Dependency Unavailable
- GIVEN a Supabase transport/throttling/server/response-decoding failure or an unexpected staff lookup failure
- WHEN `/api/admin/me` or any current protected module endpoint authenticates the request
- THEN it MUST return HTTP 503 through the shared mapping, emit a safe server diagnostic event, return no protected data, and never expose the internal cause in the response body.

### REQ-003: Permission and Session Integrity
The change MUST preserve established permission semantics and MUST keep retryable infrastructure failures from destroying valid client sessions.

#### AC-005: Unlinked and Disabled Permission Semantics
- GIVEN a valid unlinked user or a disabled staff user
- WHEN `/api/admin/me` and capability-protected operations execute
- THEN the unlinked user MUST remain `role=user` with no capabilities, the disabled user MUST remain `role=disabled` with no capabilities, and neither state may gain access through error fallback.

#### AC-006: Admin 503 Session Preservation
- GIVEN an initialized admin session with a non-empty access token
- WHEN `/api/admin/me` responds with HTTP 503
- THEN the admin store MUST enter its generic failed/retryable state, expose no server capabilities or backend error detail, retain the token and provider session, and MUST NOT invoke sign-out; the existing HTTP 401 path MUST still clear and sign out the invalid session.

### REQ-004: Safe and Complete Consumer Migration
Every current protected handler MUST consume the same authentication error contract, and diagnostics MUST remain free of credentials and user PII.

#### AC-007: Protected Consumer and Disclosure Regression Gate
- GIVEN the protected entry points in bootstrap, commerce, content, media, sitecontent, and staff
- WHEN focused tests inject unauthorized, unavailable, unlinked, and disabled outcomes
- THEN all entry points MUST use the shared mapping, tests MUST prove 401/503 consistency and fail-closed responses, and captured public responses and diagnostic events MUST contain no bearer token, authorization header, email address, provider response body, or database connection string.

## Amendments

None.
