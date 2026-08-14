# 前後端整合提案

> 受控變更：`minimal-cart-integration` Revision 1  
> 規範來源：[`specs/changes/minimal-cart-integration/spec.md`](specs/changes/minimal-cart-integration/spec.md)；本文件是 implementation strategy，衝突時以受控規格為準。  
> 狀態：Draft — 架構方向已複核；受控規格尚未批准，任何 slice 都未取得 apply 授權。  
> 目標：移除正式流程中的 fixture，讓前台與後台透過 Go application backend 使用同一組商業規則，同時維持 Cloudflare Pages static-first 架構。  
> 策略：依 vertical slice 交付；每一個 slice 都包含 migration、store/service、HTTP contract、前端或 renderer、測試與驗收。

## 1. 結果定義

整合完成後：

- 公開商品與站台內容由資料庫經 Go renderer 產生到 `dist/`，再發布至 Cloudflare Pages。
- 購物車可以留在瀏覽器，但價格、優惠、運費、庫存與訂單總額一律由 Go 計算。
- 後台使用 Supabase access token 呼叫 Go；Go 負責所有 authorization。
- SQLite 與 PostgreSQL 使用等價 schema 與行為。
- 商品圖片經既有 media presign 流程直接上傳 R2，commerce 只保存圖片 metadata。
- 正式環境不使用 fixture 作為 API 失敗 fallback；失敗時顯示明確 loading、empty、error 或 retry state。
- 每個 slice 可獨立部署、測試與回滾，不等待全部 resource 完成才驗收。

### 不在本提案內

- 不新增 Nuxt、Next.js、Pages Functions 或第二個 backend。
- 不做 request-time SSR。
- 不做 incremental rendering，除非 full render 已被量測為瓶頸。
- 不建立通用 plugin/provider abstraction；只有出現第二個真實實作或測試 seam 時才抽象。
- 不把目前的假評論資料搬進資料庫；評論功能先隱藏，另案設計。

## 2. 現況基線

### Backend

- `content`：公開文章讀取與 admin upsert。
- `contact`：公開聯絡表單。
- `media`：authenticated R2 presign。
- DB 目前只有 `articles` 與 `contact_inquiries`。
- `server/tools/render` 先讀 DB，再把資料交給 renderer；renderer 本身不持有 DB。

### minimal-cart theme

`mock-data.ts` 目前同時提供：

- 商品、圖片、尺寸、顏色與庫存。
- 分類、Hero、公告、Popup。
- 優惠碼、免運門檻、運費與稅率。
- 配送與付款方式。
- demo account 與 account/order history。

因此不能只替換 ProductGrid、CheckoutDialog 與四個 content island 就宣稱 fixture 已移除。

### Admin SPA

- resource config 內含 fixture rows。
- auth store 是可切換角色的 UI demo，不是真實 session。
- API client 尚未加入 Bearer token、PATCH、abort、runtime API base 與結構化錯誤。
- ResourceListPage 直接以 row index 修改 config rows，不適合 server pagination 或 concurrent mutation。
- Dashboard KPI 與 tasks 仍是靜態資料。

### Repository evidence ledger

本表只記錄目前 repository 可直接證明的狀態；`missing` 代表尚未實作，`unverified` 代表必須先取得部署或產品證據，兩者都不能當成既有能力。

| Claim | Repository evidence | Status | 對本提案的影響 |
|---|---|---|---|
| Go 已提供 articles、contact、media 的最小 API | `server/internal/bootstrap/app.go` 只註冊 `/healthz`、`/api/articles`、`/api/admin/articles`、`/api/contact`、`/api/media/presign` | existing | Catalog、order、site content、identity 的所有 route 都是新增工作 |
| DB 只有初始內容/聯絡資料 | `db/migrations/sqlite/001_init.sql`、`db/migrations/postgres/001_init.sql` | existing | `002`–`005` 必須雙 driver 新增，不能假設已有 commerce schema |
| OpenAPI 只覆蓋現有四組 API | `contracts/openapi.yaml` | existing but incomplete | 每個 slice 先更新 contract，再實作 producer/consumer |
| Renderer 目前只接 articles | `server/tools/render/main.go`、`server/internal/render/render.go` 的 `RenderAll([]content.Article)` | existing | 保留「tools 組合、renderer 純輸入」邊界，將輸入擴成明確 `render.Input` |
| 公開首頁的主要區塊仍由 fixture/island 供應 | `site/themes/minimal-cart/shared/lib/mock-data.ts` 及 ProductGrid、Hero、AnnouncementBar、WelcomePopup 等 imports | existing demo | Slice 1/3 必須輸出 semantic static snapshot，不能只換掉 fetch |
| 公開 policy/operational claims 被硬編 | `FooterPageDialog.vue` 寫死隱私、付款、退貨、保固、配送、promo 等陳述；`templates/home.html` 寫死 24H 到貨/兩年保固 | unsafe content baseline | Slice 3 逐項由 product/legal/ops 核准後進 published content；未核准或能力不存在就隱藏，不可直接搬入 CMS 當真 |
| 結帳金額由瀏覽器計算 | `shared/stores/cart.ts` 與 `islands/CheckoutDialog/CheckoutDialog.vue` | unsafe demo | Slice 2 的 quote/order service 是價格、優惠、運費、稅與庫存唯一 authority |
| Admin auth 與資料列是 demo | `admin/src/stores/auth.ts` 預設 owner；`admin/src/config/resources/*.ts` 內含 rows | existing demo | Slice 0 先接 session/capabilities；各資源在所屬 slice 才移除 rows |
| Admin API client 未送 token 且缺 PATCH | `admin/src/lib/api-client.ts` | missing | Slice 0 必須先完成，否則後續 admin slice 不可安全接線 |
| Supabase verifier 已有 remote validation primitive | `server/internal/auth/supabase.go` 呼叫 `/auth/v1/user`，從 response `app_metadata.role` 建 principal | partial | Slice 0 可保留 remote verification 或改 JWKS；兩者都必須 fail closed，並建立 AAL/session claims 的可信來源 |
| R2 presign route 存在 | `server/internal/modules/media/media.go`、`POST /api/media/presign` | existing primitive | Slice 1 補 product metadata、content type/key/size policy 與 object-level authorization |
| Full render 發布流程已有文件 | `workflows/publish.md`、`server/tools/render`、`server/tools/publish` | partial | Admin trigger、`publish_runs`、Cloudflare deployment status 回寫、last-known-good 尚屬 `missing` |
| Direct Upload CLI 未鎖版 | `server/tools/publish/main.go` 執行 `npx wrangler`，repo manifest/lockfile 查無 Wrangler pin | unsafe tooling baseline | Slice 3 若保留 Direct Upload，必須使用已批准且鎖版的 tooling；admin runtime 不執行 `npx` |
| CI 目前只驗證 Go | `.github/workflows/ci.yml` 執行 gofmt、archcheck、test、vet | existing but incomplete | Release gate 必須加入或另有證據涵蓋 admin/theme build、render 與 dist inspection |
| Admin production hosting/origin | repository 沒有可確認的部署拓撲 | unverified | Slice 0 前完成 Admin hosting Gate；不得預設 CORS 或 cookie/token 拓撲 |
| Commerce 與 staff module | `server/internal/modules` 查無對應實作 | missing | 依 Slice 1/4 建立；不為未存在的第二 provider 預先抽象 |

### Producer → consumer trace

| Flow | Producer/input | Validation and authority | Owner/storage | Contract/transport | Consumer/result | Current gap / fallback disposition |
|---|---|---|---|---|---|---|
| Catalog publish | Admin product form + R2 upload metadata | Go 驗證 slug/SKU/price/variant/media policy；publish capability | `commerce` + DB；image bytes 在 R2 | Admin catalog API；renderer `render.Input` | Go renderer -> semantic catalog HTML + island snapshot -> `dist/` | API/schema/module `missing`；Slice 1 移除 catalog fixture，失敗保留上一版 `dist/`，不顯示假商品 |
| Checkout/order | Browser 只送 variant、quantity、promo、shipping/payment choice、PII | Go 重新定價、驗 promo、檢庫存與 method availability；DB transaction + idempotency | `commerce` + order tables | `POST /api/checkout/quote`、`POST /api/orders` | Checkout receipt、admin fulfillment、後續 customer history | 現在由 client 計價並寫 local order；Slice 2 禁止 fallback 到 mock price/order |
| Site content publish | Admin placement-specific content form | `content` 驗 payload schema、approval/version、publish capability | `content` + `site_content` | Admin content API；renderer `render.Input` | Hero/announcement/footer static HTML、policy static routes + safe island data | 現在由 fixture/硬編 copy 供應；Slice 3 失敗保留 last-known-good publish |
| Admin identity/staff | Supabase access token；owner/manager 發起 staff mutation | Go 驗 token identity；Slice 0 以 immutable bootstrap role，Slice 4 改由 active staff row 即時授權並檢 target/last-owner invariant | Supabase Auth identity + `staff` profile/outbox/audit | Bearer token；`/api/admin/me`；staff API | Router/sidebar 能見度與 server enforcement | 現在 UI 可切 owner；Slice 0 先移除 production switcher，Slice 4 完成 canonical role lifecycle |
| Media upload | Authenticated admin 請求 presign，再直傳 R2 | Go 驗 capability、content type、size、object-key namespace；R2 驗簽名 | R2 bytes；business owner 保存 metadata | presign request/response + direct upload | Product/content editor preview；renderer 產公開 URL | Presign primitive 已有，owner linkage/policy `missing`；任何驗證失敗不得建立 metadata |

### Field ledger

