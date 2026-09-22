# Security Review Receipt

Revision: 23

Status: encrypted edge-credential cutover and 200/403/200/200 proxy proof,
initial-owner authorization, no-payment commerce smoke, and R2 media proof
passed on the temporary Pages origins. The prior storefront plaintext
`EDGE_SECRET` incident was remediated by holder rotation; the Resend Emails row
shows the matching test subject as Delivered and the holder received it in spam.
Remote admin source release and normal-inbox placement remain pending. Findings
below retain their dated historical context.

Reviewed:

- admin/public/_redirects
- README.md
- skills/site-deployment-browser/SKILL.md
- skills/site-deployment-browser/references/deployment-value-refresh.md
- docs/deployment-guide.html
- docs/environment-configuration.md
- specs/changes/deployment-dashboard-ledger/**

Revision 14: scanned the reusable skill/reference for this installation's
account, UID, project, branch, hostname, email, token name, and dated receipt.
The skill derives targets anew; actual non-secret values remain in the
installation-specific ledger. This documentation edit does not close the live
plaintext edge-credential incident or authorize using its old value.
The scoped governance and skill validators, `git diff --check`, and full
`go run ./server/tools/verify` passed. The encrypted-row repair, holder's
R2 key handoff, verified mail domain, and delivery proof remain open.
The reusable skill now warns that a Text row can leak its literal through a
tool's automatic accessibility output even without a reveal click; type audits
must suppress/redact values or be handed to the holder.

Findings:

- Severity: high (open credential incident)
  Evidence: On 2026-09-19 the current storefront Pages Production table still
  displayed `EDGE_SECRET` as plain **Text**, while storefront `DATABASE_URL`
  and admin Pages `EDGE_SECRET` displayed encrypted **Secret**. Its plaintext
  value was exposed by the provider UI during the audit and is deliberately
  omitted from every repository artifact and user-facing report.
  Risk: A dashboard reader may acquire the hop credential; four healthy
  endpoint statuses do not establish safe secret storage.
  Control: Stop further deployment. The holder creates a new credential
  off-chat and directly enters it into Railway and both Pages projects,
  removing/recreating the storefront Text row as Secret. The emergency
  cutover does not preserve the exposed value in `EDGE_SECRET_PREVIOUS`; a
  short API interruption is acceptable. After all three fresh deployments,
  recheck encrypted types and health 200 / direct 403 / both Pages 200.

- Severity: none
  Evidence: A valid Supabase Auth login initially had no matching
  `public.staff_members` record and was denied. With current explicit user
  authorization, exactly one active owner link was created for the exact Auth
  user ID; a fresh public admin session showed `owner` and protected controls.
  Risk: Email fallback, a client-side role claim, or automatically promoting
  an Auth account would create an impersonation/privilege-escalation path.
  Control: The resolver remains exact-ID-only and active-role-based. The
  updated skill and guides require current confirmation for first-owner
  creation or promotion and require a fresh server-visible role check; no
  password, token, or connection secret was recorded.
- Severity: none
  Evidence: Railway deployment `c904769a-d36e-4a8b-8581-251f56d79b7b`
  succeeded; `/healthz` returned `200`; an uncredentialed direct product API
  request returned `403`; storefront and admin Pages product API requests each
  returned `200` after their separate retry deployments.
  Risk: Treating any one of these responses as full routing proof would mask a
  missing edge credential or an unprotected origin.
  Control: The evidence and operator sheet require all four outcomes as one
  proof set and do not turn a saved variable name or a Pages response alone
  into secret-handling authority.

- Severity: none
  Evidence: The owner created a clearly labelled active smoke product. The
  storefront's initial static fallback showed no products before hydration,
  then the curatory island fetched same-origin `/api/products`, displayed the
  product, added it to the cart, and reached the checkout contact step.
  Risk: Treating the pre-hydration static fallback as a failed publication
  could lead an operator to rotate credentials or redeploy unnecessarily.
  Control: The runbook now requires waiting for storefront hydration and
  checking the Pages product endpoint before diagnosing a catalog failure.
  The static Pages render/deploy-hook path remains for refreshing no-JS/SEO
  output; no order, customer data, or payment was submitted during the smoke.
- Severity: low (open release boundary)
  Evidence: The local source removes `admin/public/_redirects` and the local
  build emits no redirect asset, but admin Pages deployment
  `87d0566f-ce84-4828-9699-97f1fa1c3997` built remote commit `1665e08` and
  still reported the old `/* /index.html 200` infinite-loop warning.
  Risk: The released admin source has not yet demonstrated the intended Pages
  SPA fallback without the malformed rule.
  Control: The user's later `staging` release must include the deletion; then
  verify the warning is absent and a representative deep link returns `200`.

- Severity: none
  Evidence: The storefront renderer and admin Vue build remain separated:
  only the storefront build may receive `DATABASE_URL`, while the admin build
  never receives it. Both Pages Functions use an independent `API_ORIGIN` and
  the operator-entered edge credential.
  Control: The source-to-destination sheet, Secret-versus-Text gate, endpoint
  map, and no-payment smoke boundary remain the release procedure.

- Severity: medium (pending configuration)
  Evidence: On 2026-09-19 the bucket had exact admin-origin PUT/Content-Type
  CORS, a test-only public development URL, and an active scoped Object Read
  & Write Account API token. It still had zero objects, while the active
  Railway service had no R2 runtime-variable names. The media
  client obtains a presigned URL through the protected API and sends a direct
  browser `PUT` with `Content-Type`.
  Risk: A wildcard CORS rule, bucket-wide administrator token, token copied to
  Pages, or premature `r2.dev` production use would expand access beyond the
  image-upload need. Without the required four Railway values, storage stays
  disabled and the API correctly returns the configuration error.
  Control: The updated handoff uses a bucket-scoped Object Read & Write token
  stored only as Railway Secrets; a single admin Pages origin with `PUT` and
  `Content-Type`; a public base URL only on Railway and the storefront build;
  and an actual verified-image/public-fetch proof before calling the path live.

- Severity: none (documentation control)
  Evidence: The R2 operator material now maps each of the five values to the
  observed Chinese **and English** dashboard labels: account ID under
  **概觀（Overview）-> 帳戶詳細資訊（Account details）**; a bucket-scoped
  **Object Read & Write** Account API token under **管理 API 權杖（Manage API
  Tokens）-> 帳戶 API 權杖（Account API Tokens）**; bucket name under
  **貯體（Buckets）**; public URL under bucket **設定（Settings）**; and CORS
  under **CORS 原則（CORS Policy）**. It also distinguishes Railway Text,
  Railway Secret, and storefront-build-only destinations.
  Risk: Treating all `R2_*` variables as secrets, copying the key pair to
  Pages, using a broad user token, or omitting CORS can leave storage disabled
  or disclose credentials to a browser-facing surface.
  Control: `R2_ACCOUNT_ID`, bucket, and public URL are explicitly Text;
  only the key pair is Secret and holder-entered. The exact CORS rule limits
  the current admin origin to `PUT` with `Content-Type`; `r2.dev` is marked
  test-only, while a production public origin requires the separate
  custom-domain confirmation.

- Severity: medium (pending configuration)
  Evidence: The existing Go bootstrap switches from its log sender to Resend
  only when `RESEND_API_KEY` is present. `RESEND_FROM` controls the provider
  sender; `CONTACT_NOTIFY_TO` is only the management recipient for persisted
  public contact inquiries, while commerce messages use the order customer
  email. At that historical checkpoint, the Resend account was signed in, but its
  Domains and API Keys lists are empty; no sender domain or delivery exists.
  Risk: A full-access or unscoped key, unverified/mismatched sender domain,
  API key copied to Pages, or a blind retry after a contact send failure could
  enable unwanted sending, leak a credential, or duplicate a saved inquiry.
  Control: The handoff requires an owned Verified domain, generated DNS rows
  in the authoritative Cloudflare zone, domain-scoped **Sending access**, a
  Railway-only API-key Secret, exact-domain sender Text, protected
  server-only contact recipient, and current confirmation before the one
  external delivery test. Resend automatic Cloudflare authorization and every
  DNS/key/email mutation remain holder-confirmed actions.

Validation:

- Revision 16: an authorized non-sensitive smoke upload reached authenticated
  presign, R2 PUT, API verify, product association, and a public image loaded
  at 864 × 1152. The initial `.png` demo filename held JPEG bytes and was
  rejected because browser MIME and upload extension differed. A disposable
  correctly named JPEG copy was used without changing the source or bypassing
  server validation. The documentation stores no presigned URL, complete
  object key, R2 key, or private recipient. A proposed contact send was
  stopped before submission because its actual management recipient was not
  explicitly confirmed; there is no email-delivery proof. The reusable skill
  and installation docs were scanned for personal account/email/UID labels,
  connection strings, API-key shapes, and secret literals; owner-specific
  labels were removed while public test URLs and non-secret receipts remain.

- A focused credential-pattern scan covers the ledger artifacts and the skill
  reference; no connection-string credential, private token, or API-key
  literal is recorded.
- The provider workflow keeps runtime literals, GitHub credentials, and
  cookies out of repository files, screenshots, logs, and reports.
- The R2 procedure records public identifiers and public URLs only. Access
  keys, secret keys, and presigned URLs remain holder-only and must never be
  copied into the agent, Pages variables, repository, or evidence.
- The Resend procedure records only dashboard labels, variable names, and
  non-secret states. API keys, private recipient addresses, DNS values,
  provider cookies, and email-share URLs stay out of the repository, browser
  reports, and evidence.
- `npm --prefix admin run typecheck` passed; `npm --prefix admin run
  build:only` had already passed after the rewrite removal and emitted no
  `admin/dist/_redirects` asset.
- `npm --prefix site/themes/curatory run build:check` passed after rerunning
  outside the sandbox that had blocked esbuild child-process startup.
- `go test ./server/internal/auth ./server/internal/bootstrap
  ./server/internal/modules/commerce`, `git diff --check`,
  `go run ./server/tools/speccheck deployment-dashboard-ledger`, and
  `go run ./server/tools/verify` passed after the revision 10 documentation
  update.
- After the revision 11 R2 field-map update, `git diff --check`,
  `go run ./server/tools/speccheck deployment-dashboard-ledger`,
  `go run ./server/tools/scopecheck`, targeted config/bootstrap/media/
  commerce tests, `go run ./server/tools/verify`, and the deployment skill's
  UTF-8 validator passed. These checks validate the controlled artifacts and
  the disabled-until-configured code path; they do not replace live R2 upload
  evidence.
- After the revision 12 Resend handoff update, focused
  config/bootstrap/mail/contact/commerce tests, admin typecheck,
  `git diff --check`, the deployment skill UTF-8 validator,
  `go run ./server/tools/speccheck deployment-dashboard-ledger`,
  `go run ./server/tools/scopecheck`, a credential scan including the Resend
  key prefix, and `go run ./server/tools/verify` passed. They validate code
  wiring and documentation controls; they do not replace holder login,
  domain/DNS verification, secret entry, deployment, or authorized delivery
  evidence.
- Revision 16 checks: controlled speccheck, scopecheck, skill UTF-8 validator,
  `git diff --check`, and full `go run ./server/tools/verify` passed. A scoped
  file scan across README, docs, the deployment skill, and this controlled
  change found no known holder names/domains/account refs, connection strings,
  Resend/Supabase/AWS-style key literals, private-key headers, or JWTs. The
  only email-shaped string is a reserved `example.com` sender example in
  README. Public Pages test URLs and commit/deployment IDs are deliberately
  retained; these are not access credentials.
- Revision 17: after the holder identified and authorized the management
  recipient, one labeled contact POST returned HTTP 201 after fresh Railway
  deployment. No destination literal, API key, message share URL, or provider
  credential is in these artifacts. Because the handler may use a log sender
  when its key is empty and Resend's dashboard timed out, keep provider event
  and inbox proof pending. The inquiry is persisted before the send call;
  never blindly repeat this synthetic POST.
- Revision 18: the holder independently reported the exact marked notification
  in their spam folder. This resolves mailbox-arrival uncertainty but not
  Resend Emails/Logs observation or normal-inbox placement. No second POST
  was made, and no recipient literal, provider key, or email-share link was
  stored. A single spam result does not identify its cause.
- Revision 19: the holder supplied a Resend Emails list screenshot with the
  matching controlled-test subject marked Delivered. This closes the provider
  event acceptance alongside holder-confirmed spam-folder receipt; it makes
  no normal-inbox placement claim. The screenshot and pictured private address
  were not copied into the repository or receipts. No new message was sent.
- Revision 20: read-only Resend Insights and public/Cloudflare DNS inspection
  showed a missing DMARC TXT record for the actual sending domain. The code
  already sends both HTML and plain text; no unsupported template change was
  made. The prefilled `p=none` DNS form remains unsaved, so no new DNS/security
  policy is live. An organization-domain DMARC policy can affect subdomains;
  publication needs explicit holder confirmation. No commit, DNS mutation,
  provider key, private recipient, full message header, or repeat send occurred.
- Revision 21: after the holder's authorization, one monitoring-only DMARC TXT
  was saved in the selected Cloudflare sending zone and found through public
  DNS. This does not change R2 API credentials, server code, or the prior mail
  result. A stricter DMARC policy and any new external send remain outside this
  action. Installation docs record only the public DNS policy and status; the
  reusable skill contains no current site's domain, account, recipient, key,
  or deployment receipt. No Git commit or repeat contact submission occurred.
- Revision 22: the holder separately confirmed the private management
  destination and authorized one new synthetic contact test. A single POST
  returned 201 and its Resend event was Delivered. Fresh Resend Insights
  recognized the monitoring DMARC policy; receiver authentication and mailbox
  placement remain unverified. The marker is synthetic, and the repository
  records no recipient address, API credential, full header, or share URL.
  No retry, order, payment, new R2 token, or Git commit occurred.
- Revision 23: read-only public checkout data showed an empty payment-method
  list despite two shipping choices. An offline method must not be silently
  enabled: it becomes public, and a real order would persist private contact
  and shipping data, decrement inventory, and attempt notification. The agent
  stopped before any admin-token extraction, database write, order, or payment.
  The reusable skill distinguishes checkout-first-step smoke from a placed
  order, and the installation guide records the precise pending owner choice.
- Revision 24: the owner explicitly authorized a public test-only manual
  no-charge method and one holder-entered marked order. The Supabase SQL
  Editor read only payment setting fields and the non-sensitive smoke product
  baseline; a collision-guarded insert returned exactly one sandbox/ready/
  enabled/zero-fee method. The public method/bootstrap and a side-effect-free
  server quote confirmed availability. No admin bearer token, database
  credential, customer field, order access token, or key was copied. The
  browser handoff requests the holder's controlled contact/delivery data and
  a single marked submit; until that happens no stock/order/mail effect is
  claimed. Public marketing copy advertising unconfigured payment types is
  called out as a pre-launch risk. No Git commit or real charge occurred.
- Revision 25: a holder-supplied order page permitted narrow order/item,
  stock and notification metadata readback. Exactly one pending/unpaid manual
  order persisted and decremented one smoke unit. No name, email, phone,
  address, raw buyer note, order access token, full order ID, message body, or
  recipient was copied into repository artifacts. The one order-placed log
  was skipped for a missing template; production template metadata was empty.
  Admin Retry would copy that log's empty rendered subject/body and cannot
  recover it. No new template, second order, external mail, paid status,
  fulfillment, restock, or Git commit was performed. A truthful template and
  controlled-recipient/second-order confirmation are required first.
- Revision 25 follow-up: the holder explicitly confirmed the exact normal
  `order_placed` subject/body and enabled it through the authenticated owner
  admin. Only one code row now appears enabled. The message uses supported
  placeholders and does not claim payment or dispatch. The recipient for
  future order mail comes from each customer's checkout email, not the
  management contact setting. No recipient, rendered PII, secret, or order
  access token was copied into the repository; saving the template sent no
  email. The old skipped log remains unsent. A new order and external mail
  are separate, still-unperformed side effects at that point.
- Revision 25 second-order follow-up: the holder subsequently submitted a
  controlled order after the normal template was enabled. Read-only queries
  limited to order/item/inventory and notification metadata found two orders
  total, latest pending/unpaid `manual_test`, one item, total NT$61, and stock
  2 to 1. One `order_placed` log is `sent:mail`, with nonempty rendered content
  and no logged error. The matching Resend subject showed Delivered and its
  identifier matched the holder's open order page in memory. The proposed
  buyer-note marker was absent; do not use it as corroboration. No full order
  ID, access token, customer/recipient/address, rendered body, or provider
  credential was copied into the repository. Delivered does not prove inbox
  placement, which remains awaiting holder observation. No third order,
  replay/retry, payment, fulfillment, restock, source release, or Git commit.
- Revision 25 mailbox follow-up: the holder's spam-folder screenshot visibly
  matched the second order's normal subject and total and also listed both
  prior contact-test messages. The second-order `sent:mail` and Resend
  Delivered outcomes now have actual mailbox-arrival proof, specifically
  **spam** placement. The post-DMARC contact message also reached spam;
  receiver authentication verdicts and filtering reason remain unknown.
  The screenshot and its full order ID, recipient/sender, and message excerpt
  were not stored in the repository. No mail replay, provider configuration,
  DNS change, third order, payment, fulfillment, or Git commit occurred.
