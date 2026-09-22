# Deployment Dashboard Ledger Specification

Change ID: deployment-dashboard-ledger
Revision: 25
Status: Accepted
Decision authority: Repository owner/user
Approval basis: The user requested the full Railway/Pages/Supabase/R2/Resend deployment flow, skill, and guide across the 2026-09-16 walkthrough, including detailed handoffs, first-owner authorization, and no-payment commerce smoke. On 2026-09-19 the user required a de-identified skill and chose Cloudflare Pages development URLs as temporary test origins. On 2026-09-20 the holder confirmed a new encrypted edge credential, entered R2/Resend Railway values, and authorized the Railway and Pages deployments. The holder subsequently authorized a labeled R2 smoke image upload and product association, and requested a personal-info/key audit of skill and docs. The holder later identified the owner-controlled management recipient and authorized one labeled contact notification test after a new Railway deployment. The holder confirmed the unique test message arrived in their spam folder. The holder then supplied a Resend Emails screenshot showing the matching test subject with Delivered status. The user asked to improve spam placement and allowed local skill/document changes without a commit. The following instruction authorized the already described Cloudflare DNS/DMARC step; one monitoring-only TXT record was published and publicly resolved. No R2 key or mail code was changed, and the instruction does not authorize copying secrets, a blind resend, or releasing the still-local redirect-source repair. The holder subsequently confirmed the current private management destination and authorized exactly one new labeled post-DMARC contact test; it returned 201 and the matching Resend event was Delivered. Mailbox placement and receiver authentication are still pending. Read-only live checkout data exposed zero public payment methods. The user explicitly authorized a publicly visible test-only manual/no-charge option and one marked order with holder-entered controlled details, without gateway payment, fulfillment, cancellation/restock, source publishing, or Git commit.
Repository baseline: 1665e08327a4656a0e3457058f4ee7c5fd8ac182
Supersedes: project-deployment-browser-skill

Revision 12 authority: the user explicitly requested that Resend be connected
as part of the same deployment path. This authorizes the reusable
domain/DNS/key/Railway handoff and delivery-proof documentation, but not
account creation, DNS mutation, key creation, secret entry, or email delivery
without current action-time authorization.

Revision 25 authority: the holder confirmed one order was placed. This permits
read-only diagnosis and local skill/document evidence. It does not authorize
a second order, a new customer-facing email template, or another external send
without the holder's specific approval of recipient and message behavior.
Subsequently the holder chose a normal customer-facing `order_placed` template
and confirmed the exact prepared copy immediately before it was enabled. The
template configuration alone sends no email. This approval does not authorize
a second order, payment, fulfillment, or a blind notification retry; the
holder identified the test mailbox as controlled, but a new order still needs
its own explicit decision and holder-entered details. The holder subsequently
submitted that second controlled order; read-only order, inventory, notification,
and provider-event verification is in scope. This does not authorize a third
order, payment, fulfillment, retry, restock, or source release.

## Outcome

The project deployment skill and HTML guide contain a current, non-secret
ledger, a safe recovery path for the Railway Git branch mismatch, and a
repeatable provider-value refresh runbook. The ledger distinguishes historical
Railway projects from the current selected workspace, preserves real
non-secret identifiers, variable names, destinations, branch/deployment state,
and endpoint verification boundaries. The runbook separates derived public
values from operator-only secrets, records staged versus live state, gives
both Pages projects their own Function path, and maps each endpoint result to
the next safe check. It makes `staging` the explicit source target, documents
Railway's staged-change lifecycle, and does not treat a successful health check
as proof of the protected API path. For this repository's long-lived Railway
Go service, the runbook makes Shared pooler with Session mode and URI format
the current
Supabase database choice, separates it from the non-secret Supabase/JWT
settings, and forbids treating the staged safe batch as a completed database
handoff. Runtime values remain in the named provider consoles and are never
copied into repository files or commits. The admin Pages build has no
top-level 404 file, so it relies on Cloudflare Pages' default SPA fallback
rather than a malformed catch-all rewrite. A field-by-field operator sheet
states the exact source path, destination, entry type, stop condition, deploy
action, and expected result for Railway, storefront Pages build/Function, and
admin Pages build/Function, including the storefront-only build use of
`DATABASE_URL`. 

