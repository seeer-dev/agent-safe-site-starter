# Admin 後台架構評估報告

> 評估範圍：`admin/` 前端 + `server/` Go 後端 + `contracts/openapi.yaml`
> 評估維度：維護性、AI agent 易改、邊界、約定、複用、新增站台擴充性
> 日期：2026-08-13（2026-08-14 兩輪對照原始碼查證並修訂）

**修訂紀錄**：第二輪查證推翻了第一輪的三個結論——(1) 補完 11 條路由**不足以**支撐型別生成，既有 42 條也有 status/schema 漂移；(2) `roles.ts` **不是零風險刪除**，build-time checker 會讀它；(3) swaggo 方案**不可行**（產 Swagger 2.0，與現有 OAS 3.1 不相容）。並新增第 8b 節（三項功能缺陷 + 契約閘缺口），優先序整體重排。

**相關文件**：`docs/backend-optimization.md` 涵蓋 Go 後端（正確性缺陷、上線拓撲、可觀測性、效能）。兩份都指出 OpenAPI 缺 11 條後台路由，補齊時一次做完。

**注意兩處交集**：
- 本文第 5 節的 `listKey` 與後端文件 P3-2 的分頁是同一次前端改動——`ResourceListPage.vue` 目前**沒有任何分頁 state**（`pageSize` 只是 `:757` 的一行顯示文字），加分頁時會一併重寫取列表的路徑。
- 本文建議的 admin 型別生成，前提是後端文件 P1-1 的正式 API base 已決定——`api-client.ts:3` 的 `API_BASE = '/api'` 目前硬編碼，跨源部署時要能由建置期注入。

---

## 目錄

