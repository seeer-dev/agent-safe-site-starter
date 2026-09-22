# Deployment Dashboard Ledger Evidence

Revision 14 (2026-09-19) de-identified the reusable deployment skill and
worksheet while keeping actual non-secret values in this installation's ledger.
The two Pages development origins are temporary test URLs, not full-loop proof.
Revision 13 (2026-09-19) rechecked the live provider dashboards and public
endpoints. The 2026-09-16 sections below are historical snapshots, not a
claim that R2/Resend are still wholly unconfigured or that the old edge
credential has been safely stored.

## 2026-09-19 live delta and release stop

- Local HEAD and remote `main`/`staging` refs are
  `bac7c78fd6c5b4793e6705cefbdc7cabb56095bb`. The current Railway
  service displayed a newer active successful GitHub deployment
  `c8da4326-ecdf-44ce-9eb1-fa7d37597f91`. Fresh read-only requests
  returned `/healthz` 200, uncredentialed direct `/api/products` 403, and
  storefront/admin Pages `/api/products` 200 each.
- The storefront Pages Production `EDGE_SECRET` row is **Text／文字** despite
  the functional proxy proof. Its literal was visible in the dashboard and
  is not reproduced here. Storefront `DATABASE_URL` and admin Pages
  `EDGE_SECRET` are encrypted **Secret／秘密**. Emergency holder-only rotation
  and encrypted recreation are pending; AC-003 is reopened as pending. Do
  not retain the old edge value through `EDGE_SECRET_PREVIOUS`.
- R2 bucket `agent-safe-site-starter-assets` then had a test-only public
  development URL, exact admin-origin PUT/Content-Type CORS, and an active
  bucket-limited Object Read & Write Account API token
  to the bucket. It still has zero objects. Railway Variables still lists
  only the original 11 service variables, no five `R2_*` names. Whether the
  holder retained the one-time key pair is unknown; replacement token
  creation, saving variables, deployments, and image proof have not happened.
- Resend was signed in; Domains showed `No domains
  yet` and API keys shows `No API keys yet`. The Cloudflare account lists
  three unrelated candidate DNS zones, so the owner must choose the sender
  domain/subdomain. No key, DNS, mail recipient, or message was handled.
- All setup tabs were opened for holder handoff. No production setting was
  changed in this audit. The local deletion of `admin/public/_redirects`
  remains uncommitted; the admin build warning/deep-link proof is still open.

Revision 12 was opened on 2026-09-16 after the user requested Resend be
connected as part of the same deployment recovery. It retains the completed
revision 10 owner/commerce evidence and the pending R2 boundary while adding a
separate pending Resend proof boundary. This record keeps credential literals
out of the repository: it contains only provider identifiers, variable names,
deployment identifiers, and observed non-secret outcomes.

Revision 10 was updated on 2026-09-16 after the user explicitly authorized
the first active owner link for a verified Supabase Auth account and the
no-payment commerce smoke test. This record keeps credential literals out of
the repository: it contains only provider identifiers, variable names,
deployment identifiers, and observed non-secret outcomes.

## R2 media handoff boundary

- The Cloudflare bucket `agent-safe-site-starter-assets` exists in APAC, but
  its dashboard displayed no CORS policy, public development URL, or custom
  domain.
- The current Railway Variables page displayed 11 service-variable names and
  no `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`,
  or `R2_PUBLIC_BASE_URL` row.
- The Go application starts with disabled object storage unless the first four
  R2 values are present. The browser media client sends a direct presigned
  `PUT` with `Content-Type`, so this is a configuration handoff rather than a
  workaround to add to Pages Functions.
- Pending proof is deliberately explicit: the credential holder must save the
  R2 settings, deploy Railway, rebuild the storefront with the public base
  URL, then use an active owner to verify an allowed image and load it from
  the public origin. No token, access key, secret key, or presigned URL is
  recorded.

## Resend mail handoff boundary

- Local inspection confirms the existing Go bootstrap selects the Resend sender
  only when `RESEND_API_KEY` is non-empty; otherwise it uses the local log
  sender. `RESEND_FROM` selects the sender, `CONTACT_NOTIFY_TO` is only the
  contact-form management recipient, and commerce notifications use each
  order's customer email.
- The agent opened the Resend **Domains** page, which redirected to the
  unauthenticated Resend login screen. No Resend account, domain, DNS row, API
  key, Railway value, deployment, message, or private recipient was read or
  created.