| Concept | Canonical contract/storage | Current consumer shape | Decision / required reconciliation |
|---|---|---|---|
| Product identity | `id` opaque；`slug` public unique；`base_sku` unique | Theme 要 `id/slug/sku`；admin product rows 有 SKU、未見 slug | Slice 1 contract 同時提供三者；admin form 新增 slug，不從 name 在 client 靜默重算 |
| Category | DB category `id/slug/name/sort_order/enabled`；product 引用 `category_id` | Theme `Category` 是固定 union，CategoryFilter 讀 CATEGORIES fixture | Slice 1 把 type 改為 contract string/ID 並由 render snapshot 供 label/order；不在 client 維護第二份 enum |
| Images | ordered `images[]`，每項含 id/key/url/alt/sort；第一張可衍生 `primary_image` | Theme 同時要 `image` 與 `images[]` | Server/render adapter 明確產 primary + ordered list；禁止只存逗號字串 |
| Variants/options | 每個可購買組合有 `variant_id/sku/options/stock/enabled` | Theme 分開 `sizes[]`、`colors[]`，cart 保存 selected values | Slice 1 調整 UI 由 variant matrix 選出 stable `variant_id`；不得以 size/color 猜庫存列 |
| Price/currency | integer minor unit；TWD；`price/original_price` | Theme `number`，現有 demo 含小數訂單金額 | API 整數；formatting 只在 UI；migration/validation 拒絕負值與浮點 payload |
| Ratings/reviews | 本提案沒有 source of truth | Theme 要 rating、breakdown、reviewList | Slice 1 隱藏評論 UI 或使用明確 unavailable state；不得把假評論搬到 DB |
| Inventory | variant `stock` 是 runtime authority；static snapshot 只作展示 | Product/card/cart 都讀 fixture stock | Checkout 必須 quote/order 重驗；snapshot 過期時顯示 server correction，不接受 client stock |
| Site content | placement + validated typed payload + draft/published version | Hero/announcement/popup/footer 各自讀 constants | Slice 3 為每個 placement 定義 OpenAPI schema/adapter；未知欄位拒絕 |
| Policy/feature copy | reviewed `policy_page`/`feature_strip` blocks + approval metadata | Footer dialog/home template 硬編能力與法律陳述 | Slice 3 只發布已核准且與實際 payment/shipping/PII 能力一致的 copy；icon 可由 trusted key UI mapping |
| Order identity | `id` opaque；`order_number` display-only unique | Theme `Order.id` 同時作顯示/查詢 | Contract 分欄；tracking/auth 不得只靠可猜的 `order_number` |
| Totals/payment | `subtotal/discount_total/shipping_total/tax_total/grand_total` + independent payment status | Theme 用 `discount/shipping/tax/total`，client 可自行組裝 | Slice 2 以 server response 覆蓋 UI；未驗證 provider 不可回 `paid` |
| Order state fields | `fulfillment_status/payment_status/return_status` 三欄分離 | Admin 用 `status/payment_status/return_request_status` | Slice 2 改 consumer 使用 canonical names；不在 API adapter 長期保留兩套 aliases |
| Concurrency | mutable aggregate 有 integer `version`；mutation 帶 `expected_version` | Admin 文案/動作使用 `expected_status` | 統一用 `expected_version`，狀態仍是 validation input；stale mutation 回 409 |
| Time | DB/API canonical `*_unix` integer；UI 依 locale format | Theme 用 `placedAt/timestamp` number；admin 欄位命名不一致 | OpenAPI 固定 Unix seconds 並註明單位；adapter 不混 ISO string/milliseconds |
| Boolean | DB portable `INTEGER CHECK (0,1)`；JSON true/false | Admin fixture 可能以 label/string 表示狀態 | API 一律 boolean；badge/label 由 UI mapping 產生 |
| Staff role | token claim role + server-derived `capabilities[]` | Admin fixture 使用 `role_label`，auth store 可切 role | `/api/admin/me` 回 canonical role/capabilities；`role_label` 只作顯示，不作 authorization |
| Capability names | §7 的 domain/action names，例如 `orders.fulfill` | Admin resource 目前使用 `twcommerce.update` | Slice 0/各資源 slice 改成 server matrix 的 canonical capability；禁止 generic update 暗含 refund/staff manage |
| PII | 結構化 name/email/phone/address；list response 最小化/遮罩 | Theme local order 保存完整地址與電話 | Slice 2/4 明訂 detail capability、log redaction；retention/deletion 仍是上線 Gate |

### Fixture disposition ledger

| Fixture/consumer | Current purpose | Final disposition | Slice | Negative evidence |
|---|---|---|---|---|
| `shared/lib/mock-data.ts` 的 PRODUCTS/CATEGORIES | Catalog、variant、stock | 改由 renderer snapshot；互動只接 snapshot IDs | 1 | production imports 不再引用 PRODUCTS/CATEGORIES |
| Cart store 的 promo/tax/shipping constants | Client-side pricing | cart 只保存 selections；quote response 顯示金額 | 2 | 搜尋 pricing constants 與本地 totals 計算為空 |
| Checkout 的 SHIPPING_METHODS/PAYMENT_METHODS | 選項與建單 | 改由 quote 回可用選項；submit 再驗證 | 2 | Checkout 不 import `mock-data`，無 fake paid path |
| Hero/announcement/popup constants | Public content | 改由 published render input | 3 | 四個 content consumer 不 import constants；HTML 無 JS 仍可讀 |
| `Footer.vue`/`FooterPageDialog.vue` hardcoded copy | Footer navigation、privacy/terms/shipping/contact/about/FAQ | 改成 bounded reviewed policy blocks，或未核准頁面從 production nav 移除 | 3 | 搜尋 hardcoded demo promo/電話/保固/付款陳述為空；published pages 有 approval evidence |
| `templates/home.html` feature claims | 24H 出貨、鑑賞期、保固等營運承諾 | 由 reviewed `feature_strip` snapshot 提供，或維持隱藏 | 3 | template 不再寫死未驗證營運承諾 |
| `DEMO_ACCOUNTS` 與 local fake orders | Login/history/tracking | 真實 identity/order API 完成前從 production UI 隱藏 | 4 | production bundle/search 無 demo credential/order records |
| `admin/src/config/resources/*.ts` rows | Admin resource demo data | config 只留 metadata，rows 由 API state 提供 | 資源所屬 slice | 每完成一資源即檢查該 config 無 rows；Slice 5 全域確認 |
| Admin role selector/default owner | Demo authorization | production 使用 token + `/api/admin/me`；dev 模式必須顯式且不可進 production build | 0/4 | Topbar 無 production role switch；server tests 證明 401/403 |

`localStorage` 仍可保存 cart identifiers、theme preference 與無權威性的 UI 狀態；它不得保存可被系統視為已付款、可履約訂單、有效權限或 canonical price 的資料。

## 3. 固定架構決策

以下決策直接沿用 repository architecture，不另行選型：

1. Go 是唯一 application backend。
2. 瀏覽器不直接查 PostgreSQL 或 Supabase Database。
3. Local 使用 SQLite；production 使用 PostgreSQL。
4. 任何 schema 變更同時提供 SQLite 與 PostgreSQL migration。
5. `auth.Principal` 在 handler/service 邊界顯式傳遞。
6. Module 不直接 import 另一個 business module；bootstrap 與 tools 負責組合。
7. 公開內容採 full static publish：

   ```text
   database -> Go render composition -> renderer -> dist/ -> Cloudflare Pages
   ```

8. Renderer 接收 render input，不直接開 DB、不呼叫 module service、不發 HTTP。
9. 金額使用整數 TWD；不使用浮點數保存價格或訂單金額。
10. API contract 以 `contracts/openapi.yaml` 為準；Go、admin 與 theme 都必須符合 contract。

### Authority matrix

| Concern | Authoritative producer/source | Derived caches or consumers | Forbidden authority |
|---|---|---|---|
| Product descriptive data/publish state | `commerce` DB + service validation | renderer snapshot、admin/theme view models | theme fixture、admin config rows |
| Price/currency | `commerce` DB；quote/order service 在 request 時讀取 | static product price 僅為展示 snapshot | cart payload、localStorage、hidden input |
| Inventory | variant row + order transaction | rendered stock hint、admin list | client-computed stock、stale HTML |
| Promo/shipping/tax/payment availability | Go quote service + enabled configuration | checkout quote response | hardcoded browser constants |
| Order totals/status | Go order service + DB transaction/event log | receipt、admin/customer read models | client totals、UI-only status mutation |
| Authenticated identity | verified Supabase token 的 signature/iss/aud/exp/sub，或明確 dev verifier | `auth.Principal` identity fields | email domain、request body identity、未驗證 JWT claims |
| Admin role/status | Final state 以 active `staff_members` row 為準；Slice 0 尚未開放 role mutation 時暫以受信任 `app_metadata.role` bootstrap | `/api/admin/me` role/capabilities；Supabase metadata mirror | UI role selector、`user_metadata`、role request body、role mutation 後的舊 JWT claim |
| Authorization | Go service capability/object checks + server role matrix | router/sidebar visibility | frontend hidden buttons |
| Public content | published `content` DB state | `dist/` last successful snapshot | fixture fallback after render/API failure |
| Media object identity | media policy + R2 object key；business owner metadata | renderer public URL | arbitrary client URL/key |
| PII retention/deletion | 尚待產品/法遵 Gate | masked read models、audit metadata | client local fake history或無期限默認保存 |

### Static/runtime ownership matrix

| Data/behavior | Delivery mode | Freshness and failure behavior |
|---|---|---|
| Hero、announcement、popup、footer | Static semantic HTML + optional island snapshot | 僅 publish 更新；deploy 失敗保留上一個成功版本，不 fallback fixture |
| Product name/description/images/options | Static semantic HTML + island snapshot | 可短暫落後 DB；draft/archived 永不進 snapshot |
| Display price | Static snapshot | 只供瀏覽；quote 顯示變價並要求使用者確認 server 結果 |
| Inventory availability | Runtime quote/order | HTML stock 只是提示；API 不可用就停止結帳，不猜測庫存 |
| Cart contents | Browser local state | 只保存 product/variant IDs、quantity、promo input；載入後由 snapshot/quote reconcile |
| Promo、shipping、tax、payment choices | Runtime quote | API error 顯示 retry/error；不得沿用 mock 或過期可用性 |
| Order creation/payment/fulfillment | Runtime Go API | transaction/idempotency；未知結果用 idempotency key 查回，不在 client 補造成功 |
| Admin/customer order history | Runtime authenticated API | loading/empty/error 分離；不得顯示 local fake orders |
| Admin capabilities | Runtime `/api/admin/me` + server enforcement | token/session 失效即清除 UI session；server 永遠再檢查 |