The R2 media handoff is now also explicit: the deployment guide distinguishes
the public account identifier from the credential pair, directs the holder to
create a bucket-scoped Object Read & Write token, gives the browser-upload
CORS rule, limits `r2.dev` to the temporary testing path, and maps every R2
value to Railway or the storefront Pages build. Completion requires actual
media verification and a public product-image fetch, not merely a saved
variable name.

The Resend handoff is equally explicit: the existing Go sender wiring remains
the only mail path, while the operator sheet distinguishes the owned verified
domain, generated DNS rows, minimal Sending access API key, exact-domain
sender, and contact-form management recipient. It records that the private
API key and notification mailbox are Railway-only protected values, explains
why **pages.dev** cannot be a production sender domain, and separates a boot
check from a currently authorized real-delivery proof.


The runbook also treats Supabase Auth and application authorization as separate: the first owner is a current-confirmation-only, exact-UID link to an active `staff_members` role, and a fresh public-admin session must prove the server recognizes that role before commerce testing continues.

## Scope

In scope:

- Updates to `skills/site-deployment-browser/SKILL.md`.
- A supporting reference under
  `skills/site-deployment-browser/references/`.
- Remove the invalid, tracked `admin/public/_redirects` SPA rewrite.
- Corrections to `README.md` and `docs/environment-configuration.md` where
  earlier deployment advice would otherwise contradict the operator sheet.
- Updates to `docs/deployment-guide.html`.
- Record the confirmation-gated first-owner authorization procedure and non-secret result.
- Record and complete the bounded R2 media handoff without storing an R2 credential.
- Record and complete the bounded Resend domain/DNS/key/Railway handoff
  without storing a Resend key or sending an email before current confirmation.
- This controlled-change directory and the task scope file.

Out of scope:

- Provider account creation, dashboard mutations, DNS changes, key creation,
  deploys, email sends, payment actions, or environment-variable saves without action-time
  user confirmation.
- Changes to application runtime behavior, provider choices, database schema,
  admin UI, public site UI, or deployment infrastructure.
- Persisting screenshots, passwords, connection strings, API keys, deploy-hook
  URLs, or other credentials into the repository.
- Transcribing a provider secret from one dashboard, log, browser field, or
  chat message into another destination.

## Requirements

### REQ-001: Record the completed provider batch at the dashboard boundary

The reusable skill and reference MUST contain the installation-independent
source/destination, deployment, and verification procedure without any owner's
account, project, UID, branch, hostname, or deployment receipt. The installation's
guide/ledger MUST record the observed non-secret targets, branch/deployment
status and endpoint results, distinguish pending from proved work, and keep
runtime secrets in provider consoles rather than a commit.

#### AC-001: Ledger is current and contains no runtime values

- GIVEN the Railway, Cloudflare Pages, and Supabase dashboard batch has been
  filled and verified
- WHEN the skill and guide are updated
- THEN the skill is de-identified while the installation-specific documents
  preserve actual non-secret project identifiers, variable destinations,
  current status, verification results, and no secret literal or runtime-value
  commit.

### REQ-002: Provide a safe Railway source-branch recovery path

The skill and guide MUST explain that Cloudflare Pages and Railway configure
their production branches independently, derive this installation's chosen
release branch from current source/settings (the ledger records `staging` for
this installation), and give an observable dashboard path plus a
confirmation-gated Railway CLI fallback when the branch picker cannot load.
They MUST direct the operator to capture the current workspace/project/
environment/service target before any mutation, verify GitHub App repository
access before reconnecting anything, and prohibit using a local-directory
deploy as evidence of Git branch alignment.