- A production mail sender needs an owned domain or sending subdomain; a
  Cloudflare Pages `pages.dev` hostname cannot supply the necessary DNS proof.
  The reusable handoff records the exact Resend/Cloudflare path, least
  privilege key, Railway-only destination types, and confirmation gate.
- A genuine contact-form test would persist an inquiry before it sends
  external mail. It remains pending explicit current authorization and a
  recipient the user controls; no test data or email was generated.

## Current source and public-route result

The dashboard's current target displayed workspace `gracious-liberation`,
project `ff5c31fd-eb59-48f7-ad46-12b468e02e42`, production environment
`845024a8-3a49-485d-ac67-d8d899ee4c27`, and service
`6fc1aea1-5efc-4ce0-9ccc-ef688d59f72c`.

- Source Settings displays `seeer-dev/agent-safe-site-starter`, branch
  `staging`, and Auto Deploy enabled.
- The earlier `main -> staging` source change produced successful deployment
  `709106e7-9697-4fcb-b05e-708c84511681` from commit
  `1665e08327a4656a0e3457058f4ee7c5fd8ac182`.
- The active public origin is
  `https://agent-safe-site-starter-production.up.railway.app`.
- Remote ref inspection confirmed both `main` (`83acded`) and `staging`
  (`1665e08`) exist. The source-selection result and public-routing result
  remain separate checks.

## Authorized credential handoff and deployment outcome

- The credential holder reported completing the direct handoff of
  `DATABASE_URL` and `EDGE_SECRET` without exposing a value to the agent.
  We intentionally did not reopen a plaintext value field to inspect it.
- Railway applied 13 staged changes. Deployment
  `c904769a-d36e-4a8b-8581-251f56d79b7b` succeeded, and the Variables UI
  displayed 11 service-variable names: the nine non-secret rows plus
  `DATABASE_URL` and `EDGE_SECRET`.
- The Railway checks returned `/healthz` = `200` and an uncredentialed direct
  `/api/products` = `403`. This distinguishes a live origin from the
  protected API route.
- Storefront Pages retry deployment
  `580ffc26-8d3b-46ec-8e4a-f912d9c4b348` succeeded. Its
  `/api/products` endpoint returned `200`.
- Admin Pages retry deployment `87d0566f-ce84-4828-9699-97f1fa1c3997`
  succeeded. Its `/api/products` endpoint returned `200`.
- Together, the four outcomes `200`, `403`, `200`, and `200` prove the two
  Pages Functions are separately forwarding through the protected Railway
  origin. They do not expose or prove a credential literal or storage value.
- The first authenticated admin session was correctly denied because
  `public.staff_members` was empty. With current explicit authorization, one
  active owner was linked to the exact authenticated user ID; a fresh public
  admin session displayed `owner` and protected controls.
- That owner created an active, clearly labelled smoke product. After the
  storefront's curatory island hydrated, it fetched the same-origin Pages
  `/api/products` route and displayed the product; the temporary static HTML
  fallback's initial zero-count did not represent the hydrated storefront.
- The smoke product was added to the cart and the public checkout's first
  contact-information step displayed the line item and total. No contact data
  was entered, no order was created, and no payment was submitted.

## Secret-storage guard and Supabase choice

- An earlier Cloudflare audit found secret-classified names in **Text** rows.
  The workflow now requires the credential holder to rotate any such value,
  delete the plaintext row, recreate it as **Secret** / encrypted, and enter
  it directly at the named destination. No pre-rotation value is recorded.
- The completed endpoint proof is functional evidence only. During every
  future release, the credential holder must still confirm the Cloudflare
  table type is **Secret**, never **Text**, without disclosing the value.
- This long-lived Go service uses Supabase **Shared pooler -> Session mode ->
  URI** (the dashboard control is labelled `Session pooler`) for
  `DATABASE_URL`. Direct connection remains conditional on verified Railway
  outbound IPv6; Transaction pooler remains a future short-lived-runtime
  exception.

## Admin SPA fallback release boundary

- The repository removes `admin/public/_redirects`; a local admin build emits
  no `_redirects` asset.
- The authorized admin retry built remote `staging` commit `1665e08`, which
  predates that local deletion. Cloudflare therefore still reported the old
  `/* /index.html 200` infinite-loop warning.
- The user will later release the local source update to `staging`. Only then
  can the warning be confirmed absent and a representative admin deep link be
  recorded as `200` for REQ-004 / AC-004.

