# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-09-16 inside a linked worktree
(baseline 1665e08) because the primary working tree is held by
`deployment-dashboard-ledger`; the two `applies_to` sets are disjoint.
Gates: `speccheck` ok (40 specs), `SCOPE_CHANGE_ID=api-response-envelope
scopecheck` ok (21 files), `verify` ok (archcheck, migration-parity, go test
green, commerce/staff/media -count=10, vet), admin vitest 252/252,
`vue-tsc --noEmit` clean in admin + both themes, admin build succeeds.
Three controls were mutation-verified (httpx data drop, resend To
substitution, R2 BaseEndpoint substitution) — each observed red for the
named trigger and green after restoration.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | httpx.JSON now emits {ok:status<400,data:value} and httpx.Error/ErrorWithCause emit {ok:false,error:{code:codeForStatus(status),message}} through a shared writeEnvelope; every handler (~280 call sites), auth.WriteError, and the edge guard produce the shape without per-call edits. httpx_test.go pins both shapes, the non-2xx ok:false guard, the 15-entry status-to-code map plus the 'error' fallback, and cause non-disclosure. Mutation-verified: dropping Data from JSON turned TestJSONSuccessEnvelope and TestJSONNon2xxKeepsOkFalse red; restored green. |
| REQ-002 | passed | admin api-client request() unwraps env.data and throws ApiError(status, error.message, error.code) — the exported ApiResponse<T> interface now describes the wire envelope; media-api presign/verify unwrap via shared unwrap/readError helpers; curatory api() unwraps and reads errorMessage() across envelope+legacy shapes; minimal-cart api.ts gained the same unwrap/readErrorMessage helpers applied to all twelve call sites. All four tolerate legacy bare bodies and legacy string errors for mixed-version deploys. |
| REQ-003 | passed | The exempt paths never touch httpx.JSON: /healthz writes plain text at bootstrap/app.go:112, the ECPay acknowledgement body writes text at ecpay_http.go:59-61, and CORS OPTIONS writes a bare 204 at app.go:275. The selected diff contains no app.go or ecpay_http.go changes, so their wire formats are byte-identical. |
| REQ-004 | passed | mail/resend_test.go (new) drives the real resend-go SDK against an httptest server via the client's exported BaseURL: TestResendSendRequestContract asserts POST /emails, Bearer auth, and from/to/subject/html/text fields; TestResendSendProviderError proves a provider 422 surfaces as an error. storage/r2_test.go gained TestPresignPutProducesSignedR2URL proving offline SigV4 presigning: virtual-hosted URL media-bucket.acct123.r2.cloudflarestorage.com, object key path, AWS4-HMAC-SHA256 algorithm, signature present, credential scope, X-Amz-Expires=600, and the Content-Type header. Mutations verified red (wrong To, wrong BaseEndpoint) then restored green. |
| AC-001 | passed | TestJSONSuccessEnvelope decodes {ok:true,data:{order:{id:'TW-1'}}} with no error key; TestJSONNon2xxKeepsOkFalse proves a 409 JSON write emits ok:false with the payload under data; TestErrorEnvelope asserts {ok:false,error:{code:'forbidden',message:'forbidden'}} at status 403; TestErrorCodeMapping pins all fifteen status codes plus the 599 fallback. |
| AC-002 | passed | Public messages are unchanged end-to-end: TestWriteErrorUnauthorized/Unavailable assert error.message equals 'unauthorized'/'service unavailable' plus ok:false; the media verify tests assert the fixed sentinels and the generic 503 message land in error.message; the stale-version commerce test still finds 'stale version' in the body. |
| AC-003 | passed | api-client.test.ts: legacy {error:'forbidden action'} mock still rejects via ApiError (existing test), the new 'unwraps the shared response envelope' test resolves {ok:true,data:{order}} to the inner payload, and 'surfaces envelope error code and message' asserts ApiError(status=409, code='conflict', message='stale version'). media-api.test.ts adds envelope unwrap for presign and error.message extraction for verify; existing legacy mocks keep passing. 252 vitest tests green; vue-tsc clean in admin and both themes; admin build succeeds. |
| AC-004 | passed | app.go and ecpay_http.go are absent from the selected diff; /healthz remains a plain-text 200, the ECPay callback returns its text/plain ack body, and OPTIONS returns an empty 204 — verified by source inspection at app.go:112/275 and ecpay_http.go:59-61, all outside httpx's write path. |
| AC-005 | passed | go test ./server/internal/platform/mail -v ran TestResendSendRequestContract and TestResendSendProviderError: the httptest server observed POST /emails with Authorization 'Bearer re_test_key' and the full field set; the 422 stub produced a wrapped 'resend send' error. Mutation: sending To=[wrong@example.com] turned the contract test red; restored and re-passed. |
| AC-006 | passed | go test ./server/internal/platform/storage -run TestPresignPutProducesSignedR2URL passed with dummy credentials and zero network: the presigned URL parses as media-bucket.acct123.r2.cloudflarestorage.com/uploads/u1/temp.webp with X-Amz-Algorithm=AWS4-HMAC-SHA256, non-empty X-Amz-Signature, X-Amz-Credential scoped to fake-access-key, X-Amz-Expires=600, and Headers Content-Type image/webp. Mutation: pointing BaseEndpoint at example.com turned the host assertion red; restored and re-passed. |