#### AC-002: A continuation can diagnose and safely recover the mismatch

- GIVEN Pages uses `staging`, a Railway account or workspace may have changed,
  and the service source is stale or its branch picker cannot load
- WHEN an operator continues through the skill or guide
- THEN they can verify the remote refs, capture the active target, distinguish
  the independent settings, inspect GitHub App access, select and apply
  the chosen release branch through the dashboard or confirmation-gated CLI route, and verify
  the deployed source before separately verifying public routing and endpoints
  without recording a credential.

### REQ-003: Make provider-value refresh reproducible without secret handling

The skill and guide MUST provide a detailed, field-by-field refresh map for
Railway, storefront Pages, and admin Pages. It MUST name the exact dashboard
source path, destination, entry type, and safe derivation rule for every
required value; distinguish Pages build variables from Function runtime
variables; identify every operator-only secret and its destination without its
contents; state whether a value is absent/staged/live; and make deployment
stop conditions explicit. It MUST record the Supabase connection-method
decision for the current long-lived Railway Go service (Shared pooler with
Session mode and URI, shown in the dashboard as Session pooler; direct only
after verified outbound IPv6; Transaction pooler only as a
future short-lived exception), require a fresh deployment after Function-
variable updates, and describe the four-request edge proof plus the
no-payment commerce smoke boundary.

#### AC-003: A future release can refresh and diagnose the deployment values

- GIVEN the Railway workspace, public origin, or either Pages project has
  changed
- WHEN an operator follows the skill and guide
- THEN they can distinguish the staged non-secret batch from the missing
  `DATABASE_URL`/`EDGE_SECRET` handoff, retrieve each value at its named
  source, put it in the named Railway/storefront/admin destination with the
  correct text-or-secret type, select Shared pooler with Session mode and URI
  for the current Railway runtime, deploy only after every stop condition
  passes, redeploy the
  affected providers, distinguish 200/403/500/502 outcomes, and stop before
  payment while accurately recording any remaining handoff.

### REQ-004: Keep the admin Pages SPA fallback valid

The admin Pages deployment MUST not contain the known invalid catch-all
rewrite that Cloudflare ignores as an infinite loop. When no top-level
`404.html` exists, the deployment MUST rely on Cloudflare Pages' default SPA
fallback for non-asset navigation paths.

#### AC-004: An admin deep link is served without the invalid redirect rule

- GIVEN the admin Pages build starts from the repository source
- WHEN Cloudflare builds and deploys the admin project
- THEN the build does not emit the invalid `/* /index.html 200` rule, and a
  representative non-asset admin route returns 200 through the SPA fallback.


### REQ-005: Make first-owner authorization explicit and safe

The skill and deployment documents MUST distinguish Supabase Auth
authentication from application staff authorization. They MUST require an
exact `auth.users.id` to `public.staff_members.supabase_user_id` mapping with
active status and a supported role; prohibit email fallback; make the first
owner creation/promote action subject to current user confirmation; and
require a fresh public-admin session to prove the server grants that role.

#### AC-005: A first owner can be authorized without weakening identity checks

- GIVEN an intended administrator can sign in through Supabase Auth but the
  matching staff row is absent
- WHEN a responsible user gives current authorization to create the first
  owner
- THEN the operator can create exactly one active owner linked by the exact
  user ID, reload the public admin session, verify the server displays
  `owner`, and record the non-secret outcome without using email matching or
  persisting a credential.

### REQ-006: Make R2 media deployment reproducible and safely verifiable

The deployment skill and guides MUST identify the actual R2 prerequisites for
this repository's direct-browser upload path: a bucket-scoped Object Read &
Write credential pair for Railway only, a narrow browser-upload CORS rule,
a public image origin, the complete Railway runtime-variable set, and the
storefront-build-only `R2_PUBLIC_BASE_URL`. They MUST distinguish the public
Account ID and public base URL from credential material, prohibit R2 secrets
from Pages and the browser bundle, and label `r2.dev` as a temporary
non-production test origin rather than an implicit production choice.