目前首頁的空 island placeholder 不符合目標 ownership：Slice 1/3 必須讓關鍵 catalog/content 在 JavaScript 未執行時仍存在於 HTML。互動 island 是 progressive enhancement，不是公開內容的唯一 producer。

## 4. 實作前的產品 Gate

這些答案會改變產品行為、成本或資料責任，必須在對應 slice 開工前確認。

| Gate | 建議預設 | 最晚決定時間 |
|---|---|---|
| 付款語意 | MVP 只啟用已確認可實際履約的離線付款，例如取貨付款；信用卡、ATM 虛擬帳號、LINE Pay 保持 disabled，禁止模擬「已付款」 | Slice 2 |
| 顧客帳號 | 先做 guest checkout；暫時隱藏 mock login、account history 與 track-order，之後再接 Supabase customer auth | Slice 4 |
| 庫存時點 | 建單 transaction 內扣庫存；取消訂單時補回。導入線上金流後再增加 reservation expiry | Slice 2 |
| 退貨庫存 | 預設收到退貨不自動補回可售庫存；驗貨後由具 capability 的明確 restock action 決定數量，避免瑕疵品被自動販售 | Slice 2 |
| 稅務與捨入 | 確認售價含稅/未稅、課稅基礎與發票責任；所有 TWD 金額以整數、逐行計算後按同一規則捨入。在確認前移除目前 client hardcoded 5%，不得把它當正式規則 | Slice 2 |
| 配送範圍 | MVP 建議只支援台灣與明確郵遞區號/配送 method 規則；若支援跨境，必須先補 zone、幣別、稅務與禁運條件 | Slice 2 |
| Admin step-up auth | `staff.manage`、`orders.refund`、offline payment confirmation、payment settings 建議要求 Supabase `aal2` 或等價 recent re-auth；若 MVP 不提供，對應 high-risk mutation 保持 disabled | Slice 2/4 |
| 法務/營運文案 | Privacy、terms、退貨、配送、保固、付款、資料加密/第三方分享等陳述逐項指定 approver 與有效日期；能力未實作或未核准就不發布 | Slice 3/上線前 |
| 發布策略 | 若後台只需「已送出」，可用 Deploy Hook 並顯示 `trigger_requested`；若 UI 要顯示部署 success/failure，建議改用/搭配 Pages API 保存 deployment ID 並查 status。兩者不可混稱「發布成功」；save draft 不觸發 | Slice 3 |
| Admin hosting | 預設與公開站同一 Pages domain 的 `/admin/`；若分站部署，再設定明確 admin origin | Slice 0 |
| PII 保存 | 上線前定義訂單姓名、電話、地址、操作紀錄的保存與刪除政策 | 上線前 |

在 Gate 未確認前，可以完成不依賴該決策的 contract 與 catalog slice，但不能假裝付款或會員流程已完成。

## 5. Module ownership

| Owner | 資料與責任 |
|---|---|
| `content` | 現有 articles；新增 site content slots、draft/publish lifecycle、site-wide `publish_runs` receipt、公開 render read model |
| `commerce` | products、images、variants、inventory、promos、shipping methods、payment settings、orders、order items、order events、members |
| `staff` | admin staff profile、Supabase auth identity linkage、role lifecycle；延後到 Slice 4 |
| `media` | 既有 R2 presign；不保存 product/content business metadata |
| bootstrap/tools | 跨 owner 組合、route wiring、render input assembly |

`commerce` 合併的理由是 catalog、quote、inventory 與 order creation 需要同一個 transaction boundary，不是因為不同 module 必然要互相 import。

`site_content` 直接由既有 `content` module 擁有，避免建立兩套 publish/CMS lifecycle。

檔案依責任拆分即可，不強制每個 entity 都建立空的 `model.go + store.go + service.go + http.go`。

## 6. Schema 提案

Migration 依 slice 分開，不把所有功能塞進單一 `002_commerce.sql`：

```text
002_catalog.sql
003_orders.sql
004_site_content.sql
005_identity.sql
```

SQLite 與 PostgreSQL 使用相同 migration number 與等價 constraint。Boolean 沿用現有 `INTEGER CHECK (value IN (0,1))` 形式，降低 driver 差異。

### 6.1 Catalog

`product_categories`

- `id TEXT PRIMARY KEY`、`slug TEXT NOT NULL UNIQUE`、`name TEXT NOT NULL`。
- `sort_order`、`enabled INTEGER CHECK (enabled IN (0,1))`、`version`、timestamps。
- Renderer 只輸出 enabled 且至少有 published product 的 category；`all` 是 UI synthetic filter，不存成 category row。

`products`

- `id TEXT PRIMARY KEY`：opaque internal ID。
- `slug TEXT NOT NULL UNIQUE`。
- `base_sku TEXT NOT NULL UNIQUE`。
- `name`、`description`、`long_description`。
- `category_id TEXT NOT NULL`，foreign key 指向 `product_categories.id`。
- `publish_status`：`draft | published | archived`。
- `currency TEXT NOT NULL DEFAULT 'TWD'`。
- `price INTEGER NOT NULL CHECK (price >= 0)`。
- `original_price INTEGER NULL CHECK (original_price >= 0)`。
- `material`、`origin`、`tag`。
- `version INTEGER NOT NULL DEFAULT 1`。
- `created_unix`、`updated_unix`。

Product/category slug 只允許 canonical lowercase ASCII segment（例如 `[a-z0-9]+(?:-[a-z0-9]+)*`），拒絕 `.`、`..`、slash、encoded separator 與 reserved routes。首次 published 後 slug 在 MVP immutable；若未來要改，必須先有顯式 redirect model，不可直接造成舊 URL 404。

MVP 不建立未被 consumer 證明需要的 product revision system，改採明確 mutation 語意：

- Draft product 可反覆保存，不進 renderer。
- `publish` 將 draft 轉為 published 並要求 full render/deploy。
- Published product 的編輯動作必須在 UI/API 命名為「更新並重新發布」；成功寫入後即是下一次 full render 的 published source，不提供看似 save-draft、實際可能被其他 full render 洩漏的路徑。
- 若產品需要「已發布版本保持不動，同時準備下一版 draft」，必須另案加入 product revisions；不得只靠單一 `publish_status` 假裝同時保存兩版。

`product_images`

- `id`、`product_id`、`object_key`、`alt_text`、`sort_order`。
- Metadata 由 commerce 保存；檔案 bytes 走 media presign -> R2，公開 URL 由 backend/renderer 依 object key 與 CDN base 組合。
- Object key 必須由 server 產生並限制在 principal/用途 namespace；只有上傳後由 server 驗 object 存在/size，並以 bounded read + image decode/magic bytes 驗實際格式與尺寸，才能把 metadata 關聯到 product。不可只信 client header 或 R2 metadata 的 content type。
- MVP allowlist 為必要的 raster image formats；SVG 預設拒絕，除非另有 sanitizer/CSP/asset-origin 設計。公開 asset origin 與 admin/API origin 分離，並設定安全 content type/sniffing headers。

`product_variants`

- `id`、`product_id`、`sku UNIQUE`。
- `size_value`、`size_label`、`color_value`、`color_label`、`color_swatch`。
- `stock INTEGER NOT NULL CHECK (stock >= 0)`。
- `enabled INTEGER NOT NULL CHECK (enabled IN (0,1))`。
- 沒有尺寸/顏色的商品也建立一個 default variant，讓 cart/order 永遠引用 `variant_id`。

庫存狀態由 variant stock 衍生，不與 `draft/published/archived` 混在同一欄。

### 6.2 Orders and pricing

`orders`

- `id TEXT PRIMARY KEY`：opaque internal ID。
- `order_number TEXT NOT NULL UNIQUE`：顯示用，例如 `TW-1042`。
- `member_id TEXT NULL`：guest checkout 可為空。
- customer name/email/phone 與結構化配送地址。
- `currency`、`subtotal`、`discount_total`、`shipping_total`、`tax_total`、`grand_total`。
- `fulfillment_status`、`payment_status`、`return_status` 分離。
- `shipping_method_id`、`promo_code_snapshot`、`payment_reference`。
- `idempotency_key TEXT NOT NULL UNIQUE`。
- `idempotency_request_hash TEXT NOT NULL`；同 key 重送相同 canonical request 回原結果，同 key 不同 payload 回 `409`。
- `version INTEGER NOT NULL DEFAULT 1`。
- `created_unix`、`updated_unix`。

`order_items`

- 保存下單當時的 product/variant/SKU/name/options/unit price/quantity/line total snapshot。
- `returned_quantity`、`restocked_quantity` 預設 0，constraint 為 `0 <= restocked_quantity <= returned_quantity <= quantity`，支援可證明且不重複的退貨回補。
- 不依賴日後可能改名或改價的 product row。

`order_events`

- append-only 保存 status transition、actor user ID、reason、timestamp 與必要 metadata。
- 不以可整包覆寫的 `timeline_json` 充當 audit log。

MVP 狀態機固定如下；UI 只能反映它，不能自行擴張 transition：

```text
fulfillment_status:
  pending -> processing -> shipped -> delivered
  pending | processing -> cancelled

payment_status:
  unpaid -> paid -> refunded

return_status:
  none -> requested
  requested -> approved | rejected
  approved -> received
  received -> refunded
```

