# Consumer Reachability Receipt

- Change ID: `auth-error-separation`
- Revision: `1`
- Generated: `2026-08-14`

## Summary

This receipt records consumer reachability verification across all protected HTTP entry points and the admin auth store.

## Protected Consumer Inventory

All 41 protected handler entry points calling `auth.Principal` across 7 production files have been migrated to the shared `auth.WriteError` handler:

1. `server/internal/bootstrap/app.go` (`GET /api/admin/me`)
2. `server/internal/modules/commerce/http.go` (26 handlers: orders, products, members, promos, payment/shipping methods)
3. `server/internal/modules/content/http.go` (`POST /api/admin/articles`)
4. `server/internal/modules/media/http.go` (`POST /api/media/verify`)
5. `server/internal/modules/media/media.go` (`POST /api/media/presign`)
6. `server/internal/modules/sitecontent/http.go` (6 handlers: ListAll, Create, Update, Delete, Publish, Approve)
7. `server/internal/modules/staff/http.go` (5 handlers: List, Create, Update, Delete, UpdateStatus)

## Admin Store Consumer Verification

The admin auth store (`admin/src/stores/auth.ts`) was verified against `/api/admin/me` error conditions:

1. **HTTP 401 Unauthorized**:
   - `auth.ts` invokes `invalidateUnauthorizedSession`.
   - Clears access token, clears capabilities, sets status to `unverified`.
   - Calls `signOut` on the session adapter.
   - Tested by `admin/src/stores/auth.test.ts: it('signs out the real session on /admin/me 401 and ignores the stale token')`.

2. **HTTP 503 Service Unavailable**:
   - `auth.ts` catches `ApiError` with status 503.
   - Sets `status` to `failed`.
   - Clears `serverCaps` (`[]`).
   - Retains `token` and `getAccessToken()` for retry.
   - Does **NOT** call `signOut` on the session adapter.
   - Sets safe generic `verifyError` (`目前無法驗證身分，請稍後再試。`).
   - Tested by `admin/src/stores/auth.test.ts: it('enters failed state and preserves session when /admin/me returns 503')`.

## Go Test Execution Record

```text
go test ./server/internal/auth ./server/internal/bootstrap ./server/internal/modules/commerce ./server/internal/modules/content ./server/internal/modules/media ./server/internal/modules/sitecontent ./server/internal/modules/staff -count=1
PASS
ok  	github.com/example/ai-site-starter/server/internal/auth	0.068s
ok  	github.com/example/ai-site-starter/server/internal/bootstrap	0.020s
ok  	github.com/example/ai-site-starter/server/internal/modules/commerce	2.195s
ok  	github.com/example/ai-site-starter/server/internal/modules/content	0.019s
ok  	github.com/example/ai-site-starter/server/internal/modules/media	0.159s
ok  	github.com/example/ai-site-starter/server/internal/modules/sitecontent	0.706s
ok  	github.com/example/ai-site-starter/server/internal/modules/staff	0.259s
```

## Admin Test Execution Record

```text
npm test --prefix admin -- --run src/stores/auth.test.ts
✓ src/stores/auth.test.ts (15 tests) 304ms
Test Files  1 passed (1)
Tests  15 passed (15)
```
