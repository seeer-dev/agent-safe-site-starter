# API Envelope Status Discriminator Delivery Plan

Change ID: api-envelope-status-discriminator
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| Envelope ships `ok` boolean at the single write funnel | `httpx.Envelope.OK` written by `writeEnvelope` since `api-response-envelope` revision 1 | Swap the field at the funnel; ~280 call sites unchanged (REQ-001) |
| Envelope detection must not collide with payloads | Bare payloads may carry their own `status` field (e.g. `{status:"pending"}`) | Detection is value-based: only `status === "success"`/`"error"` is treated as an envelope (REQ-002, AC-002) |
| No HTTP-layer pagination exists today | List endpoints filter via query params but return full result sets | `meta` is a reserved contract, not a retrofit — first paginated endpoint fills `Envelope.Meta` (REQ-003) |
| Revision-1 `ok` shape never deployed | Superseding baseline is a local commit on `staging` | No `ok`/`status` mixed window; clients tolerate only pre-envelope bare bodies |
| Worktree mode required | Primary tree is dirty under `deployment-dashboard-ledger` | All work happens in this linked worktree; scope via `SCOPE_CHANGE_ID` |

## Scope lock

- `server/internal/httpx/httpx.go`
- `server/internal/httpx/httpx_test.go`
- `server/internal/auth/auth_test.go`
- `server/internal/modules/commerce/http_test.go`
- `admin/src/lib/api-client.ts`
- `admin/src/lib/api-client.test.ts`
- `admin/src/lib/media-api.ts`
- `admin/src/lib/media-api.test.ts`
- `site/themes/curatory/shared/lib/api.ts`
- `site/themes/minimal-cart/shared/lib/api.ts`
- `specs/changes/api-envelope-status-discriminator/**`

## Dependency-ordered slices

### Slice 1: Status discriminator + meta field at the funnel (REQ-001, REQ-003, AC-001, AC-003)

Outcome: `httpx.JSON` emits `{status:"success", data}` /
`{status:"error", data}` for non-2xx; `httpx.Error`/`ErrorWithCause` emit
`{status:"error", error:{code,message}}`; `Envelope.Meta` serializes as
top-level `meta` only when set. `httpx_test.go` pins the discriminator, the
`meta` contract, and the absence of `ok`.

Edits: `httpx.go`, `httpx_test.go`.

Acceptance evidence: httpx tests; mutation — wrong discriminator literal →
red.

### Slice 2: Server test migration (REQ-001, AC-001)

Outcome: Go tests that decode the envelope assert `body["status"]` /
`env.Status`; error assertions keep their expected messages and codes.

Edits: `auth_test.go`, `commerce/http_test.go`.

Acceptance evidence: `go test ./server/...` green.

### Slice 3: Client surfaces (REQ-002, AC-002)

Outcome: admin `request()` unwraps on `status === "success"` and throws
`ApiError` on `"error"`; `media-api`, curatory `api()`, minimal-cart helpers
use the same value-based detection; bare payloads with their own `status`
field pass through; legacy bodies still tolerated.

Edits: the four client files plus their test mocks.

Acceptance evidence: `npx vitest run`, `npm run typecheck`, `npm run build`.

### Slice 4: Gates + close-out

Outcome: `speccheck`, `SCOPE_CHANGE_ID=api-envelope-status-discriminator
scopecheck`, `verify` all green inside the worktree; evidence and control
statuses filled; commit on the worktree branch; cherry-pick to primary.

## Risks

- None beyond revision 1: the discriminator rename is a pure wire-shape
  change inside the funnel; payloads, codes, and messages are untouched.