## Completion boundary

The historical functional proxy, initial-owner authorization, and no-payment
commerce proof remain recorded. Five boundaries remain:

1. The user releases the local `admin/public/_redirects` deletion with the
   later `staging` source update, then verifies no redirect warning and a
   `200` admin deep link.
2. The labelled active smoke product remains in production for repeatable
   testing. Drafting or deleting it is a separate public-data change and needs
   current user authorization; no cleanup was inferred from the smoke test.
3. Rotate the storefront's plaintext edge credential and recreate it as an
   encrypted Secret, update Railway/admin, redeploy all three, then rerun the
   endpoint and type checks. This blocks the next release.
4. R2 CORS/public test URL/scoped token exist, but the holder's direct key
   entry, Railway/Pages handoff, and actual verified product image do not.
5. Resend login exists, but owned-domain Verified state, scoped key handoff,
   fresh Railway deployment, and authorized delivery proof do not.

## Revision 12 validation

- Local inspection confirmed the production path is already Go → Resend:
  bootstrap selects `mail.ResendSender` only with `RESEND_API_KEY`; the
  contact service uses `CONTACT_NOTIFY_TO`, and commerce notifications use the
  order email. Official Resend domain, API-key, Cloudflare DNS, email, and log
  guidance informed the field map without copying a provider value.
- `go test ./server/internal/config ./server/internal/bootstrap
  ./server/internal/platform/mail ./server/internal/modules/contact
  ./server/internal/modules/commerce` passed; `npm --prefix admin run
  typecheck` passed.
- `git diff --check`, `go run ./server/tools/speccheck
  deployment-dashboard-ledger`, `go run ./server/tools/scopecheck`, and the
  deployment skill UTF-8 validator passed. A focused credential scan including
  the Resend key prefix found no literal.
- `go run ./server/tools/verify` passed after the Resend documentation and
  controlled-change update; it covered architecture, migration parity,
  controlled specs/scope, all Go tests, repeated commerce/staff/media tests,
  and vet.
- `go test ./server/internal/auth ./server/internal/bootstrap
  ./server/internal/modules/commerce` passed, as did the focused current
  `config`, `bootstrap`, `media`, and `commerce` package set.
- `npm --prefix admin run typecheck` passed, and
  `npm --prefix site/themes/curatory run build:check` passed after rerunning
  outside the sandbox that had blocked esbuild child-process startup.
- After the R2 Chinese-dashboard field-map update, `git diff --check`,
  `go run ./server/tools/speccheck deployment-dashboard-ledger`,
  `go run ./server/tools/scopecheck`, and `go run ./server/tools/verify`
  passed. The verifier covered architecture, migration parity, controlled
  specs/scope, Go tests, and vet.
- `skills/site-deployment-browser` passed the skill validator in UTF-8 mode;
  Windows' default CP950 decoder cannot read this multilingual skill, so the
  UTF-8 invocation is the relevant validation route.
- A focused fixed-string scan of the deployment documents, skill, and
  controlled artifacts found no connection-string credential, private token,
  service-role key, or literal deploy-hook URL.

## Revision 13 validation

- The browser re-audit used provider UI for variable **names/types only**
  after detecting the storefront Text incident. The reported and persisted
  record contains no edge value, R2 key pair, Resend key, database URI, or
  private mailbox. The exact dashboard links, bilingual field map, emergency
  rotation, and saved-versus-lost one-time-key fork were added to the skill,
  reference, environment guide, and HTML guide.
- Fresh read-only HTTP statuses were 200 / 403 / 200 / 200 for origin health,
  direct origin products, storefront products, and admin products.
- Gate results for this revision are recorded below after local verification;
  they cannot replace the pending cloud rotation, image, or email proofs.

## Revision 14 validation

- The reusable skill and worksheet were scanned for this installation's
  account/user/project/bucket/token/branch/URL/date markers: no matches.
  Installation-specific non-secret values remain in README/guide/ledger.
  A later type-only recheck found the storefront edge credential still Text;
  the skill now requires suppressing value-bearing accessibility output or
  handing type inspection to the holder.
- Two public Pages test tabs loaded; the storefront showed the existing
  labeled smoke product. Current Railway Variables still showed 11 service
  variables, with no five R2 names. Resend Domains/API Keys remained empty;
  the R2 public test URL and narrow admin-origin CORS remained visible.
  Cloudflare storefront Production still showed `EDGE_SECRET` as Text. No
  secret was intentionally read, copied, or persisted; these observations do
  not close the security/media/mail gates.
