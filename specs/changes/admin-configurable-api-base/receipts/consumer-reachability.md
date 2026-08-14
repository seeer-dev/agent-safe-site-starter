# Consumer Reachability Receipt: Admin Configurable API Base

- **Change ID**: `admin-configurable-api-base`
- **Revision**: 2
- **Date**: 2026-08-14
- **Auditor**: Antigravity Implementation Agent

## 1. Consumer Integration Verification

All admin HTTP request dispatchers were audited and verified to consume the single centralized `getApiBase()` resolver from `admin/src/lib/api-config.ts`.

### 1.1 `admin/src/lib/api-client.ts`
- **Call Sites**:
  - `request<T>(path: string, options?: RequestInit)`
  - Target URL calculation: `const base = getApiBase(); const normalizedPath = path.startsWith('/') ? path : `/${path}`; const url = `${base}${normalizedPath}``
- **Verification**:
  - `admin/src/lib/api-client.test.ts` executes mocked-fetch tests for `api.get`, `api.post`, `api.put`, `api.patch`, `api.del`.
  - Proves default `/api` prefix resolution (`/api/admin/me`), custom HTTPS prefix resolution (`https://api.example.com/api/admin/products`), un-slashed path normalization (`admin/orders` -> `https://api.example.com/api/admin/orders`), bearer token injection, and error handling.

### 1.2 `admin/src/lib/media-api.ts`
- **Call Sites**:
  - `presignUpload(req: PresignRequest, signal?: AbortSignal)` -> fetches `${getApiBase()}/media/presign`
  - `verifyUpload(req: VerifyRequest, signal?: AbortSignal)` -> fetches `${getApiBase()}/media/verify`
  - `uploadToR2(url: string, ...)` -> directly PUTs to the presigned R2 storage URL without prefixing
- **Verification**:
  - `admin/src/lib/media-api.test.ts` executes mocked-fetch tests for `presignUpload` and `verifyUpload` with default `/api` and custom `https://api.example.com/api` prefix.
  - Verifies that both endpoints reach the configured prefix while `uploadToR2` directly uploads to the presigned URL.

## 2. Dual-Mode Build Reachability
- Executed `npm --prefix admin run typecheck`: Passed with 0 errors.
- Executed `npm --prefix admin run build` with `ADMIN_API_BASE` unset: Passed, emitted production bundle.
- Executed `npm --prefix admin run build` with `ADMIN_API_BASE=https://api.example.com/api`: Passed, emitted production bundle with inlined prefix.

## 3. Conclusion
Both admin consumers (`api-client.ts` and `media-api.ts`) share the single authoritative resolver, preserving complete endpoint paths and local proxy fallback.
