# AC-015 walkthrough — admin surfaces cover every entity

Each screen was opened via the resource registry in the built admin
bundle; capabilities gate every route and action.

| Entity | Screen | Ops exercised |
|---|---|---|
| 總覽 | dashboard | /admin/stats + orders/products/comments KPIs + task links |
| 商品 | minimal-cart-products | list/create/edit (variants editor, dynamic category select, media uploader), 上架/草稿/封存, bulk |
| 分類 | categories | list/create/edit/delete, is_active + sort_order |
| 訂單 | minimal-cart-orders | list/detail + status transitions incl. 結案(completed), return flow, restock |
| 評論 | comments | list + approve/reject with optional reply (reply key) |
| 會員 | minimal-cart-members | existing resource, unchanged |
| 優惠 | minimal-cart-promos | percent/fixed/freeshipping + min_subtotal + usage_limit |
| 公告文章 | articles | slug-keyed upsert (POST+PUT alias), publish/draft, pinned, publish_at |
| 前台內容 | minimal-cart-content | existing governed blocks, unchanged |
| 付款方式 | tw-commerce.methods | fee field added to form + column |
| 配送方式 | minimal-cart-shipping | fee/free_threshold/enabled/sort |
| 通知模板 | notification-templates | 5 lifecycle codes, enable/disable, subject/body |
| 通知日誌 | notification-logs | read-only list + detail |
| 商店設定 | store-settings (/settings) | version-checked draft save + publish |
| 人員 | staff | existing resource, unchanged |

Denied/unauthenticated access: every route carries `caps` and the API
enforces capabilities server-side (403 on missing capability, verified
by existing auth tests).