- `go run ./server/tools/speccheck deployment-dashboard-ledger`,
  `go run ./server/tools/scopecheck`, the skill creator's UTF-8 validator,
  `git diff --check`, and `go run ./server/tools/verify` all passed. Verify
  covered architecture, migration parity, Go tests (including repeated
  commerce/staff/media tests), and `go vet`. An initial `python` invocation
  did not exist on this host; the installed Python 3.12 executable validated
  the skill successfully.

## Revision 15 live deployment (2026-09-20)

- With the holder's action-time confirmation, Railway's 9 staged changes were
  applied to the selected production service. Its new deployment
  `e500ca98-93d7-4e95-867c-e6377232e20f` became Active/Online, replacing
  `c8da4326-ecdf-44ce-9eb1-fa7d37597f91`. The service then showed 19
  variable names, including database, edge, all five R2, and all three Resend
  names, and no remaining staged changes. Values were neither expanded nor
  copied. The holder had confirmed one new shared edge credential and encrypted
  Secret type for both Pages Function rows.
- The latest successful Production `staging` deployment of commit `bac7c78`
  was retried in each Pages project *after* Railway became Active. Storefront
  `cba35b37-9dc3-4e5c-a6c5-065f5e018c4e` and admin
  `ca05ef2b-1788-4518-bb7f-f58c98e17207` both reported success. During
  admin's build the stable admin `/api/products` briefly returned 403; after
  success it returned 200. Fresh read-only checks after both successes:
  Railway `/healthz` 200, direct uncredentialed `/api/products` 403,
  storefront `/api/products` 200, admin `/api/products` 200.
- A direct public GET of `/res/minimal-cart-products` returned 200, and an
  existing browser session displayed the 商品 management heading at that URL.
  The local `_redirects` source deletion was not included in commit `bac7c78`,
  so a clean-session redirect-warning check after its future release remains.
- The owner-selected Resend sending domain visibly remained Verified. The Railway Resend
  variable names are live but no outbound event was attempted; mail delivery
  remains unproved. R2 variables and the storefront public base are deployed,
  but no authenticated presign/PUT/verify/product-image/fetch had yet been
  attempted at that checkpoint. No provider secret, recipient, or presigned URL
  was read into this evidence.

## Revision 16 R2 and privacy follow-up (2026-09-20)

- With the holder's current authorization, a non-sensitive image was uploaded
  via the authenticated admin and associated with the labeled active smoke
  product. The first attempt failed before upload: the repository's `.png`
  example had JPEG magic bytes, so browser `image/png` disagreed with the
  uploader's `.jpg` extension. A disposable `.jpg`-named copy of the same JPEG
  bytes passed presign, browser R2 PUT, and API verify. The source asset and
  server MIME check were not changed. A fresh admin readback showed the
  verified image association; the storefront's public R2 image loaded at
  864 × 1152. No full object key, presigned URL, or credential is recorded.
- The reusable skill and installation docs now describe the exact MIME mismatch
  recovery and independent media proof. Personal mailbox, Auth UID, provider
  account/org identifiers, sender-domain label, and token display name were
  redacted from the installation documents/ledger. Public Pages test URLs and
  deployment receipts remain non-secret operational identifiers.
- A labeled contact notification was proposed but **not submitted**. The
  management recipient behind `CONTACT_NOTIFY_TO` was not explicitly
  identified for an external send. No inquiry was created and no mail event
  was sent by this test; obtain destination confirmation and fresh authorization
  before testing. Resend remains pending, as does release of the local admin
  redirect-source repair.

### Revision 17 bounded contact test (2026-09-20)

- The holder identified an owner-controlled management destination and
  authorized one marked notification. Railway applied its single staged
  `CONTACT_NOTIFY_TO` change; deployment
  `7e9ab485-9dc9-4d72-868f-89fa21cadd48` became Active with no staged
  changes. Fresh origin `/healthz` and storefront `/api/products` each
  returned 200.
- Exactly one synthetic `POST /api/contact` through the storefront Pages
  Function returned 201. The handler first persists `contact_inquiries`, then
  calls the configured sender; a 201 is not Resend delivery proof because a
  blank `RESEND_API_KEY` selects the log sender. The Resend dashboard event
  page timed out, so no provider event or inbox arrival was observed. Do not
  repeat the POST without inspecting the existing inquiry/event. No private
  destination address, token, or email-share URL is recorded here.