#### AC-006: A real image crosses the full R2 media path

- GIVEN the existing bucket begins without a CORS policy, public origin, or
  Railway R2 runtime variables
- WHEN the credential holder configures the scoped R2 token, CORS policy,
  temporary test public URL or approved custom domain, Railway values, and a
  fresh storefront build
- THEN an active owner can upload an allowed raster image from the public
  admin, the server verifies it into `verified/product-images/`, a product
  can reference that verified key, and the storefront can fetch the public
  image without exposing a credential. The evidence records names, routes,
  statuses, and public URLs only.

### REQ-007: Make Resend mail deployment reproducible and safely verifiable

The deployment skill and guides MUST map the existing Resend sender to the
operator’s real provider actions: an owned verified sending domain or
subdomain, the generated Cloudflare DNS records, a domain-scoped **Sending
access** API key, exact-domain **RESEND_FROM**, and server-only
**CONTACT_NOTIFY_TO**. They MUST distinguish a contact-form management
notification from a customer order notification, prohibit all Resend values
from Pages/browser surfaces, and require current confirmation before a real
email/test-data action.

#### AC-007: One bounded Resend delivery can be proved without secret leakage

- GIVEN the operator has an owned domain in the authoritative Cloudflare zone
- WHEN the holder verifies it in Resend, enters the minimal Railway values,
  completes a fresh Railway deployment, and gives current approval for one
  controlled delivery test
- THEN the system can observe a labelled contact-form send or another
  explicitly approved flow in Resend **Emails** / **Logs**, record only
  non-secret outcome/status, and avoid blind retries because the contact
  inquiry is persisted before the mail attempt.

### REQ-008: Distinguish an offline-test quote from a verified order loop

The deployment skill and installation guides MUST give an authorized,
single-row recovery for an empty payment-method table without extracting an
admin token or database credential. They MUST name the selected Supabase SQL
Editor as the empty-table configuration path, distinguish public payment
availability from quote and persisted order, explicitly label the test method
as not charging customers, and state that DB-only configuration does not
require a new Pages or Railway deployment. They MUST require a holder-entered
marked order, idempotent one-time submission, authorized readback of order and
inventory, and an enabled, truthful `order_placed` template before an order
meant to prove automatic mail. A missing template's skipped log MUST not count
as a sent notification; the current Retry path reuses empty rendered content
and cannot recover that log. The runbook MUST require new authority for an
additional order/recipient rather than silently retrying or forging a log.
Private contact fields and order access tokens MUST remain out
of skill, documentation, and evidence. Gateway payment, fulfillment, and
cancel/restock are outside this authorization.

#### AC-008: A marked order can be traced without claiming payment

- GIVEN the public checkout initially has available shipping but no ready
  payment method
- WHEN the owner authorizes a public test-only manual option and a holder
  submits one marked order with controlled details
- THEN the public method list and server quote show that option, the order
  appears once as unpaid in authorized readback, stock changes transactionally,
  and notification outcome is checked separately without exposing PII or
  claiming that payment or fulfillment occurred. A skipped event proves a
  configuration gap, not mail closure; acceptance remains pending until an
  authorized, correctly templated order has a sent event. A prior skipped
  order is retained and not repeated without fresh consent.

## Amendment log

