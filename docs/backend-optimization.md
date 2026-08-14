# 後端優化評估

針對 `server/` 的一次通盤審視。每一項都標註了實際位置、觸發條件、以及建議的修法。
排序依「資料正確性 > 上線阻斷 > 可觀測性 > 效能」，不是依實作難度。

審視範圍：`server/`（29,009 行 Go / 80 檔，其中 `server/internal/` 26,138 行 / 63 檔）、`db/migrations/`、`contracts/openapi.yaml`。
不含 `admin/` 與 `site/themes/`（見 `docs/admin-architecture-review.md`）。

> **修訂紀錄**：初版依「效能優先」排序。經 Codex 對照原始碼獨立審查後修訂——新增一項既有的併發正確性缺陷（P0-0）、四項錯誤語意問題，並重排優先順序。初版對 P0-1 快取風險、P1-2 影響面、P1-3 郵件觸發條件、P1-4 阻斷點的描述均已收斂。

---

## 優先順序總表

| # | 項目 | 類別 | 影響面 |
|---|---|---|---|
| **P0-0** | **併發取消訂單有實際失敗的測試** | **正確性** | **資料一致性** |
| P0-1 | contact 先寫入再寄信，失敗回 400 且外洩內部訊息 | 正確性 | 重複資料、資訊外洩 |
| P0-2 | staff lookup 吞掉所有 error，DB 故障偽裝成無權限 | 正確性 | 診斷失能 |
| P0-3 | CI 沒有真 PostgreSQL 驗收 | 正確性 | 上線風險 |
| P1-1 | 正式環境路由拓撲未定（API base + CORS） | 上線阻斷 | admin 上不了線 |
| P1-2 | Postgres 連線池未設定 | 上線阻斷 | 連線耗盡 |
| P2-1 | 可觀測性：無 status/request-id/client-IP，5xx 不留 log | 可觀測性 | 出事查不到 |
| P2-2 | PII 遮罩用 byte 切片，中文只保留 2 字且可能切出無效 UTF-8 | 正確性 | 遮罩失準 |
| P3-1 | 認證每受保護請求 1 次外部 HTTP + 1 次 DB 查詢 | 效能 | 延遲 |
| P3-2 | 列表查詢無 LIMIT，且 admin 端實際上沒有分頁 | 效能 | 記憶體、延遲 |
| P3-3 | 商品圖片 N+1 | 效能 | 延遲 |
| P3-4 | `ORDER BY updated_unix` 無索引 | 效能 | 延遲 |
| P4-1 | DB 錯誤判斷靠字串比對 | 韌性 | 換 driver 時失效 |
| P4-2 | 公開端點無應用層限流 | 韌性 | 濫用 |
| P4-3 | OpenAPI 缺 11 條已實作路由 | 契約 | 已失真 |
| P4-4 | commerce 單一模組 2,358 行 | 維護性 | 持續累積 |

---

# 第一順位：資料正確性

## P0-0 併發取消訂單有實際失敗的測試

**位置**：`server/internal/modules/commerce/service_test.go:643`

這是既有缺陷，不是新引入的。實測：

```console
$ go test ./server/internal/modules/commerce -run '^TestConcurrentOrderCancellationRestocksOnce$' -count=20
--- FAIL: TestConcurrentOrderCancellationRestocksOnce (0.02s)
    service_test.go:643: cancel error = invalid order status transition
--- FAIL: TestConcurrentOrderCancellationRestocksOnce (0.02s)
    service_test.go:643: cancel error = invalid order status transition
FAIL    github.com/example/ai-site-starter/server/internal/modules/commerce    0.385s
```

20 次跑出 2 次失敗，**非決定性**。兩個請求同時取消同一張訂單時，落敗的那個在 service 層先撞上 `ErrInvalidTransition`，而不是回傳一致的 stale/conflict 結果。