- 建單預設 `fulfillment_status=pending`、`payment_status=unpaid`、`return_status=none`；只有真實 provider success/verified webhook 或受權限保護的 offline-payment action 能進入 `paid`。
- Fulfillment 的 `delivered/cancelled`、return 的 `rejected/refunded`、payment 的 `refunded` 是 terminal；若未來需要 reopen/partial refund，另案擴充 schema/contract，不以任意字串繞過。
- `pending|processing -> cancelled` 在同一 transaction 內只補回一次庫存並寫 event。已出貨後不能走 cancel，必須走 return flow。
- `return_status none -> requested` 只允許已送達訂單；`received` 不自動 restock，是否回補與數量依退貨庫存 Gate 的明確 action/event。
- Fulfillment、payment、return 各自更新自己的欄位與同一 aggregate `version`；一條狀態機不能暗中覆寫另一條。
- Paid order 取消或退貨不會自動假設退款成功；只有實際 refund/offline confirmation 成功後才進入 `payment_status=refunded` 與終態 `return_status=refunded`。
- `requested` 的 customer producer 延到 Slice 4；Slice 2 若保留 admin 建立 request，必須記錄 actor/reason，不能直接跳到 approved。

目標 status 與 `expected_version` 只存在 mutation request，不保存成 order column。Store 使用：

```sql
UPDATE orders
SET fulfillment_status = ?, version = version + 1, updated_unix = ?
WHERE id = ? AND version = ?
```

affected rows 為 0 -> `409 conflict`。

### 6.3 Promotions, shipping, payment settings

`promos`

- code、type、value、enabled、starts/expires、最低金額與使用條件。
- `percent` 使用 integer basis points（0–10000）；`fixed` 使用 integer TWD；`freeship` 不接受含糊的浮點 value。
- Go service 驗證期間、條件與折扣上限。
- MVP 每張訂單最多一個 promo；計算順序固定為 items subtotal -> promo discount -> shipping -> tax -> grand total，任何階段不得低於 0。稅額的課稅基礎依產品 Gate 定案。
- 公開 API 不列出所有 active promo codes。

`shipping_methods`

- id、name、description、fee、free_threshold、enabled、sort_order。
- Quote 與 order service 讀取同一份規則。

`payment_methods`

- 只保存 method 與 admin-controlled `enabled`。
- provider、environment、credential readiness 由 backend configuration 計算，不把 secret 或可能漂移的 readiness 寫入 DB。
- 沒有真實 provider flow 與 verified webhook 的 method 不得顯示 available。

### 6.4 Site content

`site_content`

- `id`、`key`、`placement`；`UNIQUE(key, publish_status)`，同一 logical key 最多各有一筆 draft 與 published copy。
- `title`、`body`、`payload_json`。
- `publish_status`：`draft | published`。
- Optional `approved_by`、`approved_unix`、`review_due_unix`；policy/feature placements publish 時必填且 approver 必須具對應 capability。`approved_by/unix` 由 service 從 `Principal`/clock 寫入，不接受 request body 冒名。
- `sort_order`、`version`、`created_unix`、`updated_unix`。

Save draft 只 upsert draft row；publish transaction 驗證 draft 後，把內容複製/upsert 到 published row。Renderer 永遠只讀 published row，因此其他資源觸發 full render 時也不會洩漏 draft。

任何會改變 policy/feature claim 的 draft mutation 都清除既有 approval；approve request 帶 `expected_version`，publish 只接受 approval version 與目前 draft version 相同的資料。

`payload_json` 只承載目前真實存在的 placement payload，例如：

- hero：desktop/mobile image、eyebrow、CTA、bounded `stats[]`（label/value）；若不採用 stats，Slice 3 必須移除對應 UI，不留 fixture。
- announcement/popup：icon、highlight、倒數時間、popup copy。
- footer：link/page key 與顯示文字。
- `policy_page`：只支援現有 page keys 與 bounded ordered blocks（heading/paragraph/list），另存 `approved_by`、`approved_unix`、`review_due_unix`；沒有 approval evidence 不可 publish。
- `feature_strip`：bounded icon key/title/body/order；24H 到貨、保固、退貨等 claims 同樣受 approval Gate。

Go service 依 placement 驗證 payload schema；不要建立任意 page-builder abstraction。

`title/body` 預設視為純文字並由 template escape；URL 預設只允許 same-origin relative path，必要的 external HTTPS host 另列 allowlist。若未來需要 rich HTML，必須另定 allowlist sanitizer 與測試，不能直接把 admin HTML 注入 template 或 island JSON。

`publish_runs`

- 由既有 `content` module 擁有 site-wide publish receipt，欄位包含 `id`、`requested_by`、`source_kind`、`source_id`、`source_version`、`provider`、optional `external_deployment_id`、`status`、safe `error_code`、timestamps。
- `status`：`trigger_requested | active | succeeded | failed | cancelled`；只有查到 Cloudflare deployment terminal status 才能寫 `succeeded/failed/cancelled`。
- Deploy Hook URL 或 Pages API token 只存在 server secret config，不保存到 row、不回傳 browser、不寫 log。
- Pages API token 只授予目標 account/project 所需的 Pages Read/Write；若採 trigger-only Hook，不另外配置 API token，也不聲稱能追蹤 terminal status。
- `content` 不 import `commerce`；bootstrap 在 catalog/content publish 成功後，把 source receipt 交給 publish service。Renderer 仍由 Pages build/既有 tools 組合所有 module read models。

### 6.5 Members and staff

`members`

- `auth_user_id TEXT NULL UNIQUE`、email、name、phone、status、tier、notes。
- order count/total spent 先由 query 聚合；若之後證明效能不足，再加入可重建 projection。
- 若需要精確 tag filter，使用 `member_tags`，不以逗號字串做模糊比對。

`staff_members`

- `auth_user_id TEXT NOT NULL UNIQUE`、email UNIQUE、display_name、role、status、version。
- Slice 4 上線後，active staff row 是 admin role/status 的即時 authority；verified JWT 提供 identity，bootstrap/authz resolver 依 `sub` 載入 staff row 後建立顯式 `auth.Principal`。
- 建立或變更 staff role 由 Go 先更新 canonical staff row，再透過受信任的 Supabase admin adapter 同步 `app_metadata.role`；metadata 是 mirror，不可反過來覆蓋 DB role。
- 禁止 browser 直接修改 metadata。
- Service 必須阻止非 owner 提權、移除最後一位 owner 或自行提升權限。

`staff_auth_sync_jobs`

- Durable outbox：`id`、`staff_id`、`staff_version`、desired role/status、`pending|succeeded|failed`、attempt count、safe error code、next-attempt/timestamps。
- Supabase sync 失敗不回滾已生效的降權/停用；Go 仍依 canonical staff row fail closed，outbox 可安全重試。Service-role credential 只在 server config，不能進 job payload/log。

## 7. Authentication and authorization

### Identity source

- Production：Supabase access token。
- Dev：既有 dev token。
- Verifier 必須透過現有 Supabase `/auth/v1/user` remote validation 或 local JWKS 驗證，可信地建立 issuer/audience/expiry/subject/AAL/session contract；任何網路、key refresh、claim/schema 錯誤都 fail closed，不信任單純 decode 的 JWT。
- Slice 0 到 Slice 4 以前不提供 runtime role mutation，暫以受信任 `app_metadata.role` bootstrap；永遠不讀 `user_metadata` 的角色或 capabilities。
- Slice 4 啟用 staff lifecycle 後，bootstrap/authz resolver 以 token `sub` 查 active `staff_members`；row role/status 才是即時 authority，舊 JWT 的 metadata 不能保留已撤銷權限。
- Supabase `app_metadata.role` 只保存 role mirror；capability matrix 由 Go 定義，避免在 staff row、token 各存一份 capability list。
- Admin metadata update 只能由 server-side admin adapter 執行；browser refresh session 只為取得新 claim/UI 狀態，不是 authorization 生效條件。

### Principal

```go
type Principal struct {
    UserID string
    Email  string
    Role   string
    AAL    string
    SessionID string
}
```

`auth.Can(principal, capability)` 依 server-side role matrix 判定；high-risk service 另檢查已確認的 AAL/re-auth policy。`Principal` 仍在 handler/service 邊界顯式傳遞，不藏進 `context.Context`。

建議 capability 至少分開：

- `catalog.read | catalog.write | catalog.publish | catalog.archive`
- `inventory.adjust`
- `orders.read | orders.fulfill | orders.returns | orders.refund`
- `members.read | members.write`
- `promos.read | promos.write`
- `payment_settings.read | payment_settings.write`
- `content.read | content.write | content.approve | content.publish`
- `staff.read | staff.manage`

Handler 負責解析 token 與 request；Service 接收 `auth.Principal` 並執行 authorization。Frontend route guard 只改善 UX，不能取代 service authorization。

### Admin session contract

```text
GET /api/admin/me
Authorization: Bearer <supabase-access-token>
```

回傳 user ID、email、role 與 server 計算後的 capabilities。Admin auth store 以此結果作 UI gate，移除 production role switcher。

Admin 不自行從 JWT decode role 來決定權限；每次 session 建立/refresh 與 role-sensitive navigation 都以 `/api/admin/me` 結果為準。Slice 4 後即使 client 尚未 refresh token，server 也依 canonical staff row 執行降權或停用。

## 8. HTTP contract

所有 path 先寫入 `contracts/openapi.yaml`。Go 1.22+ ServeMux 與 OpenAPI 使用 `{id}`，不使用文件式的 `:id`。

### 8.1 Admin catalog

```text
GET    /api/admin/products
GET    /api/admin/products/{id}
POST   /api/admin/products
PUT    /api/admin/products/{id}
PATCH  /api/admin/products/{id}/publish-status
POST   /api/admin/products/bulk

GET    /api/admin/categories
POST   /api/admin/categories
PUT    /api/admin/categories/{id}
PATCH  /api/admin/categories/{id}/enabled

POST   /api/media/presign
```

