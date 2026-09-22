# Deployment Dashboard Ledger Plan

Change ID: deployment-dashboard-ledger
Revision: 14
Status: Applying

Normative specification: [`spec.md`](spec.md)

## Scope lock

- `skills/site-deployment-browser/SKILL.md`
- `skills/site-deployment-browser/references/deployment-value-refresh.md`
- `skills/site-deployment-browser/references/**`
- `admin/public/_redirects`
- `README.md`
- `docs/deployment-guide.html`
- `docs/environment-configuration.md`
- `specs/changes/deployment-dashboard-ledger/**`
- `.ai/scope.json`

## Slices

### Slice 1: Diagnose the branch mismatch

Verify the remote `main` and `staging` refs, current Pages production branch,
Railway GitHub integration, service-page failure, and current workspace/project/
environment/service identity without copying a secret or changing a provider
setting.

Covers REQ-002, AC-002.

### Slice 2: Update the live ledger and recovery instructions

Record actual Supabase, Railway, and Cloudflare Pages non-secret state,
branch/deployment status and results in the installation's HTML guide/ledger;
keep only reusable source/destination rules in the skill. Add the independent
branch-setting explanation, workspace hand-off guard, GitHub App access check,
confirmation-gated dashboard and CLI source-change paths, staged-change
lifecycle, exposure-aware verification, and separate Pages-origin check. Keep
all runtime values in provider dashboards.

Covers REQ-001, AC-001, REQ-002, AC-002.

### Slice 12: De-identify the agent handoff and separate test from release

Remove this installation's accounts, UIDs, project IDs, branch, domains,
emails, token names and dated receipts from the skill and reference. Keep the
current site-specific Pages test URLs and pending R2/Resend/security/source
boundaries in README, environment guide, HTML guide and evidence. Include the
later owner-chosen production-domain cutover without treating payment as part
of the no-payment acceptance. Validate skill and governed repository checks.

Covers REQ-001, AC-001, REQ-002, REQ-003, REQ-006, REQ-007.

### Slice 3: Security and verification

Check the updated files for known provider runtime values, run the controlled
spec gate, skill validation, and repository verification, and retain safe
dashboard/public-site screenshots as session evidence only. Record the
user-applied source change only through its non-secret deployment evidence;
do not turn that prior confirmation into authorization for a public-domain or
Pages-origin update.

Covers AC-001, AC-002.

### Slice 4: Repeatable value-refresh handoff

Record the active public Railway origin and health result, the staged
non-secret Railway change set, the two independent Pages Function destinations,
and the outstanding operator-only secret handoff. Add an exact operator sheet:
each row must state the dashboard source path, destination, text-or-secret
type, current state, and safe entry rule; distinguish the storefront's Go
renderer build secrets from the admin's Vue-only build; and state the explicit
stop-before-Deploy gate. Include deployment order, endpoint proof, failure
map, credential-rotation boundary, and no-payment admin-to-storefront smoke
boundary. For the current long-lived Railway Go service, record `Connect ->
Direct connection string -> Shared pooler -> Session mode -> URI` (dashboard:
`Session pooler`) as the `DATABASE_URL`
handoff; reserve direct connection for verified outbound IPv6 and Transaction
pooler for a future short-lived runtime. Do not represent the pre-handoff 500
responses as a completed proxy test.

Covers REQ-001, AC-001, REQ-003, AC-003.

### Slice 5: Admin SPA fallback repair

Reproduce the Cloudflare build warning, verify that neither the source nor the
build output contains a top-level 404 page, remove the invalid tracked
catch-all rewrite, and update the guide so it documents Pages' default SPA
fallback. Rebuild the admin project, redeploy it, and request a representative
deep link.

Covers REQ-004, AC-004.

### Slice 6: Encrypted-secret type recovery

Treat a Cloudflare row name as insufficient evidence for the database or edge
handoff. Add an explicit **Secret** / encrypted versus **Text** type check to
the skill, canonical operator sheet, and environment guide. When a credential
has been saved as text, require operator-led rotation, deletion, and encrypted
recreation without the agent copying the value between dashboards.

Covers REQ-001, AC-001, REQ-003, AC-003.

### Slice 7: Authorized deployment completion evidence

After the credential holder reports direct entry complete and the user
authorizes the deployment, apply the full Railway staged change set, retry each
Pages production deployment, and record only non-secret deployment identifiers
and HTTP statuses. Require the four endpoint proof: Railway health `200`,
uncredentialed direct API `403`, storefront Pages API `200`, and admin Pages
API `200`. Then, with the authorized exact-ID-linked owner, create a labelled
active smoke product, wait for storefront hydration rather than judging only
the static fallback, add it to the cart, and reach checkout without customer
data, order creation, or payment. Keep the locally repaired admin redirect
as the separate remaining source-release boundary.

Covers REQ-001, AC-001, REQ-003, AC-003, REQ-004, AC-004.