這代表併發取消的結果**取決於兩個 goroutine 的交錯時序**：有時回「版本衝突」（可重試），有時回「狀態轉換非法」（語意上像是使用者操作錯誤）。呼叫端無法據此決定要不要重試。

**為什麼排第一**：這是資料一致性問題，且**已經有測試在證明它壞掉**。在它綠燈之前做任何效能優化，都是在會動的地基上蓋東西——快取、分頁、索引都可能讓時序改變，使這個 flake 的表現方式改變而更難查。

**建議修法**：先確認期望語意。取消競賽的落敗方應該回哪一種錯誤？
- 若視為「已經是目標狀態」→ 應該冪等成功
- 若視為「版本過期」→ 應該一致回 conflict，讓呼叫端重試

決定之後，把狀態轉換檢查移到與版本檢查同一個交易邊界內，讓兩者原子地一起判定，而不是先在 service 層檢查記憶體中的 stale 狀態、再到 store 層做樂觀鎖。

**同時要做**：整包 `go test ./...` 必須在 `-count=5` 以上穩定綠燈才算數。單次通過對併發測試沒有意義。

---

## P0-1 contact：先寫入再寄信，失敗回 400 且外洩內部訊息

**位置**：`server/internal/modules/contact/contact.go:49`（Submit）、`:89`（Handler）

```go
if err := s.store.Insert(ctx, in); err != nil {
    return fmt.Errorf("save inquiry: %w", err)
}
if s.notifyTo == "" {
    return nil
}
return s.mailer.Send(ctx, ...)   // ← 寄信失敗，inquiry 已經寫進去了
```

handler 端把**所有**錯誤壓成同一個出口：

```go
if err := h.service.Submit(r.Context(), input); err != nil {
    httpx.Error(w, http.StatusBadRequest, err.Error())   // ← 全部 400，且直接吐 err.Error()
}
```

三個問題疊在一起：

1. **重試會重複新增**。寄信失敗 → 使用者看到錯誤 → 再送一次 → 資料庫多一筆 inquiry。沒有冪等鍵。
2. **錯誤分類錯誤**。DB 故障和 Resend 故障都是 5xx，卻回 400，等於告訴前端「你的輸入有問題」。
3. **內部訊息外洩**。`err.Error()` 直接進 response body，瀏覽器會看到 `save inquiry: ...` 或 `resend send: ...` 這類含內部結構的字串。

**建議修法**：
- 在 service 層區分 typed error：validation（400）、internal（500）。handler 依型別映射，internal 只回固定訊息，細節寫 server log。
- 寄信改成非阻斷：inquiry 寫入成功就回 200，寄信失敗只記 log。若通知不可漏，加一張 outbox 表 + 重試，而不是讓使用者面對寄信失敗。
- 或者明確定義一個「已保存但未通知」狀態並回 202。

---

## P0-2 staff lookup 吞掉所有 error

**位置**：`server/internal/auth/resolver.go:48`

```go
row, err := r.lookup.GetBySupabaseUserID(ctx, p.UserID)
if err != nil {
    // No linked staff row: valid Supabase user, no admin capabilities.
    p.Role = "user"
    p.Capabilities = nil
    return p, nil        // ← 所有 error 都走這條，包含 DB 斷線
}
```

註解說的是「沒有連結的 staff row」，但這個分支同時吞掉了**資料庫故障**。

安全上仍然是 fail-closed（給 0 權限），這點沒問題。問題在**診斷**：Supabase PostgreSQL 短暫不可用時，全站管理員會突然一起變成 `role: "user"`，收到 403。沒有任何 log、沒有 5xx、監控看到的是「一切正常，只是大家都沒權限」。這種故障極難定位。

**建議修法**：只吞 not-found，其餘往上傳：

```go
row, err := r.lookup.GetBySupabaseUserID(ctx, p.UserID)
if errors.Is(err, ErrStaffNotFound) {
    p.Role = "user"
    p.Capabilities = nil
    return p, nil
}
if err != nil {
    return Principal{}, fmt.Errorf("staff lookup: %w", err)   // → 503，並留 log
}
```

