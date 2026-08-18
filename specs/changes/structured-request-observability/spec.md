# Structured Request Observability Specification

Change ID: structured-request-observability
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved the pre-launch operability batch on 2026-08-18 after review found the access log cannot distinguish a 200 from a 500 and business modules emit no diagnostics at all.
Repository baseline: e1b7cc623dbb5edebc1132bd3bc05a260d71cb0b
Supersedes: none

## Outcome

A failing request is identifiable from the logs alone: which request it was,
what it returned, where it came from, and how long it took. A server-side
failure leaves a diagnostic even though the browser is told nothing.

No route, schema, authentication semantic, or response body changes.

## Scope

In scope:

- The access-log middleware and the record it emits.
- Response status capture.
- Correlation identity for a request.
- Making the existing `media` verify-handler comment about server-side logging
  true.

Out of scope:

- Metrics, tracing, log shipping, or any external system.
- Rate limiting or any use of client address for enforcement.
- Changing what a client receives in any response body.

## Decisions and invariants

- Logging is diagnostic only. No value logged here may be used for an
  authorization or rate-limiting decision in this change.
- `X-Forwarded-For` is attacker-controllable. It is recorded as a claim, never
  resolved into a trusted client identity, and the transport peer address is
  recorded separately so the two can never be confused.
- A response body never gains detail. Where a diagnostic is added, the public
  message stays exactly as it was.
- Log records must not contain credentials, tokens, authorization headers,
  email addresses, or connection strings.

## Requirements

### REQ-001: Requests are individually identifiable

Every request MUST be logged with a correlation identifier that is stable for
that request, and that identifier MUST be derivable by a caller that supplied
one.

#### AC-001: Request record carries correlation and origin

- GIVEN any request reaching the API
- WHEN the access log emits its record
- THEN the record MUST contain the method, path, response status, duration, a
  request identifier, the transport peer address, and any forwarded-for claim
  recorded as an untrusted claim; and WHEN the caller supplied `X-Request-Id`
  that value MUST be used, otherwise a generated one MUST be present.

### REQ-002: Response status is observable

The logging layer MUST record the status code actually written, including the
implicit 200 written by a handler that never calls `WriteHeader`.

#### AC-002: Status capture covers explicit and implicit writes

- GIVEN a handler that writes an explicit status and a handler that writes a
  body without calling `WriteHeader`
- WHEN each is served through the middleware
- THEN the recorded status MUST be the explicit code in the first case and 200
  in the second, and the bytes written MUST reach the client unchanged.

### REQ-003: Server-side failures leave a diagnostic

A response of 500 or above MUST leave a server-side record, and any comment
claiming server-side logging MUST be true.

#### AC-003: Failures are diagnosable without widening the response

- GIVEN a handler that fails internally and returns a generic public message
- WHEN the response is served
- THEN a server-side record for that request MUST exist and be correlatable by
  request identifier, the public body MUST remain the generic message, and the
  `media` verify handler's stated server-side logging MUST actually occur.

#### AC-004: Log records disclose nothing sensitive

- GIVEN a request carrying an `Authorization` header, a cookie, and a query
  string, served by a handler that fails with an internal error containing a
  connection string and an email address
- WHEN the record is emitted
- THEN it MUST NOT contain the bearer token, the raw authorization header, the
  cookie, the connection string, or the email address, and the assertion MUST
  fail if any of them appears.
