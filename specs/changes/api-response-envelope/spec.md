# API Response Envelope Specification

Change ID: api-response-envelope
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner directed on 2026-09-16 that the API's heterogeneous bare-payload responses be replaced by a unified status/response envelope, that the implementation be carried through to a commit, and that Resend and R2 provider paths carry executable readiness evidence.
Repository baseline: 1665e08327a4656a0e3457058f4ee7c5fd8ac182
Supersedes: none

## Outcome

Every JSON API response shares one envelope shape, and the Resend/R2 provider
integrations carry runnable contract evidence:

```json
// success — any 2xx written via httpx.JSON
{ "ok": true, "data": <payload> }

// error — httpx.Error / httpx.ErrorWithCause / auth.WriteError / edge guard
{ "ok": false, "error": { "code": "<machine-code>", "message": "<human>" } }
```

`ok` is the discriminator. `error.code` is a stable machine code derived from
the HTTP status; `error.message` keeps today's human-readable text.
`data` preserves each endpoint's existing payload — the resource keys
(`{order: ...}`, `{products: ...}`, `{items: ...}`) move inside `data`
unchanged, so payload semantics do not change, only the outer shape.

## Scope

In scope:

- `server/internal/httpx/httpx.go`: `JSON` wraps every value as
  `{ok:true, data}`; `Error`/`ErrorWithCause` emit
  `{ok:false, error:{code,message}}` with a status-derived code.
- New `server/internal/httpx/httpx_test.go` covering both shapes.
- The three client surfaces: `admin/src/lib/api-client.ts`,
  `admin/src/lib/media-api.ts`, `site/themes/curatory/shared/lib/api.ts`,
  `site/themes/minimal-cart/shared/lib/api.ts`.
- Server test files that decode or assert response bodies.
- Provider-readiness contract tests: `server/internal/platform/mail/resend_test.go`
  (new) and an offline `PresignPut` signing test in
  `server/internal/platform/storage/r2_test.go`.

Out of scope:

- Renaming or restructuring endpoint payloads inside `data`.
- Per-endpoint machine codes beyond the status-derived set.
- Non-JSON protocol endpoints: `/healthz` (plain text), the ECPay
  `1|OK` callback acknowledgement and browser-return redirect, and CORS
  `OPTIONS` 204 keep their existing wire formats.
- Adding `RESEND_API_KEY` or R2 credentials to any environment.
- Documentation owned by the in-flight `deployment-dashboard-ledger` change.

## Decisions and invariants

- The envelope is produced once, inside `httpx.JSON`/`httpx.Error`; handlers
  keep calling the same helpers and cannot bypass the shape.
- HTTP status remains the transport-level truth; `ok` mirrors
  `status < 400` so a stray non-2xx passed to `JSON` still produces
  `ok:false` rather than a lying success envelope.
- Error codes are derived from status (400 `bad_request`, 401 `unauthorized`,
  403 `forbidden`, 404 `not_found`, 405 `method_not_allowed`, 409 `conflict`,
  413 `payload_too_large`, 415 `unsupported_media_type`, 422
  `unprocessable_entity`, 429 `too_many_requests`, 500 `internal_error`,
  502 `bad_gateway`, 503 `service_unavailable`, 504 `gateway_timeout`,
  fallback `error`). Handlers need no per-call code plumbing.
- Public error messages stay exactly the strings callers see today;
  `ErrorWithCause` keeps internal causes in server logs only.
- Clients tolerate mixed-version deploys: they prefer the envelope and fall
  back to legacy bare payloads / `{"error":"msg"}` while Pages and the origin
  roll independently.
- The Pages Function proxy and the dev-server proxy pass bodies through
  untouched; they need no change.

## Requirements

### REQ-001: One envelope at the single write funnel

`httpx.JSON` MUST emit `{"ok":true,"data":<value>}` for success statuses and
`{"ok":false,"data":<value>}` for any non-2xx it is asked to write.
`httpx.Error` and `httpx.ErrorWithCause` MUST emit
`{"ok":false,"error":{"code":<status-derived>,"message":<public msg>}}`.

#### AC-001: Success and error shapes are produced centrally

- GIVEN `httpx.JSON` called with 200 and 404 statuses and `httpx.Error`
  called with a 4xx status
- WHEN bodies are decoded
- THEN the success body is `{"ok":true,"data":...}`, the non-2xx JSON body
  carries `ok:false`, and the error body carries `ok:false` plus
  `error.code` matching the status map and `error.message` equal to the
  public message.

#### AC-002: Public message text is preserved

- GIVEN an existing error site such as `auth.WriteError` or the edge guard
- WHEN the envelope change is applied
- THEN `error.message` MUST equal the previously emitted `error` string and
  `error.code` MUST be the mapped code for that status.

### REQ-002: All client surfaces consume the envelope

The admin API client, the admin media client, and both storefront theme
clients MUST unwrap `data` on success and read `error.message` (and
`error.code` where surfaced) on failure, while tolerating a legacy bare
payload during mixed-version deploys.

#### AC-003: Envelope-first, legacy-tolerant reads

- GIVEN mocked responses in both new and legacy shapes
- WHEN each client resolves success and rejects on error
- THEN enveloped `data` is returned for new shapes, legacy bodies still
  resolve, and error messages come from `error.message` (new) or `error`
  (legacy).

### REQ-003: Protocol endpoints keep their wire format

`/healthz`, the ECPay `1|OK` acknowledgement and browser-return redirect,
and CORS `OPTIONS` MUST NOT be wrapped in the JSON envelope.

#### AC-004: Exempt endpoints are unchanged

- GIVEN the exempt endpoints
- WHEN they respond
- THEN their bodies/content types match the pre-change formats byte-for-byte.

### REQ-004: Provider paths carry executable contract evidence

The Resend sender MUST have a test that drives the real SDK against a local
HTTP server and asserts the request contract (POST `/emails`, bearer auth,
from/to/subject/html/text fields) plus error surfacing on a non-2xx provider
response. The R2 store MUST have a test that proves `PresignPut` produces a
correctly signed Cloudflare R2 URL offline (host, bucket/key path, signature
parameters, content-type header) without network access.

#### AC-005: Resend contract test executes

- GIVEN a `ResendSender` whose client base URL points at a test server
- WHEN `Send` runs against a 200 and a 422 stub
- THEN the test server observes the exact request contract and the 422
  surfaces as an error.

#### AC-006: R2 presign contract test executes offline

- GIVEN `NewR2` built with dummy credentials
- WHEN `PresignPut` is called
- THEN the returned URL carries the R2 endpoint host, the bucket/key path,
  SigV4 parameters, and the content-type header — all without a network
  call.

## Amendments

None.