`bootstrap/app.go:252` 的 `mediaVerifierAdapter` 已經正確做了這個區分（DB 故障回 5xx，不塌成 400）。resolver 應該跟隨同一個模式——這是 repo 內既有的正確範例。

---

## P0-3 CI 沒有真 PostgreSQL 驗收

**位置**：`.github/workflows/ci.yml`

CI 只跑 `go test ./...`，全部走 SQLite。但正式預設是 Supabase PostgreSQL（`architecture.yaml:24`）。

以下全部是 **driver 敏感**、在 SQLite 上測不出來的：

- unique / foreign-key 錯誤的識別（見 P4-1，目前靠字串比對，兩個 driver 訊息完全不同）
- migration 014（order_items）、015（media_gc）的實際套用
- 交易隔離層級與鎖行為——SQLite 是 `SetMaxOpenConns(1)` 全序列化，**根本不會產生真正的併發**
- `ON CONFLICT` 語意差異

第三點特別重要：**P0-0 那個併發測試在 SQLite 下是被序列化的**，真正的 Postgres 併發行為從來沒被驗證過。

**建議修法**：CI 加一個 `services: postgres:16` job，跑 clean migrate + 同一套測試 + 併發測試 `-count=10`。這是 P0-0 能否算真正修好的前提。

---

# 第二順位：上線阻斷

## P1-1 正式環境路由拓撲未定

**位置**：`admin/src/lib/api-client.ts:3`、`server/internal/bootstrap/app.go:209`、`.env.example:5`

初版把這寫成純 CORS 問題，**不完整**。真正的阻斷點更前面：

```ts
const API_BASE = '/api'      // api-client.ts:3 — 硬編碼相對路徑
```

開發時 `admin/vite.config.ts` 用 proxy 把 `/api` 代到 `:8080`，是同源。但正式環境如果 admin 部署在 Cloudflare Pages、API 在 Railway，這個相對路徑會打到 **Pages 自己**，得到 404——**根本走不到 Railway，CORS 連被觸發的機會都沒有**。

CORS 是第二層問題：`withCORS`（`app.go:209`）只比對單一 `SITE_ORIGIN`，而 `.env.example:5` 指向前台 `:4173`。

**這兩件事必須一起決定，順序是先路由再 CORS**：

1. **先決定拓撲**。admin 正式環境要 (a) 與 API 同源部署，還是 (b) 獨立網域打跨源 API？
2. **(a) 的話**：需要一個反向代理或路徑分流把 `/api` 導到 Go，`API_BASE` 維持相對路徑，CORS 不用改。
3. **(b) 的話**：`API_BASE` 要能由建置期環境變數注入（目前不行），且 `SITE_ORIGIN` 要擴充成 `ALLOWED_ORIGINS` 清單。維持嚴格 allowlist，不要因為有 `Authorization` header 就退化成 `*`。

初版建議的「admin 與 API 都掛 Railway」目前**沒有任何 repo 或 architecture.yaml 的支持**——那是一個提議，不是既有設計。決定之後應該寫進 `architecture.yaml` 與 `docs/`。

---

## P1-2 Postgres 連線池未設定

**位置**：`server/internal/platform/database/database.go:22`

```go
db, err := sql.Open(sqlDriver, dsn)
if dialect == SQLite {
    db.SetMaxOpenConns(1)      // ← 只有 SQLite 有設定
}
```

Postgres 路徑完全沒設 `SetMaxOpenConns` / `SetMaxIdleConns` / `SetConnMaxLifetime` / `SetConnMaxIdleTime`。`database/sql` 的 `MaxOpenConns` 預設**無上限**，`MaxIdleConns` 預設只有 2。

在 Supabase 上，無上限的連線在流量尖峰會打爆連線配額；`MaxIdleConns=2` 又讓連線不斷被建立與丟棄。

