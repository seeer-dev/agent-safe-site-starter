# Security review — log record non-disclosure

Change ID: structured-request-observability
Revision: 1
Covers: AC-004
Reviewed: 2026-08-18

## Scope

Whether the new per-request record or the new 5xx diagnostic can carry a
credential, a token, an authorization header, a cookie, an address, or a
connection string.

## Controls

1. **Only the path is logged, never the query string.** A caller can put
   anything in a query value, and `/api/admin/orders?token=…` is exactly the
   shape that would otherwise land in a log line.
2. **No header is logged.** `Authorization` and `Cookie` are never read by the
   middleware. The only header values that reach a record are `X-Request-Id`
   and `X-Forwarded-For`, both filtered to `[A-Za-z0-9_-]` and capped at 64
   characters.
3. **The forwarded value is named `forwarded_for_claim`.** It is
   attacker-controllable, so it is never recorded as `client_ip` and the
   transport peer is recorded separately under `peer`. Nothing in this change
   consumes either for a decision.
4. **`ErrorWithCause` redacts before logging.** URL-shaped runs, JWT-shaped
   tokens, and email-shaped tokens are replaced, and the result is truncated at
   200 runes — rune-based, because byte slicing can emit invalid UTF-8 into a
   log. This mirrors `auth.WriteError`'s existing posture.
5. **The public body never widens.** `ErrorWithCause` writes exactly what
   `Error` would; the cause is a second, server-only destination.

## Observation

`TestRecordDisclosesNothingSensitive` drives a request carrying a bearer token
in both the `Authorization` header and the query string, a session cookie, into
a handler that fails with a cause containing a PostgreSQL connection string and
an email address. The record contains none of the four secrets, and still
contains `connect`, `failed`, and `status=500`, so an over-broad redactor
cannot pass either.

## A defect found in this review's own gate

The first version of the assertion was **not sensitive to URL redaction**.
Removing `diagnosticURLPattern` left the test green, because the chosen DSN
(`postgres://user:secretpw@db.internal:5432/app`) contained
`secretpw@db.internal`, which the *email* pattern matched and redacted. The
test passed for the wrong reason.

The fixture now uses `postgres://svcuser:secretpw@localhost:5432/app`. The host
has no dot-TLD, so the email pattern cannot match any part of it and only URL
redaction can remove it. Re-running the mutation after the fix:

```console
$ # diagnosticURLPattern replacement commented out
--- FAIL: TestRecordDisclosesNothingSensitive
    record leaked "postgres://svcuser:secretpw@localhost:5432/app":
    ... cause="connect postgres://svcuser:secretpw@localhost:5432/app for [redacted-email] failed"
```

Restored, the test is green. This is recorded rather than quietly corrected
because a gate that passes for the wrong reason is the specific failure class
this repository has been tracking.

## Residual

Redaction is pattern-based, so a secret in an unanticipated shape can pass
through. The structural fix — never stringifying a raw external error and
logging only allowlisted stage identifiers — is larger than this change and is
recorded as a known limit rather than claimed as done.

`slog` writes to the default handler. Whoever configures log shipping owns the
transport; this change does not choose a destination.
