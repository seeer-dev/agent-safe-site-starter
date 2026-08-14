# Security Review Receipt: Admin Configurable API Base

- **Change ID**: `admin-configurable-api-base`
- **Revision**: 2
- **Date**: 2026-08-14
- **Auditor**: Antigravity Implementation Agent

## 1. Threat Model & Asset Protection

- **Protected Assets**: Admin bearer authentication tokens (`Authorization: Bearer <token>`) and administrative endpoint calls (`/admin/*`, `/media/*`).
- **Transport Security**:
  - Admin requests carry bearer tokens. Unencrypted HTTP transport to remote hosts would risk credential leakage via network eavesdropping.
  - Policy: Remote absolute URLs MUST use `https://`. Unencrypted `http://` is strictly limited to exact canonical local development loopback hosts (`localhost`, `127.0.0.1`, `[::1]`).
  - Raw Host Enforcement: To prevent WHATWG URL parser normalization from widening loopback boundaries, raw HTTP strings are inspected before parsing to reject alternative numeric/octal/hex/alias spellings (e.g. `127.0.0.2`, `0177.0.0.1`, `0x7f000001`, `localtest.me`).
  - Unit tests in `admin/src/lib/api-config.test.ts` verify that remote HTTP and non-canonical loopback URLs are rejected and fall back to `/api`.

## 2. Input Validation & Injection Prevention

- **Forbidden Components**:
  - RelativePrefix Grammar: Strictly `/api` with optional trailing slashes (`/api`, `/api/`, `/api///`). All other root-relative paths (e.g. `/custom/api`, `/admin`, `/api/v1`) are rejected and fall back to `/api`.
  - Standalone Dot-Segments (`.`, `..`, `%2e`, `%2E`, `.%2e`, `%2e.`, `%2e%2e`): Segment-aware regex `/(?:^|\/)(?:\.|\.\.|%2e|%2e%2e|\.%2e|%2e\.)(?:\/|$)/i` rejects complete dot-segments to prevent path traversal, while allowing legitimate in-segment dots and encoded dots (e.g. `v1..beta`, `v1%2ebeta`, `v1.0.0`).
  - Protocol-relative URLs (`//example.com/api`): Rejected to prevent scheme confusion or host redirection.
  - Credentials / Userinfo (`https://user:pass@api.example.com/api`): Rejected via parser credentials validation (`parsed.username || parsed.password`), while allowing valid in-segment `@` path characters (e.g. `v1/@scope/api`, `/@scope/api`).
  - Query Strings (`?query=1`): Rejected to prevent unexpected parameter injection.
  - Fragments (`#hash`): Rejected to prevent client-side routing confusion.
  - Control Characters & Whitespace: Rejected to prevent request smuggling or header splitting.
  - Non-root relative paths (`api/test`): Rejected.
  - Absolute paths not ending in `/api`: Rejected (e.g. `https://api.example.com`, `https://api.example.com/v1/admin`, `https://api.example.com/v1/@api`).

## 3. Silent Safe Fallback & Information Leakage

- On any validation failure or malformed input, the resolver silently returns `/api`.
- The resolver NEVER logs raw untrusted input, user tokens, or malformed URL strings to the console or logs, preventing token/credential leakage in client error logs.

## 4. Conclusion

The security contract meets all requirements of Revision 2 and has been verified with 30 automated unit tests in `admin/src/lib/api-config.test.ts`.