**建議修法**：把四個參數都做成可設定，走 `config.Config` 讀環境變數。

**但具體數值不要寫死成通則。** `MaxIdle = MaxOpen`、以及 pgx 是否需要 `default_query_exec_mode=simple_protocol`，都取決於實際部署拓撲——你是直連 Postgres、走 Supabase 的 session pooler、還是 transaction pooler？repo 裡沒有足以定案的證據。

正確做法是先確認 pooler 模式，再定值。若確定走 transaction pooling，才需要處理 prepared statement 的問題。這件事跟 P1-1 是同一個「正式拓撲未定」的根，建議一起決定。

---

# 第三順位：錯誤語意與可觀測性

## P2-1 可觀測性

**位置**：`server/internal/bootstrap/app.go:227`

```go
log.Printf("http %s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
```

缺少：**狀態碼**（沒包 `ResponseWriter`，看不出 200 還是 500）、**request ID**、**來源 IP**、**結構化輸出**。

更嚴重的是**業務模組完全沒有任何 log**——全 `server/internal/modules/` grep 不到一行 `log.` 或 `slog.`。`media/http.go:119` 的註解寫「Internal error details are logged server-side」，**但那行 log 不存在**，內部錯誤細節是直接丟掉的。

**為什麼這排在效能優化之前**：沒有觀測就無法驗證快取有沒有生效、限流有沒有誤擋、P0-0 的併發問題在正式環境是否重現。先做效能優化等於在黑箱裡調參數。

**建議修法**：換成 `log/slog`（標準庫，無新依賴）。middleware 包一層記錄 status code、request id（優先取 `X-Request-Id`）、client IP、duration。handler 的 5xx 路徑補上 `slog.Error` 帶原始 error。這也是 P0-1、P0-2 修法的前提——把細節寫 log 才能不外洩給瀏覽器。

---

## P2-2 PII 遮罩用 byte 切片

**位置**：`server/internal/modules/commerce/service.go:920`

```go
func maskShippingAddress(addr string) string {
    addr = strings.TrimSpace(addr)
    if len(addr) <= 6 {
        return strings.Repeat("*", len(addr))
    }
    return addr[:6] + strings.Repeat("*", len(addr)-6)
}
```

`len()` 和切片都是 **byte 語意**，不是字元語意。對中文地址：

- UTF-8 中文一字 3 bytes → `addr[:6]` 只保留 **2 個字**，不是註解意圖的 6 個字
- 混合 ASCII 與中文時（例如門牌號在前），第 6 個 byte 可能落在 rune 中間，**切出無效 UTF-8**
- `strings.Repeat("*", len(addr)-6)` 的星號數量洩漏 byte 長度而非字元長度

現有測試只斷言「有遮罩且非空」，抓不到這個。

**建議修法**：改用 `[]rune(addr)` 做切片，並補上中文、混合、以及邊界（正好 6 bytes）的測試案例。`maskPhone`（`service.go:928`）同樣是 byte 切片，但電話號碼是 ASCII 所以目前安全——仍建議一併改成 rune 語意，避免之後有人塞進非 ASCII 字元。

---

# 第四順位：效能

> 這一段的每一項在程式碼事實上都成立，但都應該排在正確性與可觀測性之後。

## P3-1 認證每受保護請求 1 次外部 HTTP + 1 次 DB 查詢

**位置**：`server/internal/auth/supabase.go:29`、`server/internal/auth/resolver.go:38`

**適用範圍要講清楚**：只有呼叫 `Authenticator.Principal` 的**受保護 handler** 會走這條路徑。公開端點（`GET /api/products`、`POST /api/contact` 等）即使帶了 `Authorization` header 也不會觸發驗證。

在受保護路徑上，supabase 模式下每次請求會：

1. 打一次 `GET {SUPABASE_URL}/auth/v1/user`（外部網路，timeout 5s）
2. 查一次 `staff_members` 拿 role