- Product list 支援 `page`、`page_size`、`status`、`category_slug`、`query`；response 同時提供 `category_id/category_slug/category_name`，避免 consumer 再查 fixture mapping。
- Category mutation 使用 stable ID/version；disable 前需處理仍引用它的 draft/published products，不能讓已發布商品失去 render label。
- Mutation 回傳新 `version`。
- Archive 使用明確 publish status；不以名稱為 DELETE、實際卻 disable 的隱晦語意。
- Bulk request 每筆帶 ID 與 expected version，response 逐筆回報成功/衝突/驗證失敗。

### 8.2 Checkout and public order creation

```text
POST /api/checkout/quote
POST /api/orders
```

`quote` request 只包含：

- `product_id`
- `variant_id`
- `quantity`
- optional `promo_code`
- `shipping_method_id`

Go 回傳 canonical items、availability、subtotal、discount、shipping、tax、total、可用 payment methods 與 validation errors。若使用者選定付款方式後重新 quote，可帶 optional `payment_method_id` 驗證它仍可用。

`POST /api/orders` 接收同一組 cart identifiers、customer/shipping fields、`shipping_method_id`、`payment_method_id` 與：

```text
Idempotency-Key: <client-generated-random-id>
```

Order service 在同一 transaction 內重新計價、驗證 promo、確認/扣除庫存、建立 order/items/events。Request 不接受 authoritative price、total、payment status 或 payment reference。

Idempotency scope 與行為：

- Client 產生不可預測的 random key；server 記錄 canonical request hash。
- 同 key + 同 hash 重送時回同一張訂單與相同成功語意。
- 同 key + 不同 hash 回 `409 idempotency_key_reused`，不可回舊訂單或建立新訂單。
- Server timeout/未知結果時，client 只可用同 key 重試；不可更換 key 並自行顯示成功。

刪除原提案的：

```text
GET /api/promos/active
```

Promo code 只能透過 quote 驗證，不公開列出。

### 8.3 Admin orders

```text
GET    /api/admin/orders
GET    /api/admin/orders/{id}
PATCH  /api/admin/orders/{id}/fulfillment
PATCH  /api/admin/orders/{id}/return
POST   /api/admin/orders/{id}/restock
POST   /api/admin/orders/{id}/offline-payment
GET    /api/admin/dashboard
```

- Service 驗證合法 state transition。
- Mutation 帶 `expected_version`；stale write 回 `409`。
- 所有狀態改變寫入 `order_events`，包含 principal、reason 與 timestamp。
- Refund 與 offline-payment 使用獨立 capability，不能只靠通用 `commerce.update`。
- Restock request 逐 item 帶已驗收 quantity、reason、`expected_version` 與 idempotency key；service 驗不超過 received quantity，並要求 `inventory.adjust`。

### 8.4 Admin site content

```text
GET    /api/admin/site-content
POST   /api/admin/site-content
PUT    /api/admin/site-content/{id}
POST   /api/admin/site-content/{id}/approve
POST   /api/admin/site-content/{id}/publish
POST   /api/admin/site-content/{id}/unpublish

GET    /api/admin/publish-runs
GET    /api/admin/publish-runs/{id}
POST   /api/admin/publish-runs
```

Save draft 與 publish 分離。Publish 成功保存 published copy 後建立 `publish_run` 並觸發已選定的 Cloudflare transport；trigger 被接受只能顯示「資料已保存、部署已送出」。只有 Pages status 為 success 才顯示「已發布」；terminal failure 顯示「資料已保存、部署失敗」，不可假裝公開站已更新。Unpublish 移除 published copy、保留 draft，並走同一 full publish receipt。

`POST /api/admin/publish-runs` 對「目前 published desired state」建立新的 run，供明確 retry/operator publish 使用；它也接受 `Idempotency-Key`，同 key 不得重複觸發。不得覆寫舊 run 或把失敗改寫成成功。List/detail 只回 safe status/error code 與外部 deployment link，不回 Hook URL、API token 或 provider raw logs。

正常頁面載入不呼叫 `GET /api/site-content/published`；published snapshot 由 renderer 寫入 `dist/`。

### 8.5 Members and staff

這組 API 放到 Slice 4，等 customer/staff identity lifecycle 確認後才實作：

```text
GET    /api/admin/members
GET    /api/admin/members/{id}
PUT    /api/admin/members/{id}
PATCH  /api/admin/members/{id}/status

GET    /api/admin/staff
POST   /api/admin/staff
PUT    /api/admin/staff/{id}
```

### 8.6 Response conventions

- List：`{ items, total, page, page_size }`。
- Validation error：`400` + stable error code + field errors。
- Unauthenticated：`401`。
- Authenticated but forbidden：`403`。
- Missing object：`404`。
- Stale version/state：`409`。
- Request body 設定大小限制並拒絕 unknown fields。
- Public order creation 加入 rate limit/abuse protection；日誌不可包含完整電話、地址或 token。

## 9. Renderer and publishing

### Render composition

`server/tools/render` 與 `server/tools/dev`：

1. 開啟 DB。
2. 在一個一致的 read snapshot/transaction 內讀取 published articles、site content、categories 與 catalog read models；SQLite/PostgreSQL adapter tests 必須證明 render 不會混到跨版本資料。
3. 建立單一 `render.Input`。
4. 呼叫 `renderer.RenderAll(input)` 寫入 staging output。
5. 所有 page/assets/manifest 驗證成功後才 promote 成 `dist/`；任何失敗保留上一個完整 output，不留下半套站點。

Renderer 不 import SQL adapter，也不直接查 DB。

Renderer 至少產生首頁/catalog、`/products/{slug}/` 與 `/categories/{slug}/` static routes；所有 output path 由已驗證 slug 組成並在寫檔前確認仍位於 `dist/`。Product cards/header/category links 使用真實 `<a href>`，dialog/filter 只作 progressive enhancement。

### Theme data flow

- Header navigation、Hero、公告、Footer 與 published catalog 由 renderer 先輸出可讀的 semantic HTML，避免公開頁只剩空 island placeholder；cart/account/dialog controls 再由 island 增強。
- Privacy/terms/shipping/about/FAQ 等 policy content 產生真實 static routes（例如 `/privacy/`），Footer 使用可複製/可爬取的 `<a href>`；dialog 若保留只能 progressive enhance 同一份 snapshot，不能成為唯一內容載體。
- 需要互動的 island 接收同一份安全編碼 snapshot，mount 後再增強搜尋、篩選、carousel、dialog 與 cart；page load 不 fetch content API。Snapshot 必須由 Go encoder/template 安全嵌入並處理 `<`/`</script>`，禁止用字串串接產生 executable script。
- Cart localStorage 只保存 `product_id`、`variant_id`、`quantity` 與 promo input，不保存可被當成 authoritative 的整個 Product/price。
- Checkout 開啟時呼叫 quote；submit 再由 server 重新計算。
- Live API 失敗時停止結帳並顯示 retry，不 fallback 到 mock price、stock、promo 或 payment method。

### Publish flow

```text
admin publish
  -> Go validates and saves published state
  -> Go creates publish_runs(trigger_requested)
  -> Cloudflare trigger starts a Pages build
  -> Pages build runs go run ./server/tools/render
  -> dist/ -> Cloudflare Pages
  -> optional Pages API status sync updates active/succeeded/failed/cancelled
```

Catalog snapshot 可能短暫落後 DB；quote/order 永遠使用當前 DB，因此不會接受過期價格或庫存。

Deploy Hook 的成功 response 只代表 trigger 已被接受，不能寫 `succeeded`。選擇「可追蹤發布」時，server 必須以 scoped Pages API token 保存/查詢 Cloudflare deployment ID 與 terminal status；選擇「只觸發」時，admin 只顯示 `trigger_requested` 並連到 Cloudflare deployment log，不宣稱公開站已更新。

`server/tools/publish` 的 Direct Upload 是 AI/CI/operator 路徑，與 Pages Git build/Deploy Hook 是不同 transport；兩者共用 renderer/`dist/` acceptance，但不得混用 success receipt。

## 10. Admin integration

### API client

- Runtime API base；避免把 `/api` prefix 重複組合。
- 自動加入 Supabase Bearer token。
- 支援 GET/POST/PUT/PATCH/DELETE。
- 結構化處理 400/401/403/404/409/5xx。
- 支援 AbortController，切 resource 時取消舊 request。
- 401 清除 session 並導向登入；403 顯示權限不足；409 重新載入該 row。

### Resource state

- `ResourceDef` 只保留欄位、filter、form 與 action metadata。
- API rows、loading、error、pagination、filter、selection 放在 page/store state，不再修改 config constant。
- Selection 使用 stable row ID/version，不使用 array index。
- Form 使用獨立 draft copy，成功後以 response 更新 row。
- Row/bulk mutation 有 pending/partial failure receipt。
- Loading、error、empty、success、keyboard、responsive state 都要可驗收。

### Auth UI

- Production 移除 Topbar role selector。
- Sidebar/router 依 `/api/admin/me` capabilities 隱藏不可用功能。
- 即使 UI 隱藏，server service 仍獨立檢查 capability。

### Local development

- Theme Vite 與 admin Vite 不得同用 5174；admin 改用另一個固定 port。
- 若 admin 與 site 同 origin，以 dev proxy 呼叫 `/api`。
- 若 production 分 origin，Config 支援明確 allowlist，並加入 PUT/PATCH/DELETE/OPTIONS、`Authorization`、`Content-Type`、`Idempotency-Key` 等 contract 必要 headers；不可使用 wildcard origin 搭配 credentials。

## 11. Vertical slices

### Dependency ordering

