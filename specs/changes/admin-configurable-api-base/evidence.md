# Evidence

## Delivery status

This controlled change is **Accepted** with full passing evidence and receipts across all requirements and acceptance criteria.

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | api-config.ts implements resolveApiBase and getApiBase with default fallback to /api; verified with 30 unit tests in api-config.test.ts. |
| REQ-002 | passed | Security grammar enforced in resolveApiBase rejecting bare origins, non-loopback HTTP, non-/api relative paths, userinfo, query, fragment, and standalone dot-segments; verified in api-config.test.ts and receipts/security-review.md. |
| REQ-003 | passed | api-client.ts and media-api.ts adopt getApiBase(); dual-mode typecheck and Vite build pass cleanly with unset config and with custom ADMIN_API_BASE. |
| AC-001 | passed | Unit tests in api-config.test.ts verify resolveApiBase returns /api when input is undefined, null, empty string, or whitespace. |
| AC-002 | passed | Unit tests in api-config.test.ts verify resolveApiBase resolves valid remote HTTPS prefixes and loopback HTTP prefixes ending in /api. |
| AC-003 | passed | Unit tests in api-config.test.ts verify trailing slashes are trimmed on valid relative /api and absolute prefixes ending in /api. |
| AC-004 | passed | Unit tests in api-config.test.ts verify rejection of bare origins, non-/api relative paths, remote HTTP, non-canonical loopbacks, protocol-relative URLs, userinfo, query, fragment, and standalone dot-segments with silent fallback to /api; recorded in receipts/security-review.md. |
| AC-005 | passed | Mocked-fetch tests in api-client.test.ts and media-api.test.ts verify that api methods and media upload/verify endpoints dispatch to getApiBase() preserving paths; recorded in receipts/consumer-reachability.md. |
| AC-006 | passed | vue-tsc --noEmit and vite build passed with 0 errors both with unset ADMIN_API_BASE and with ADMIN_API_BASE=https://api.example.com/api. |