兩者都沒快取。後台列表頁進場打 5 支 admin API，就是 5 次跨網際網路往返 + 5 次 DB 查詢。

**修法要比初版謹慎。** 初版說「依 token hash 快取整個 Principal，改動小、風險低、登出即時生效」——這三點都太樂觀：

- **目前沒有 logout/invalidation 介面**。要做到「登出即時生效」得先建這個介面，不是附帶效果。
- **以 token hash 為 key，就無法依 user 失效**。staff 被停權時要立刻踢掉，需要一個 `userID → tokens` 的次級索引，否則只能等 TTL 過期。
- **TTL 直接等於停權與權限調整的 stale window**，這是必須明講並由產品面接受的取捨，不是實作細節。

**建議順序**：
1. **先做本地 JWKS 驗簽**（`{SUPABASE_URL}/auth/v1/.well-known/jwks.json`，JWKS 本身快取數小時）。這讓第 1 項變成純 CPU 運算、完全不出網路，且**不引入任何 staleness**——JWT 過期時間由簽發端控制。`Verifier` 介面不用動，這正是 `bootstrap/app.go:47` 的 switch 留的縫。
2. **再處理 staff capability 快取**，並且明確定義失效契約：TTL 上限、停權時的主動失效路徑、次級索引。這一步不做也可以先上線——它只是一次 DB 查詢，遠比外部 HTTP 便宜。

---

## P3-2 列表查詢無 LIMIT，且 admin 端實際上沒有分頁

**位置**：`commerce/store.go:131`（ListProducts）、`:158`（ListPublishedProducts）、`:453`（ListMembers）、`:535`（ListOrders）

四支都沒有 `LIMIT` / `OFFSET`。另外 `ListMyOrders`（`service.go:893`）重用了無上限的 `ListOrders`，所以會員訂單列表同樣不設限。

`orders` 是寬表（25 個欄位，含 `items_json`、`timeline_json` 兩個 JSON blob），一萬筆就是一萬筆全部進記憶體再序列化。

**admin 端的狀況比初版描述的更糟。** 初版說 `pageSize: 20` 是「純前端切頁」——**不是**。`ResourceListPage.vue` 沒有任何 slice 或 page state，全部 rows 直接交給 `ResourceTable`。`pageSize` 唯一的用途是 `ResourceListPage.vue:757` 顯示「每頁 20 筆」這行文字。也就是說**前後端都沒有分頁**，只有一個誤導人的標籤。

**建議修法**：在三個 Filter 加 `Limit` 與 `Cursor`，走 **keyset 分頁**：

```sql
WHERE (updated_unix, id) < (?, ?)
ORDER BY updated_unix DESC, id DESC
LIMIT ?
```

OFFSET 在深頁時一樣要掃過前面所有列，對這個 workload 沒好處。

這會同時動到 `contracts/openapi.yaml` 與 admin 前端（需要真的實作分頁 UI，不只是改標籤），屬於跨層變更，建議獨立成一個 controlled change。**先補完 P4-3 那 11 條缺的路由再加分頁參數**，不要在失真的契約上繼續擴充。

---

## P3-3 商品圖片 N+1

**位置**：`server/internal/modules/commerce/service.go:740`

```go
func (s Service) enrichProductsListWithImages(ctx, products []Product, deriveURLs bool) {
    for i := range products {
        enhanced, err := s.enrichProductWithImages(ctx, products[i], deriveURLs)  // ← 每件一次 SELECT
    }
}
```

`enrichProductWithImages`（`service.go:712`）內部呼叫 `ListProductImages(ctx, p.ID)`。N 件商品 = 1 + N 次查詢。公開的 `GET /api/products` 與後台的 `GET /api/admin/products` 都走這條。

SQLite 本地感覺不出來（`database.go:45` 鎖成單連線，本來就序列化），到 Postgres 就是 N 次網路往返。