### Slice 8: Initial owner authorization record

Document the distinction between Supabase Auth and application staff
authorization. Preserve the exact-ID-only resolver boundary, require current
user confirmation before first-owner creation or promotion, record the
authorized non-secret outcome, and prove it with a fresh public-admin session
that displays the server-granted `owner` role. The resulting owner can run
the separate, user-authorized no-payment commerce smoke boundary.

Covers REQ-005, AC-005.

### Slice 9: R2 media handoff and proof

Record the observed disconnected R2 state, the exact dashboard source,
least-privilege Account API token choice, narrow presigned-PUT CORS policy,
temporary `r2.dev` versus custom-domain boundary, Railway and storefront Pages
destinations, secret classifications, deployment order, and failure map.
After the credential holder performs the cloud changes, prove the direct
admin -> presign -> R2 PUT -> verify -> product -> storefront-image path with
an allowed non-sensitive test image. Do not copy a token, secret key, or
presigned URL into any artifact.

Covers REQ-006, AC-006.

### Slice 10: Resend mail handoff and proof

Inspect the existing Go mail sender, then document the actual Resend
Domains/API Keys and Cloudflare DNS handoff with English UI labels and Chinese
operator meaning. Record the owned-domain requirement, manual DNS default,
domain-scoped Sending access key, exact sender rule, Railway-only
Secret-versus-Text destinations, contact-versus-order recipient behavior,
deployment gate, and failure map. Keep all keys and private recipients out of
the repository. Do not create an account/key, save a value, mutate DNS, or
send email without current action-time authorization. After the holder
completes the provider side, prove one bounded delivery in Resend Emails/Logs
without blindly repeating the contact endpoint because it persists before
mail send.

Covers REQ-007, AC-007.

### Slice 11: Re-audit, emergency edge rotation, and remaining provider handoffs

Separate the 2026-09-16 historical ledger from the latest provider state.
Record only the current non-secret Railway/Pages endpoint and source evidence,
R2 scoped-token/CORS/test-origin status, and empty signed-in Resend lists.
The storefront Pages plaintext `EDGE_SECRET` reopens the correct-type release
gate: pause provider deployments, guide the holder through generating a fresh
credential and direct encrypted Railway/storefront/admin entry, then deploy
and replay four endpoint checks. Do not preserve the exposed value as a
rotation-window previous secret. After current holder choices/confirmations,
complete the R2 key handoff and image proof and the Resend domain/DNS/key/mail
proof; preserve each as pending until independently observed.

Covers REQ-001, AC-001, REQ-003, AC-003, REQ-006, AC-006, REQ-007, AC-007.

### Slice 13: Authorized no-gateway order loop

Read-only check live public payment/bootstrap data and the selected
Supabase payment rows. After the owner's specific choice, create only one
clearly labeled sandbox/manual/no-charge row in an empty table, using a guarded
SQL Editor insert without reading credentials or contact records. Prove
public availability and a side-effect-free server quote, then hand the
checkout page to the holder for controlled contact/shipping entry and exactly
one marked submission. Before mail-inclusive checkout, check production
notification template metadata and, with separate customer-facing-copy
authorization, establish one truthful enabled `order_placed` template via
the admin create action. Only after submission, inspect non-sensitive order,
inventory, and notification evidence through authorized views. A skipped
missing-template log cannot be retried into a send because its rendered
content is empty; ask for a second order/recipient authorization rather than
forging a log. Do not collect
PII, access token, gateway payment, fulfillment, cancellation, or restock.
Update the reusable skill, installation guides, and this evidence ledger;
leave the order acceptance pending until independently proved.

Covers REQ-008, AC-008.

