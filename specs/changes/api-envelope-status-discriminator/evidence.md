# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-09-16 inside a linked worktree
(baseline 418957a) because the primary working tree is held by
`deployment-dashboard-ledger`; the two `applies_to` sets are disjoint.
This change supersedes only the `ok` discriminator from
`api-response-envelope`; all other revision-1 outcomes remain accepted.
Gates: `speccheck` ok (41 specs), `SCOPE_CHANGE_ID=api-envelope-status-
discriminator scopecheck` ok (14 files), `verify` ok (archcheck,
migration-parity, go test green, commerce/staff/media -count=10, vet),
admin vitest 254/254, `vue-tsc --noEmit` clean in admin + both themes,
admin build succeeds. One mutation (statusLabel forced to error) observed
red for TestJSONSuccessEnvelope and green after restoration.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Envelope.OK was replaced by Status string derived via statusLabel (success below 400, error otherwise) so the discriminator can never disagree with transport status; Envelope gains Meta serialized as top-level meta only when set. httpx_test.go pins success status, non-2xx error status, the error body shape, absence of the ok key, and meta serialization. Mutation-verified: forcing statusLabel to always return error turned TestJSONSuccessEnvelope red; restored green. |
| REQ-002 | passed | All four client surfaces detect the envelope by status value, not key presence: api-client request() returns env.data on success and throws ApiError on error; media-api unwrap() returns data only when status is the literal success; curatory api() and minimal-cart unwrap() apply the same value check. Bare payloads carrying their own status field (e.g. pending) fall through to the legacy path unchanged, proven by new collision tests in api-client.test.ts and media-api.test.ts. |
| REQ-003 | passed | Envelope.Meta is declared with omitempty and documented as the reserved sibling of data for the offset-pagination contract {page, per_page, total, total_pages}; no endpoint adopts pagination in this change, so Meta is only ever written through the in-package writeEnvelope path pinned by TestMetaSerialization. |
| AC-001 | passed | TestJSONSuccessEnvelope decodes {status:success,data:{order:{id:TW-1}}} and asserts no ok key; TestJSONNon2xxKeepsErrorStatus proves a 409 JSON write emits error status with the payload under data; TestErrorEnvelope asserts {status:error,error:{code:forbidden,message:forbidden}} at 403 with no ok key; TestErrorCodeMapping still pins the fifteen-entry status map plus the 599 fallback. |
| AC-002 | passed | api-client.test.ts: the legacy {error:forbidden action} mock still rejects via ApiError, the envelope success test resolves {status:success,data:{order}} to the inner payload, the envelope error test asserts ApiError(status=409, code=conflict, message=stale version), and the new collision test proves a bare {status:pending} payload resolves unchanged. media-api.test.ts mirrors this for presign/verify including its own {status:processing} passthrough. 254 vitest tests green; vue-tsc clean in admin and both themes; admin build succeeds. |
| AC-003 | passed | TestMetaSerialization writes an envelope with Meta via writeEnvelope and asserts top-level meta {page:1,per_page:20,total:137,total_pages:7} appears beside data, then writes a normal JSON response and asserts meta is absent. |