1. [現狀總覽](#1-現狀總覽)
2. [核心問題：API 契約斷層](#2-核心問題api-契約斷層)
3. [Swagger → 前端型別：可行路徑](#3-swagger--前端型別可行路徑)
4. [新增站台的路由、權限、資源擴充](#4-新增站台的路由權限資源擴充)
5. [後端響應架構與 types](#5-後端響應架構與-types)
6. [元件庫拆分與維護](#6-元件庫拆分與維護)
7. [ResourceListPage 拆解計畫](#7-resourcelistpage-拆解計畫)
8. [Config 系統統一](#8-config-系統統一)
8b. [功能缺陷（優先於結構重構）](#8b-功能缺陷優先於結構重構)
9. [AI Agent 易改性檢查表](#9-ai-agent-易改性檢查表)
10. [優先級與建議路線圖](#10-優先級與建議路線圖)
11. [與 SiteCore 的關係](#11-與-sitecore-的關係)

---

## 1. 現狀總覽

### 架構事實

| 層 | 技術 | 現狀 |
|---|---|---|
| Go 後端 | `net/http` ServeMux, 無框架 | 單站、無多租戶 |
| API 回應 | `httpx.JSON(w, status, map[string]any{...})` | 無標準信封，直接 JSON |
| OpenAPI | `contracts/openapi.yaml` 手寫 | 1285 行，42 operationId |
| 前端 | Vue 3 + Pinia + Vue Router | config-driven CRUD |
| 前端 API client | `api.get<T>(path)` 泛型 fetch | 不解析信封、不生成型別 |
| 前端型別 | `Record<string, any>` 31 處（另 5 處在測試） | 無 swagger 生成 |
| 前端 config | `RES` (資源) + `PROFILE` (導航) 雙定義 | 手動同步 |
| 前端頁面 | `ResourceListPage.vue` 1094 行 | god component |

### 後端回應格式（無信封）

```go
// 成功
httpx.JSON(w, 200, map[string]any{"products": products})
httpx.JSON(w, 200, map[string]any{"items": siteContent})
httpx.JSON(w, 200, map[string]any{"order": order})

// 錯誤
httpx.Error(w, 403, "forbidden")
// → {"error": "forbidden"}
```

**問題**：每個端點的回應 key 不同（`products`、`items`、`order`），前端用 `Object.values(data).find(Array.isArray)` 猜測，脆弱。

### OpenAPI spec 現狀

- 手寫 YAML，與 Go handler 沒有自動同步
- 有完整的 schema 定義（`Product`、`AdminProductResponse`、`Order`、`SiteContent` 等）
- 有 operationId（`adminMinimalCartProductsList` 等）
- **前端完全不用這份 spec** — 所有型別手寫，所有 API 路徑手寫

---

## 2. 核心問題：API 契約斷層

這是所有其他問題的根源。目前有**三份事實**，互相不一致：

```
contracts/openapi.yaml   ← 手寫，可能過時
       ↕ (無自動同步)
Go handlers              ← 真實行為
       ↕ (無型別生成)
admin/src/config/*.ts    ← 手寫路徑 + Record<string,any>
```

### 斷層清單

| 斷層 | 影響 | AI agent 會犯的錯 |
|---|---|---|
| OpenAPI ↔ Go：手寫 spec 可能過時 | 加了端點忘了改 spec | agent 讀 spec 以為有/沒有某端點 |
| Go ↔ 前端：回應 key 不一致 | `products` vs `items` vs `order` | agent 不知道新端點的 key 是什麼 |
| 前端 config ↔ Go：ops 字串無驗證 | `ops.status: 'adminMinimalCartOrdersStatusUpdate'` 在 Go 裡不存在 | agent 改了 Go operationId 但前端沒跟 |
| 前端型別 ↔ Go DTO：無生成 | `AdminProductResponse` 有 `product_images` 但前端用 `images` | agent 用錯欄位名 |

### 具體案例：OpenAPI 缺 11 條已實作的後台路由

契約斷層不是假設，現在就已經發生。以下路由在 `bootstrap/app.go` 有註冊、admin 的 resource config 有引用，但 `contracts/openapi.yaml` 裡**完全沒有**：

| 路由 | 註冊於 | admin 引用的 operationId |
|---|---|---|
| `PATCH /api/admin/products/{id}/status` | `app.go:158` | `adminMinimalCartProductsUpdateStatus` |
| `POST /api/admin/products/bulk` | `app.go:159` | `adminMinimalCartProductsBulkUpdate` |
| `GET/PUT/PATCH /api/admin/members*` | `app.go:169-171` | `adminMinimalCartMembers*`（3 個） |
| `GET/POST/PUT/DELETE /api/admin/promos*` | `app.go:174-177` | `adminMinimalCartPromos*`（4 個） |
| `GET/PUT /api/admin/payment-methods*` | `app.go:180-181` | `adminTwCommerceMethods*`（2 個） |

契約覆蓋率約 2/3。這 11 個 operationId 目前指向不存在的條目，而**沒有任何機制會抓到**——`check-resource-contracts.mjs` 比對的是原始碼字串，不是 OpenAPI 文件。

### 更嚴重：既有的 42 條也已經漂移

補完 11 條**不等於**契約就正確了。現存的條目本身就跟 Go 對不上。以 admin products 為例（`openapi.yaml:347-415` vs `commerce/http.go:385/407/426/445/458`）：

| 項目 | OpenAPI 宣告 | Go 實際行為 |
|---|---|---|
| list / get / update 的 response schema | `$ref: Product`（**公開** schema） | `toAdminProductResponse(...)`（admin DTO） |
| `POST` create 的 status | `200` | `http.StatusCreated`（**201**） |
| `DELETE` 的 status 與 body | `204`，無 body | **200** + `{"id": "..."}` |
| `AdminProductResponse`（`openapi.yaml:962`） | 有定義，含 `product_images` | **沒有任何 operation 引用它** |

也就是說 `AdminProductResponse` 是一個定義好卻沒被接上的孤兒 schema，而 admin 的 product 端點在契約上仍宣稱回傳公開的 `Product`。

**這改變了第 3 節的施工順序**：不能只補 path。要先**全面 audit 既有 42 條的 status code 與 request/response schema**，否則型別生成只會把錯誤的契約合法化——生成出來的型別會信誓旦旦地說 create 回 200、delete 回 204、admin 列表回公開 Product，全部是錯的。**帶著錯誤型別的編譯期檢查比沒有型別更危險**，因為它會讓人停止懷疑。

### 型別缺口

`AdminProductResponse` 在 spec 裡定義完整（`openapi.yaml:962`，含 `product_images` 陣列與說明），但前端拿不到這份型別：

```ts
// products.ts:24 — 回應被當成無型別的 bag
rowMap: (raw: Record<string, any>) => ({
  ...raw,
  updated_at: formatUnix(raw.updated_unix),
}),
```

目前欄位名剛好都對得上（`product_images` 前後端一致，且 `check-resource-contracts.mjs` 有守住 legacy `image`/`images` 不得出現在表單）。但這是**靠腳本斷言維持的，不是靠型別**。如果 Go 改了 DTO 欄位名，編譯期不會發現，只有那支字串比對腳本可能會抓到——而它只覆蓋少數幾個欄位。

---

## 3. Swagger → 前端型別：可行路徑

### 現狀：為什麼前端沒用 OpenAPI？

1. Go 後端不用 swaggo annotation，spec 是手寫的
2. 前端是 Vue（不是 React），SiteCore 的 `generate-openapi-client.mjs` 生成 React/TS 但依賴 `apienv` 信封
3. 沒有人建過生成管線

### 建議：從 OpenAPI 生成 TypeScript 型別

**第一步（低成本、高收益）**：只生成型別，不生成 client。

```bash
# 用 openapi-typescript（npm 套件，框架無關）
npx openapi-typescript contracts/openapi.yaml -o admin/src/lib/api-types.ts
```

**產物形狀要先講清楚**：`openapi-typescript` 預設產出的是 `paths` / `components` / `operations` 三個型別映射，**不是一堆可以直接 import 的具名 interface**。而且 `AdminMinimalCartProductsListResponse` 這個名字目前不存在於 spec 裡（那是 operationId，不是 schema 名稱）。實際用法要自己接一層 alias：

```ts
// api-types.ts —— 生成物
import type { components, operations } from './api-types.gen'

// 手寫的 alias 層
export type AdminProduct = components['schemas']['AdminProductResponse']
export type AdminProductsList =
  operations['adminMinimalCartProductsList']['responses'][200]['content']['application/json']
```

**前端使用**：
```ts
// 之前
const data = await api.get<Record<string, any>>(r.api.list)
const arr = Object.values(data).find(v => Array.isArray(v))

// 之後
import type { AdminProductsList } from '@/lib/api-types'
const data = await api.get<AdminProductsList>(r.api.list)
const arr = data.products  // 有型別、有自動補全
```

參考：<https://github.com/openapi-ts/openapi-typescript>

**收益**：
- 欄位名拼錯 → 編譯期報錯
- Go DTO 加了欄位 → 重新生成 → 前端知道
- AI agent 改 API 回應 → 先改 OpenAPI → 生成 → 前端型別同步

**成本**：
- 加一個 dev dependency（`openapi-typescript`）
- 加一個 npm script（`gen:types`）
- 改 33+ 處 `Record<string, any>` → 用生成的型別（可漸進）

### 前置條件：OpenAPI spec 必須準確

目前 spec 是手寫的，可能跟 Go 不同步。兩個選項：

**選項 A**：加 swaggo annotation 到 Go handler，用 `swag init` 生成

**這個選項目前不可行，不要照著做。** 三個具體障礙：

1. **版本不相容**。`contracts/openapi.yaml:1` 是 `openapi: 3.1.0`，但 `swaggo/swag` 穩定版產生的是 **Swagger 2.0**。它不會直接取代現有的 OAS 3.1 YAML，中間需要一層 Swagger2 → OAS3 轉換，而轉換有資訊損失（`nullable`、`oneOf`、複合 schema 都會失真）。文件裡沒有定義轉換策略，也沒有定義誰是 source of truth（轉換後的檔案能不能手改？改了下次生成會不會被覆蓋？）。參考：<https://github.com/swaggo/swag>
2. **成本被低估**。annotation 是 **operation 級**，不是 handler 級。實際有 **53 個 operation**，不是「~20 個 handler」。
3. **`make swagger` 不存在**。`Makefile` 目前只有 `dev / migrate / seed / render / preview / archcheck / scopecheck / verify` 八個 target。

另外：**本 repo 的 `AGENTS.md` 並沒有 swagger 相關規則**（全文 36 行）。若要導入，得先把規則加進 `AGENTS.md`——而它是受保護治理路徑，需要一份 controlled change。

**若要走「Go 為真實來源」這條路，應該選 OAS3-native 的方案**，而不是 swaggo。在選定之前，維持手寫 spec + 加強驗證閘（見下方 B-4）是比較務實的路。

**選項 B**：保持手寫 spec，加一個 contract test 驗證 spec ↔ Go 同步
- 優點：不用改 Go
- 成本：寫測試比寫 annotation 更難維護

### 不建議的方案

| 方案 | 為什麼不 |
|---|---|
| 複製 SiteCore 的 `generate-openapi-client.mjs` | 依賴 `apienv` 信封（`{data, meta, error}`），AI-go-starter 沒有信封 |
| 加 `apienv` 信封到 Go 後端 | 大改動，所有 handler + 所有前端呼叫端都要改，收益不明顯 |
| 用 Zod runtime 驗證 | config 是編譯期常數；runtime 驗證對 config 無意義 |

---

## 4. 新增站台的路由、權限、資源擴充

### 現狀：單站架構

AI-go-starter 是**單站系統**：
- 一個 Go binary、一個 DB、一組路由
- `SITE_THEME` 環境變數選主題（`site/themes/<name>/`）
- 沒有 site registry、沒有 per-site 路由命名空間
- admin 前端服務所有資源，不區分站

### 新增主題（現有架構可做）

```
1. site/themes/<new-theme>/  ← 新主題目錄（templates + islands）
2. SITE_THEME=<new-theme>    ← 環境變數切換
3. admin/src/config/resources/<new>.ts  ← 新資源 config（如果新主題有新資料模型）
4. admin/src/config/resources/index.ts  ← 加入 RES
5. admin/src/config/profile.ts          ← 加入 PROFILE（如果新資源要出現在導航）
```

**問題**：步驟 3-5 是三個手動步驟，key 必須一致，沒有編譯期檢查。

### 建議：讓 RES 成為單一來源

```ts
// config/resources/products.ts — 加導航欄位
export const productsResource: ResourceDef = {
  label: '商品',
  desc: '商品目錄與庫存',
  navIcon: 'Package',           // ← 新增
  navSection: 'primary',        // ← 新增
  readCap: 'twcommerce.read',   // ← 新增
  // ...其他不變
}

// config/profile.ts — 自動衍生
import { RES } from './resources'
export const PROFILE = [
  { key: 'dashboard', label: '總覽', icon: 'LayoutDashboard', section: 'primary',
    component: 'MinimalCartDashboardPage', caps: ['twcommerce.read'] },
  ...Object.entries(RES).map(([key, r]) => ({
    key,
    label: r.label,
    icon: r.navIcon ?? 'Package',
    section: r.navSection ?? 'primary',
    component: 'ResourceListPage' as const,
    caps: r.readCap ? [r.readCap] : [],
  })),
]
```

**新增資源只需 1 步**：在 `config/resources/` 加檔案 + 在 `index.ts` 加入 RES。導航自動出現。

### 權限擴充

現狀：`server/internal/auth/resolver.go` 硬編碼 role → caps 映射。

```go
case "owner": return []string{"twcommerce.read", "twcommerce.create", ...}
case "manager": return []string{"twcommerce.read", "twcommerce.create", ...}
```

**`admin/src/config/roles.ts` 沒有任何 runtime import，但它不是可以直接刪的孤兒檔。**

兩個事實要分開看：

1. **runtime 確實沒用到**。沒有任何 `.vue` / `.ts` import 它（`types.ts:2` 的註解自承是 "Direct port of the mockup"）。前端授權判斷走的是 `stores/auth.ts:150` 打 `/api/admin/me` 拿伺服器算出的 caps——這條路徑是對的。
2. **但 build-time 有東西讀它**。`admin/scripts/check-resource-contracts.mjs:13` 讀進這個檔，`:191-209` 對它做斷言（`content.approve` 必須存在且 owner/manager 都有、不得發明 server 沒授予的 `content.*` cap）。**直接刪檔會讓 `npm run check:resource-contracts` 失敗**——而它現在是 PASS 的。

**漂移比初版描述的更多。** 實際比對 `roles.ts` 的 manager 與 `resolver.go:93`：

```
Go 有但 roles.ts 缺：media.upload、staff.read、twcommerce.admin
```

三個，不是兩個。owner 那組同樣缺 `media.upload` 與 `twcommerce.admin`。而 checker 只驗 `content.*` 這一組，所以這些漂移全部溜過去了。

**建議**：
- Go resolver 保持真實來源（已經是）
- **刪 `roles.ts` 要連同 `check-resource-contracts.mjs:13,191-209` 一起改**，把那幾條 capability 斷言改成直接讀 `resolver.go`，或移進新的語意閘（見第 9 節 B-4）。這是一次協調變更，不是單檔刪除
- 加 capability 時：改 Go resolver → 前端 resource config 的 `cap` 欄位用同一個字串。**兩處，不是三處**
- 更好的作法：從 Go 匯出 capability 常數清單生成一份 TS union type，讓 config 裡拼錯的 `cap` 在編譯期報錯，同時取代 checker 那幾條手寫斷言

### 路由擴充

現狀路由是手寫的 4 條：
```ts
{ path: '/', component: DashboardPage }
{ path: '/res/:resourceKey', component: ResourceListPage }
{ path: '/states', component: StatesPage }
{ path: '/:pathMatch(.*)*', redirect: '/' }
```

**這已經足夠**。新資源不需要新路由 — `/res/<key>` 動態路由覆蓋所有資源。唯一要改的是 PROFILE（導航），而如果 PROFILE 從 RES 衍生，連這都不用改。

---

## 5. 後端響應架構與 types

### 現狀問題

Go 回應沒有標準信封：
```go
httpx.JSON(w, 200, map[string]any{"products": products})  // list
httpx.JSON(w, 200, map[string]any{"order": order})         // single
httpx.JSON(w, 200, map[string]any{"status": "deleted"})    // action result
httpx.Error(w, 403, "forbidden")                            // error → {"error": "..."}
```

前端對應：
```ts
// list：猜 array 在哪 —— 兩處都這樣寫
// ResourceListPage.vue:45
const arr = Object.values(data).find((v) => Array.isArray(v))
// DashboardPage.vue:79
const arr = Object.values(res).find((v) => Array.isArray(v))

// single：知道 key 但手寫
const order = data.order  // 但 data 是 Record<string,any>，沒有型別

// error：api-client 有 ApiError class，OK
```

這個猜法有一個沉默的失敗模式：**如果回應含有兩個陣列欄位，`find` 只會拿到第一個**。

失效方式是「穩定地拿錯」而不是「隨機」——Go 的 `encoding/json` 對 `map[string]any` 會**依鍵名字典序**輸出，所以拿到的永遠是字典序在前的那個陣列。這反而更難察覺：不會間歇性壞掉，而是一直安靜地顯示錯的資料。目前每支端點都只回一個陣列所以還沒觸發。

### 建議：不加信封，但加型別 + 加 listKey

**不加 `apienv` 信封的理由**：
- AI-go-starter 是輕量單站，信封的收益（分頁 meta、統一錯誤碼）目前不需要
- 加信封要改所有 Go handler + 所有前端呼叫端，成本遠大於收益
- AGENTS.md 說「不為假設的未來 provider 加 seam」

**但要做兩件事**：

#### 5a. 在 ResourceDef 加 `listKey`

```ts
// types.ts
export interface ResourceDef {
  listKey?: string  // 'products' | 'orders' | 'items' | ...
}

// ResourceListPage.vue:45
const arr = r.listKey ? data[r.listKey] : Object.values(data).find(v => Array.isArray(v))
```

每個 resource config 明確宣告自己的 list key，不再猜。`DashboardPage.vue:79` 也要一起改——它現在是同一段猜測邏輯的第二個複本。

#### 5b. 從 OpenAPI 生成回應型別

```ts
// 生成出的型別
export interface AdminMinimalCartProductsListResponse {
  products: AdminProductResponse[]
}

// 前端使用
const data = await api.get<AdminMinimalCartProductsListResponse>(r.api.list)
const arr = data.products  // 有型別
```

### Go DTO → OpenAPI → 前端型別的同步鏈

目標鏈路：

```
Go handler (真實行為)
    ↓ 語意契約閘驗證一致性（見 8b-4）
OpenAPI spec (contracts/openapi.yaml，手寫但受閘保護)
    ↓ openapi-typescript
admin/src/lib/api-types.gen.ts + 手寫 alias 層
    ↓ import
config/resources/*.ts + pages/*.vue
```

**注意這裡刻意不畫 `swaggo annotation`**。第 3 節說明了原因：swaggo 產 Swagger 2.0，與現有 OAS 3.1 不相容。在選定 OAS3-native 方案之前，spec 維持手寫，靠契約閘保證它不漂移。

**AI agent 的工作流**：
1. 改 Go handler + DTO
2. 同步改 `contracts/openapi.yaml`（path、status、schema）
3. 跑契約閘 —— 沒改對會紅燈
4. 跑 `npm run gen:types`（生成 api-types.gen.ts）
5. 改前端 config —— 型別會追蹤欄位變更

第 3 步是關鍵。手寫 spec 之所以會漂移，正是因為現在沒有任何東西在第 3 步擋人。

---

## 6. 元件庫拆分與維護

### 現狀

```
admin/src/components/
├── ui/           ← 10 個通用元件
│   ├── Badge.vue       ✅ 使用中
│   ├── Button.vue      ✅ 使用中
│   ├── Checkbox.vue    ✅ 使用中
│   ├── ConfirmDialog.vue  ⚠️ 混了 approve 專屬邏輯
│   ├── EmptyState.vue  ❌ 未使用
│   ├── Input.vue       ✅ 使用中
│   ├── Modal.vue       ✅ 使用中
│   ├── Panel.vue       ❌ 未使用
│   ├── Select.vue      ✅ 使用中
│   └── Textarea.vue    ✅ 使用中
├── resource/     ← 1 個資源元件
│   └── ResourceTable.vue
├── layout/       ← 4 個版面元件
├── auth/         ← 1 個認證元件
└── MediaUploader.vue  ← 600 行，獨立
```

### 問題

1. **ConfirmDialog 混了 approve 專屬邏輯**（`expiryInput` prop）
2. **EmptyState 和 Panel 未使用** — 死碼
3. **表單欄位渲染是 ResourceListPage 裡的 v-if 鏈** — 加新 widget 要改頁面 template

### 建議（小專案尺度）

不需要拆成獨立套件或建 widget registry。只需要：

#### 6a. 刪死碼

刪除 `EmptyState.vue` 和 `Panel.vue`（未使用）。如果未來需要，再建。

#### 6b. ConfirmDialog slot 化

```vue
<!-- ConfirmDialog.vue — 只管 confirm/cancel/reason -->
<ConfirmDialog :open="..." :title="..." @confirm="..." @cancel="...">
  <template #extra>
    <!-- approve 的 expiry input 由呼叫端注入 -->
    <div v-if="needExpiry">
      <label>生效時間</label>
      <Input type="datetime-local" v-model="expiry" />
    </div>
  </template>
</ConfirmDialog>
```

通用邏輯在元件，特殊邏輯在呼叫端。加新 action 的額外輸入不用改 ConfirmDialog。

**但這不是「改一個元件」的工作。** 目前 `ConfirmDialog.vue` 內含：expiry 驗證、必須為未來時間的檢查、值的保存、focus 管理、時區處理、以及**用驗證結果 gate 住 confirm 按鈕**。如果把輸入移到呼叫端，slot 就需要一份 validity contract（呼叫端要能回報「我這段輸入還不合法」），否則單純加一個 `#extra` slot 會**失去 confirm gating**——使用者可以在 expiry 無效的情況下按下確認。checker 與測試也要一起搬。

**建議把這項延後**：目前只有 approve 一個 action 需要額外輸入。等**第二個**這種 action 真的出現時再泛化，那時才知道 validity contract 該長什麼樣。現在做等於為單一案例設計抽象。

#### 6c. 抽 FieldRenderer — 一個小元件，不是 registry

把 ResourceListPage template 裡的 v-if 鏈（~70 行）抽成一個 `FieldRenderer.vue`：

```
admin/src/components/resource/FieldRenderer.vue
```

props: `field: FieldDef`, `modelValue`, `readOnly`, `disabled`
內容就是那段 v-if 鏈搬過去，加 `emit('update:modelValue')`。

**不是 widget map registry** — 就是一個元件包著 v-if 鏈。加新 widget 時在這個元件加一個 `v-else-if` 分支，比在 1094 行的頁面裡改清楚得多。`FieldWidget` union 加一個值，TypeScript 會追蹤。

ResourceListPage 的表單區從 70 行 v-if 鏈變成一行：
```vue
<FieldRenderer
  :field="fd"
  :modelValue="formData[fd.k]"
  :readOnly="isFieldReadOnly(fd)"
  :disabled="mutating"
  @update:modelValue="formData[fd.k] = $event"
/>
```

（初版此處寫 `submitting`——**該變數不存在**，`ResourceListPage.vue:30` 的實際名稱是 `mutating`。）

**搬移時必須保留的契約**，這些不是可選的樣式細節：
- `label` / `id` 的關聯與 `help` 文字
- `readOnly` 與 `roOnEdit`（僅編輯時唯讀）兩種不同的唯讀語意
- MediaUploader 的 `aria-labelledby` 綁定

這些如果在搬移時掉了，是 a11y 迴歸，而現有測試不一定會抓到。

---

## 7. ResourceListPage 拆解計畫

### 現狀：1094 行，6 個職責

```
ResourceListPage.vue
├── 資料載入 + rowMap (30 行)
├── 表單 modal (120 行) ← seedFieldValue, buildFormPayload, openForm, saveForm
├── Confirm dialog + action 執行 (160 行) ← resolveActionEndpoint, buildStatusBody, buildPostBody, runConfirm
├── Restock modal (200 行) ← 完全自成一區
├── Bulk actions + receipt (50 行)
└── State machine 顯示 (30 行)
```

### 建議：拆成 2 個 composable + 1 個元件（不改 UI）

小專案不需要 4 個 composable。restock 是唯一夠大夠獨立的區塊，值得自成一個 composable；其餘的 action/form 邏輯留在頁面裡但抽成函數就夠了。

```
admin/src/
├── pages/
│   └── ResourceListPage.vue       ← ~600 行（表單 + action + 編排，可接受）
├── composables/
│   └── useRestockModal.ts         ← restock 的 200 行狀態 + 邏輯
├── components/resource/
│   └── FieldRenderer.vue          ← 表單欄位 v-if 鏈（~70 行 template）
```

**為什麼只拆 restock**：
- restock 有自己的 modal、自己的 state（10+ 個 ref）、自己的 idempotency 邏輯，是頁面裡最大且最自成一區的區塊
- 表單和 action 雖然長，但跟頁面的 rows/selected/resource 緊密互動，拆出去反而要傳一堆參數
- FieldRenderer 是純渲染元件，抽出去頁面 template 少 70 行

**但成本不要低估**（初版寫「零耦合」是錯的）：

- restock **不是零耦合**。它讀 `resource`、`rows`、`api`、`mutating`、`loadRows`。抽成 composable 需要把這些當參數或注入傳進去，介面不會小。
- **`check-resource-contracts.mjs:347+` 直接字串檢查 restock 的實作細節**，例如斷言 `restockIdempotencyKey.value = crypto.randomUUID()` 必須出現在 `ResourceListPage` 裡（確保 idempotency key 只在開 modal 時產生一次，而非每次送出）。**把 restock 搬進 composable 會讓這個檢查失敗**，必須在同一次變更裡改 checker 與相關測試。初版的「1 檔拆出 200 行」漏算了這部分。
- **1094 → 約 600 行是目標，不是已驗證的結果**。實際數字取決於上面那些注入介面有多大。

**拆完後頁面結構**：
```
ResourceListPage.vue (~600 行)
├── script: 載入 + selection + form + action（留在頁面）
├── script: useRestockModal()（從 composable 拿）
├── template: 頁面編排 + 表單 modal + confirm dialog
├── template: <FieldRenderer />（表單欄位渲染）
└── template: restock modal（用 composable 的 state）
```

### action 分支鏈 — 保持函數，不建 registry

現狀的 `resolveActionEndpoint` / `buildStatusBody` / `buildPostBody` 用 if 鏈比對 `a.op`。小專案裡這三個函數加起來 ~50 行，改成 registry 反而增加間接層。

**保持現狀**，但加一個註解標明加新 action type 要改哪幾個函數：

```ts
// 加新 action type 時，在以下三個函數加分支：
// 1. resolveActionEndpoint — 加 endpoint 解析
// 2. buildStatusBody 或 buildPostBody — 加 body 建構
// 3. resource config 的 ops 定義對應 operationId
```

---

## 8. Config 系統統一

### 現狀的雙定義

```
config/profile.ts     ← 導航（key, label, icon, section, caps）
config/resources/     ← 資源（label, api, cols, form, actions）
```

加資源要改兩邊，key 必須一致。

### 建議：RES 為單一來源，PROFILE 自動衍生

在 `ResourceDef` 加 3 個可選欄位：

```ts
export interface ResourceDef {
  // ...現有欄位
  navIcon?: string       // 'Package' | 'ShoppingBag' | ...
  navSection?: SectionKey  // 'primary' | 'secondary' | 'settings'
  readCap?: Capability   // 'twcommerce.read' | 'content.read' | ...
  listKey?: string       // 'products' | 'orders' | 'items' | ...
}
```

`profile.ts` 改成從 `RES` 衍生（見第 4 節程式碼）。

**可行性已確認**：`RES` 的 8 個 key 與 `PROFILE` 的 8 個資源項目 key 完全一致（`minimal-cart-products`、`tw-commerce.methods`、`staff` 等），順序也相同。`dashboard` 不在 `RES` 裡，維持手寫在陣列開頭即可。

**但「新增資源只需 1 步／只改 1 檔」不成立**，實際還是要新增 resource 檔 + 改 `index.ts` 兩處，衍生只省掉第三處。五個施工注意事項：

- **`navLabel` 是必要的，不能直接用 `label`。** PROFILE 與 resource 的 label 目前有**刻意差異**：導航寫「人員」而 resource 是「員工」、導航「優惠」而 resource「優惠活動」。直接衍生會改動 UI 文案。需要 `navLabel?: string` 並 fallback 到 `label`。
- **需要 `navVisible`。** 不是所有 resource 都該出現在導航。沒有這個欄位，未來任何新增的內部或隱藏資源會被自動曝光——這是一個會安靜發生的權限/UX 問題。
- **icon 映射有兩份。** `Sidebar.vue` 與 `MobileNav.vue` 各自維護一份字串 → lucide 元件的對照。加新 icon 要改兩處，衍生 PROFILE 不會解決這件事。應先合併成單一 icon registry。
- **順序是隱性依賴。** 衍生依賴 `Object.entries(RES)` 的插入順序決定側邊欄排列。目前剛好對，但建議加 `navOrder?: number` 明確排序。
- **`MOBILE_KEYS` 是第三份手寫清單**（`profile.ts` 底部），衍生時不會自動跟上。改成 `navMobile?: boolean`，或明確保留手寫並註明原因。

另外一個型別細節：`RES: Record<string, ResourceDef>` 會**擦掉 literal key 型別**，衍生出的 PROFILE key 會退化成 `string`，路由與 caps 都失去編譯期檢查。改成 `as const satisfies Record<string, ResourceDef>` 才能保住 key 的字面量型別。

### 刪除 ResourceDef.rows

`rows: Record<string, any>[]` 是 mockup 時代遺留，runtime 不用。改成 `rows?:` 或刪除。

---

## 8b. 功能缺陷（優先於結構重構）

以下三項不是「架構不夠好」，是**功能沒做完或做錯**。它們應該排在元件拆分之前。

### 8b-1. 篩選器是無功能的裝飾 UI

`ResourceListPage.vue:736-757` 的 toolbar 會依 `resource.filters` 渲染 Select 與 Input，但：

```vue
<Select
  :options="f.opts ?? []"
  modelValue=""          <!-- ← 永遠是空字串，寫死 -->
  style="min-width:120px"
/>
<Input
  v-else
  :placeholder="f.l"     <!-- ← 連 v-model 都沒有 -->
  style="min-width:120px"
/>
```

**沒有 `@update`、沒有 v-model、沒有 query state、沒有任何東西會把選擇送給後端。** 使用者可以操作它們，但什麼都不會發生。

`products.ts:49-52` 定義了「狀態」「分類」兩個篩選器，`orders.ts` 也有——這些 config 目前全部沒有作用。

### 8b-2. 分頁完全不存在

`pageSize: 20` 唯一的用途是 `ResourceListPage.vue:757` 顯示「每頁 20 筆」這行文字。沒有 page state、沒有 slice、沒有 server query。**前端顯示一個數字，後端回傳全部資料。**

這跟後端的 `LIMIT` 缺失是同一件事的兩半，見 `docs/backend-optimization.md` P3-2。兩邊必須一起做——只做後端分頁而前端沒有翻頁 UI，使用者會看不到第一頁以外的資料。

### 8b-3. selection 用 row index 當 identity

`ResourceListPage.vue:65`：

```ts
const selected = ref<Set<number>>(new Set())
```

選取狀態記的是**陣列索引**，不是資料的穩定 ID。目前沒出事，是因為列表既不能排序也不能篩選也不能翻頁——`rows` 在使用者操作期間不會重排。

**但 8b-1 和 8b-2 一旦實作，這就是一個資料正確性 bug**：篩選後索引 3 指向的是不同的商品，而 bulk action 會對錯誤的資料執行。批次操作包含「批次上架」「批次轉草稿」這類會改變上架狀態的動作。

**修法必須跟 8b-1 / 8b-2 同一批做**：改用穩定的 resource ID，並明確定義換頁／換篩選時 selection 要清除還是保留。

### 8b-4. 缺少語意層級的契約閘

`check-resource-contracts.mjs`（475 行）是 **source-string assertions**——它讀 `.vue` 和 `.ts` 的原始碼文字做 `includes()` 比對。這帶來兩個問題：

- **抓不到真正的契約漂移**。它現在是 PASS 的，卻沒發現第 2 節那 11 條缺失路由，也沒發現既有 42 條的 status/schema 漂移。
- **阻礙安全重構**。因為它斷言的是實作細節（變數名、特定程式碼片段的存在），任何合法的重構都會讓它紅燈——見第 7 節 restock 的例子。

**建議**：加一個真正解析 OpenAPI 的閘，最低限度驗證：
1. 列舉 Go router 的 method + path（`bootstrap/app.go` 可以靜態解析），與 OpenAPI 的 path 集合雙向比對
2. operationId 唯一，且前端 config 引用的 operationId 都存在
3. response status code 集合一致

schema 與 runtime 的一致性再用 handler contract test 補（打真實 handler、比對回應形狀）。這個閘建立後，`check-resource-contracts.mjs` 裡那些字串斷言大部分可以退場。

---

## 9. AI Agent 易改性檢查表

以「AI agent 要做 X，需要改幾個檔、會不會踩坑」為度量：

| 任務 | 現狀 | 改善後 |
|---|---|---|
| 加新資源 | 改 3 檔（resource + index + profile），key 手動同步 | 改 1 檔（resource + index），profile 自動衍生 |
| 加新表單欄位類型 | 改 ResourceListPage template 的 v-if 鏈（在 1094 行裡） | 改 FieldRenderer.vue 加一個 v-else-if 分支 |
| 加新 action type | 改 resolveActionEndpoint + buildStatusBody + buildPostBody（在 1094 行裡找） | 同左，但頁面只有 ~600 行，函數有註解標明 |
| 改 API 回應欄位 | 改 Go DTO + 改 OpenAPI（可能忘）+ 改前端 rowMap（用 `any` 不會報錯） | 改 Go DTO + 改 swaggo + 跑 gen:types → 前端型別自動更新，編譯期報錯 |
| 加新 capability | 改 Go resolver + 改 resource config 的 cap；另有一份死碼 roles.ts 會誤導 agent 去改它 | 改 Go resolver + 改 resource config 的 cap（roles.ts 已刪） |
| 改 restock 邏輯 | 在 1094 行裡找 restock 相關的 200 行 | 只讀 useRestockModal.ts |
| 改表單 payload | 在 1094 行裡找 buildFormPayload | 在 ~600 行頁面裡找（有 FieldRenderer 分擔 template） |
| 加新主題 | 加 site/themes/ + 加 resource config + 加 profile | 加 site/themes/ + 加 resource config（profile 自動衍生） |

---

## 10. 優先級與建議路線圖

> **修訂說明**：初版把結構重構排在 P0、契約排在 P1。經對照原始碼審查後重排——**功能缺陷與契約失真優先於元件拆分**。理由是拆分本身不會讓後台更能用，而篩選器、分頁、契約漂移是現在就在影響正確性與上線的。

### P0 — 恢復「可驗收、可上線」

| # | 項目 | 改動範圍 | 收益 |
|---|---|---|---|
| 1 | 決定正式 API base / 跨源拓撲，實作精確 CORS | api-client + config + Go CORS | **admin 目前上不了正式環境**（見文件開頭交集說明與 `backend-optimization.md` P1-1） |
| 2 | Audit 既有 42 條 OpenAPI 的 status/schema 漂移 + 補 11 條 | openapi.yaml | 型別生成的前置；現況會把錯契約合法化 |
| 3 | 實作篩選器 + 真實分頁（與後端 LIMIT 一起） | ResourceListPage + Go store | 修復無功能 UI（8b-1、8b-2） |
| 4 | selection 改用穩定 ID | ResourceListPage | 第 3 項一旦做完，index identity 就是資料正確性 bug（8b-3） |
| 5 | 加語意契約閘 | +1 script | 防止再漂移（8b-4） |

### P1 — 契約斷層，做了以後所有改動都更安全

**前提是 P0-2 已完成**。在契約還有 status/schema 漂移時生成型別，等於把錯誤固化成編譯期「事實」。

| # | 項目 | 改動範圍 | 收益 |
|---|---|---|---|
| 6 | 選定 OAS3-native 的 source-of-truth 流程 | 決策 + 工具評估 | **不要採用 swaggo**（產 Swagger 2.0，與現有 OAS 3.1 不相容） |
| 7 | OpenAPI → TypeScript 型別生成 + 手寫 alias 層 | +1 dev dep + 1 script + 漸進改 any | 欄位名拼錯編譯期報錯 |
| 8 | PROFILE/RES 整合 | types + profile + icon registry | 同時處理 navLabel / navVisible / navOrder / mobile policy |
| 9 | ResourceDef 加 listKey | types + 8 configs + DashboardPage | 不再猜 array 在哪 |

**第 9 項要單獨看待**：`listKey` 只是 runtime 的取值修正，**它本身不提供任何編譯期型別追蹤**。`ResourceListPage` 仍然是動態 `resource` + 無型別的 row 資料。要真正的型別安全，`ResourceDef` 需要 generic row/response 參數，或每個 resource 提供 typed decoder。不要把 listKey 和型別生成算成同一件事的兩半。

### P2 — 清理債，不急但做了少踩坑

| # | 項目 | 改動範圍 | 收益 |
|---|---|---|---|
| 10 | 抽 useRestockModal composable | 1 composable + **同批改 checker 與 tests** | restock 邏輯獨立 |
| 11 | 抽 FieldRenderer 元件 | +1 元件，頁面 template 少 70 行 | 加 widget 改一個小元件 |
| 12 | 刪 `config/roles.ts` | 1 檔 + **協調 checker `:13,191-209`** | 移除會誤導 agent 的假權限表 |
| 13 | 刪 EmptyState + Panel | 2 檔 | 少死碼（這兩個才是真正零風險） |
| 14 | 刪 ResourceDef.rows | types + 8 configs | 少一個必填欄位 |
| 15 | admin/AGENTS.md | +1 檔 | agent 進來知道怎麼改 |
| 16 | ConfirmDialog slot 化 | 1 元件 + validity contract + tests | **等第二個需要額外輸入的 action 出現再做** |

第 12 項初版寫成「零風險單檔刪除」，**是錯的**——`check-resource-contracts.mjs` 會讀它並做斷言，直接刪會讓現在 PASS 的檢查失敗。仍然值得做（假權限表對 AI agent 的誤導性很高），但要當成協調變更處理。

### 不建議做（過度工程）

| 項目 | 為什麼不 |
|---|---|
| 拆 4 個 composable | 小專案不需要；form/action 跟頁面緊密耦合，拆出去要傳一堆參數 |
| Action handler registry | 3 個函數加起來 50 行，registry 增加間接層不划算 |
| apienv 信封 | 大改動，收益不明顯，AGENTS.md 說不為假設加 seam |
| Zod runtime 驗證 config | config 是編譯期常數，TypeScript 已足夠 |
| Widget map registry | 一個 v-if 鏈元件就夠了，不需要動態 component map |
| Test helper 套件 | 3 個測試檔各自建 Pinia/Router 只有 ~10 行重複，不值得抽 |
| 多租戶 / per-site admin | 現在是單站，加 multi-tenancy 是架構變更 |
| 複製 SiteCore 的 admin packages | React + pnpm workspace + apienv 信封，跟 Vue 單站不匹配 |

---

## 11. 與 SiteCore 的關係

> **本節未經驗證。** 下表描述的是本 repo 之外的另一個專案，我在查證時發現本機 `D:\Projects\SiteCore` 的 root `package.json` 宣告的是 **Vue 3.5.39，找不到 React**，與下表的「React 19」不符。可能是版本已變更、或指的是子目錄/另一個 checkout。**使用本節結論前請先確認對象。** 下表其餘欄位同樣未查證。

### 事實（待確認）

| | AI-go-starter | SiteCore |
|---|---|---|
| 前端框架 | Vue 3 | React 19（**與本機實際不符，待確認**） |
| 後端 | Go `net/http` | Go Gin |
| API 信封 | 無（直接 JSON） | `pkg/apienv`（`{data, meta, error}`） |
| OpenAPI | 手寫 YAML | swaggo annotation → `swagger.json` |
| 前端型別生成 | 無 | `generate-openapi-client.mjs` → 生成 TS |
| 共享 admin 套件 | 無 | `packages/admin-*` 8 個 npm 套件 |
| 多站 | 單站 + 主題切換 | 多站（`sites/<name>/`） |
| 資源定義 | `RES` + `PROFILE` 雙定義 | `ResourceDefinition` 單定義 + helper functions |

### 結論：獨立但學習

**不應該複製 SiteCore 的 admin packages**：
- 框架不同（Vue vs React）
- 信封不同（無 vs apienv）
- 規模不同（單站 vs 多站平台）

**應該學習的模式**：
- OpenAPI → 型別生成管線（SiteCore 用 `generate-openapi-client.mjs`，AI-go-starter 可用 `openapi-typescript`）
- 資源定義為單一來源（SiteCore 的 `ResourceDefinition` 已經是單定義）
- operationId 作為前後端契約錨點（SiteCore 的 `adminStaffMembersListOperation` 常數）
- audience-prefix 路由（`/api/admin/`、`/api/me/`、`/api/public/`）— AI-go-starter 已經有 `/api/admin/`

### 未來如果需要多站

如果 AI-go-starter 未來要支援多站，正確的演進路徑是：

1. Go 後端加 site context（request-scoped site ID）
2. 路由加 site 命名空間（`/api/sites/{siteId}/admin/...`）
3. 資源 config 改成 per-site registry
4. admin 前端加 site selector

**但這是架構變更，不應該現在預先做**。AGENTS.md 明確說「不為假設的未來 provider 加 seam」。

---

## 附錄：建議的目標目錄結構

```
admin/src/
├── config/
│   ├── resources/
│   │   ├── index.ts          ← RES（單一來源）
│   │   ├── products.ts       ← 含 navIcon, navSection, readCap, listKey
│   │   ├── orders.ts
│   │   └── ...
│   ├── machines.ts
│   └── tones.ts
│   (profile.ts 改成從 RES 衍生的薄檔)
├── composables/
│   └── useRestockModal.ts    ← restock 的 200 行（唯一拆出的 composable）
├── components/
│   ├── ui/                   ← 通用元件（刪 EmptyState, Panel）
│   ├── resource/
│   │   ├── ResourceTable.vue
│   │   └── FieldRenderer.vue ← 表單欄位 v-if 鏈（新增）
│   ├── layout/
│   └── auth/
├── lib/
│   ├── api-client.ts
│   ├── api-types.ts          ← 從 OpenAPI 生成（新增）
│   ├── types.ts              ← config 型別（加 listKey, navIcon, navSection, readCap）
│   ├── utils.ts
│   └── auth/
├── pages/
│   ├── ResourceListPage.vue  ← ~600 行（form + action + 編排）
│   ├── DashboardPage.vue
│   └── StatesPage.vue
└── stores/
    ├── auth.ts
    ├── layout.ts
    └── theme.ts
```

**跟現狀的差異**：只加了 3 個檔（`useRestockModal.ts`、`FieldRenderer.vue`、`api-types.ts`），刪了 2 個檔（`EmptyState.vue`、`Panel.vue`），改了 `profile.ts`（從手寫改成衍生）。其餘結構不動。