**建議修法**：加一支 `ListProductImagesByProductIDs(ctx, ids []string)`，單次 `WHERE product_id IN (...)` 撈完再在記憶體 group by。索引 `idx_product_images_product(product_id, sort_order)` 已存在（`db/migrations/sqlite/013_media_verification.sql:38`），直接可用。

注意動態長度的 IN 子句要先組好對應數量的 `?` 再過 `database.Bind`（`database.go:53` 是逐字元替換）。

**做這項之前先做 P3-2**：分頁把基數限制住之後，N+1 的上限才可控（20 筆 vs 全表）。

---

## P3-4 `ORDER BY updated_unix` 沒有對應索引

**位置**：`db/migrations/{sqlite,postgres}/002_commerce.sql`

現有索引涵蓋 `products(status)`、`products(category)`、`products(slug)`、`orders(status)`、`orders(member_id)`、`orders(idempotency_key)`、`orders(access_token)`。三支列表查詢都以 `ORDER BY updated_unix DESC` 收尾，而 `updated_unix` 上沒有任何索引。

初版斷言「目前每次列表都是全表掃描 + 記憶體排序」——**這需要 EXPLAIN 才能確認**，planner 的實際選擇取決於資料分佈與統計資訊。正確的說法是：沒有可用於排序的索引，因此排序無法從索引取得順序。

**建議索引**（比初版完整——初版漏了 members、payment_status、category、以及無 filter 路徑）：

```sql
-- 無 filter 路徑
CREATE INDEX idx_orders_updated            ON orders   (updated_unix DESC, id DESC);
CREATE INDEX idx_products_updated          ON products (updated_unix DESC, id DESC);
CREATE INDEX idx_members_updated           ON members  (updated_unix DESC, id DESC);
-- 有 filter 路徑
CREATE INDEX idx_orders_status_updated     ON orders   (status,         updated_unix DESC, id DESC);
CREATE INDEX idx_orders_payment_updated    ON orders   (payment_status, updated_unix DESC, id DESC);
CREATE INDEX idx_orders_member_updated     ON orders   (member_id,      updated_unix DESC, id DESC);
CREATE INDEX idx_products_status_updated   ON products (status,         updated_unix DESC, id DESC);
CREATE INDEX idx_products_category_updated ON products (category,       updated_unix DESC, id DESC);
```

`id` 放在索引尾端，讓 keyset 分頁的 tiebreaker 也能走索引。SQLite 和 Postgres 兩份都要寫（`AGENTS.md:13`）。索引加完後應以 EXPLAIN 驗證 planner 真的採用。

---

# 第五順位：韌性與契約

## P4-1 DB 錯誤判斷靠字串比對

**位置**：`server/internal/platform/database/database.go:96`（IsUniqueViolation）

```go
msg := strings.ToLower(err.Error())
return strings.Contains(msg, "unique constraint") || ... || strings.Contains(msg, "23505")
```

**影響面要說準。** 初版說這兩支函式是「訂單冪等性的判定核心」，過重了：

- `IsUniqueViolation` 在 `CreateOrderTxWithIdempotency`（`store.go:666`）裡是 **concurrent unique-insert race 的 fallback 分支**。主路徑是先在交易內查 idempotency key；DB 的 unique constraint 才是最後保證。
- 因此辨識失敗的後果是**合法的 replay 變成 500**，而不是突破 unique index 造成重複建單。嚴重度低於初版描述。
- `database.go:112` 是 `IsForeignKeyViolation`，它**不參與** `CreateOrderTxWithIdempotency`。初版把兩支函式一起稱為冪等核心是錯的。
- `IsUniqueViolation` 另外還用於 restock 冪等與 shipping method 重複檢查。

仍然值得修，理由是韌性而非正確性：driver 或 DB 版本改了訊息措辭就會靜默失效，而測試用的是同一組字串所以抓不到。`strings.Contains(msg, "23505")` 也會誤中任何內容含這五個數字的錯誤訊息。