### Revision 18 mailbox-arrival confirmation (2026-09-20)

- The holder reported the exact one-time smoke marker and message in their
  spam folder. This is positive evidence the notification reached the chosen
  mailbox; it also records an inbox-placement concern. No second contact POST
  was submitted. The recipient literal and message content are deliberately
  absent from repository evidence.
- Resend **Emails/Logs** was not available for independent inspection after
  the dashboard timed out. The normative AC-007 event observation remains
  pending even though mailbox arrival is proved. Do not attribute the spam
  classification to any specific DNS/content/reputation cause from one test.

### Revision 19 Resend event confirmation (2026-09-20)

- The holder supplied the Resend **Emails** list showing the same controlled
  contact test subject with **Delivered** status. This resolves the provider
  event observation for REQ-007 and AC-007; the earlier dashboard timeout is
  historical. The matching one-time message had already been located in the
  holder's spam folder. No second POST, Resend retry, or order was created.
- `Delivered` is a provider/receiving-server outcome, not proof of normal
  inbox placement. Spam placement remains a separate production deliverability
  quality concern. The screenshot itself, pictured private recipient, message
  body, token, and share URL are not stored in the repository.

### Revision 20 mail deliverability diagnosis (2026-09-20)

- Read-only inspection of the matching Resend message's **Insights** showed
  one **Needs attention** finding: **No DMARC record found**. The sender's
  authoritative Cloudflare zone listed no `_dmarc` among its seven DNS records;
  a public TXT lookup returned name-not-found. Resend marked the existing
  plain-text alternative and message size as healthy. This establishes a
  specific DNS configuration gap, not the sole cause of spam placement.
- The Cloudflare Add record form has a monitoring-only TXT `_dmarc` /
  `v=DMARC1; p=none;` / Auto TTL **prefilled but unsaved**. The user permitted
  local skill/document storage and prohibited a commit. Do not equate that
  with authorization to publish DNS. No DNS change, new email, or code
  deployment occurred. A later authorization would require fresh DNS proof
  and a separately approved new-message authentication/inbox check.
- The reusable skill and installation docs now lead with provider Insights,
  public DNS, and receiving-header evidence rather than assuming a Go mail
  template change will cure spam. No private address, full header, screenshot,
  or key was written into repository artifacts.

### Revision 21 monitoring DMARC publication (2026-09-20)

- The holder's new instruction authorized the previously described Cloudflare
  DNS/DMARC step and continuation. In the selected sending domain's DNS zone,
  the agent saved exactly one `_dmarc` TXT with `v=DMARC1; p=none;` and Auto
  TTL. The record appeared in Cloudflare's table; a fresh public TXT lookup
  returned that value. No R2 API token, Railway setting, Pages deployment,
  mail code, Git commit, or second email was changed.
- The old Resend Insights warning and spam-folder result describe the message
  sent before publication. New-message DMARC authentication and inbox
  placement remain unproved. One further external send needs its own current
  permission and explicit recipient confirmation; the prior inquiry must not
  be replayed blindly.

### Revision 22 single post-DMARC mail test (2026-09-20)

- The holder confirmed the current Railway management recipient as theirs and
  explicitly permitted a single new test. Public storefront `/api/products`
  returned `200`, and the Cloudflare-published monitoring TXT remained
  publicly resolvable. Exactly one synthetic contact POST with a distinct
  `DEPLOYMENT_DMARC_20260920_B1` marker returned `201`; no retry, customer
  data, order, payment, or Git commit was involved.
- The matching new Resend Emails row showed **Delivered**. Its own **Insights**
  placed **Include valid DMARC record** under **Doing great**, displaying
  `p=none`; the earlier message had warned that no record was found. This
  proves Resend recognized the published policy on the fresh message, not a
  receiving-provider `dmarc=pass` verdict or normal-inbox placement. The
  holder's mailbox location and SPF/DKIM/DMARC summary are requested and
  remain pending. No recipient literal, key, or full header was retained.

### Revision 23 full-commerce readiness audit (2026-09-20)

- At the owner's request to verify the whole no-online-payment loop, six fresh
  public read-only checks returned Railway health `200`, uncredentialed direct
  products `403`, storefront and admin Pages products `200`/`200`, storefront
  home `200`, and admin product deep link `200`. These do not release the
  still-local admin redirect-source repair.
