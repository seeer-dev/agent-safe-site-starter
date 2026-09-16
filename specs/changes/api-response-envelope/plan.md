# API Response Envelope Delivery Plan

Change ID: api-response-envelope
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| Single write funnel already exists | `httpx.JSON`/`httpx.Error`/`httpx.ErrorWithCause` are the only response writers; `auth.WriteError` and the edge guard delegate to `httpx.Error` | Wrapping inside `httpx` converts ~280 call sites with no handler edits (REQ-001) |
| Success payloads are heterogeneous | Handlers emit `{order:…}`, `{products:…}`, `{items:…}`, bare objects | `data` keeps each payload verbatim; only the outer shape unifies (REQ-001) |
| Errors are uniform strings | `httpx.Error` writes `{"error": msg}` | `error.{code,message}` preserves the message text (AC-002) |
| Admin client anticipated an envelope | `api-client.ts` declares `ApiResponse<T> { data, error? }` but returns raw bodies | Client unwrapping aligns code with its declared contract (REQ-002) |
| Mixed-version deploy windows are real | Pages and the Railway origin deploy independently | Clients must tolerate legacy bare bodies, not hard-require the envelope (AC-003) |
| Protocol endpoints are not JSON | `/healthz` text, ECPay `1|OK` ack, `OPTIONS` 204 | Exempt by design; never wrap (REQ-003, AC-004) |
| Resend sender has no executable evidence | `mail/` has no tests; SDK exposes `Client.BaseURL` | Point the real SDK at `httptest` to prove the request contract offline (REQ-004, AC-005) |
| R2 presign is offline signing | `s3.PresignClient` produces URLs without network | A dummy-credential test proves host/bucket/key/signature wiring (REQ-004, AC-006) |
| Worktree mode required | Primary tree is dirty under `deployment-dashboard-ledger` | All work happens in this linked worktree; scope via `SCOPE_CHANGE_ID` |

## Scope lock

- `server/internal/httpx/httpx.go`
- `server/internal/httpx/httpx_test.go` (new)
- `server/internal/platform/mail/resend_test.go` (new)
- `server/internal/platform/storage/r2_test.go`
- `admin/src/lib/api-client.ts`
- `admin/src/lib/api-client.test.ts`
- `admin/src/lib/media-api.ts`
- `admin/src/lib/media-api.test.ts`
- `site/themes/curatory/shared/lib/api.ts`
- `site/themes/minimal-cart/shared/lib/api.ts`
- `server/internal/auth/auth_test.go`
- `server/internal/modules/commerce/http_test.go`
- `server/internal/modules/commerce/model_test.go`
- `server/internal/modules/commerce/product_images_test.go`
- `server/internal/modules/commerce/service_security_test.go`
- `server/internal/modules/commerce/service_test.go`
- `server/internal/modules/commerce/shipping_methods_test.go`
- `server/internal/modules/media/verify_http_test.go`
- `server/internal/modules/sitecontent/http_test.go`
- `server/internal/modules/staff/http_test.go`
- `specs/changes/api-response-envelope/**`

## Dependency-ordered slices

### Slice 1: Envelope at the funnel (REQ-001, AC-001, AC-002)

Outcome: `httpx.JSON` emits `{ok, data}`; `httpx.Error`/`ErrorWithCause`
emit `{ok:false, error:{code,message}}`; `codeForStatus` maps the HTTP
status set. `httpx_test.go` pins both shapes, the status→code map, and the
non-2xx-JSON `ok:false` guard (AC-001). Error messages keep their existing
public text (AC-002).

Edits: `httpx.go`, new `httpx_test.go`.

Acceptance evidence: httpx tests; mutation check — revert `data` wrapping →
decode assertions red.

### Slice 2: Server test migration (REQ-001, AC-001)

Outcome: every Go test that decodes a response body reads `body["data"]` /
`body["error"].(map)["message"]`; error assertions keep their expected
messages.

Edits: the ten listed `server/internal/..._test.go` files.

Acceptance evidence: `go test ./server/...` green.

### Slice 3: Client surfaces (REQ-002, AC-003)

Outcome: admin `request()` returns `body.data` and throws
`ApiError(status, error.message)`; `media-api` unwraps presign/verify;
curatory `api()` unwraps; minimal-cart helpers unwrap. All four tolerate
legacy bare bodies.

Edits: the four client files plus their test mocks.

Acceptance evidence: `npx vitest run`, `npm run typecheck`, `npm run build`.

### Slice 4: Provider contract tests (REQ-004, AC-005, AC-006)

Outcome: `resend_test.go` drives the real SDK at an `httptest` server and
asserts the request contract plus 4xx surfacing (AC-005); `r2_test.go`
gains an offline `PresignPut` signing test (AC-006).

Acceptance evidence: `go test ./server/internal/platform/... -v` shows the
named tests pass; mutation — wrong auth-header expectation → red.

### Slice 5: Exempt-endpoint check + gates + close-out (REQ-003, AC-004)

Outcome: `/healthz`, ECPay `1|OK` ack, and `OPTIONS` verified unchanged;
`speccheck`, `SCOPE_CHANGE_ID=api-response-envelope scopecheck`, `verify`
all green inside the worktree; evidence and control statuses filled; commit
on the worktree branch.

## Risks

- Legacy-tolerant clients make rollback safe; removing tolerance later is a
  follow-up decision, not part of this change.
- Envelope wraps storefront public responses; the static render path does
  not consume them (renderer reads the DB), so `dist/` output is unchanged.