| Revision | REQ/AC | Old meaning | New meaning | Reason | Approval basis | Invalidated evidence and replay |
|---|---|---|---|---|---|---|
| 2 | REQ-001, AC-001, REQ-002, AC-002 | Record the completed dashboard-variable ledger only. | Preserve the ledger and add a safe, confirmation-gated Railway branch-alignment recovery path. | A live diagnosis confirmed Pages on `staging`, Railway on `main`, a failed branch picker, and an otherwise healthy API/edge path. | The 2026-09-16 user request. | Re-read artifacts, replay secret scan and governance/skill validation, and retain the security review receipt. |
| 3 | REQ-001, AC-001, REQ-002, AC-002 | Treat recorded Railway IDs and a selected source label as sufficient continuation evidence. | Distinguish historical/current Railway targets, require current-target capture, staged-change application, deployment evidence, exposure-aware health verification, and separate Pages-origin confirmation. | The user switched Railway workspace; the staged `main -> staging` change deployed successfully, but the current service remains unexposed. | The user's 2026-09-16 request to finalize the whole deployment process. | Re-read artifacts, replay secret scan and governance/skill validation, preserve the security review receipt, and verify the observed deployment details. |
| 4 | REQ-001, AC-001, REQ-003, AC-003 | Treat the current public-routing result as either fully completed or unavailable. | Record the live public origin and health result, label the non-secret Railway set as staged, document the operator-only database/edge handoff, add the admin Pages project, and supply a repeatable value map, endpoint proof, failure map, and no-payment smoke boundary. | The public Railway domain was generated and healthy, but a fresh service still lacks its database and edge secrets; the user requested a detailed, reusable refresh procedure rather than a misleading completion claim. | The user's 2026-09-16 request to run what is safe directly and write detailed release instructions. | Re-read artifacts, replay governance/skill/security checks, verify the public health and current pre-handoff endpoint statuses, and retain the security review receipt. |
| 5 | REQ-004, AC-004 | Keep the existing catch-all rewrite as a required SPA configuration. | Remove the invalid rewrite and rely on the documented Pages default SPA fallback because the build contains no top-level 404.html. | A successful authorized admin redeploy reported that the rule was ignored as an infinite loop, even though a representative deep path happened to fall back to the SPA. | The user's request to run the full flow through and leave a usable deployment procedure. | Reproduce the warning, verify the source/asset state, remove the rule, run the admin build, redeploy, and request a representative deep link. |
| 6 | REQ-003, AC-003 | Refer generically to a pooler connection string and imply the transaction-pooler preparation. | Record the observed Session-pooler URI choice for this long-lived Railway Go service; distinguish it from the staged Supabase/JWT values, preserve direct-connection IPv6 and future Transaction-pooler exceptions, and forbid deployment before both operator-only names exist. | The user selected the Session-pooler path in Supabase and asked that this decision be retained in the skill and guide after noticing Railway still had only the nine staged non-secret variables. | The user's 2026-09-16 request to record the choice in the skill and documentation. | Re-read the runtime pool implementation and official provider guidance; update the skill, guide, and refresh reference; rerun security/governance/skill validation and preserve non-secret evidence. |
| 7 | REQ-001, AC-001, REQ-003, AC-003 | Give a value map and deployment outline that still requires the operator to infer the exact dashboard field, Pages build-versus-Function destination, and final deploy gate. | Add an exact source -> destination operator sheet, make all required entry types and stop conditions explicit, and correct README/environment-configuration guidance so storefront-only `DATABASE_URL` use and the valid admin fallback cannot be misread. | The user correctly reported that the prior documents did not show a trustworthy end-to-end fill-and-deploy procedure. | The user's 2026-09-16 request for every variable's source, destination, and deployment action to be recorded. | Re-read provider guidance and repository build paths; update all deployment documents/skill; replay secret scan, governance/skill validation, and repository verification. |
| 8 | REQ-001, AC-001, REQ-003, AC-003 | Treat a named Cloudflare variable as adequate proof of a safe secret handoff. | Make encrypted **Secret** type versus plain **Text** an explicit deployment gate; require rotation, deletion, and encrypted recreation if a database URI or edge credential appears as Text. | The dashboard audit showed that names alone did not prevent secret values from being saved in text fields. | The user's follow-up demand for a trustworthy, complete fill-and-deploy process. | Re-read the provider table without extracting values; update the skill/reference/docs, replay credential scanning and governed verification, and leave the actual rotation to the credential holder. |
| 9 | REQ-001, AC-001, REQ-003, AC-003, REQ-004, AC-004 | Treat the operator sheet's future deployment steps as the current state. | Record the authorized live deployment IDs and four endpoint outcomes without a credential literal; keep the remote admin redirect warning and user-authenticated commerce smoke as explicit remaining boundaries. | The credential holder completed the handoff and the user authorized application of the staged change set and Pages retries. The endpoints proved the protected proxy path, while the remote source still predates the local redirect repair. | The user's 2026-09-16 "settings are complete; try it" authorization. | Re-read dashboard deployment status, run the four non-secret HTTP checks, update docs/evidence, rerun governed verification, and leave source release plus authenticated commerce smoke pending. |
| 10 | REQ-003, AC-003, REQ-005, AC-005 | Treat a successful Supabase Auth session as sufficient evidence that an administrator may use the admin surface, and treat the first static storefront snapshot as the complete catalog view. | Require a current-confirmation-only, exact UID-to-active-staff owner link, prohibit email fallback, prove the role through a fresh public-admin session, and distinguish static fallback from hydrated live catalog before cart/checkout proof. | The first authenticated account was correctly denied because `staff_members` was empty; once safely authorized, the active smoke product initially appeared absent only before the curatory island fetched the Pages API. | The user's explicit 2026-09-16 authorization to create the active owner link, run the no-payment product smoke, and record both in the skill and documents. | Inspect resolver/tests and exact staff row, reload public admin, create authorised labelled smoke data, wait for storefront hydration, reach checkout without order/payment, update docs/evidence/security receipt, and rerun governed verification. |
| 11 | REQ-006, AC-006 | Treat a generic R2 variable list as sufficient deployment guidance. | Add the exact token scope, CORS contract, public-origin boundary, destination/type map, deployment order, failure map, and actual media proof requirement. | The live R2 bucket had no CORS policy or public origin and Railway had no R2 runtime variables, so image upload could not complete. | The user's 2026-09-16 request to make R2 work end-to-end and record the process in the skill and documents. | Inspect the storage code and dashboard state; update the operator sheet and guides; then require a credential-holder deployment plus actual presign, upload, verify, product-image, and public-fetch evidence without persisting a credential. |
| 12 | REQ-007, AC-007 | Treat Resend as a generic RESEND_* note without a source, destination, DNS, or delivery boundary. | Record the current Go sender wiring, Resend Domains/API Keys and Cloudflare DNS paths, domain-scoped Sending access key, exact sender, contact-recipient distinction, Railway-only types, and confirmation-gated delivery proof. | The user asked that Resend be connected as part of the same full deployment flow. | The user's 2026-09-16 Resend request. | Inspect production mail wiring and official provider guidance; update the skill/guides/ledger, scan for API-key literals, run governance/skill/repository verification, and leave login/domain/key/DNS/deploy/delivery evidence pending. |
| 13 | REQ-001, AC-001, REQ-003, AC-003, REQ-006, AC-006, REQ-007, AC-007 | Treat 2026-09-16 provider snapshots and functional endpoint proof as current complete configuration. | Record the 2026-09-19 live delta: newer staging commit/deployment, already-configured R2 CORS/test URL/scoped token but missing Railway values/media proof, signed-in empty Resend account, and plaintext storefront `EDGE_SECRET` as a blocking credential incident. Reopen AC-003 until rotation and encrypted recreation. | The latest audit showed cloud state drift and a live security gap that four healthy endpoints cannot disprove. | The user's 2026-09-19 request to complete the deployment and update skill/docs with login and one-time-value handoffs. | Recheck non-secret UI and HTTP statuses, update provider ledger and security receipt without literals, validate the skill/governance/repository; wait for holder rotation and bounded R2/Resend proofs before restoring acceptance. |
| 14 | REQ-001, AC-001, REQ-002, REQ-003, REQ-006, REQ-007 | Keep site-specific IDs, URLs, branch and dated incident state in the reusable skill. | Derive the target from each installation's dashboards, use its Pages URLs as temporary full-loop test origins, and keep actual non-secret identities only in the installation ledger. Retain pending security, R2, mail and admin-release boundaries. | Other agents must not copy this owner's account or URL as a default, and proxy proof is not a full loop. | User's 2026-09-19 de-identification and Pages test-origin request. | Scan skill/reference for identities and secret literals; validate skill and governed repository checks; preserve pending acceptance. |
| 17 | REQ-007, AC-007 | Keep the contact test blocked on a recipient not yet identified for this send. | Record the holder-confirmed destination without its literal, the fresh Railway deployment, one marked HTTP 201 contact submission, and the still-pending Resend event/inbox proof. | A successful API response does not by itself prove provider sending; the configured sender can be a log fallback. | The holder provided the recipient and authorized one test after updating Railway. | Do not retry the POST; inspect Resend event and recipient inbox when available, then replay the acceptance evidence without recording PII or credentials. |
| 18 | REQ-007, AC-007 | Treat HTTP 201 without provider-event or inbox evidence as unproved delivery. | Record the holder's matching-message spam-folder receipt as mailbox-arrival proof, while keeping provider event and normal-inbox placement separate and pending. | An actual mailbox receipt resolves the log-sender ambiguity but exposes an inbox-placement issue; it does not satisfy the specified Resend Events/Logs observation. | The holder reported the exact unique test message in spam. | Update only the mail evidence and reusable spam-check guidance; do not resend or infer a cause from one test. |
| 19 | REQ-007, AC-007 | Keep the Resend event pending after an earlier dashboard timeout. | Record the holder-provided Resend Emails list showing the matching test subject as Delivered; pass the provider-event acceptance while preserving the separately observed spam-placement concern. | The provider event closes the specified mail evidence gap, while receiving-server delivery is distinct from normal-inbox placement. | The holder supplied the Resend dashboard screenshot after confirming the same message in spam. | Replay mail traceability/security and documentation checks without storing the pictured recipient or resending. |
| 20 | REQ-007, AC-007 | Treat Delivered plus spam placement as a generic code defect without diagnosing provider insight or DNS. | Add a de-identified deliverability diagnostic and record one missing-DMARC observation in installation docs; keep the safe initial DNS record explicitly unsaved until current authorization. | Resend Insights flags missing DMARC and public DNS agrees; the existing mail sender already includes plain text, so a speculative code change is not the demonstrated repair. | The user requested improvement, allowed local saves, and explicitly prohibited a commit; no DNS publication permission was established. | Recheck the skill, installation ledger, privacy, and governed verification locally. DNS publication and a new external test remain separate authorization boundaries. |
| 24 | REQ-008, AC-008 | Checkout-first-step smoke stopped at an empty payment table; no owner choice or order authority was established. | Record the explicit test-only manual method decision, the guarded one-row live configuration and public quote proof, and the still-pending holder-entered marked order, inventory and notification checks. | The user explicitly agreed to a full no-gateway loop after the empty-table blocker was explained. | The user's 2026-09-20 consent to publicly enable a visibly labeled no-charge option and submit one controlled marked order, without a Git commit. | Recheck public payment/bootstrap/quote and the de-identified docs; only mark AC-008 passed after one holder-submitted order plus non-sensitive admin/inventory/mail evidence. |
| 25 | REQ-008, AC-008 | Treat a placed unpaid order and a checked notification log as possible full closure even when mail was skipped. | Require an enabled truthful order-placed template before mail acceptance; record the first order's transactional stock proof and skipped/template-not-found event, prohibit retrying the empty rendered log, and hold full closure for a separately authorized order/email. | The holder placed the first order; narrow DB readback found the production template table empty and one skipped order event, while development-only seed contains templates. | The holder's 2026-09-20 confirmation of the placed order authorizes read-only diagnosis and local documentation, not a second order or new customer email. | Recheck safe order/template/log fields and update the skill, guides, security receipt, and gate; require new action-time consent for an enabled template and second order. |