**建議修法**：改用型別化錯誤。pgx 有 `*pgconn.PgError`（`errors.As` + `pgErr.Code == "23505"`），`modernc.org/sqlite` 有 `*sqlite.Error` 帶 `Code()`。兩個 driver 都已在 `go.mod`，不需新依賴。字串比對只留作 fallback。

**前提是 P0-3**：沒有 Postgres CI，這個改動無法被驗證。

---

## P4-2 公開端點無應用層限流

**位置**：`server/internal/bootstrap/app.go:206` — middleware 只有 `withCORS` 與 `withAccessLog`

全專案 grep 不到 rate limiter。無需驗證即可呼叫的端點：`POST /api/contact`、`POST /api/orders`、`POST /api/quote`、`GET /api/orders/{id}`。

**兩點修正**：

1. **`/api/contact` 不是每次成功都寄信**。`contact.go:64` 有 `if s.notifyTo == "" { return nil }`，且 `bootstrap/app.go:66` 在 `RESEND_API_KEY` 為空時用 `LogSender`。所以「別人可以花你的錢」只在**同時設定了 `RESEND_API_KEY` 與 `CONTACT_NOTIFY_TO`** 時成立。
2. **不能從 repo 推論部署邊緣也沒有限流**。Cloudflare 與 Railway 都可能在前面擋掉一部分。這裡的結論只能是「應用程式內沒有」。

`GetOrderByAccessToken`（`store.go:577`）的設計本身是對的（token 存 SHA-256、id + token 必須同時命中、空 token 直接 ErrNotFound），token 由 crypto/rand 產生，暴力破解不現實。

**建議修法**：加 `withRateLimit` middleware。但有兩個約束不能忽略：

- **單機記憶體 token bucket 只適用單 replica**。Railway 一旦擴到多實例，每個實例各自計數，實際限額變成 N 倍。要跨實例就需要共享狀態。
- **取 client IP 不能直接寫「取 `X-Forwarded-For` 最右側」**。正確做法要先定義 trusted proxy chain——信任幾層、哪些來源。寫錯的話限流可以被偽造的 header 繞過。

先決定部署拓撲（與 P1-1 同一個根），再實作。

---

## P4-3 OpenAPI 缺 11 條已實作的後台路由

**位置**：`contracts/openapi.yaml`

程式比對：`bootstrap/app.go` 註冊 **53** 條路由，`openapi.yaml` 有 **42** 個 operationId，缺口正是以下 11 條，且沒有多餘的契約條目。

| 路由 | 註冊於 | admin 引用的 operationId |
|---|---|---|
| `PATCH /api/admin/products/{id}/status` | `app.go:158` | `adminMinimalCartProductsUpdateStatus` |
| `POST /api/admin/products/bulk` | `app.go:159` | `adminMinimalCartProductsBulkUpdate` |
| `GET/PUT/PATCH /api/admin/members*` | `app.go:169-171` | `adminMinimalCartMembers*`（3 個） |
| `GET/POST/PUT/DELETE /api/admin/promos*` | `app.go:174-177` | `adminMinimalCartPromos*`（4 個） |
| `GET/PUT /api/admin/payment-methods*` | `app.go:180-181` | `adminTwCommerceMethods*`（2 個） |

`admin/scripts/check-resource-contracts.mjs` 抓不到——它比對的是原始碼字串，不是 OpenAPI 文件。

**這是 P3-2 分頁的前置**：先把基線補完整，再加 pagination contract。詳見 `docs/admin-architecture-review.md` 第 2、3 節。

---

## P4-4 commerce 模組單檔過大

`commerce/service.go` 2,358 行、`store.go` 1,409 行、`service_test.go` 5,714 行，同時擁有 products、orders、members、promos、payment_methods、shipping_methods、returns、order_events 八組資料。

**不建議現在動。** 列在這裡是提醒：**下一個商務功能不要再往 commerce 塞**。

初版說「orders 是最乾淨的切點，它跟 products 只透過 SKU 和庫存增減耦合」——**這個判斷過度簡化了**：

