# Supabase Google and LINE OAuth runbook

Application code only starts OAuth. Supabase verifies the provider and
issues the session. Go authorizes `/api/admin/me` and member routes from
the Supabase access token. Provider client secrets stay in Google, LINE,
and the Supabase dashboard — never in `.env`, Vite define, the browser
bundle, Go config, the database, or logs.

Browser flags are fail-closed:

```
AUTH_GOOGLE_ENABLED=false
AUTH_LINE_ENABLED=false
```

Only `1`, `true`, `yes`, or `on` enable a button. Empty or any other
value keeps the provider hidden. Public login still requires
`AUTH_MODE=supabase` plus `SUPABASE_URL` and `SUPABASE_PUBLISHABLE_KEY`.

Redirects use the current application origin only (`window.location.origin`).
Do not add a path, query, or hash. Register those exact origins in
Supabase Auth URL configuration (for example `http://localhost:4173`,
`http://localhost:5174`, and the production site/admin origins).

Email/password remains available whenever Supabase auth is enabled.

## Google (built-in Supabase provider)

1. Google Cloud Console: create an OAuth client (Web application).
2. Authorized redirect URI must be the **Supabase callback**, not the
   storefront:
   `https://<project-ref>.supabase.co/auth/v1/callback`
3. Copy the Google client ID and secret into Supabase Dashboard →
   Authentication → Providers → Google. Enable the provider there.
4. Add the public and admin application origins to Supabase Redirect URLs.
5. Set `AUTH_GOOGLE_ENABLED=true` in the operator environment (not a
   committed secret). Rebuild the admin app and the theme islands.

The browser calls `signInWithOAuth({ provider: 'google' })`.

## LINE (`custom:line` OIDC)

Do **not** set `AUTH_LINE_ENABLED=true` until the LINE channel has
**email permission**. Without it, LINE will not return an email and
member/staff identity is incomplete.

1. LINE Developers Console: create a LINE Login channel.
2. Request and wait for the email-address permission (`email` scope).
3. Callback / redirect URI is again the **Supabase** callback:
   `https://<project-ref>.supabase.co/auth/v1/callback`
4. In Supabase Dashboard → Authentication → Providers, add a **Custom
   OIDC** provider:
   - Identifier / slug: `line` so the browser provider is `custom:line`
   - Issuer: `https://access.line.me`
   - Client ID / secret: from the LINE channel (dashboard only)
   - Scopes: `openid profile email`
   - PKCE: enabled, method **S256**
5. Add the same application origins to Supabase Redirect URLs.
6. Only then set `AUTH_LINE_ENABLED=true` and rebuild.

The browser calls `signInWithOAuth({ provider: 'custom:line' })` and
requests `openid profile email`. Application code does not store LINE
access or refresh tokens.

## After the callback

Supabase returns the user to the application origin. The existing
session adapter reads the Supabase session (`detectSessionInUrl`) and
sends the Supabase access token to Go as `Authorization: Bearer`.

- Public member routes succeed for that principal.
- Admin still requires an active linked staff row from `/api/admin/me`.
  An OAuth identity with no staff row stays forbidden.
- Cancel or provider error returns the user to the signed-out form.

## Token verification and session revocation

Go verifies Supabase access tokens according to `SUPABASE_VERIFIER_MODE`:

- `remote` (default / rollback): Go calls `/auth/v1/user` on every protected request.
  Server-side sign-out, session deletion, and user bans take effect immediately.
- `jwks`: Go verifies asymmetric `ES256` / `RS256` tokens locally using a cached key
  set from `<SUPABASE_URL>/auth/v1/.well-known/jwks.json`. This eliminates the per-request
  network round-trip. However, because verification occurs locally, server-side
  sign-out or session termination is not observed until the access token expires.
  To restore immediate revocation checks, set `SUPABASE_VERIFIER_MODE=remote` and redeploy.

## Secret-negative check

The Vite define surface may include only:

- `AUTH_MODE`
- `SUPABASE_URL`
- `SUPABASE_PUBLISHABLE_KEY`
- `AUTH_GOOGLE_ENABLED`
- `AUTH_LINE_ENABLED`

Never define `DEV_AUTH_TOKEN`, service-role keys, Google client secrets,
or LINE channel secrets into the browser.
