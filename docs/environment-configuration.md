# Environment configuration

How local development and online deployment get their configuration, and why
a file in this repository can never supply a production value.

## The boundary

`APP_ENV` is read from the **process environment** before any dotenv file is
opened (`server/internal/config/config.go`). That single ordering is the whole
mechanism:

| Runtime | Authoritative source | Repository dotenv files |
|---|---|---|
| Local Go API, tools, renderer | process env → `.env.development.local` → `.env` | read |
| Railway Go API | Railway process environment, `APP_ENV=production` | **ignored** |
| Cloudflare Pages Go render/build | Pages build environment, `APP_ENV=production` | **ignored** |
| Local admin / public-theme Vite | process env + local dotenv profile | read |
| Production admin / public-theme Vite build | provider build environment | **ignored** |

Consequences worth stating plainly:

- A dotenv file **cannot** put a process into production mode. Only the
  process environment can set `APP_ENV=production`.
- A dotenv file **cannot** rescue a production deployment that is missing a
  required value. Validation fails closed instead
  (`Config.Validate`), so a misconfigured deploy stops rather than silently
  falling back to SQLite or dev authentication.
- There is no `.env.production` to deploy. Do not create one.

## Current provider handoff (2026-09-20)

The earlier walkthrough changed in the provider dashboards. Treat
this checkpoint as the latest observed state; the detailed, repeatable
[source → destination sheet](../skills/site-deployment-browser/references/deployment-value-refresh.md#operator-worksheet)
contains the exact pages and bilingual controls.

This document is the **current installation's ledger**, not the reusable
skill's defaults. The temporary test origins are
[storefront](https://agent-safe-site-starter.pages.dev/) and
[admin](https://agent-safe-site-starter-admin.pages.dev/). The media image
path is proved on these origins. A holder-provided mailbox screenshot shows
the second order notification and both contact tests in spam; their matching
Resend Emails rows showed `Delivered`. Normal-inbox placement remains open. The
[de-identified skill](../skills/site-deployment-browser/SKILL.md) and its
worksheet require another installation's agent to discover its own values.

| Layer | Observed now | Still required |
|---|---|---|
| Railway + Pages proxy | On 2026-09-20 the holder confirmed a fresh common edge credential and encrypted **Secret／秘密** in both Pages Production Functions. Railway's 9 staged changes were applied; the new deployment `<railway-deployment-id>` became Active/Online. It now lists 19 service variables, including the database, edge, five R2, and three Resend names, with no staged changes. Storefront retry `<storefront-pages-deployment-id>` and admin retry `<admin-pages-deployment-id>` both succeeded from `staging` commit `bac7c78`. After both succeeded: origin health 200; uncredentialed direct products 403; storefront/admin Pages products 200/200. | This proves the encrypted edge cutover and proxy paths, not the correctness of unseen R2/Resend credential values. Recheck all four codes after the next release or value rotation. |
| Cloudflare R2 | Bucket `<r2-bucket-name>` has exact admin-origin PUT/Content-Type CORS, a test-only public `r2.dev` base, and a bucket-scoped Object Read & Write token. After Railway and storefront Pages deployment, an authorized non-sensitive smoke image completed authenticated presign → browser PUT → API verify, was saved on the active smoke product, and loaded on the storefront from the public R2 hostname (`naturalWidth=864`, `naturalHeight=1152`). No key or presigned URL was inspected or recorded. | Media path proved on the temporary Pages/R2 URLs. For a custom-domain cutover, update public base/CORS and re-prove; this does not prove mail delivery. |
| Resend | The owner-selected sending domain is **Verified**. The first authorized labeled contact POST returned `201`; the holder found that message in **spam**, while Resend Emails showed **Delivered** and its Insights warned **No DMARC record found**. After a monitoring-only `_dmarc` TXT was published, the holder explicitly confirmed the current management recipient and authorized **one new** test. Storefront `/api/products` returned `200`, then exactly one fresh `POST /api/contact` with marker `DEPLOYMENT_DMARC_20260920_B1` returned `201`. Resend Emails marked that new subject **Delivered**; its Insights now puts **Include valid DMARC record** under **Doing great** and displays `p=none`. A holder-provided screenshot now places **both** contact-test messages in spam. No private recipient or key is recorded. | The new DNS policy is publicly visible and recognized by Resend, but the post-DMARC message still landed in spam. This does **not** identify the receiving mailbox's `dmarc=pass` verdict or a root cause. Ask for a redacted SPF/DKIM/DMARC pass/fail summary and spam reason only; do not resend the persisted inquiry or change DNS again on this evidence alone. |
| Checkout → order | The initial 2026-09-20 bootstrap had two shipping methods and **zero** payment methods. With the owner's consent, one guarded insert into the selected Supabase DB created `pm-manual-smoke` / `manual_test`, labeled `測試用人工付款（不請款）`, sandbox/ready/enabled/zero fee. Pages API and bootstrap exposed only that row. A side-effect-free quote for `s2s` was NT$1 + NT$60 shipping = NT$61. The first holder-entered order used another shipping method: one smoke item, `pending`/`unpaid`, `manual_test`, subtotal NT$1 + shipping NT$120 = NT$121; stock 3 → 2. Its suggested buyer-note marker was absent; the holder's order page identified it. After enabling the normal template, the holder submitted a second controlled order. Narrow readback found **two** total orders; the latest has one smoke item, `pending`/`unpaid`, `manual_test`, subtotal NT$1 + shipping NT$60 = NT$61; stock 2 → 1. The second suggested marker was also absent; the holder's exact order page was compared with the Resend subject in memory without recording its full ID. `sold_count` is not proof of pending orders. No private field or access token was recorded. | First order: `skipped/template not found`; it was not retroactively sent and Retry copies empty content. The owner approved one enabled truthful `order_placed` template before the second order. Second order: exactly one `order_placed` log, `sent:mail`, rendered subject/body nonempty, zero logged errors. Resend Emails shows the matching order subject as **Delivered**; the holder's screenshot independently shows that same order notification in **spam**. This proves automatic sending and mailbox arrival, **not** normal-inbox placement. No third order, gateway charge, paid status, fulfillment or restock occurred. Disable/replace the publicly visible test payment before real-payment launch; static copy still advertises unverified card/LINE Pay/COD. |
| Admin SPA source | A fresh browser session loaded `/res/minimal-cart-products` with the 商品 heading and a public GET returned 200 on 2026-09-20. The local deletion of `admin/public/_redirects` is still uncommitted, and this Pages retry uses the old `bac7c78` source. | Do not treat a single working deep link as release of the source repair. After the source change is reviewed/published to `staging`, rebuild admin Pages and check the redirect warning and a clean-session deep link again. |

### Temporary Pages test-loop verdict (2026-09-20)

**Proved within the no-gateway test scope:** authenticated owner access and
active product publishing → R2 upload/verification and storefront image →
hydrated live catalog/cart/checkout → server quote and holder-submitted
`manual_test` unpaid order → persisted order and one-unit stock decrement →
enabled `order_placed` template → one backend `sent:mail` event → matching
Resend `Delivered` event → actual mailbox arrival in **spam**. The earlier
order's missing-template `skipped` log remains unsent. Evidence for these
distinct hops is recorded in the table above; neither a static fallback nor
a successful 200 alone substitutes for the later order/mail observations.

**Outside or not yet accepted:** gateway charge, paid status, fulfillment,
normal-inbox placement, custom site/media domains, updated static/SEO catalog
snapshot, and the still-local admin redirect-source release. The public test
payment option must be disabled or replaced and inaccurate static payment
claims corrected before a real launch. No third order or mail replay is
needed to state the test-loop result.

No provider key, old edge value, connection string, or private recipient is in
this repository. A 200 proxy response does not validate secret storage, R2,
or email. `r2.dev` is for this test URL only; switch to an approved custom
R2 hostname before a production media cutover.

### Repeatable test-only checkout handoff

The admin **商店（Store）→ 付款方式（Payment methods）** page can edit an existing
method, but it cannot create one from an empty list. For a new installation,
read the public `/api/storefront/bootstrap` and `/api/payment-methods` first.
If both have no method and the owner approves an unpaid public test option,
open the **correct Supabase project → SQL Editor／SQL 編輯器 → New query／新增查詢**;
confirm **Primary Database／主要資料庫**, then read only the seven non-sensitive
payment columns. If a suitable row exists, use the admin edit form instead.
For an empty table, use a guarded single-row insert into
`public.payment_methods` with a unique test ID, `method=manual_test`, explicit
`測試用人工付款（不請款）` label, `sandbox`, `ready`, `enabled=true`, zero fee, and
current Unix seconds for `updated_unix`. Query returns the inserted non-secret
fields; a generic SQL success banner is not proof. This is an operator action
against live DB configuration, not a secret or a migration. Verify Pages public
payment API and bootstrap show exactly one method, then call `/api/quote` with
one current product SKU, one available shipping method, and `manual_test`;
confirm the backend amount and zero payment fee. Do not assume the static
credit-card, LINE Pay, or COD marketing text is true.

Before a mail-inclusive checkout, open the authenticated admin
**商店（Store）→ 通知模板（Notification templates）**. Unlike payment methods,
this resource has a **新增（Create）** action. A new production database may
have no rows because `server/tools/dev` seeds templates only for local
development. Read only template `code`, enabled flag, and booleans for
nonempty subject/body. After the owner approves the actual customer-facing
copy, add one enabled `order_placed` template with a truthful subject/body
using supported `{{customer_name}}`, `{{order_id}}`, and `{{total}}` placeholders.
The regular template can omit test language, but it must not conflate order
creation with payment or shipment. The local dev seed's “盡快為您出貨” wording
is not appropriate for an unpaid order. This installation's owner approved
the normal wording, and the authenticated admin list showed exactly one enabled
`order_placed` row on 2026-09-20. A DB template does not require a new
Railway/Pages deployment and does not send a previous skipped order.

Next open [the public checkout](https://agent-safe-site-starter.pages.dev/checkout/)
with the marked product in the cart. The holder, not the agent, supplies their
own controlled name, `09`-format mobile, email, and actual test delivery
details; add a unique non-private marker to the buyer note. Submit once, then
inspect the order result and the authorized admin order list by marker.
Record only order status, `unpaid`, quantity/total, stock delta, and
notification outcome; avoid persisting a full order ID when a dated count and
status suffice. If the submit times out, check whether an order was
already created before any retry; the browser generates an idempotency key per
checkout view. An order persists private data and decrements stock even if
best-effort mail fails. If the log says `skipped` because the template is
missing, the admin Retry action cannot recover it: retry copies the original
empty rendered subject/body. Do not retry it blindly or forge a log. A
additional holder-submitted order needs new consent and another stock decrement.
For the verified second order, the buyer-note marker was absent; match the
holder's exact order page to the new Resend subject in memory and retain only
the count/time/status, never the full order ID. Treat DB `sent:mail`, Resend
`Delivered`, and holder-confirmed inbox/spam as three separate observations.
Cancellation, restock, paid status, fulfillment, and
real payment require separate decisions. Do not put contact fields, order
access token, Resend recipient, or provider credentials into this file.

### Mail deliverability follow-up (DNS published)

The earlier message was accepted by Resend and reached spam. Its **Emails →
message → Insights** report specifically said **No DMARC record found** at
the time; Cloudflare's DNS list and public lookup then confirmed the gap.
That historical warning does not prove DMARC alone caused spam placement.
The current mail implementation already sends a plain-text alternative
alongside HTML.

On 2026-09-20, after the holder authorized this DNS change, the sending
domain's Cloudflare **網站（Websites）→ DNS → 記錄（Records）** received one
**TXT**, **名稱（Name）** `_dmarc`, **內容（Content）**
`v=DMARC1; p=none;`, **TTL** 自動（Auto）. The new row was visible in the zone;
`Resolve-DnsName -Name _dmarc.<sending-domain> -Type TXT` returned the same
value in a fresh public lookup. This is monitoring-only: it does not request
quarantine/rejection or guarantee inbox delivery. No R2 API token, Railway
variable, Pages deployment, or mail code changed for this DNS-only step.
Do not invent a `rua` mailbox or tighten the policy before checking every
legitimate sending source. The separately authorized single new contact POST
returned `201`; Resend marked its message `Delivered`, and fresh Insights
lists **Include valid DMARC record** under **Doing great** with `p=none`.
This is Resend's DNS-policy analysis, not a receiver-side authentication
verdict. A later holder-provided screenshot locates the new message in spam,
alongside the earlier contact test and the second-order notification. Receiver
SPF/DKIM/DMARC pass/fail and spam reason remain unknown; request only a
redacted summary, not full headers or addresses. Do not send another copy or
assume DMARC was the sole cause of either classification.

Sources: [Resend DMARC implementation](https://resend.com/docs/dashboard/domains/dmarc),
[Resend subdomain policy discovery](https://resend.com/blog/how-dmarc-applies-to-subdomains).

## Local development

```bash
cp .env.development.example .env.development.local
# edit .env.development.local
go run ./server/tools/dev
```

Precedence is **process environment > `.env.development.local` > `.env`**. The
first source that supplies a name wins; nothing overwrites a value already in
the process environment.

`.env` is the legacy location. It still works — if you already have one, you
do not need to do anything. To move over, copy the values you want into
`.env.development.local`; anything you leave behind is still picked up from
`.env`. Nothing in this repository will rename, rewrite, or read out your
existing `.env`.

All real profiles (`.env`, `.env.local`, `.env.development`,
`.env.development.local`, `.env.production`, `.env.production.local`) are
git-ignored. The tracked `*.example` files hold placeholders only.

### Ephemeral local PostgreSQL gate

To run the full PostgreSQL 16 live integration test suite locally without
setting up an external PostgreSQL service or modifying local `.env` files:

```bash
go run ./server/tools/local-postgres-gate
```

Key characteristics and operational boundaries:

- **Single command & parity**: Starts an isolated PostgreSQL 16 instance on
  loopback (`127.0.0.1`) matching CI, injects a child-scoped `TEST_DATABASE_URL`
  into `postgres-live-gate`, and runs every required `TestPostgresLive*` test.
- **Process isolation**: `TEST_DATABASE_URL` exists only inside the child test
  process. No credentials or endpoints are written to `.env` files, repository
  files, or logs.
- **Automatic cleanup**: When tests complete (or on failure / interrupt), the
  child PostgreSQL process is stopped and all temporary runtime/data directories
  are removed automatically.
- **Binary caching & offline behavior**: On the first run, PostgreSQL 16
  binaries are downloaded to the user cache directory
  (`%LocalAppData%/embedded-postgres-binaries` on Windows,
  `~/.cache/embedded-postgres-binaries` on Linux). Subsequent runs execute
  offline using the cached binaries. First-run execution requires network
  connectivity to download the binary archive.
- **Port collisions**: Defaults to loopback port `5433` to avoid collisions
  with any default PostgreSQL service on port `5432`. If port `5433` is
  occupied, use the `-port` flag (e.g. `go run ./server/tools/local-postgres-gate -port 5434`).
- **Validation boundary**: Local test success verifies code against a real
  PostgreSQL 16 engine, but does not substitute for CI run receipts or production
  deployment validation.

## Initial admin authorization (not an environment variable)

Supabase Auth and backend staff authorization are separate. A successful
Supabase Auth login supplies a verified user ID, but the Go API grants admin
capabilities only when it finds the exact ID in
`public.staff_members.supabase_user_id`, with `status = active` and a
supported `role_label` such as `owner`. It deliberately never falls back to
email matching, Auth metadata, or a browser-supplied role.

For the first owner, open **Supabase -> Authentication -> Users** and copy
the intended account's user ID. In **Table Editor -> public.staff_members**,
check whether an active row already uses that exact ID. If not, stop for
current, explicit authorization before creating one unique row with the
intended display name/email, `role_label = owner`, that exact user ID,
`status = active`, and a current `updated_unix`. This modifies production
authorization; it is not implied by adding environment variables or by the
presence of an Auth account.

Reload the public admin Pages app in the same signed-in session. Proceed to
the product smoke test only after the server reports `owner` and exposes
protected controls. If it reports no permission, inspect the exact ID,
selected Supabase database, migrations, and staff status; do not bypass the
server-side resolver.

## Storefront catalog freshness

The curatory storefront deliberately keeps a static Pages fallback and
interactive Vue islands. The static HTML generated by `make site` reflects
the catalog at build time, while the hydrated `ShopGrid` island requests
same-origin `/api/products` through the storefront Pages Function. A newly
active product can therefore appear in the live shop/cart/checkout flow
after hydration even if the first HTML snapshot still says there are zero
products.

For an authorized smoke test, wait for the hydrated live product card/count,
then add it to cart and open checkout. Do not treat the initial static
fallback as an API failure. `CF_DEPLOY_HOOK_URL` plus
`go run ./server/tools/publish` is the separate path for refreshing the
static no-JavaScript/SEO snapshot; it is not required for the hydrated
shopping-flow check. Do not create or delete test products without the
user's current authorization.

For the project-specific record of the branch handoff, missing-variable
gate, protected proxy proof, first-owner recovery, hydration behaviour, and
the still-pending remote admin source release, read
[`deployment-value-refresh.md`](../skills/site-deployment-browser/references/deployment-value-refresh.md#operator-worksheet).

## Deployment ownership

In the table below, **Pages (build)** means the storefront project whenever a
value is needed by the Go renderer. The separate admin Vue build receives only
its browser-safe auth configuration and `ADMIN_API_BASE` behavior; it does not
inherit `DATABASE_URL`, `DB_DRIVER`, or other renderer-only values.

| Variable | Railway (Go API) | Pages (build) | Pages (fn env) † | Browser bundle | Secret |
|---|:--:|:--:|:--:|:--:|:--:|
| `APP_ENV=production` | ● | ● | | | |
| `HTTP_ADDR` | ● | | | | |
| `SITE_ORIGIN` | ● | | | | |
| `EDGE_SECRET` | ● | | ● | | ● |
| `EDGE_SECRET_PREVIOUS` | ● | | | | ● |
| `API_ORIGIN` | | | ● | | |
| `PUBLIC_SITE_URL` | ● | ● | | | |
| `PUBLIC_API_BASE` | ● | ● | | | |
| `ADMIN_API_BASE` | | ● | | ● | |
| `DB_DRIVER` | ● | ● | | | |
| `DATABASE_URL` | ● | ● | | | ● |
| `DB_MAX_OPEN_CONNS` | ● | | | | |
| `DB_MAX_IDLE_CONNS` | ● | | | | |
| `DB_CONN_MAX_LIFETIME` | ● | | | | |
| `DB_CONN_MAX_IDLE_TIME` | ● | | | | |
| `AUTH_MODE` | ● | ● | | ● | |
| `SUPABASE_URL` | ● | ● | | ● | |
| `SUPABASE_PUBLISHABLE_KEY` | ● | ● | | ● | |
| `SUPABASE_VERIFIER_MODE` | ● | | | | |
| `AUTH_GOOGLE_ENABLED` | | ● | | ● | |
| `AUTH_LINE_ENABLED` | | ● | | ● | |
| `DEV_AUTH_TOKEN` | | | | | ● (never in production) |
| `R2_ACCOUNT_ID` | ● | | | | |
| `R2_ACCESS_KEY_ID` | ● | | | | ● |
| `R2_SECRET_ACCESS_KEY` | ● | | | | ● |
| `R2_BUCKET` | ● | | | | |
| `R2_PUBLIC_BASE_URL` | ● | ● | | | |
| `RESEND_API_KEY` | ● | | | | ● |
| `RESEND_FROM` | ● | | | | |
| `CONTACT_NOTIFY_TO` | ● | | | | ● |
| `CF_DEPLOY_HOOK_URL` | ● | | | | ● |

† **Pages Functions environment** — runtime variables on each Pages project
(Settings → Environment variables), consumed by `functions/api/[[path]].js`
when a request hits `/api/*`. They never enter a build output or browser
bundle. `API_ORIGIN` is the Railway public URL the proxy fetches;
`EDGE_SECRET` is the same hop credential the Railway side enforces — set it
on the storefront project AND the admin project.

## R2 media deployment

The media path is deliberately not a browser-to-database path. An authorized
admin session asks the Go API for a short-lived upload URL, the browser PUTs
the bytes to R2, the API verifies the object, and a product may then reference
the verified image. The path remains disabled until Railway has **all four**
runtime values `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`,
`R2_SECRET_ACCESS_KEY`, and `R2_BUCKET`; a saved name without a fresh Railway
deployment is not proof.

The following names match the current Chinese Cloudflare dashboard. The
observed bucket is `<r2-bucket-name>`; replace only the bucket
name if a future deployment deliberately uses another bucket.
Every navigation label below is written as **中文（English）**: use the left
label on a Chinese dashboard and the parenthesized label after switching the
dashboard to English.

| Railway variable | Cloudflare 後台（中文／English） | 複製／輸入什麼 | Railway 型別與去處 |
|---|---|---|---|
| `R2_ACCOUNT_ID` | **儲存空間和資料庫（Storage & databases）** → **R2 物件儲存（R2 Object Storage）** → **概觀（Overview）** → **帳戶詳細資訊（Account details）** → **帳戶 ID（Account ID）** | 顯示的帳戶 ID | **Text**。這是帳戶識別碼，不是密鑰。 |
| `R2_ACCESS_KEY_ID` | **R2 物件儲存（R2 Object Storage）** → **概觀（Overview）** → **管理 API 令牌（Manage API Tokens）** → **帳戶 API 權杖（Account API Tokens）** → **建立 Account API 權杖（Create Account API Token）** | 建立完成頁的 **Access Key ID** | Railway **Secret**。 |
| `R2_SECRET_ACCESS_KEY` | 同一個建立完成頁 | 只顯示一次的 **Secret Access Key** | Railway **Secret**；不要貼到聊天室、截圖或文件。 |
| `R2_BUCKET` | **R2 物件儲存（R2 Object Storage）** → **概觀（Overview）** → 表格欄位 **貯體（Buckets）** | `<r2-bucket-name>` | **Text**。 |
| `R2_PUBLIC_BASE_URL` | `<r2-bucket-name>` → **設定（Settings）** → **公用開發 URL（Public Development URL）** → **啟用（Enable）**（測試）；正式環境改用 **自訂網域（Custom Domains）** → **新增（Add）** | Cloudflare 顯示的 **Public Bucket URL** 或已連線的自訂圖片網域；移除結尾 `/` | **Text**：Railway 與 storefront Pages 的 **Production build** 各一份。不要設在 admin 或任何 Pages Function。 |

建立 Account API 權杖時，選 **Object Read & Write**，並限制為
**只套用指定 bucket** 的 `<r2-bucket-name>`。兩個 key 是
一組憑證：即使 Access Key ID 看似可辨識，也要和 Secret Access Key
一起存為 Railway Secret。不要用既有的全帳號使用者 token，也不要把這
兩個值設到 Cloudflare Pages。

### 瀏覽器上傳 CORS

在 bucket 的 **設定（Settings）** → **CORS 原則（CORS Policy）** →
**新增（Add）** 貼入以下 JSON 後按 **儲存（Save）**。這是目前 admin Pages
網域所需的最小權限；日後若 admin 換成
自訂網域，只替換 `AllowedOrigins` 的完整 HTTPS origin，不能改成 `*`。

```json
[
  {
    "AllowedOrigins": ["https://agent-safe-site-starter-admin.pages.dev"],
    "AllowedMethods": ["PUT"],
    "AllowedHeaders": ["Content-Type"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3600
  }
]
```

### 套用與驗收順序

1. 持有人先儲存上述 CORS；測試時再在 **公用開發 URL（Public Development URL）** 依 Cloudflare
   提示輸入 `allow` 啟用 `r2.dev`。`r2.dev` 只供測試，正式環境使用
   **自訂網域（Custom Domains）** 並完成 DNS/公開存取確認。
2. 持有人建立 bucket-scoped Account API token，直接把 key pair 填到
   Railway 的兩個 Secret 欄位；再填其餘三個 Text 值。
3. 在 Railway 按 **Deploy**，等待新的 deployment 成功；之後在 storefront
   Pages 的 Production **build** 只新增 `R2_PUBLIC_BASE_URL`，並觸發新的
   storefront deployment。admin Pages 不需要 R2 變數。
4. 以已登入且具 `media.upload` 權限的 admin 上傳一個非敏感測試圖片。
   成功必須依序看到 presign、R2 PUT、API verify、`verified/product-images/`
   物件、商品圖片關聯與前台公開圖片；任何一段缺失都不算 R2 已打通。

常見判讀：API 回「R2 is not configured」表示 Railway 四個 runtime 值至少
一個缺失或尚未部署；瀏覽器 CORS/preflight 失敗表示 origin、`PUT` 或
`Content-Type` 與 CORS 規則不一致；公開圖片 404 通常是
`R2_PUBLIC_BASE_URL`、物件 key 或 storefront build 尚未重新部署。更完整的
逐步重跑與失敗表在
[`deployment-value-refresh.md`](../skills/site-deployment-browser/references/deployment-value-refresh.md#r2-媒體上傳中文後台逐欄位交接)。

## Resend mail deployment

This project already has the production sender wired: when Railway has a
non-empty `RESEND_API_KEY`, bootstrap uses Resend; otherwise it deliberately
uses the local log sender. A live key by itself is not sufficient—production
must also set a sender whose domain is verified in Resend. `RESEND_FROM`
otherwise falls back to the example address in source and will not make a
valid production sender.

Resend currently presents its dashboard in English. The following uses Chinese
explanations with the exact English labels shown in Resend and Cloudflare.
Choose an **owned** domain or a dedicated sending subdomain such as
`mail.example.com`; do not try to verify a Cloudflare Pages `*.pages.dev`
hostname.

| Railway variable | 來源與後台路徑（中文說明／English UI） | 輸入什麼 | Railway 型別與去處 |
|---|---|---|---|
| `RESEND_API_KEY` | Resend **API Keys** → **Create API Key**；名稱可標示為 Railway production，權限選 **Sending access**，若 UI 提供 domain 限制則只選已驗證的寄信網域 | 只顯示一次的 key | Railway **Secret**。只由持有人直接輸入，不可放 Pages、瀏覽器、程式碼、截圖或文件。 |
| `RESEND_FROM` | Resend **Domains** → 已驗證的 domain 詳情；不需要另外建立 sender identity | `品牌名稱 <orders@已驗證網域>`；`@` 後的 domain／subdomain 必須和已驗證項目完全相同 | Railway **Text**，但仍只在 server 使用。它不是 API key，也不能放到 Pages。 |
| `CONTACT_NOTIFY_TO` | 不是從 Resend 取值；由站方指定可收網站詢問的管理信箱 | 一個由站方控制的收件信箱 | Railway **Secret**（保護私人收件地址），server-only。它只接收 `POST /api/contact` 的聯絡表單通知。 |

### 網域與 Cloudflare DNS

1. 在 Resend **Domains** → **Add Domain** 輸入已擁有的 domain 或寄信
   subdomain。若尚未有可控制 DNS 的網域，停在這一步；`pages.dev`
   不能替代它。
2. 預設採手動 DNS：Resend domain 詳情會列出本次實際需要的 DNS rows。
   到 Cloudflare **網站（Websites）** → 該 zone → **DNS** →
   **記錄（Records）** → **新增記錄（Add record）**，逐列複製 Name、Type、
   Content／Mail server、Priority 和 TTL。不要用文件範例值取代 Resend
   畫面生成的值。
3. Resend 的寄信驗證包含 SPF 與 DKIM；依當前畫面也可能顯示回傳路徑用的
   MX row。所有產生的 DNS rows 都必須存在。若畫面要求 CNAME，Cloudflare
   必須選 **僅 DNS（DNS only）**，不能開橘雲 proxy。
4. 回 Resend 按 **Verify DNS Records**，只在 domain 狀態顯示
   **Verified** 後進下一步。DNS 通常較快但可能傳播更久；若仍 pending，
   先比對 record name/value 和 DNS zone，再使用 Resend 的重新驗證。
5. Resend 的 **Sign in to Cloudflare** 自動設定會授予 Resend 對 DNS 的
   存取權；這是另一條可選路徑，必須由帳號持有人在當下確認授權範圍。它
   不是手動路徑的必要條件。

### 部署與有副作用的驗收

1. 網域 Verified 後，持有人建立最小權限的 **Sending access** key，
   直接填 Railway `RESEND_API_KEY` Secret；再填 `RESEND_FROM` 和
   `CONTACT_NOTIFY_TO`。三個名稱都在 Railway service，**不**在 storefront、
   admin 或 Pages Functions。
2. 在 Railway 套用整組變更並等待一個新的成功 deployment；先確認
   `/healthz` 為 `200`。這只證明 API 可運行，不代表寄信成功。
3. 寄一封真實測試信會對外傳送並在資料庫留下資料，須由使用者在當下確認，
   並使用其控制的寄件／收件帳號。最小範圍是明確標記的網站聯絡表單測試：
   `POST /api/contact` 成功會回 `201`；這只代表詢問已保存且設定的
   sender 沒有回錯，若 API key 為空會落到 log sender。仍須在 Resend
   **Emails** 看 `sent`／`delivered`，由持有人確認收件匣。儀表板不可用
   時保留「送達待驗」，不要公開分享 Resend 的 email-share URL。
4. 聯絡表單會**先**寫入 `contact_inquiries` 才嘗試寄信；若 API 回錯或
   Resend 記錄失敗，先查 Resend **Emails**／**Logs**，不要直接重複提交。
   訂單通知是另一條流程：它寄給下單時的 customer email，測試它需要另外
   授權建立測試訂單與傳送外部 email。

| 現象 | 可能原因 | 下一個安全動作 |
|---|---|---|
| Railway 部署成功但沒有 Resend request | `RESEND_API_KEY` 未保存、尚未部署，或沒有觸發聯絡／訂單流程 | 只核對名稱和 deployment；不要讀 key。 |
| Resend `403` 或 domain mismatch | `RESEND_FROM` 的 domain／subdomain 不等於 **Verified** domain，或 key 未被授權該 domain | 修正 sender 或重新建立 domain-scoped Sending access key，再部署。 |
| Resend `401` | key 遺失、失效或貼到錯誤 Railway environment | 持有人直接替換 Railway Secret，部署後再做一次授權的測試。 |
| `POST /api/contact` 回錯但詢問已存在 | 寫入已成功、寄信階段失敗 | 用明確測試標記查 Resend **Emails**／**Logs**；不要盲目重送。 |
| Resend 顯示 `sent` 但尚未 `delivered` | 收件伺服器尚在處理或有暫時性遞送問題 | 看該 email 的 event／log；不要把 `sent` 誤判為 inbox 已收到。 |

### Connection pool bounds

`database/sql` defaults to an **unlimited** number of open connections and two
idle ones. Unlimited lets a traffic spike exhaust the provider's connection
allowance; two idle makes steady traffic churn connections. Both are wrong for
a hosted PostgreSQL.

The Go API therefore always applies bounds. Defaults are
`DB_MAX_OPEN_CONNS=10`, `DB_MAX_IDLE_CONNS=10`, `DB_CONN_MAX_LIFETIME=30m`,
`DB_CONN_MAX_IDLE_TIME=5m`. An unset, zero, negative, or unparseable value
falls back to its default, so no configuration mistake can produce an
unbounded pool. Idle above open is clamped to open.

**These defaults are a conservative floor, not a sizing recommendation.** The
right numbers depend on what sits in front of PostgreSQL:

- Direct connections, or a **session** pooler: size `DB_MAX_OPEN_CONNS` to the
  instance's allowance divided by the number of API replicas.
- A **transaction** pooler: the pooler owns the real connection count, so keep
  this small. Note that pgx's prepared statements need
  `default_query_exec_mode=simple_protocol` in the DSN under transaction
  pooling. Choosing the pooler mode is an operator decision this repository
  does not make.

SQLite is unaffected and stays at one connection. That is deliberate: several
lock guards are documented as SQLite no-ops precisely because its access is
serialized.

### Edge-to-origin authentication

Edge rate limiting and WAF rules only apply to traffic that reaches the edge.
If the origin is directly reachable, they can be bypassed by connecting to it,
and an origin hostname is discoverable through Certificate Transparency logs
and historical DNS rather than only by guessing.

`EDGE_SECRET` closes that. When set, the API refuses any request that does not
carry it in `X-Edge-Secret`, so edge protection stops depending on the origin
being undiscoverable. The value is injected by the Pages Function
`functions/api/[[path]].js`, which both Pages projects deploy from the repo
root: browsers call same-origin `/api/*`, the Function stamps the header and
forwards to `API_ORIGIN`, and requests that never traverse it are refused.
A Cloudflare Transform Rule on an `api.` subdomain remains a valid
alternative injection mechanism, but the Function path is the documented
default — no Transform Rule or `api.` DNS record is needed.

It authenticates the **hop, not the caller**. It proves a request traversed the
proxy; it says nothing about who sent it, and nothing in this repository uses
it for an authorization decision.

Operational notes:

- **Opt-in.** Unset means the check is absent and behavior is exactly as
  before, so local development and existing deployments are unaffected.
- **The health path is exempt.** The platform probes `/healthz` directly rather
  than through the edge, so guarding it would fail every probe.
- **Every rejection looks the same.** Absent, empty, and wrong all return the
  same 403 body, so a prober cannot learn whether the header name is right.
- **Planned rotation of an uncompromised value is zero-outage.** The origin accepts the non-empty values of
  `EDGE_SECRET` and `EDGE_SECRET_PREVIOUS` as a set, so: set
  `EDGE_SECRET_PREVIOUS` to the outgoing value and `EDGE_SECRET` to the new
  value on Railway and redeploy (both are accepted) → set `EDGE_SECRET` to the
  new value on both Pages projects and redeploy → remove
  `EDGE_SECRET_PREVIOUS` on Railway and redeploy. `EDGE_SECRET_PREVIOUS`
  exists only for the window — remove it once rotation completes.
  **Do not use this procedure for the currently exposed Text value.** Rotate
  that credential directly in Railway and both Pages Secrets without accepting
  the old value as previous; temporary `/api/*` interruption is acceptable.

## The browser boundary

The **storefront** Cloudflare Pages build holds server-only values because the
Go renderer runs in that project. Its `DATABASE_URL` is an encrypted build
secret and must not reach `dist/`. The separate **admin** Pages project is a
Vue-only build: it needs public Supabase configuration, but it must not receive
`DATABASE_URL` merely because the storefront needs it. Only an explicit
allowlist crosses either browser boundary:

Cloudflare's variable form may start on **Text** (`文字`). For both the
storefront `DATABASE_URL` build value and the `EDGE_SECRET` Function value on
each Pages project, explicitly select **Secret** / **Encrypt** before saving.
After saving, verify the table type is Secret rather than Text. If a secret was
ever stored as Text, treat it as compromised: rotate the database password or
edge credential, delete the text row, and enter the replacement as an encrypted
Secret without copying it through chat or another provider dashboard.

- Both admin and public theme: `AUTH_MODE`, `SUPABASE_URL`,
  `SUPABASE_PUBLISHABLE_KEY`, `AUTH_GOOGLE_ENABLED`, `AUTH_LINE_ENABLED`
- Admin only: `ADMIN_API_BASE`

The list is `BROWSER_SAFE_KEYS` in `admin/vite.config.ts` and
`site/themes/minimal-cart/vite.config.ts`. Both builds construct their
`define` map from it, so adding a variable to the build environment does not
expose it — a name has to be added to the allowlist deliberately.

`SUPABASE_PUBLISHABLE_KEY` is a public client identifier, not a server secret.
Row-level security, not key secrecy, is what protects Supabase data.

`SUPABASE_VERIFIER_MODE` is a server-only setting (`remote` | `jwks`). It defaults
to `remote` (verifying tokens via `/auth/v1/user`). When set to `jwks`, the Go API
caches public keys from `<SUPABASE_URL>/auth/v1/.well-known/jwks.json` and verifies
asymmetric JWTs locally. In `jwks` mode, server-side sign-out or session revocation
is not observed until the access token expires. Setting `SUPABASE_VERIFIER_MODE=remote`
and redeploying immediately restores per-request `/auth/v1/user` checks.

OAuth client secrets belong in the Supabase dashboard. They are never
environment variables here and never enter a bundle.

### Checking it

Build with a sentinel in a server-only variable and confirm it is absent:

```bash
RESEND_API_KEY=SENTINEL_CHECK npm --prefix admin run build:only
grep -r SENTINEL_CHECK admin/dist/ ; # expect no matches
```

## Adding a variable

1. Add it to `.env.development.example` and/or `.env.production.example` with a
   placeholder and an ownership comment.
2. Read it in `server/internal/config/config.go` if the Go side needs it.
3. Only if a browser genuinely needs it, and only if it is not a secret, add it
   to `BROWSER_SAFE_KEYS` in the relevant `vite.config.ts`.
4. Update the ownership table above.

Step 3 is the one that needs a second opinion. Everything reachable from a
browser bundle is public.
