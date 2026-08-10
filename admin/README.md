# Admin workspace

Vue 3 SPA 後台，從 SiteCore/web 抽出的類 shadcn 後台骨架。與 `../site/` 靜態站台並列，是獨立的 surface，獨立部署，不影響 Cloudflare Pages 靜態站台。

## 結構

- `apps/admin` — Vue SPA 後台（Vue Router + Pinia + TanStack Vue Query + TanStack Vue Form + Vue I18n + Tailwind CSS 4）
- `packages/admin-ui` — 類 shadcn 元件庫（Button/Dialog/Sheet/Table/Tabs/Panel/Pager/Chip…）+ theme.ts 主題套用
- `packages/admin-shell` — AdminShell.vue（側欄 + 頂列 + 麵包屑殼）
- `packages/admin-fields` — 表單欄位元件
- `packages/admin-theme` — 語意 CSS variables（tokens.css / tailwind.css / shell.css / components.css）
- `packages/admin-theme-mint` / `admin-theme-nippon` / `admin-theme-tailwind` — 三套主題變體
- `packages/admin-auth` — Pinia session store
- `packages/admin-transport` — 網路層骨架（**stub，待接 starter Go API**）
- `packages/api-client` — API 型別骨架（**stub，待從 starter OpenAPI 重新生成**）

## 狀態

- 後台目前以 `runtime.fixture.ts` 假資料驅動，所有業務頁（orders/commissions/messaging/content/dashboard…）可瀏覽。
- `admin-transport` 與 `api-client` 是空殼 stub，方法呼叫會 throw `transport.not_wired`。接 starter Go API（`server/internal/modules/content|contact|media`）時需重新生成 `api-client` 並重寫 `admin-transport`。
- `apps/admin/src/domains/` 與 `pages/` 是 SiteCore 網紅分潤業務的範本，保留作為參考；之後依 starter 模組長新頁時再替換。

## 指令

```bash
pnpm install
pnpm dev:admin      # Vite dev server (port 5173)
pnpm build:admin    # vue-tsc + vite build
pnpm check          # typecheck + lint + unit tests
```

## 邊界

- 後台不直接查 PostgreSQL/Supabase，一律透過 Go API。
- 後台是獨立部署目標，不混入 `dist/` 靜態站台。
- `admin-ui`/`admin-shell`/`admin-fields`/`admin-theme` 是純展示/框架整合包，不呼叫 API、不引入業務領域。
