# AC-016 consumer reachability — dashboard statistics

- `DashboardPage.vue` fetches `/admin/stats` (server-aggregated
  products/orders/pending_comments/revenue) alongside
  `/admin/orders`, `/admin/products`, `/admin/comments`.
- KPIs rendered: 待處理訂單, 待出貨, 退貨待審, 低庫存商品, 總營收
  (revenue from stats), 待審評論 (pending_comments from stats with
  list fallback).
- Task list links into orders/products/comments resources for
  actionable follow-through.
- `DashboardPage.test.ts` mounts the page with mocked endpoints and
  asserts all six KPI values render — `['1','1','0','1','4280','1']`.
  The test first failed (4 vs 6 KPIs) when the stats/consumption rows
  were added, then passed after the update — the assertion exercises
  the new consumption path.
- Live check: `GET /api/admin/stats` with dev-admin returned real
  counters (200) on the seeded DB.
