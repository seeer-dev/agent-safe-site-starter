# Authentication Error Separation Delivery Plan

Change ID: auth-error-separation
Revision: 1
Status: Applying

Normative specification: [`spec.md`](spec.md)

## Repository Reality and Baseline

| Area | Observed state at proposal time | Delivery implication |
|---|---|---|
| Staff resolver | `server/internal/auth/resolver.go` converts every staff lookup error into `role=user` with no capabilities | Introduce an auth-owned missing-link result and propagate all other causes as unavailable |
| Module boundary | `staff.ErrNotFound` belongs to the staff module; `auth` must not import that business module | Translate the sentinel in `server/internal/bootstrap/staff_lookup.go`, matching the existing adapter pattern used for media errors |
| Protected HTTP consumers | Bootstrap plus commerce, content, media, sitecontent, and staff call `Principal(r)`; each currently maps every error to HTTP 401 | Add one shared mapping contract and migrate every current call site in the same slice |
| Supabase verifier | Explicit provider rejection returns `ErrUnauthorized`, while transport and decode errors return ordinary wrapped errors | Classify dependency failures explicitly so the shared HTTP boundary can return 503 without treating bad credentials as availability failures |
| Admin consumer | `admin/src/stores/auth.ts` signs out only on HTTP 401; other failures enter `failed` and retain the session | Preserve production behavior and add a regression test for the new server-side 503 path |
| Logging | Existing access logging records method/path/duration but not status or request ID | Add only a bounded auth-unavailable diagnostic event here; defer the global observability model to its own proposal |

## Scope Lock

Implementation is limited to the auth package, the bootstrap staff adapter and `/api/admin/me`, the seven current files containing protected handler entry points, their focused tests, and the admin auth-store regression test named in `control.json`. No database, migration, role policy, CORS, deployment, or OAuth configuration file is authorized.

- `server/internal/auth/**`
- `server/internal/bootstrap/app.go`
- `server/internal/bootstrap/staff_lookup.go`
- `server/internal/bootstrap/staff_lookup_test.go`
- `server/internal/modules/commerce/http.go`
- `server/internal/modules/commerce/http_test.go`
- `server/internal/modules/content/http.go`
- `server/internal/modules/media/http.go`
- `server/internal/modules/media/media.go`
- `server/internal/modules/media/verify_http_test.go`
- `server/internal/modules/sitecontent/http.go`
- `server/internal/modules/sitecontent/http_test.go`
- `server/internal/modules/staff/http.go`
- `server/internal/modules/staff/http_test.go`
- `admin/src/stores/auth.test.ts`
- `specs/changes/auth-error-separation/**`

## Dependency-Ordered Slices

### Slice 1: Typed Producer and Adapter Errors
- Add auth-owned classifications for a valid missing staff link and authentication dependency unavailability.
- Translate only `staff.ErrNotFound` at `staffLookupAdapter`; preserve and wrap every other cause.
- Update the capability resolver so only the typed missing-link result produces a user principal with no capabilities.
- Classify Supabase credential rejection separately from provider transport, throttling/server, and response-decoding failures.
- Add focused tests for cause preservation, no email fallback, disabled staff, development capabilities, and provider classifications.
- Covers: REQ-001, REQ-002, AC-001, AC-002, AC-003.

### Slice 2: Shared HTTP Mapping and Protected Consumers
- Implement one shared HTTP mapping for errors returned by `auth.Principal`: credential rejection to generic 401, typed dependency unavailability to generic 503, and an internal-safe fallback for unexpected auth errors.
- Emit a stable, bounded diagnostic event on the unavailable path without tokens, headers, emails, response bodies, or connection strings.
- Replace the hard-coded 401 branch at every current `Principal(r)` consumer in bootstrap, commerce, content, media, sitecontent, and staff.
- Exercise `/api/admin/me` plus representative module endpoints for 401, 503, no protected response data, and non-disclosure. Record full call-site reachability in a receipt rather than claiming representative tests cover uninspected consumers.
- Covers: REQ-002, REQ-004, AC-003, AC-004, AC-007.

