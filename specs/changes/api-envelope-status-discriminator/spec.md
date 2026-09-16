# API Envelope Status Discriminator Specification

Change ID: api-envelope-status-discriminator
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner directed on 2026-09-16 that the `ok` boolean discriminator shipped by `api-response-envelope` be replaced by a conventional status discriminator per OpenAPI-era API conventions (JSend-style `status` + `data`/`error`), and that the pagination metadata contract be planned into the envelope.
Repository baseline: 418957ac65531ab80f1cb0728d07a37a562f97a4
Supersedes: api-response-envelope

## Outcome

Every JSON API response shares one envelope shape using a `status` string
discriminator:

```json
// success — any 2xx written via httpx.JSON
{ "status": "success", "data": <payload> }

// error — httpx.Error / httpx.ErrorWithCause / auth.WriteError / edge guard
{ "status": "error", "error": { "code": "<machine-code>", "message": "<human>" } }

// paginated success — reserved contract, adopted per endpoint as needed
{ "status": "success", "data": <payload>, "meta": { "page": 1, "per_page": 20, "total": 137, "total_pages": 7 } }
```

`status` is the discriminator: the string `"success"` when the HTTP status is
below 400, `"error"` otherwise. `data` preserves each endpoint's existing
payload; `error.code`/`error.message` keep the revision-1 error contract.
`meta` is a reserved top-level sibling of `data` for non-resource metadata;
offset pagination is the planned first occupant.

This change supersedes only the revision-1 discriminator choice (`ok`
boolean) and adds the `meta` contract. Every other `api-response-envelope`
outcome — `data` wrapping, the status-derived `error.code` map, exempt
protocol endpoints, provider contract tests, and legacy-tolerant clients —
remains accepted behavior and is not reimplemented here. The `ok` shape was
never deployed, so no mixed `ok`/`status` window exists; clients keep
tolerating pre-envelope bare payloads only.

## Scope

In scope:

- `server/internal/httpx/httpx.go`: `Envelope` gains `Status` (`"success"` /
  `"error"`) replacing `OK`, plus a reserved `Meta` field.
- `server/internal/httpx/httpx_test.go`: pin the `status` discriminator and
  the `meta` serialization contract.
- The three client surfaces: `admin/src/lib/api-client.ts`,
  `admin/src/lib/media-api.ts`, `site/themes/curatory/shared/lib/api.ts`,
  `site/themes/minimal-cart/shared/lib/api.ts` — value-based envelope
  detection.
- Server and client test files that assert the discriminator.

Out of scope:

- Re-opening any other revision-1 outcome (payload keys, error codes,
  message text, exempt endpoints, provider tests).
- Retrofitting pagination onto existing list endpoints — this change plans
  the `meta` contract; adoption happens when a paginated endpoint needs it.
- Documentation owned by the in-flight `deployment-dashboard-ledger` change.

## Decisions and invariants

- The discriminator follows the JSend-era convention the owner asked for:
  `status` is the string `"success"` or `"error"`. RFC 9457 problem details
  was evaluated and rejected — it would change the media type to
  `application/problem+json` and rename every field, conflicting with the
  requested status-then-data shape.
- HTTP status remains the transport-level truth; `status` mirrors
  `status < 400` so a stray non-2xx passed to `JSON` still produces
  `"error"` rather than a lying success envelope. The JSend three-state
  `fail`/`error` split is deliberately not adopted — the HTTP status already
  carries the 4xx/5xx distinction and clients only need a binary check.
- Envelope detection on the client is value-based, not key-presence-based:
  `status === "success"` unwraps `data`, `status === "error"` rejects, and
  any other `status` value (including a bare payload that happens to carry a
  `status` field, e.g. `{status: "pending"}`) falls through to the legacy
  bare-payload path untouched.
- Pagination contract: `meta` is a top-level sibling of `data`, omitted when
  absent. Paginated endpoints use offset pagination —
  `{"page", "per_page", "total", "total_pages"}` — matching the admin's
  page-numbered list UI; default `per_page` is 20 with a hard maximum of 100.
  No endpoint adopts it in this change.
- Clients tolerate mixed-version deploys: they prefer the envelope and fall
  back to legacy bare payloads / `{"error":"msg"}` while Pages and the origin
  roll independently.

## Requirements

### REQ-001: `status` discriminator at the single write funnel

`httpx.JSON` MUST emit `{"status":"success","data":<value>}` for success
statuses and `{"status":"error","data":<value>}` for any non-2xx it is asked
to write. `httpx.Error` and `httpx.ErrorWithCause` MUST emit
`{"status":"error","error":{"code":<status-derived>,"message":<public msg>}}`.
No response may carry an `ok` key.

#### AC-001: Success and error shapes carry the status discriminator

- GIVEN `httpx.JSON` called with 200 and 404 statuses and `httpx.Error`
  called with a 4xx status
- WHEN bodies are decoded
- THEN the success body is `{"status":"success","data":...}`, the non-2xx
  JSON body carries `"error"` status, and the error body carries
  `"error"` status plus `error.code` matching the status map and
  `error.message` equal to the public message — with no `ok` key present.

### REQ-002: All client surfaces detect the envelope by status value

The admin API client, the admin media client, and both storefront theme
clients MUST unwrap `data` on `"success"` and read `error.message` (and
`error.code` where surfaced) on `"error"`, while tolerating a legacy bare
payload during mixed-version deploys. Envelope detection MUST be value-based
(`status === "success"` / `status === "error"`) so a bare payload that
carries a `status` field of its own is never mistaken for an envelope.

#### AC-002: Value-based detection with legacy tolerance

- GIVEN mocked responses in the new and legacy shapes
- WHEN each client resolves success and rejects on error
- THEN enveloped `data` is returned for new shapes, legacy bodies still
  resolve, error messages come from `error.message` (new) or `error`
  (legacy), and a bare payload carrying a non-envelope `status` value is
  returned unchanged.

### REQ-003: Envelope reserves a top-level `meta` key for pagination

The `Envelope` type MUST carry a `Meta` field serialized as top-level `meta`
(omitted when nil). The pagination contract is offset-based:
`{"page", "per_page", "total", "total_pages"}` with default `per_page` 20 and
maximum 100. No endpoint adopts pagination in this change.

#### AC-003: `meta` serialization contract is pinned

- GIVEN an envelope written with and without `Meta`
- WHEN bodies are decoded
- THEN `meta` appears as a top-level sibling of `data` when set and is
  absent otherwise.

## Amendments

None.