```text
Admin hosting/auth transport Gate ──> Slice 0 contract/auth foundation
                                           ├──> Slice 1 catalog ──> Slice 2 quote/order
                                           ├──> Slice 3 site content/publish
                                           └──> Slice 4 staff identity
Payment/inventory/tax/shipping Gates ────────────────────> Slice 2
Publish Gate ────────────────────────────────────────────> Slice 3
Customer/staff lifecycle Gate + Slice 2 ────────────────> Slice 4 customer history
PII policy + Slices 1–4 ────────────────────────────────> Slice 5 cleanup/release
```

- Slice 0 固定 identity、error、version 與 admin transport contract，是所有 admin mutation 的共同前置。
- Slice 1 必須先提供 canonical variant/price/inventory，Slice 2 才能安全 quote/order。
- Slice 3 在 Slice 0 後可與 Slice 1/2 並行，因為由 `content` 擁有且只在 tools/bootstrap 組合。
- Slice 4 的 staff lifecycle 只依賴 Slice 0；customer history 依賴 Slice 2 已有 orders。
- Slice 5 只清除已被真實 producer 取代或已從 production UI 移除的 fixture，不能提早刪除仍被未完成流程使用的 demo 資料。

### Slice 0 — Contract、auth 與開發基礎

**Controlled-spec coverage**

`REQ-001`、`REQ-002`、`REQ-008`、`REQ-011`、`REQ-012`；`AC-001`、`AC-002`、`AC-003`、`AC-004`、`AC-015`、`AC-021`、`AC-023`。

**Outcome**

Admin 以可驗證 principal 呼叫 Go，OpenAPI、server enforcement 與 UI capability 使用同一個 contract；本 slice 不宣稱 commerce 已可用。

**Entry → validation → owner → contract → result**

Supabase access token → verifier 以 `/auth/v1/user` 或 local JWKS 驗 token，建立可信 identity/AAL/session 並在 Slice 0 讀 immutable bootstrap role → `auth`/bootstrap → `/api/admin/me` 與共用 error schema → admin 取得 role/capabilities；每個 service 仍自行檢查 capability。

**Hard dependencies**

- 確認 Admin hosting/origin 與 production token transport。
- 為 payment、customer account、inventory/return restock、tax/rounding、shipping scope Gate 指定 owner 與決策期限；它們只阻擋各自的後續 slice，不阻擋 Slice 0 的 auth foundation。
- 若需要新增 production auth dependency，先取得明確批准。

**Allowed paths**

- `.ai/scope.json`
- `contracts/openapi.yaml`
- `server/internal/auth/**` 與相鄰 tests
- `server/internal/bootstrap/app.go`、`server/internal/config/**`
- `admin/src/lib/api-client.ts`、`admin/src/stores/auth.ts`、admin router/sidebar/Topbar 的 auth wiring
- `admin/vite.config.ts`；package manifest/lockfile 僅限已批准 dependency

**Implementation scope**

- 確認 admin hosting Gate；為其餘 product Gates 記錄 owner、採納的預設或明確 blocked 狀態。
- 更新 OpenAPI 共用 response/error 與 `GET /api/admin/me`。
- 建立 server-side role/capability matrix。
- Admin 接 Supabase/dev session、Bearer token 與 `/api/admin/me`。
- 若登入實作需要新增 production dependency，先取得明確批准；不得在本 slice 靜默加入。
- API client 補 PATCH、runtime base、structured errors。
- 修正 CORS methods/origin allowlist 與 Vite port 衝突。

**Acceptance evidence**

- HTTP/auth tests：無 token -> 401；有效 token 但無 capability -> 403；錯 project/signature/audience/expiry 與 remote verifier outage 被拒絕。
- `/api/admin/me` response 與 admin 顯示的 role/capabilities 相同；production build 查無 role switcher/default-owner path。
- `go test` 的 auth/bootstrap targeted tests、admin `npm run typecheck` 與 `npm run build` 通過。
- OpenAPI example 與實際 handler response 的 error shape 相同。

**Rollback / safe failure**

- Auth/config 失敗時 default deny；不退回 UI role selector 或無驗證 principal。
- Dev verifier 只能由明確 local config 啟用，production config 啟用時 startup fail closed。
- 本 slice 無資料 migration；可回退 admin wiring，但不可回退 server-side authorization checks。

### Slice 1 — Catalog：admin -> DB -> static storefront

**Controlled-spec coverage**

`REQ-001`、`REQ-003`、`REQ-008`、`REQ-010`、`REQ-011`、`REQ-012`；`AC-001`、`AC-002`、`AC-005`、`AC-006`、`AC-015`、`AC-016`、`AC-019`、`AC-020`、`AC-021`、`AC-022`、`AC-023`。

**Outcome**

一個 published product 從 admin 表單與 R2 metadata，經 Go/DB/renderer 成為可讀 static storefront；draft/archived 與假評論不會外洩。

**Entry → validation → owner → contract → result**

Admin product/media input → Go 驗 slug/SKU/整數價格/variant/media policy/capability → `commerce` + R2 metadata → catalog admin API + `render.Input` → semantic product HTML 與同源 island snapshot。

**Hard dependencies**

- Slice 0 的 principal/capability/error/version contract。
- R2/CDN public URL 組合規則與 upload size/content-type policy。

**Allowed paths**

- `.ai/scope.json`、`contracts/openapi.yaml`
- `architecture.yaml`（登記 `commerce` module ownership）
- `db/migrations/sqlite/002_catalog.sql`、`db/migrations/postgres/002_catalog.sql`
- `server/internal/modules/commerce/**`、`server/internal/modules/media/**` 中 catalog upload policy 的最小修改
- `server/internal/bootstrap/app.go`、`server/internal/render/**`、`server/tools/render/**`、`server/tools/dev/**`、catalog seed 的精確路徑
- `admin/src/config/resources/products.ts` 與 product/category list/form/upload 所需的精確共用元件/API state paths
- `site/themes/minimal-cart` 的 product types、Header catalog navigation、ProductGrid/ProductDetail/ProductCard/CategoryFilter、render templates 與 catalog fixture imports

**Implementation scope**

- 雙 driver `002_catalog.sql`。
- `commerce` catalog model/store/service/handler 與測試。
- Product admin list/create/update/publish/archive/bulk。
- Category admin list/create/update/enable/disable，並供 renderer 產 filter label/order。
- Product image presign + R2 upload + post-upload verification/metadata，限制實際格式、pixel/byte size、object key；SVG 預設拒絕。
- Renderer 取得 published products/images/variants。
- Home/catalog 與 ProductDetail 使用 published snapshot。
- Renderer 產 product/category static routes，ProductDetail dialog 與 static detail page 共用同一 snapshot。
- 只移除 catalog 相關 fixture；其他未完成區域明確維持 demo 或隱藏。
- Slice 1 以 local/CI full render 證明 storefront；production admin-to-Cloudflare trigger/status 由 Slice 3 補齊。在 Slice 3 完成前，production product publish 必須走已記錄的 operator/CI publish path，UI 不得假稱已部署。

**Acceptance evidence**

- E2E artifact：Admin 新增商品與圖片 -> publish -> render -> `dist/` 出現正確 slug/SKU/image/variant；draft/archived 不存在。
- SQLite/PostgreSQL migration 與 store tests 證明 product/category slug syntax/immutability/unique、SKU unique、category FK/disable invariant、整數非負價格、variant stock constraint。
- Media tests 證明偽造 content type、超限 bytes/dimensions、非圖片與 SVG 被拒絕，未驗證 object 不會建立 product metadata。
- 關閉 JavaScript 檢查公開 HTML 仍可讀商品並可用 Header/category/product links 導航；島資料與 static routes 使用同一 snapshot，output path 無法逸出 `dist/`。
- `rg` 證明 production catalog consumers 不再 import PRODUCTS/CATEGORIES/reviews fixture。
- Targeted Go tests、admin build、theme `build:check`、render 與 dist inspection 通過。

**Rollback / safe failure**

- Migration 只新增表；回退應停用新 route/UI，不破壞或刪除已建立 catalog rows。
- Render/upload 失敗不切換公開版本，保留 last-known-good `dist/`；Slice 3 前只回報 DB publish 與 operator/CI render 結果，不假造 Cloudflare success receipt。
- 禁止以 mock catalog 補位；尚無 published 商品時輸出真實 empty state。

### Slice 2 — Quote、order creation 與 admin fulfillment

**Controlled-spec coverage**

`REQ-004`、`REQ-005`、`REQ-008`、`REQ-009`、`REQ-010`、`REQ-011`、`REQ-012`；`AC-007`、`AC-008`、`AC-009`、`AC-010`、`AC-015`、`AC-016`、`AC-017`、`AC-018`、`AC-019`、`AC-020`、`AC-021`、`AC-022`、`AC-023`。

**Outcome**

Server 成為價格、優惠、運費、付款可用性、庫存與訂單狀態的唯一 authority；重試不重複建單，競爭不超賣。

**Entry → validation → owner → contract → result**

Browser variant/quantity/promo/配送付款選擇與 PII → Go 重新計價、驗 availability/capability/idempotency 並在 transaction 內扣庫存 → `commerce` → quote/order/admin transition API → customer receipt、admin fulfillment、append-only events。

**Hard dependencies**

- Slice 1 的 canonical product/variant/inventory。
- 付款語意、庫存/return restock、稅務/捨入、配送範圍、admin step-up Gate；只啟用可實際履約且符合 re-auth policy 的 payment/refund action。
- PII log redaction 規則；retention Gate 可延至上線前，但資料收集範圍必須先最小化。

**Allowed paths**

- `.ai/scope.json`、`contracts/openapi.yaml`
- `db/migrations/sqlite/003_orders.sql`、`db/migrations/postgres/003_orders.sql`
- `server/internal/modules/commerce/**` 的 quote/order/promo/shipping/payment 邏輯與 tests
- `server/internal/bootstrap/app.go`、必要 config/rate-limit wiring
- Theme cart store、CartDrawer、CheckoutDialog、order receipt 及相關 types
- Admin orders/payment/promos resource、dashboard commerce read model 與精確共用 API state paths

**Implementation scope**