## Traceability

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 2, 3, 4, 7 | Inspect ledger and operator sheet, scan for runtime values, verify the four non-secret endpoint outcomes, run speccheck, skill validation, and verify |
| REQ-002, AC-002 | 1, 2, 3 | Inspect ref and provider state, review recovery instructions, security review receipt, run speccheck, skill validation, and verify |
| REQ-003, AC-003 | 4, 3, 7 | Inspect the exact source/destination/type/stop-condition sheet and the provider-state ledger, confirm the Shared-pooler/Session-mode decision and storefront-only build secret match the long-lived pgx/database/sql runtime and build paths, confirm no secret literal is recorded, verify the four endpoint outcomes, wait for storefront hydration, complete the no-payment cart/checkout proof, run speccheck, skill validation, and verify |
| REQ-003, AC-003 | 7 | Document the distinction between the static Pages fallback and curatory island's live Pages Function fetch so a post-admin smoke test does not treat pre-hydration HTML as a failed publication |
| REQ-004, AC-004 | 5, 3, 7 | Inspect the Cloudflare build warning and source asset, run the admin build, deploy the user-released source, and request a representative deep link; record the currently pending remote-source boundary until then |
| REQ-001, AC-001, REQ-003, AC-003 | 6, 3 | Inspect the encrypted-Secret type gate and remediation path, scan changed artifacts for credentials, run speccheck/scopecheck/verify, and leave external rotation as a credential-holder handoff |
| REQ-005, AC-005 | 8, 3 | Inspect resolver/tests and docs, confirm the exact UID is linked to one active owner row only after current authorization, reload public admin to observe `owner`, retain the security review receipt, and run speccheck/skill validation/verify |
| REQ-006, AC-006 | 9, 3 | Inspect config/bootstrap/media upload code and the current R2 dashboard state, verify the source-to-destination/type/CORS sheet, then observe a fresh deployment, verified media key, product association, and public image fetch without recording credentials |
| REQ-007, AC-007 | 10, 3 | Inspect config/bootstrap/contact/commerce mail paths and official Resend domain/API-key/Cloudflare DNS guidance, verify the exact source/destination/type and recipient distinction, scan for API-key literals, then observe a currently authorized delivery proof without recording credentials or retrying a persisted inquiry blindly |
| REQ-008, AC-008 | 13, 3 | Verify one authorized test-only method in the selected DB and public Pages response, quote without personal data, holder-submitted unpaid order and stock delta. For mail acceptance, confirm enabled truthful template before a separately authorized order, then sent log and provider event. A skipped log keeps acceptance pending. Keep PII and access token out of artifacts. |

## Risks and controls

- Risk: a live dashboard value is copied into documentation while recording
  the run. Control: record names, destinations, and non-secret status only.
- Risk: a provider branch, workspace target, or deployment status becomes
  stale. Control: record the observed target and deployment identifiers, and
  distinguish deployment completion from public routing verification.
- Risk: a fallback command silently changes the production source or deploys a
  local directory. Control: require action-time confirmation, capture current
  service IDs from the selected workspace, inspect the source after mutation,
  and prohibit `railway up` as evidence of Git branch alignment.
- Risk: a safe staged change is mistaken for an applied runtime configuration
  or a health response is mistaken for an API/edge proof. Control: label
  absent/staged/live separately and require the four endpoint checks after the
  operator-only database and edge entries are deployed.
- Risk: a future operator chooses Transaction pooler because it is visible in
  the Supabase Connect dialog, then mistakes the nine staged Supabase/JWT
  settings for a database connection. Control: record Shared pooler -> Session
  mode -> URI (dashboard: Session pooler) as this long-lived Railway Go
  service's default, require both secret names before deploy, and make
  direct/Transaction alternatives conditional.
- Risk: removing a SPA rewrite exposes deep links to 404. Control: verify
  there is no top-level 404 page, rely only on Cloudflare's documented default
  SPA fallback, and request a representative deep link after deployment.
- Risk: a screenshot exposes credentials. Control: keep screenshots on safe
  deployment or public-site pages and leave them out of the repository.
- Risk: a static fallback is mistaken for the hydrated live storefront.
  Control: distinguish the last Pages render from the curatory island's
  same-origin API fetch; wait for hydration before changing deployments or
  declaring a product absent.
- Risk: an Auth user is mistaken for a staff administrator or is elevated by
  email coincidence. Control: keep the resolver's exact UID link, require
  active server-side staff status and current confirmation for first-owner
  provisioning, then prove the role in a fresh public-admin session.
- Risk: a broad R2 token, wildcard CORS rule, or public development URL is
  mistaken for a permanent production media design. Control: scope the token
  to the one bucket, allow only the admin origin's PUT plus `Content-Type`,
  keep token values off Pages, and require a custom domain before production.
- Risk: a broad Resend API key, unverified or mismatched sender domain, or
  private recipient copied into a browser-facing variable permits unwanted
  email use or fails only after an inquiry is persisted. Control: require
  Sending access restricted to the verified domain, Railway-only Secret/Text
  destinations, exact sender-domain matching, and one current-confirmation
  delivery proof inspected through Resend Emails/Logs.
- Risk: a user treats a **pages.dev** hostname or automatic DNS authorization
  as an owned mail domain. Control: require an authoritative owned Cloudflare
  zone, retain manual DNS as the default path, and require current holder
  approval for Resend Domain Connect or every actual DNS/key mutation.
- Risk: a publicly visible manual test option is mistaken for a paid order or
  a supported gateway, and a repeated submit persists customer data twice.
  Control: label it explicitly no-charge/sandbox, prove quote separately,
  require holder-entered controlled data and one marked submit, check for an
  existing order before retry, record unpaid status and stock/mail separately,
  and disable or replace the option before real-payment launch.
- Risk: development-only notification seed is mistaken for a live production
  template and an order is counted as mail-delivered despite a skipped log.
  Control: inspect template metadata before mail-inclusive checkout, use
  approved truthful copy, distinguish skipped/sent/Delivered/inbox, and never
  retry a missing-template log whose original rendered content is empty.
