# Structured Request Observability Delivery Plan

Change ID: structured-request-observability
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| Access log carries no status | `bootstrap/app.go:228` logs method, path, duration only | Wrap `ResponseWriter` to capture status |
| No correlation identity | no `X-Request-Id` handling anywhere | Accept or generate one |
| No origin recorded | `r.RemoteAddr` unused | Record peer and forwarded claim separately |
| Business modules emit nothing | `log.`/`slog.` absent from `server/internal/modules/` except mail | Add a failure diagnostic at the response boundary |
| A comment claims logging that does not exist | `media/http.go` says internal detail is "logged server-side" | Make it true rather than delete the claim |
| `auth.WriteError` already redacts and logs | `auth/auth.go` | Reuse its redaction posture; do not duplicate it |

## Scope lock

- `server/internal/bootstrap/app.go`
- `server/internal/bootstrap/observability.go` (new)
- `server/internal/bootstrap/observability_test.go` (new)
- `server/internal/httpx/httpx.go`
- `server/internal/httpx/httpx_test.go` (new)
- `server/internal/modules/media/http.go`
- `specs/changes/structured-request-observability/**`

## Dependency-ordered slices

### Slice 1: Status capture and request record

Outcome: a `ResponseWriter` wrapper records the status actually written,
defaulting to 200 for handlers that never call `WriteHeader`; the middleware
emits a structured record with method, path, status, duration, request id,
peer address, and the forwarded claim.

Acceptance evidence: table tests over explicit-status, implicit-200, and
supplied-vs-generated request id. Covers REQ-001, REQ-002, AC-001, AC-002.

Rollback: revert the middleware; the previous record format returns.

### Slice 2: Failure diagnostics

Outcome: `httpx.Error` records a server-side diagnostic for 5xx without
altering the public body; the `media` verify handler's claim becomes true.

Hard dependencies: Slice 1 provides the request id the diagnostic correlates
against.

Acceptance evidence: a 5xx served through the stack produces both a public
generic body and a server-side record. Covers REQ-003, AC-003.

Rollback: revert `httpx.Error`; public behavior is unchanged either way.

### Slice 3: Non-disclosure gate

Outcome: a test drives a request carrying an authorization header, a cookie,
and a query string into a handler failing with an internal error containing a
connection string and an email, and asserts none of them appear in the record.

Hard dependencies: Slices 1 and 2 produce the record under test.

Acceptance evidence: the assertion observed failing when a sensitive value is
deliberately routed into the record, then passing after restoration. Covers
AC-004.

Rollback: the gate is additive; removing it restores prior coverage.

## Traceability matrix

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 1 | Record-field table test |
| REQ-002, AC-002 | 1 | Explicit and implicit status cases |
| REQ-003, AC-003 | 2 | 5xx public body vs server record |
| AC-004 | 3 | Red-before-green disclosure gate |

## Risks and controls

- Risk: the forwarded header is mistaken for a trusted client identity.
  Control: it is recorded under a name that marks it a claim, and this change
  authorizes no enforcement use.
- Risk: a diagnostic leaks what the response deliberately withholds. Control:
  Slice 3 gates it, and the public body is asserted unchanged.
- Risk: logging cost on hot paths. Control: one record per request, no
  per-handler instrumentation.
