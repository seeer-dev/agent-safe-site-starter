# Evidence

## Delivery Status

Revision 1 was applied with plain apply authorization from repository owner. All implementation slices, tests, and required receipts are complete.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Staff lookup error separation implemented in auth.StaffCapabilityResolver and bootstrap.staffLookupAdapter. Valid unlinked user receives role=user and empty capabilities (TestStaffCapabilityResolverUnlinkedUser, TestStaffCapabilityResolverNoStaffRow). Staff lookup infrastructure errors return ErrUnavailable (TestStaffCapabilityResolverLookupInfrastructureFailure, TestStaffLookupAdapterPreservesInfrastructureError). |
| REQ-002 | passed | SupabaseVerifier classifies credential rejection (400/401/403) as ErrUnauthorized and provider failures (429/500/503/network/decode) as ErrUnavailable (TestSupabaseVerifierClassifications). Protected HTTP handlers use shared auth.WriteError mapping (TestCommerceHTTPAuthErrorSeparation, TestSiteContentHTTPUnavailableOnAuthFailure, TestListStaffHTTPReturns503OnDependencyUnavailable, TestVerifyHTTPAuthErrorSeparation). |
| REQ-003 | passed | Unlinked and disabled permission semantics preserved (TestStaffCapabilityResolverDisabledStaff, TestStaffCapabilityResolverUnlinkedUser). Admin auth store preserves session token and provider session on HTTP 503 while clearing server capabilities and entering retryable failed state (admin/src/stores/auth.test.ts). |
| REQ-004 | passed | All 41 protected handler entry points calling auth.Principal across 7 production files migrated to auth.WriteError. Non-disclosure verified in public response bodies and server diagnostic logs (TestWriteErrorNoSensitiveDataLeakage, receipts/security-review-rev1.md, receipts/consumer-reachability-rev1.md). |
| AC-001 | passed | Verified Supabase user without staff row receives UserID, role=user, empty StaffID, and 0 capabilities without email fallback. Proved by TestStaffCapabilityResolverUnlinkedUser, TestStaffCapabilityResolverEmailImpersonation, and TestStaffLookupAdapterTranslatesNotFound. |
| AC-002 | passed | Staff lookup failure for reasons other than ErrNotFound returns no principal and wraps ErrUnavailable while preserving the underlying cause. Proved by TestStaffCapabilityResolverLookupInfrastructureFailure and TestStaffLookupAdapterPreservesInfrastructureError. |
| AC-003 | passed | Missing/malformed bearer token or Supabase 400/401/403 returns HTTP 401 with generic error body. Proved by TestWriteErrorUnauthorized, TestSupabaseVerifierClassifications, and handler tests in commerce, content, media, sitecontent, staff. |
| AC-004 | passed | Supabase transport/5xx failures and staff lookup infrastructure errors return HTTP 503 via auth.WriteError, emit a redacted rune-bounded diagnostic, and return no protected data. Proved by TestWriteErrorUnavailable, TestWriteErrorNoSensitiveDataLeakage (rewritten to be non-vacuous), TestCommerceHTTPAuthErrorSeparation, TestSiteContentHTTPUnavailableOnAuthFailure, TestListStaffHTTPReturns503OnDependencyUnavailable, and TestVerifyHTTPAuthErrorSeparation. Recorded in receipts/consumer-reachability-rev1.md and receipts/security-review-rev1.md. |
| AC-005 | passed | Unlinked user retains role=user with 0 capabilities; disabled staff retains role=disabled with 0 capabilities. Proved by TestStaffCapabilityResolverUnlinkedUser, TestStaffCapabilityResolverDisabledStaff, and TestListStaffHTTPForbiddenWithoutReadCap. |
| AC-006 | passed | Admin store on /api/admin/me 503 enters failed status, clears server capabilities, retains token and provider session, and does not invoke signOut; 401 invalidates and signs out. Proved by admin/src/stores/auth.test.ts and receipts/consumer-reachability-rev1.md. |
| AC-007 | passed | All protected entry points in bootstrap, commerce, content, media, sitecontent, and staff use the shared auth.WriteError mapping; no Principal error path returns 401 any longer. Reviewer found the original non-disclosure test vacuous (it passed a bare sentinel) and the sanitizeDiagnostic helper performing no redaction, demonstrated the leak, and remediated both: the test now places secrets inside the error chain and asserts both redaction and retained context, and sanitizeDiagnostic redacts URL, JWT, and email shapes with rune-bounded truncation. Recorded in receipts/security-review-rev1.md including the reviewer addendum. |

## Verification Command Records

1. **Go Unit and Integration Tests**:
   - `go test ./server/internal/auth ./server/internal/bootstrap -count=1` -> PASS
   - `go test ./server/internal/modules/commerce ./server/internal/modules/content ./server/internal/modules/media ./server/internal/modules/sitecontent ./server/internal/modules/staff -count=1` -> PASS

2. **Admin Store and Typecheck Tests**:
   - `npm --prefix admin test -- --run src/stores/auth.test.ts` -> PASS (15 tests)
   - `npm --prefix admin run typecheck` -> PASS

3. **Governance Gates**:
   - `go run ./server/tools/archcheck` -> PASS
   - `go run ./server/tools/speccheck` -> PASS
   - `go run ./server/tools/scopecheck` -> PASS
   - `go run ./server/tools/verify` -> PASS