- 雙 driver `003_orders.sql`。
- Promo、shipping、payment settings、quote、orders、items、events。
- Transactional repricing、inventory update、idempotency。
- CheckoutDialog 改用 quote/order API。
- Admin order list/detail/fulfillment/return/offline-payment。
- Dashboard 的 commerce KPI/task。
- 移除 cart pricing、promo、shipping、payment 與 order fixture。

**Acceptance evidence**

- Security/service tests：竄改 request price/total 不影響 server；disabled/expired promo與 payment method 被拒絕。
- Store/concurrency tests：相同 Idempotency-Key 只建一單；競爭最後庫存不超賣；transaction 失敗不留半張 order/event。
- Mutation tests：stale `expected_version` -> 409；非法 transition 不更新 row、不寫 event。
- E2E artifact：published product -> quote -> order -> admin 看單 -> 合法出貨；receipt totals 等於 persisted snapshots。
- `rg` 證明 Checkout/cart/order production path 無 local pricing、fake order、fake paid fallback。
- Targeted Go tests、admin/theme builds 與完整 verify 通過。

**Rollback / safe failure**

- 可 feature-disable checkout/admin mutations，但保留已建立訂單與 audit events；不執行破壞性 down migration。
- 建單/扣庫存/事件任一步失敗由 transaction 全部 rollback；外部 payment 未確認時不得標記 paid。
- Client 遇 timeout 使用同一 idempotency key 查回/重試，不自行產生成功 receipt。

### Slice 3 — Site content 與發布

**Controlled-spec coverage**

`REQ-006`、`REQ-007`、`REQ-008`、`REQ-010`、`REQ-011`、`REQ-012`；`AC-011`、`AC-012`、`AC-013`、`AC-014`、`AC-015`、`AC-016`、`AC-019`、`AC-020`、`AC-021`、`AC-022`、`AC-023`。

**Outcome**

Draft site content 只有明確 publish 後才進入 full render；公開首頁即使沒有 JavaScript，也能讀到最後一次成功發布的內容。

**Entry → validation → owner → contract → result**

Admin placement form → `content` 驗 placement payload/version/capability → `site_content` → admin content API + renderer input → `dist/` -> Cloudflare Pages；publish job 狀態回到 admin。

**Hard dependencies**

- Slice 0 auth/error/version contract。
- 發布策略 Gate：選擇 trigger-only Deploy Hook 或可保存 deployment ID/status 的 Pages API；定義 secret ownership 與 last-known-good 行為。

**Allowed paths**

- `.ai/scope.json`、`contracts/openapi.yaml`
- `architecture.yaml`（更新 `content` 對 site content/publish receipts 的 ownership）
- `db/migrations/sqlite/004_site_content.sql`、`db/migrations/postgres/004_site_content.sql`
- `server/internal/modules/content/**`、`server/internal/bootstrap/app.go`
- `server/internal/render/**`、`server/tools/render/**`、`server/tools/publish/**` 與必要 deploy config/`.env.example`
- Direct Upload tooling manifest/lockfile 僅限已批准 Wrangler pin；不得順手加入其他 production dependency
- Admin content resource/editor/publish status 的精確 paths
- Theme Hero、AnnouncementBar、WelcomePopup、Footer、FooterPageDialog、home template feature strip 與 content fixture/hardcoded claims

**Implementation scope**

- 雙 driver `004_site_content.sql`。
- 既有 `content` module 新增 site content store/service/handler。
- Admin content list/edit/publish。
- Hero、Announcement、WelcomePopup、Footer 改吃 render snapshot。
- Policy pages 與 feature strip 只發布經核准且與實際 commerce/PII 能力一致的 bounded blocks；未核准頁面從 production navigation 移除。
- `publish_runs`、選定的 Cloudflare trigger/status transport 與發布失敗狀態。
- Direct Upload 若保留，將 Wrangler 鎖定在已批准的 tooling manifest/lockfile 或要求 CI 提供已驗證版本；admin API process 絕不 shell out 到 `npx`。
- 移除 site content fixture。

**Acceptance evidence**

- Tests/artifacts 證明 save draft 不改 published copy/`dist/`，publish 才觸發 full render/deploy，invalid placement payload/unknown fields 被拒絕。
- 模擬 render/Cloudflare trigger/status 失敗：public 仍是上一版；admin 只在查到 terminal failure 時顯示 failed，trigger-only 模式只顯示 requested。
- 關閉 JavaScript 檢查 Hero/announcement/footer 等關鍵內容仍在 HTML；popup enhancement 讀同一 snapshot。
- Policy pages 有可直接開啟的 static URL，Footer links 在無 JavaScript 時可導航；dialog 與 static page copy 來自同一 published row。
- `rg` 證明 site-content consumers 不再 import fixture constants。
- Policy/feature approval tests：缺 approver/date、過期 review、未實作 payment/shipping claim 都無法 publish；dist 查無 demo promo、電話、保固或虛假資料處理陳述。
- Publishing tooling test/CI 證明不會在 runtime 臨時下載未鎖版 Wrangler，且 secret 不出現在 command line/log/response。
- Targeted Go tests、admin/theme builds、render 與 dist inspection 通過。

**Rollback / safe failure**

- Migration 只新增表；停用 publish UI/route 時保留 draft/published rows。
- Render 或 deploy 失敗絕不覆蓋目前成功版本；不得 fallback 到 fixture 或把 draft 當 published。
- Deploy Hook URL 或 Pages API token 只在 server config；缺失時 startup/operation 明確失敗，不記錄或回傳 secret。

### Slice 4 — Customer/member/staff identity

**Controlled-spec coverage**

`REQ-002`、`REQ-008`、`REQ-009`、`REQ-010`、`REQ-011`、`REQ-012`；`AC-004`、`AC-015`、`AC-017`、`AC-018`、`AC-019`、`AC-020`、`AC-021`、`AC-022`、`AC-023`。

**Outcome**

Customer 只能存取自己的 order；staff role lifecycle 由 Go 維護並受 owner invariants 保護，production 不再有 demo account/role elevation path。

**Entry → validation → owner → contract → result**

Verified customer/staff token + mutation target → Go 以 subject linkage/active staff row 驗 object ownership、capability、last-owner/self-elevation invariants → `commerce` member / canonical `staff` row + Supabase metadata outbox mirror → account/order/staff API → scoped history與立即生效的 server capabilities。

**Hard dependencies**

- Slice 0 principal/capability contract；customer order history 另依賴 Slice 2。
- Customer account 與 staff invite/deactivate/re-auth/session refresh lifecycle、admin step-up Gate。
- PII retention/deletion policy 在上線前必須定案。
- Supabase server-side admin credential 的最小權限、rotation 與 hosting secret ownership；service-role value 不得進 browser、DB、log 或 committed env file。

**Allowed paths**

- `.ai/scope.json`、`contracts/openapi.yaml`
- `architecture.yaml`（登記 `staff` 與 identity sync ownership）
- `db/migrations/sqlite/005_identity.sql`、`db/migrations/postgres/005_identity.sql`
- `server/internal/auth/**`、`server/internal/modules/staff/**`、`server/internal/modules/commerce/**` 的 member/order-access 最小 paths
- `server/internal/bootstrap/app.go` 與必要 Supabase config/adapter/`.env.example` key name（不得寫 secret value）
- Admin staff/members/auth/router paths
- Theme account/login/order-history/track-order/user store 與相關 types

**Implementation scope**

- 確認 customer account 與 staff invite lifecycle。
- 雙 driver `005_identity.sql`。
- Guest order 與 Supabase customer identity 的 member linkage。
- Account order history 與安全的 order access。
- Staff profile/role 管理，經 Go 更新 Supabase `app_metadata.role`。
- 移除 DEMO_ACCOUNTS、local fake orders 與 production role switcher。

**Acceptance evidence**

- Object-level authorization tests：顧客只能讀自己的訂單；order number 單獨不足以 tracking；staff list/detail 的 PII 權限不同。
- Privilege tests：body/`user_metadata` role 無效；manager 不能自升、管理 owner 或移除最後 owner。
- Role mutation/停用後，即使使用舊 token，下一個 request 也依 canonical staff row 得到新 capability 或被拒絕；refresh session 後 JWT mirror 與 `/api/admin/me` 一致。
- Supabase metadata sync failure 產生可重試 outbox receipt，不能恢復已撤銷權限；audit 記 actor/target/result 且無 secret/完整 PII。
- `rg`/production build 證明無 DEMO_ACCOUNTS、fake order history、production role switcher。
- 雙 driver migration、targeted Go tests、admin/theme builds 與 verify 通過。

**Rollback / safe failure**

- Identity sync 失敗時 server 仍依 canonical staff row 授權並回報/重試 outbox；不得把 metadata sync 失敗假裝成功，也不得讓舊 claim 恢復已撤銷權限。
- 可隱藏 account/staff UI、停用 mutation route，但保留 profiles、linkages 與 audit records。
- 不以刪除 identity rows 作 rollback；PII deletion 只能依已確認 policy 執行。

### Slice 5 — Cleanup and release

**Controlled-spec coverage**

`REQ-010`、`REQ-011`、`REQ-012`；`AC-019`、`AC-020`、`AC-021`、`AC-022`、`AC-023`、`AC-024`。

**Outcome**

只剩明確允許的 demo/dev fixture；production contracts、bundles、migrations、render output 與部署文件彼此一致，可用證據宣稱整合完成。

**Entry → validation → owner → contract → result**

Slices 1–4 的完成證據 + repository-wide searches → scopecheck/contracts/build/render/security matrix → 各既有 owner → release artifacts/docs → 可部署且無 fabricated fallback 的 release candidate。

**Hard dependencies**

- Slices 1–4 的 acceptance evidence 完成。
- PII retention/deletion 與任何延後的 production Gate 已簽定。
- PostgreSQL validation environment 可用。

**Allowed paths**

