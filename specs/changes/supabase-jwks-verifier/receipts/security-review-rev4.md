# Security Review Receipt

- Change ID: `supabase-jwks-verifier`
- Revision: `4`
- Reviewer: Codex, independent of the implementation agent
- Review state: repository implementation accepted for verification; live-project evidence remains blocked

## Reviewed boundary

- The verifier derives one discovery URL from validated `SUPABASE_URL`; JWT
  headers and claims do not choose a network destination.
- Only ES256/P-256 and RS256/RSA-2048-or-stronger public keys are admitted.
  `HS256`, `none`, unsupported algorithms, empty `kid`, wrong key/algorithm,
  invalid signatures, and invalid required claims return `ErrUnauthorized`.
- Issuer is exact, audience contains `authenticated`, `exp` is required, `nbf`
  uses the fixed skew, and only verified `sub` plus string `email` enter the
  capability-empty initial principal.
- The HTTP client has a five-second timeout, rejects redirects, and bounds the
  token, response body, and JWKS key count. A failed refresh does not publish
  a partial set or use an expired key.
- A fresh matching key now returns while an unknown-key refresh is in flight;
  the in-flight generation channel makes concurrent unknown-key callers share
  that refresh. No negative-key cache was added because it is outside this
  controlled contract.
- `WriteError` retains the established 401/503 public bodies and applies the
  existing redaction path to unavailable errors. The verifier does not embed
  a token, claim, or provider response body in a public response.

## Independent replay

```text
go test -race ./server/internal/auth -run 'TestJWKSVerifier(ConcurrentUnknownKIDRefreshIsCoordinated|KnownKeyNotBlockedDuringRefresh|InvalidTokenDoesNotInvokeStaffResolver|WarmCacheAvoidsNetwork|DoesNotUseExpiredKeyAfterRefreshFailure)$' -count=20
PASS

go test -race ./server/internal/auth ./server/internal/config ./server/internal/bootstrap -count=1
PASS

go test ./server/... -count=1
PASS

go vet ./server/internal/auth ./server/internal/config ./server/internal/bootstrap
PASS

go mod verify
all modules verified
```

## Adversarial findings resolved before acceptance

1. The first implementation held the cache mutex across discovery I/O, so an
   unknown key could delay valid warm-cache verification. The accepted version
   fetches outside the mutex and has a barrier-based regression test.
2. The first concurrent-refresh test only exercised a cold cache and used a
   sleep. The accepted test primes a fresh generation, blocks a real refresh
   with channels, releases it deterministically, and verifies one post-prime
   fetch for ten callers.
3. Mixed-case production environments could enter the development URL branch.
   The accepted configuration uses one case-insensitive production predicate
   for dotenv, dev-auth, and JWKS HTTPS enforcement.
4. Invalid-token tests now traverse `Authenticator` with a counting staff
   lookup and prove zero resolver calls before the valid-token control.

## Remaining external boundary

This receipt does not establish that the target Supabase project issues
asymmetric tokens or that the operator accepts local-verification revocation
semantics. Those facts are intentionally reserved for S01 / AC-006.