- 建單、取消、退貨都在**同一個交易裡直接更新 products**（庫存增減），不是鬆耦合
- checkout 同時依賴 promo、shipping、payment、member 四組規則
- `mediaVerifierAdapter`（`app.go:239`）是**跨模組唯讀查詢**的範例，不是跨 aggregate 交易的等價示範。用它來論證 orders 可以切出去是不對的

真要拆，應該**依交易邊界重新設計**，而不是依檔案行數切。這是架構變更，需要獨立的 controlled change 與明確的一致性論證。

---

## 已經做對、不要動的部分

- **capability 檢查在 service 層**（commerce 14 處、sitecontent 6 處、staff 4 處），handler 只做錯誤翻譯。不要為了效能把檢查上移到 middleware。
- **staff 連結只認 `supabase_user_id`**（`resolver.go:48`），刻意排除 email 比對以防冒用，註解已寫明攻擊路徑。
- **停用員工 fail-closed**（`resolver.go:56`）——IdP session 仍有效也給 0 權限。（但同一個函式的 error 處理有 P0-2 的問題，修的時候不要動到這個分支的語意。）
- **`mediaVerifierAdapter` 對 DB 故障與驗證失敗的區分**（`app.go:252`）——DB 掛掉回 5xx 不塌成 400。這是全 repo 錯誤處理的正確範例，P0-1 與 P0-2 應該跟隨它。
- **media presign 有 capability 檢查**（`media/media.go:77` 檢查 `media.upload`）、MIME/副檔名雙向 allowlist、拒絕 SVG/WebP/AVIF。
- **`httpx.DecodeJSON`** 已有 1MB body 上限、`DisallowUnknownFields`、拒絕多個 JSON 物件。
- **`config.Validate`**（`config/config.go:92`）擋掉 `AUTH_MODE=dev` 在 production。
- **`cmd/api/main.go`** 的 HTTP timeout 四件套與 graceful shutdown 都設好了。
- **訂單樂觀鎖 + 冪等鍵 + request fingerprint**（`store.go:666`、`:834`）設計完整。（P0-0 的缺陷在 service 層的狀態檢查時機，不在這套機制本身。）

---

## 建議執行順序

**階段一 — 正確性（在這之前不要做任何效能優化）**

1. P0-3 CI 加 PostgreSQL service —— 後面所有驗證的前提
2. P0-0 併發取消 —— 先定語意，再修交易邊界，以 `-count=10` 驗證
3. P0-2 resolver error 分流 —— 小改動，解鎖故障診斷
4. P0-1 contact typed error + 寄信非阻斷

**階段二 — 上線拓撲（一個決策，兩項實作）**

5. 決定 admin 正式環境路由拓撲 → 寫進 `architecture.yaml` 與 `docs/`
6. P1-1 依決策實作 API base 與 exact CORS
7. P1-2 依 pooler 模式定連線池參數

**階段三 — 可觀測性**

8. P2-1 slog + status/request-id/client-IP + 5xx server logging
9. P2-2 PII 遮罩改 rune 語意 + 補測試

**階段四 — 效能**

10. P4-3 補齊 OpenAPI 11 條（基線）
11. P3-2 分頁 + P3-4 索引 + P3-1 分頁 contract —— 一起做
12. P3-3 N+1 批次查詢（分頁限制基數之後才做，上限才可控）
13. P3-1 認證：先 JWKS，再談 staff cache 與失效契約

**階段五 — 韌性**

14. P4-1 typed DB errors（需要 P0-3 才能驗證）
15. P4-2 限流（需要階段二的拓撲決策）
16. P4-4 commerce 拆分 —— 最後，且依交易邊界重新設計

第 5 項是整條路徑上唯一需要**人來決定**的：admin 正式環境部署在哪。它同時決定 CORS、限流的來源識別、以及連線池的規模假設。建議先做這個決定。