- 先由 repository search 產出精確 fixture/dead/docs/CI path 清單，再寫入 `.ai/scope.json`；禁止預先給 directory-wide cleanup wildcard。
- 允許 `README.md`、`contracts/openapi.yaml`、deployment docs、seed、CI 的精確必要修改；不藉 cleanup 重構 business modules。

**Implementation scope**

- `rg "mock-data|fixture|DEMO_ACCOUNTS"` 檢查 production imports。
- 更新 README、OpenAPI、seed 與 deployment docs。
- Seed 依 table 個別 idempotent，不因 articles 已存在就跳過其他 seed。
- 完整 SQLite E2E；PostgreSQL migration/integration validation。
- 移除未使用 API path、operation ID 與 dead UI。

**Acceptance evidence**

- Negative searches 與 bundle inspection 證明無假帳密、假付款成功、可購買 fixture 商品或 API failure fixture fallback。
- Contract tests/field ledger audit 證明 frontend/API/OpenAPI 的 ID、日期單位、boolean、integer TWD、version 與 error shape 一致。
- SQLite clean-migrate/seed/E2E 與 PostgreSQL clean-migrate/integration 通過；每張 seed table 個別 idempotent。
- `go run ./server/tools/verify`、admin `npm run build`、theme `npm run build:check`、render 與人工/自動 dist inspection 全部通過。

**Rollback / safe failure**

- 任一 completion gate 失敗就停止 release；不刪 production data、不做破壞性 migration rollback。
- Cleanup 只刪已被 acceptance evidence 證明無 consumer 的 fixture/dead path；若 consumer search 仍命中就保留並回到所屬 slice 修正。
- Release 失敗維持 last-known-good Pages/API deployment，修正後重新產生完整證據。

## 12. Test and validation matrix

每個 slice 都執行最窄測試，再執行完整驗證。

### Go

- Service unit tests：validation、authorization、state transition。
- Store/integration tests：SQLite migration、transaction、version conflict、idempotency。
- HTTP tests：request limits、unknown fields、status/error contract。
- PostgreSQL migration 在 CI 或可用 integration environment 驗證。
- Renderer integration tests：一致 read snapshot、slug/path containment、任一 page 失敗不 promote partial `dist/`。

```powershell
go run ./server/tools/scopecheck
go run ./server/tools/verify
```

### Admin

```powershell
Set-Location admin
npm run typecheck
npm run build
```

### Theme and publishing

```powershell
Set-Location site/themes/minimal-cart
npm run build:check

Set-Location ../../..
go run ./server/tools/render
```

檢查 `dist/`：

- published catalog/content 存在。
- draft/archived 資料不存在。
- island assets 路徑有效。
- render manifest/page count 完整，staging failure 不污染 last-known-good output。
- 無 fixture fallback 或 demo credential。
- 關鍵內容在預期的 static snapshot 中。

### Security regression cases

- 無 token、過期 token、錯誤 role。
- Role demotion/disable 後的舊 token、`user_metadata`/body role injection、AAL 不足的 high-risk action。
- object-level authorization。
- 被竄改的 price/discount/total。
- promo enumeration 不存在。
- duplicate submit/idempotency。
- stale version 與 inventory race。
- 不合法 order/return/payment transition。
- upload 偽造 content type、magic bytes、pixel/byte limit、key namespace 與 SVG policy。
- Site content XSS、external URL allowlist、island JSON `</script>` escaping。
- Product/category slug traversal、encoded separator、reserved route collision 與 renderer output containment。
- Deploy Hook/Pages API/Supabase admin secrets 不出現在 browser、DB、command line、log 或 error response。
- CORS allowlist。
- PII 不出現在 access log 或一般 list response。

## 13. Definition of Done

只有同時滿足以下條件，才能宣稱「前後台完成對接、fixture 已移除」：

- Admin 使用真實 session/token，server 是唯一 authorization authority。
- 商品 CRUD、圖片、publish 與 static storefront 完整串通。
- Quote/order 由 Go 重新計價並以 transaction/idempotency 保護。
- Admin 能查看與合法推進訂單狀態，所有操作有 audit event。
- Site content 走 content -> renderer -> dist -> Pages。
- Customer/account/track-order 要嘛完成真實 API，要嘛從 production UI 移除。
- SQLite/PostgreSQL migrations 等價。
- `contracts/openapi.yaml` 與實作一致。
- Production code 不 import fixture/mock credentials，也不在 API 失敗時顯示假交易資料。
- `go run ./server/tools/verify`、admin build、theme build、render 與 dist inspection 全部通過。

## 14. Review baseline（2026-08-11）

這是規格審查的現況 baseline，不是未來 slices 的完成證據。

| Check | Result | Interpretation |
|---|---|---|
| `codegraph` availability | unavailable | 本輪以 `rg`、routes、migrations、OpenAPI、producer/consumer source 逐項追蹤；沒有假稱做過 graph impact query |
| `go run ./server/tools/archcheck` | pass | 現有 module dependency boundary 通過 |
| `go test ./...` | pass | 現有 Go tests 通過；repository 尚無 commerce/order integration tests |
| `go vet ./...` | pass | 現有 Go code 通過 vet |
| `admin: npm run typecheck` | pass | 現有 Admin SPA typecheck 通過；不代表真實 auth/data integration 已存在 |
| `minimal-cart: npm run typecheck` | pass | 現有 theme typecheck 通過；不代表 fixture 已移除 |
| `go run ./server/tools/speccheck` | blocked by pre-existing protected-path changes | 全域輸出 6,597 行，沒有一行指向 `minimal-cart-integration`；本變更的 artifact consistency 檢查為 12 REQ、24 AC、31 scope paths、0 issue |
| `go run ./server/tools/scopecheck` | blocked by pre-existing out-of-scope changes | 全域輸出 6,606 行；列出的既有變更包含舊 admin workspace、`frontend-sample/**` 與未納入本文件工作的 theme build/config paths |
| `go run ./server/tools/verify` | blocked at speccheck | `archcheck` 先通過；更新後的 verify 在 speccheck 擋下未受控 protected changes，尚未進入後續 gate |
| admin/theme production build、render、dist inspection | not run for this plan-only review | 沒有 production/publishing code 變更；它們仍是各 slice 與 Definition of Done 的必要 acceptance evidence |

本輪只在既有文件工作範圍追加 `specs/changes/minimal-cart-integration/**`，沒有把產品碼或整個 dirty tree 加入 allowlist。完整 speccheck/scopecheck 的失敗屬 repository baseline，不應藉由擴大 scope 掩蓋。

### External contract validation

- Cloudflare 官方文件把 Pages Deploy Hook 定義為觸發新 build，且 Hook URL 本身必須視為 secret；部署的 `latest_stage.status` 需經 Pages API 取得。因此 §4/§9/§11 不再把 Hook HTTP success 當成 deployment success：[Deploy Hooks](https://developers.cloudflare.com/pages/configuration/deploy-hooks/)、[Get deployment info](https://developers.cloudflare.com/api/resources/pages/subresources/projects/subresources/deployments/methods/get/)。
- Supabase 官方文件要求 admin user update 只在 server 執行，metadata 改變後 client refresh session 才會取得更新 claim；JWT 驗證仍必須檢查 signature、issuer、audience、expiry。因而 Slice 4 以 canonical staff row 立即授權，`app_metadata.role` 只作 mirror：[updateUserById](https://supabase.com/docs/reference/javascript/auth-admin-updateuserbyid)、[JWT claims/validation](https://supabase.com/docs/guides/auth/jwt-fields)。

## 15. Review completion gate

| Required proof | Status | Evidence in this plan |
|---|---|---|
| Repository-backed claims | met | §2 evidence ledger；existing/missing/unverified 分開 |
| Real consumer trace per important flow | met | §2 catalog、checkout/order、site content、identity/staff、media traces |
| Schema/consumer field reconciliation | met | §2 field ledger + §6/§8 canonical schema/contracts |
| Explicit authority and static/runtime ownership | met | §3 authority matrix、static/runtime matrix |
| Fixture/fallback disposition | met | §2 fixture ledger + 各 slice negative evidence/safe failure |
| Dependency-ordered vertical slices | met | §11 dependency graph；每個 slice 有 outcome、trace、hard dependencies、allowed paths、acceptance、rollback |
| Narrow review scope and baseline | met with repository baseline limitation | `.ai/scope.json` 與 §14；完整 verify 被既有 dirty tree 擋住，但 plan evidence 可獨立審查 |
| Product decisions/blockers identified | met | §4；payment、inventory/return restock、tax/rounding、shipping、admin step-up、publishing、hosting、legal/ops copy、identity、PII 有最晚決定點 |
| Security-sensitive boundaries have proof requirements | met at specification level | §3 authority、§7 auth、§8 idempotency/errors、§12 security regression；尚未實作或測試的能力沒有標成完成 |
| Controlled-spec traceability and approval | traceability met; approval blocked | Revision 1 的 `control.json`、`spec.md`、`plan.md`、`evidence.md` 已建立；12 個 REQ、24 個 AC 與 slices/surfaces 已映射，但狀態仍是 Draft |

**Review verdict：架構方向通過；Revision 1 是 review-ready Draft，不是 apply-ready。**

- Draft 狀態下任何 slice 都不得開始產品碼；必須先明確批准 `minimal-cart-integration` Revision 1。
- Revision 1 批准且 GATE-001 解除後，下一個唯一 eligible slice 是 Slice 0；Slice 1 仍須等待 Slice 0 的 acceptance evidence。
- Slice 2 在 payment、inventory/return restock、tax/rounding、shipping scope、admin step-up Gate 未定前保持 blocked。
- Slice 3 在 Cloudflare trigger/status transport、last-known-good 發布責任與 legal/ops copy approver 未定前保持 blocked。
- Slice 4 在 customer/staff lifecycle、admin step-up、Supabase admin secret ownership 與 PII policy 未定前不可宣稱完成。