- Fresh `GET /api/storefront/bootstrap` exposed two available shipping methods
  but **zero** payment methods. Independent `GET /api/payment-methods` returned
  zero. The checked-in Vue checkout requires a payment selection; Go `Quote`
  and `CreateOrder` fail closed on an empty/disabled/unready method. The admin
  payment-method resource edits rows but has no create action from an empty
  list. The previous cart-to-checkout-first-step proof is only partial, not a
  placed-order loop.
- The user was asked to choose and authorize an offline/test method and to
  enter controlled test contact/shipping data. No method, order, payment,
  stock, fulfillment, or additional mail was mutated in this audit. An order
  would persist contact data, decrement stock, and enqueue a best-effort
  notification, so method choice and test-data authority are material.

### Revision 24 authorized test payment and quote (2026-09-20)

- The owner explicitly approved one publicly visible, clearly labeled
  test-only manual/no-charge payment option and one marked order with the
  holder entering controlled contact/shipping details. No gateway charge,
  fulfillment, cancellation/restock, source release, or Git commit was
  authorized. The selected Supabase SQL Editor showed zero payment rows;
  a guarded one-row insert returned `pm-manual-smoke`, `manual_test`, sandbox,
  ready, enabled, zero fee. The public Pages `/api/payment-methods` returned
  exactly that option with 200, and bootstrap independently included it once.
- Before an order, the labeled smoke product had stock 3 and sold count 0.
  A side-effect-free one-item quote for `s2s` returned subtotal NT$1, shipping
  NT$60, fee NT$0, total NT$61. No contact fields or secrets were used or
  recorded. The checkout browser was handed to the holder with a unique
  non-private buyer-note marker and one-submit instruction. The marked order,
  authorized admin readback, stock delta, and notification remain pending;
  quote is not order proof. The current public static marketing copy still
  advertises other unconfigured/unverified payment choices and must not be
  treated as a payment gateway acceptance claim.

### Revision 25 holder order and missing notification template (2026-09-20)

- The holder reported an order page. A read-only query of its exact order ID
  and item rows (excluding customer name, email, phone, address, token and
  unredacted buyer note) found one pending/unpaid `manual_test` order with one
  smoke item: subtotal NT$1, chosen shipping NT$120, fee NT$0, total NT$121.
  The earlier NT$61 quote used a different shipping method. A separate query
  found the holder's suggested buyer-note marker absent, item quantity 1,
  stock 3 → 2, sold-count still 0. The order is identified only by the
  holder-supplied page, and the full ID is deliberately not copied into this
  ledger. No second order, payment, fulfillment, or restock occurred.
- The same order had one `order_placed` notification log, `skipped`; a safe
  exact-category query identified `template not found`, and production
  `notification_templates` had zero rows. The dev seed creates templates only
  locally. No Resend order message was sent. The current Retry code copies
  original rendered subject/body; both are empty on missing-template skip,
  so clicking Retry would only create another skipped log. A truthful enabled
  template and a new controlled-recipient/second-order authorization are
  needed before mail-inclusive acceptance. Those have not yet happened.

### Revision 25 follow-up: normal order template enabled (2026-09-20)

- After reviewing the actual customer-facing copy, the holder chose a normal
  rather than test-labelled `order_placed` message and explicitly confirmed
  saving/enabling it. The authenticated admin initially displayed zero rows;
  after the create action it displayed exactly one `order_placed` row with the
  approved ordinary subject and `啟用` state. Its body describes receipt and
  total, refers to authoritative payment/shipping status, and explicitly does
  not equate order creation with payment or dispatch. No customer recipient,
  rendered message, account email, or full order ID is stored here.
- This database-only configuration did not deploy or send. The first order's
  skipped log remains historical and cannot be re-rendered by the current
  Retry action. A fresh holder-authorized order with controlled contact and
  shipping details, new notification log, Resend event, and mailbox observation
  remain required for AC-008. There was no second order, gateway charge,
  fulfillment, restock, or Git commit in this step.

### Revision 25 follow-up: holder's second order and provider event (2026-09-20)

- After the enabled normal template, the holder submitted a second controlled
  Pages checkout. Read-only narrow SQL found exactly two total orders; the new
  latest order was `pending`/`unpaid`, `manual_test`, one smoke item, subtotal
  NT$1, shipping NT$60, fee NT$0, total NT$61. Stock fell from 2 to 1. The
  first NT$121 order and its skipped log remain historical and were not resent.
