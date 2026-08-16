# Consumer Reachability Receipt

- Change ID: `supabase-jwks-verifier`
- Revision: `4`
- Reviewer: Codex

## Startup selection

`config.Load` normalizes the server-only `SUPABASE_VERIFIER_MODE`; in
`AUTH_MODE=supabase`, `bootstrap.newAuthVerifier` selects exactly one of the
existing remote verifier or `JWKSVerifier`. No handler chooses a verifier and
there is no request-time fallback from local verification to `/auth/v1/user`.

## Protected-consumer trace

`bootstrap.NewWithDB` builds one `auth.Authenticator`. Supabase mode attaches
the existing `StaffCapabilityResolver`, which receives the verified `sub`.
The same authenticator is passed to the protected bootstrap route and module
handlers found by the reachability scan:

- `GET /api/admin/me` in `server/internal/bootstrap/app.go`
- commerce protected handlers
- content publishing handler
- media presign and verification handlers
- site-content admin handlers
- staff admin handlers

No module constructs `JWKSVerifier`, reads token capability claims, or bypasses
the resolver boundary. The verifier returns role `user` with no staff ID or
capabilities; the resolver remains the authority for every staff role and
capability.

## Replay

```text
go test -race ./server/internal/auth ./server/internal/config ./server/internal/bootstrap -count=1
PASS

rg -n '\\.Principal\\(r\\)|Authenticator' server/internal --glob '*.go'
Reviewed bootstrap and protected module call paths listed above.
```
