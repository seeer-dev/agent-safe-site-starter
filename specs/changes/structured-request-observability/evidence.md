# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-08-18. All four gates were mutation-verified: each was observed failing for a named trigger and green after restoration, with no mutation left in the diff.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | withRequestObservability assigns a correlation identifier per request, honouring a sanitized caller-supplied X-Request-Id and generating one otherwise, and echoes it back in the response header. Proved by TestRequestRecordFields. |
| REQ-002 | passed | statusRecorder wraps the ResponseWriter and records the status actually written, defaulting to 200 for handlers that write a body or nothing without calling WriteHeader. Proved by TestStatusCaptureExplicitAndImplicit across explicit, implicit, and no-write cases. |
| REQ-003 | passed | httpx.ErrorWithCause records a redacted server-side diagnostic for 5xx while writing the unchanged public body, and media/http.go:120 now uses it so the comment at :84 claiming server-side logging is true. Proved by TestServerFailureLeavesDiagnostic. |
| AC-001 | passed | The record carries request_id, method, path, status, bytes, duration, and peer; a supplied identifier is used verbatim and echoed, an absent one is generated. A forwarded header is recorded as forwarded_for_claim and never as client_ip. Mutation-verified: renaming the field to client_ip turned the assertion red, and restoration returned it to green. |
| AC-002 | passed | Explicit status 418 recorded as 418; a body write without WriteHeader recorded as 200 with the body reaching the client unchanged; a handler writing nothing recorded as 200. Mutation-verified: returning a fixed 599 from statusOrOK turned the assertion red, and restoration returned it to green. |
| AC-003 | passed | A 5xx served through the stack returns the generic public message with the cause absent from the body, while the server-side record carries the cause and a correlatable request_id. Proved by TestServerFailureLeavesDiagnostic. |
| AC-004 | passed | A request carrying a bearer token, a cookie, and a query-string token, served by a handler failing with a cause containing a connection string and an email, produces a record containing none of them, while retaining controlled context (connect, failed, status=500). The assertion was initially non-sensitive to URL redaction because the chosen DSN host matched the email pattern; the fixture was changed to a host without a dot-TLD so only URL redaction can remove it. Mutation-verified after that fix: removing the URL redactor turned the assertion red, and restoration returned it to green. Recorded in receipts/security-review.md. |