- The new order has exactly one `order_placed` log with `sent:mail`, nonempty
  rendered subject/body, and no logged error. In Resend Emails, the normal
  order subject was `Delivered`; its order identifier matched the holder's
  currently open order page in memory. Neither suggested buyer-note marker
  appeared in the orders; the evidence relies on the holder page, chronology,
  and exact subject match, not a fictitious marker. No complete order ID,
  recipient, message content, customer/shipping field, or token was saved.
- `sent:mail` is backend send status, `Delivered` is provider delivery, and
  normal inbox versus spam is a separate holder observation still pending.
  No third order, repeat POST, blind Retry, payment, fulfillment, cancellation,
  restock, or Git commit occurred. AC-008's sent-event condition is met, but
  mailbox placement and the separate AC-004 source-release gate remain open.

### Revision 25 mailbox-placement follow-up (2026-09-20)

- The holder supplied a spam-folder screenshot after the second order. Its
  normal order-notification subject and NT$61 excerpt correspond to the
  order already matched against the Resend Delivered event; the two earlier
  labeled contact tests are also visible in the same spam folder. The image,
  full order identifier, message excerpt, sender/recipient addresses, and
  private headers are not copied into repository artifacts.
- This closes the second order's mailbox-arrival question as **arrived in
  spam**, not normal inbox. The post-DMARC contact test also landed in spam,
  despite Resend recognizing the public monitoring policy. The screenshot
  does not expose receiver-side SPF/DKIM/DMARC verdicts or a spam reason;
  neither code, DNS, template, nor credentials were changed. No repeat
  inquiry, third order, retry, payment, fulfillment, or Git commit occurred.

### Revision 25 source release and fallback acceptance (2026-09-22)

- The local work was committed as `847c025` (storefront bloom mark and
  transition curtain, a separately accepted change) and `a96f2a6` (this
  ledger, the de-identified deployment skill, the updated guides, and the
  `admin/public/_redirects` deletion), then pushed `bac7c78..a96f2a6` to
  `staging`. The `prelaunch-security-hardening` draft remained untracked
  and was not included.
- Storefront Pages rebuilt and began serving the new `a96f2a6` curtain
  markup within about 90 seconds of the push, confirming Auto Deploy
  picked up the released source. The admin build's content is unchanged
  apart from the absent `_redirects`, so no public marker distinguishes
  its new deployment; Auto Deploy remained enabled from the recorded
  source settings.
- A clean-session request to
  `https://agent-safe-site-starter-admin.pages.dev/res/minimal-cart-products`
  returned `200` on three consecutive checks after the release; admin root
  returned `200` and admin `/api/products` returned `200`. The deployed
  commit contains no `admin/public/_redirects`, so the infinite-loop
  warning has no source in this build; the dashboard build-log line itself
  remains holder-visible only. REQ-004 and AC-004 are recorded passed on
  this evidence.

## Traceability