### Slice 3: Admin Session Regression and Final Replay
- Add a focused admin auth-store test proving `/api/admin/me` HTTP 503 enters `failed`, clears server-derived capabilities, retains the access token/provider session, and does not call sign-out.
- Replay the existing HTTP 401 invalidation test and valid unlinked/disabled permission tests.
- Run targeted Go and admin tests, then the repository architecture, controlled-spec, scope, and verify gates.
- Record `consumer-reachability` and `security-review` receipts against the fixed revision.
- Covers: REQ-003, AC-005, AC-006, AC-007.

## Traceability Matrix

| REQ / AC | Slice | Target / Consumer | Verification | Expected Evidence |
|---|---|---|---|---|
| REQ-001 | 1 | auth resolver and bootstrap staff adapter | Focused Go unit tests | Missing link is consumed; other causes propagate as unavailable |
| AC-001 | 1 | valid Supabase user without staff row | Resolver/adapter tests | User ID retained, role user, empty staff/capabilities, no email fallback |
| AC-002 | 1 | staff database failure | Resolver/adapter tests with wrapped sentinel checks | No principal; original cause remains reachable; unavailable classification |
| REQ-002 | 1-2 | Supabase verifier and protected HTTP boundary | Verifier and HTTP tests | Credential failures produce 401; provider/staff dependencies produce 503 |
| AC-003 | 1-2 | malformed/invalid credentials | Authenticator and handler tests | Generic 401; no internal detail |
| AC-004 | 2 | `/api/admin/me` and all protected modules | HTTP tests plus consumer-reachability and security-review receipts | Generic 503, safe diagnostic, no protected data or leaked cause |
| REQ-003 | 1,3 | role resolution and admin auth store | Go resolver and Vitest store tests | Permission states preserved; availability failure remains retryable |
| AC-005 | 1,3 | unlinked and disabled principals | Existing and updated resolver/HTTP tests | No capability elevation or identity conflation |
| AC-006 | 3 | admin `/admin/me` verification | Focused Vitest 401/503 tests plus consumer receipt | 503 retains session; 401 invalidates it |
| REQ-004 | 2-3 | bootstrap, commerce, content, media, sitecontent, staff | Source inventory, full verify | No hard-coded all-errors-to-401 consumer remains |
| AC-007 | 2-3 | public error bodies and diagnostic capture | Disclosure assertions and security review | Consistent status mapping with no credential/PII leakage |

## Validation Plan

Run the narrowest affected suites first, followed by repository gates:

```text
go test ./server/internal/auth ./server/internal/bootstrap -count=1
go test ./server/internal/modules/commerce ./server/internal/modules/content ./server/internal/modules/media ./server/internal/modules/sitecontent ./server/internal/modules/staff -count=1
npm --prefix admin test -- --run src/stores/auth.test.ts
npm --prefix admin run typecheck
go run ./server/tools/archcheck
go run ./server/tools/speccheck
go run ./server/tools/scopecheck
go run ./server/tools/verify
```

No browser walkthrough is required because the admin already owns a generic failed state and this change does not alter its rendered UI. The consumer-reachability receipt must still exercise the real admin store path and prove session retention rather than relying only on Go handler tests.

## Risks and Controls

- A partial call-site migration would preserve misleading 401 behavior in some modules. Control: source inventory plus a regression search for hard-coded `Principal` error branches.
- Mapping `staff.ErrNotFound` inside `auth` would violate module boundaries. Control: bootstrap owns translation and `archcheck` must pass.
- Logging raw request or provider data could leak credentials or PII. Control: stable bounded event, captured-log assertions, and required security-review receipt.
- Treating 503 like 401 in the admin would destroy a valid session during a temporary outage. Control: explicit token/sign-out assertions and replay of the existing 401 test.
- This change does not reduce the two remote/database lookups per protected request. Authentication caching remains a separate performance change.
