# Evidence

## Delivery Status

Revision 4 is Verifying. Packets S02-S05 were implemented in a temporary
linked worktree and independently replayed; S01 remains environment-blocked.
No live Supabase signing/session claim is inferred from local fixtures.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Revision 4 configuration and bootstrap selection replayed; see `receipts/consumer-reachability-rev4.md`. |
| REQ-002 | passed | Revision 4 local verification, resolver boundary, and handler reachability replayed; see `receipts/consumer-reachability-rev4.md` and `receipts/security-review-rev4.md`. |
| REQ-003 | passed | Revision 4 bounded refresh, cache, and failure separation replayed; see `receipts/security-review-rev4.md`. |
| REQ-004 | blocked | Requires non-secret signing, issuer, audience, access-token-lifetime, protected-endpoint, outage, and explicit rollback observations from the actual Supabase deployment environment. |
| AC-001 | passed | Revision 4 mode/default/selection trace replayed; see `receipts/consumer-reachability-rev4.md`. |
| AC-002 | passed | Revision 4 trust-input, algorithm, claim, and URL-boundary review replayed; see `receipts/security-review-rev4.md`. |
| AC-003 | passed | Revision 4 warm-cache and resolver/consumer trace replayed; see `receipts/consumer-reachability-rev4.md` and `receipts/security-review-rev4.md`. |
| AC-004 | passed | Revision 4 invalid-token, 401, no-resolver, and no-fallback review replayed; see `receipts/security-review-rev4.md`. |
| AC-005 | passed | Revision 4 refresh coordination, cache bound, timeout, and failure-classification review replayed; see `receipts/security-review-rev4.md`. |
| AC-006 | blocked | Requires non-secret signing, issuer, audience, access-token-lifetime, protected-endpoint, outage, and explicit rollback observations from the actual Supabase deployment environment. |

## Replay Commands

- `go test -race ./server/internal/auth -run 'TestJWKSVerifier(ConcurrentUnknownKIDRefreshIsCoordinated|KnownKeyNotBlockedDuringRefresh|InvalidTokenDoesNotInvokeStaffResolver|WarmCacheAvoidsNetwork|DoesNotUseExpiredKeyAfterRefreshFailure)$' -count=20` passed.
- `go test -race ./server/internal/auth ./server/internal/config ./server/internal/bootstrap -count=1` passed.
- `go test ./server/... -count=1`, `go vet ./server/internal/auth ./server/internal/config ./server/internal/bootstrap`, `go mod verify`, `git diff --check`, and selected `scopecheck` passed before the final evidence update.