| Gate | Status | Proof |
|---|---|---|
| REQ-001 | passed | The deployment skill and its worksheet contain no live account, user, project, branch, hostname, bucket, email, or deployment receipt; they derive targets per installation. README and deployment guides retain this installation's dated non-secret observations, exact test Pages URLs, pending blocks, and no credential literals. |
| AC-001 | passed | The installation-specific ledger records the Active Railway deployment, successful storefront/admin retries, fresh 200/403/200/200 statuses, and authenticated R2 image upload/product/public-fetch proof. It keeps mail and unreleased admin source repair separate. The reusable skill contains no actual deployment identities. |
| REQ-002 | passed | The reusable skill derives each installation's chosen branch and requires independent Railway/Pages source alignment, GitHub App/member access checks, deployed-commit proof, and no railway up shortcut; this installation's guide records staging and its current non-secret target. |
| AC-002 | passed | The current Railway service retained staging and its new deployment e500ca98 is Active; both Pages Production retries used staging commit bac7c78. The recovery documentation preserves the identity/cache/support escalation boundary and makes public routing a separate final hop. Security review receipt: receipts/security-review.md. |
| REQ-003 | passed | The de-identified worksheet maps Supabase, Railway, storefront/admin Pages build and Function destinations, correct Secret/Text types, Session pooler URI, source/deploy order, four endpoint checks, no-payment smoke, and emergency exposed-secret cutover. Current-site values/status remain only in the guide and ledger. |
| AC-003 | passed | The holder rotated the formerly exposed edge value, confirmed encrypted Secret type in both Pages Functions and the same new value at Railway, and authorized the Railway + two Pages redeployments. All became successful/Active on 2026-09-20, followed by fresh 200/403/200/200 checks. These do not prove R2 or mail. Security review receipt: receipts/security-review.md. |
| REQ-004 | passed | The invalid tracked catch-all redirect was removed, the local admin build emits no redirect asset or top-level 404.html, and the repair shipped in commit a96f2a6 pushed bac7c78..a96f2a6 to staging on 2026-09-22. The storefront Pages rebuild served the new a96f2a6 curtain markup within ~90 seconds of push, confirming Auto Deploy built the released source. |
| AC-004 | passed | After the a96f2a6 release, a clean-session request to https://agent-safe-site-starter-admin.pages.dev/res/minimal-cart-products returned 200 on three consecutive checks, admin root returned 200, and admin /api/products returned 200. The deployed commit contains no admin/public/_redirects, so the infinite-loop warning has no source in this build; the dashboard build-log line itself remains holder-visible only. Security review receipt: receipts/security-review.md. |
| REQ-005 | passed | The skill, canonical refresh sheet, README, environment guide, and HTML guide distinguish Supabase Auth from staff authorization; require an exact UID-to-active-staff-role mapping; prohibit email fallback; require current confirmation before first-owner creation or promotion; and require a fresh public-admin role check. |
| AC-005 | passed | The first authenticated account was denied while `public.staff_members` was empty. With current explicit authorization, one active owner row was linked to the exact Auth user ID; a new public admin session displayed `owner` and protected controls. No credential was recorded. Security review receipt: receipts/security-review.md. |
| REQ-006 | passed | The bilingual R2 worksheet maps sources, narrow token/CORS/public origin, destinations, deploy order, and presign→PUT→verify→product/public-fetch proof without a site-specific bucket. It now covers actual-byte MIME/extension mismatches and records no secret or full object key. The live media path is proved on the temporary test origins. |
| AC-006 | passed | With explicit test-upload authorization, a non-sensitive correctly named JPEG copy completed authenticated presign, browser R2 PUT, API verify, and association with the labeled smoke product. A fresh admin readback retained the verified image and the storefront image loaded from the public R2 development hostname at 864 × 1152. No credential, presigned URL, or full object key was recorded. Security review receipt: receipts/security-review.md. |
| REQ-007 | passed | The reusable bilingual Resend worksheet maps owned-domain DNS, Verified status, restricted Sending key, Railway-only destinations, contact/order recipient distinction, bounded delivery proof, and evidence-led spam/DMARC diagnosis with explicit DNS permission. The holder confirmed mailbox arrival in spam and supplied a matching Resend Emails row showing Delivered. |
| AC-007 | passed | The owner-selected sending domain is Verified. After the holder explicitly identified an owner-controlled management recipient and authorized one test, Railway deployment 7e9ab485 became Active, origin health and storefront products returned 200, and exactly one labeled contact POST returned 201. The holder confirmed the unique marker arrived in spam, then supplied a Resend Emails screenshot showing that same test subject with Delivered status. This proves the approved flow's provider outcome and actual mailbox arrival; spam placement is recorded separately as a deliverability-quality finding. The recipient literal, key, screenshot, and email-share URL stay out of repository evidence, and the test was sent once. Security review receipt: receipts/security-review.md. |
| REQ-008 | passed | The de-identified skill and worksheet cover an authorized single-row manual test method, public API/quote proof, holder-entered order with narrow readback, and an enabled normal order-placed template prerequisite. They distinguish order creation from payment/fulfillment, prohibit blind retry of a skipped empty-rendered notification, and require separate consent for a second order/recipient. Installation docs record non-sensitive state only. |
| AC-008 | passed | The first holder order was pending/unpaid, with one item and stock 3 to 2; its missing-template notification was skipped, not resent. After owner approval of one enabled normal `order_placed` template, the holder submitted a second controlled order. Narrow readback found two orders total, latest pending/unpaid `manual_test`, one item, total NT$61, stock 2 to 1. Its sole order notification was `sent:mail` with nonempty rendered content and zero errors. The Resend subject matched the holder's open order page in memory and showed Delivered. A later holder-provided spam-folder screenshot showed the matching second-order message: mailbox arrival is confirmed in spam, not normal inbox. Buyer-note markers were absent. No customer data, full ID, token, screenshot, or message content is retained. Security review receipt: receipts/security-review.md. |
