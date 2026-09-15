---
name: site-deployment-browser
description: Run this starter's provider-dashboard deployment setup or audit with user login handoff, safe screenshots, and environment-variable mapping for Supabase, Cloudflare, Railway, Resend, and optional ECPay.
---

# Project deployment browser

Use this skill only for this repository's deployment path:

- public static site on Cloudflare Pages from `dist/`;
- admin Vue SPA on a second Cloudflare Pages project from `admin/dist`;
- Go API on Railway;
- Supabase PostgreSQL and Auth;
- Cloudflare R2 for media;
- Resend for transactional email;
- optional ECPay stage/production payment setup.

Read `docs/deployment-guide.html` and `docs/environment-configuration.md`
before opening provider dashboards. Inspect current repo configuration when the
guide and implementation could have drifted.

## Current deployment target

For the active deployment walkthrough, preserve these real identifiers in
notes and the guide: Supabase organization `ailazytech` (ref
`ntrysovsjzjiqfbobdai`), project `agent-safe-site-starter` (ref
`wcjqrwchajqdyarhsriw`), project URL
`https://wcjqrwchajqdyarhsriw.supabase.co`, and region `Southeast Asia
(Singapore)` (`ap-southeast-1`). Do not replace these identifiers with
anonymized placeholders. Keep passwords, connection strings, API tokens,
deploy-hook URLs, and other credentials out of the skill, guide, screenshots,
chat, and reports; record only their variable names, source locations, and
destinations.

## Live setup ledger

The following non-secret identifiers and checkpoints were observed during the
2026-09-15 deployment walkthrough. Keep these values intact when continuing
this project:

- Supabase Auth user: `ailazytech@gmail.com`, UID
  `68a03370-56ee-413a-8097-950e629ee46a`; the user list confirms the account
  exists and the project has automatic RLS enabled, Data API disabled, and
  automatic exposure of new tables disabled.
- Cloudflare account ID: `304aea33c544c2730e9fcd141dba420a`; R2 bucket
  `agent-safe-site-starter-assets` exists in APAC with public access disabled
  and no custom domain configured.
- Railway workspace: `ailazytech's Projects`; project `zealous-connection`
  (ID `359501a3-fd3e-4669-ab1e-547df21a5b56`), production environment
  (ID `6fbe5189-1322-4075-a1c0-ed874b8857b3`), service
  `agent-safe-site-starter` (ID `30a26c3a-4160-4dae-99d8-d8da64c2acea`).
  The first GitHub deployment is `Deployment successful`; the service is
  still unexposed, uses US West, and has one replica.
- Safe screenshots were captured for the Railway successful deployment and
  the Supabase Auth user list. Screenshots remain session evidence and are not
  persisted in the repository.

## Browser rule

Use a controllable browser session that is isolated from personal browsing as
far as the available browser backend supports. Prefer, in order:

1. A task-named browser session that is both controllable and visible.
2. A user-opened clean Chrome profile/tab explicitly handed to the agent.
3. A controllable agent tab marked for handoff when the backend cannot show it
   on the desktop.

Do not inspect unrelated user tabs. If the only controllable browser exposes
personal tabs, stop before screenshots or authenticated work and ask the user
to hand over a clean provider tab. Login, MFA, CAPTCHA, password-manager use,
and password entry are user-only steps.

## Safe operating loop

For each provider:

1. Open the provider dashboard or claim the clean tab the user provided.
2. Hand off for sign-in/MFA/CAPTCHA. Resume only after the user says the
   provider is authenticated.
3. Navigate to the exact settings page named in the guide.
4. Retrieve required values into short-lived task memory only. Classify each
   as public configuration, server-only secret, or one-time secret.
5. Ask action-time confirmation before creating accounts/projects/resources,
   creating keys, saving environment variables, triggering deploys, changing
   DNS, changing payment settings, uploading files, or sending test email.
6. Capture screenshots only on pages where secrets are hidden or absent.
7. Verify a non-secret status page or endpoint, then clear any secret from
   task memory.

Never print, persist, screenshot, commit, or report secret values. Report
variable names, destinations, and verification results only.

Production values belong in provider dashboards. Do not create or populate
`.env.production`. Local `.env.development.local` is in scope only when the
user explicitly asks for local development values.

## Provider map

| Provider | Open | Retrieve / configure | Destination |
|---|---|---|---|
| Supabase | `https://supabase.com/dashboard` | Project region, transaction-pooler `DATABASE_URL`, `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY`, admin user, Data API/RLS settings | Railway; Pages build variables where the guide says so |
| Cloudflare R2 | `https://dash.cloudflare.com` -> R2 | Bucket, `R2_ACCOUNT_ID`, scoped Object Read/Write token pair, `R2_PUBLIC_BASE_URL` | Railway; Pages build variable for public base URL |
| Resend | `https://resend.com` | Verified domain, `RESEND_API_KEY`, `RESEND_FROM`, `CONTACT_NOTIFY_TO` | Railway |
| Railway | `https://railway.app` | Go service deploy, public Railway origin, production variables, `/healthz` status | Railway service variables |
| Cloudflare Pages | `https://dash.cloudflare.com` -> Workers & Pages | Storefront project, admin project, build commands, `Settings -> Variables and Secrets` runtime Function variables `API_ORIGIN` and `EDGE_SECRET`, and `Settings -> Builds -> Add deploy hook` | Cloudflare Pages projects; deploy hook URL returns to Railway |
| DNS / edge proof | Cloudflare DNS and public site URL | Custom domains, `/api/products` via Pages, direct Railway origin rejection after `EDGE_SECRET` | Cloudflare DNS and verification report |
| ECPay optional | Stage first: `https://vendor-stage.ecpay.com.tw` | `MerchantID`, `HashKey`, `HashIV`, callback settings, one stage transaction | Railway variables and ECPay dashboard |

For Supabase transaction pooler URLs, append
`default_query_exec_mode=simple_protocol` as documented. For `EDGE_SECRET`
rotation, use the documented overlap sequence with `EDGE_SECRET_PREVIOUS` on
Railway before switching both Pages projects.

Cloudflare's Pages build container starts clean. For the curatory storefront
use `npm --prefix site/themes/curatory ci && make site` with output directory
`dist`; `make site` alone fails before `vite` is installed. For the admin
project use `npm --prefix admin ci && npm --prefix admin run build:only` with
output directory `admin/dist`.

## Completion report

Return a concise non-secret report:

- provider checkpoints reached;
- screenshots captured or skipped, with the reason;
- variable names configured and their destination only;
- endpoint/build/deploy verification results;
- remaining user handoffs such as login, DNS propagation, production deploy,
  or optional ECPay stage payment.
