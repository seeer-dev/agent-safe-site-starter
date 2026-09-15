# Walkthrough — admin IA port acceptance

Change ID: admin-ia-port
Revision: 1
Covers: AC-001, AC-004, AC-005, AC-008, AC-009
Performed: 2026-09-15

Each scenario below was exercised against the built components via the
Vitest/jsdom suite (`npx vitest run`, 242 tests green) and, where noted, by
direct DOM inspection of the mounted result. AC-001 was additionally
verified in a real Chromium session against `npm run dev` after a
post-acceptance layout regression report (see the regression note below).

## AC-001 — Groups expand, collapse, and track active children

Steps:

1. Mount the expanded `Sidebar` with the grouped profile.
2. Click the 顧客 group header → the inline child list toggles open; the
   chevron rotates 90°.
3. Click it again → the list collapses.
4. Navigate to `/res/minimal-cart-members` → the 顧客 group auto-expands and
   the 會員 child carries the `is-current` class; the group header itself is
   marked `active`. Flat leaves (訂單/商品/分類) carry `active` directly.
5. Collapse the sidebar (`layout.sidebarCollapsed = true`) → hovering a
   group opens the flyout panel listing the same filtered children.
6. Open the mobile drawer (`layout.mobileDrawerOpen = true`) → the same
   `Sidebar` renders inside `.drawer-panel`; tapping a child emits
   `navigate` and the drawer closes.

Observed: `Sidebar.test.ts` (11 tests) + `AdminShell.test.ts` (3 tests) —
group toggle, active-child auto-expand + highlight, collapsed flyout
structure (flyout/bridge are direct children of the positioned
`.nav-item-wrapper`; parents opt out of the leaf tooltip via
`data-has-children`), drawer outside `.app`, drawer opens, child navigation
closes the drawer, gated destinations absent from the drawer; all green.

Regression note (post-acceptance fix, verified in real Chromium on the dev
server): the owner reported the collapsed flyout stacked at the sidebar top
and the mobile layout broke entirely. Root causes and fixes:

- `.nav-item-wrapper` was not positioned, so the absolutely positioned
  `.flyout` anchored to the sticky `.sidebar` (`top:0` = sidebar top) —
  every group's flyout rendered stacked at the top. Fixed with
  `position:relative` on the wrapper; re-verified live: each group's flyout
  top now equals its hovered item top (e.g. 商店 at y=92, 系統 at y=221),
  left edge sits at the collapsed rail's right edge, and the hover bridge
  keeps the path open into the panel.
- `@media(max-width:920px)` declared `.sidebar{display:none}` which also
  hid the Sidebar copy inside the mobile drawer (blank panel), and
  `.app[data-sidebar="collapsed"]`'s `56px 1fr` columns outranked the
  mobile `1fr` override, so a persisted collapsed state left a dead gutter
  and squeezed the app. Fixed with `.mobile-drawer .sidebar{display:flex;
  position:static;...}` and `.app[data-sidebar]{grid-template-columns:1fr}`
  inside the media query. Re-verified live at 390px with collapsed state
  persisted: single-column grid (`390px`), drawer opens with labels fully
  visible, group expands, tapping 訂單 navigates to
  `/res/minimal-cart-orders` and closes the drawer.

Follow-up owner request (same review): child items in both the inline
`.nav-children` list and the collapsed `.flyout` now render their declared
icon (falling back to the parent group icon when a child omits one) at
14px/`--text-3`, brand-colored when current; verified live in expanded and
collapsed modes. The mobile drawer inherits the same markup. Also per the
same review, `.nav-label` now flexes to fill so each group's `nav-chevron`
sits flush at the item's right edge (labels truncate rather than pushing
it out); verified live: chevron right edge sits 10px (item padding) inside
every group's right border. Also per the same review, the display-only
orders state-machine block (two `.machine` diagram panels + the
`expected_version` explainer note) was removed: the mockup never included
it, `ConfirmDialog` already surfaces the `expected_version` warning at the
decision moment, and valid transitions are already expressed by per-row
buttons. `config/machines.ts`, the `StateMachineFlow` type, and the
`.machine*` CSS were deleted with it (`.note` stays — ConfirmDialog and
form notices still use it). Verified live: the orders page opens directly
into filters/table with no `.machine` node. Also per the same review,
`ResourceTable` now scrolls horizontally when columns exceed the panel
(width:max-content inside an `.rtable-wrap` overflow container, `td` capped
at 340px) and supports column pinning: `Col.pin:'left'|'right'` stacks
pinned cells outward with measured header offsets, `ResourceDef.pinActions`
pins the 動作 column right, and the select-all checkbox column auto-pins
when a left-pinned column exists. All thirteen resource configs pin their
identifier column left and their actions column right. Verified live at a
700px viewport: orders table scrolls (scrollWidth 1026 vs wrap 674), the id
column stays at the left edge and 動作 stays at the right edge after
scrolling 300px. ResourceTable.test.ts pins block: 4 tests, all green.

Navigation rearrangement (owner review of the landed IA): above the
divider, 訂單/商品/分類 are now flat top-level leaves and a single 顧客
group holds 會員+評論; below the divider the groups are 內容 (公告文章/
前台內容), 通知 (模板/日誌), 商店 (優惠/商店設定/付款方式/配送方式 —
優惠 moved in from the old business group, 商店設定 moved in from 系統),
and 系統 (人員/角色權限). The old 商店 and 交易 groups were dissolved.
Groups now carry `caps:[]` so a mixed-capability group (商店 holds both
twcommerce.read and content.read children) stays reachable by any
principal that can see one child; the recursive child filter still does
the real gating. Divider emit-once behavior is unchanged — `dividerBefore`
is declared on every below-divider group so the label still renders when
earlier universal groups are fully gated. Sidebar.test.ts updated to the
new arrangement (顧客 toggling/auto-expand, gated-group assertions);
21 layout tests green.

Dropdown clipping (owner regression report after table pinning): menus
opened inside the table were clipped/covered — `.rtable-wrap{overflow-x:
auto}` also computes overflow-y to auto (scroll containers clip both
axes), the pinned actions cell's `position:sticky` creates a stacking
context that trapped the menu's z-index under later rows, and `.panel{
overflow:hidden}` clipped the rest. `DropdownMenu` now teleports its menu
to `<body>` and positions it `fixed` from the trigger's bounding rect,
repositioning on scroll (capture — reaches nested scrollers like
`.rtable-wrap`) and resize, flipping upward when it would overflow the
viewport. The outside-click guard treats the teleported menu as inside,
and the menu carries its own keydown handler since teleported key events
no longer bubble through the root. Verified live: menu parent is
`document.body`, `position:fixed`, no clipping at the wrap edge; the last
row's menu correctly flipped upward. DropdownMenu.test.ts gained two
regression guards (body-level teleport, mousedown inside menu does not
close it); 248 tests green.

## AC-004 — Sort toggles and indicates direction

Steps:

1. Mount `ResourceTable` with a column declared `sortable` (name) and one
   without (SKU).
2. Click the name header once → rows order ascending per `zh-Hant` collation;
   the header exposes `aria-sort="ascending"` and the icon flips to the
   up-arrow.
3. Click again → rows reverse; `aria-sort="descending"`, down-arrow.
4. Click the non-sortable SKU header → nothing happens; no `aria-sort`
   attribute, no reorder.
5. Sort a numeric column (price) → numeric order, not lexicographic.
6. Sort a column containing an empty value → the empty row stays last in
   both directions.
7. Sort, then fire a row action on the first displayed row → the emitted
   index is the row's index in the original `rows` array, not its sorted
   position.

Observed: `ResourceTable.test.ts` sortable-columns block (7 tests) — all
green; products/orders/members/categories/articles/comments/notification
-templates resource configs mark representative columns `sortable`, and
columns without the flag preserve server order.

## AC-005 — Menu opens, acts, and dismisses correctly

Steps:

1. Row with three actions (edit / publish / archive-danger): the first
   action stays inline; clicking the 更多 trigger opens a `role="menu"`
   dropdown listing the remaining two, 封存 styled `danger`.
2. Choose 封存 → `rowAction` emits `(originalIndex, 'archive')`; the menu
   closes.
3. Re-open, press Escape → menu closes, nothing emitted.
4. Re-open, click outside → menu closes, nothing emitted.
5. As a principal missing `twcommerce.delete`: the 封存 item remains
   visible, disabled, and displays the reason `需要 twcommerce.delete`;
   clicking it emits nothing.
6. Filter bar: the 狀態/分類 select filters render as dropdown menus; the
   current value is check-marked; choosing an item updates the filter model;
   Escape and outside click dismiss.

Observed: `DropdownMenu.test.ts` (7 tests) + `ResourceTable.test.ts`
dropdown block (4 tests) — all green. `ResourceFilters.vue` renders
`DropdownMenu` for `w: 'select'` filters; text filters keep inputs; the
form builder still uses native selects for ordinary fields.

## AC-008 — Search jumps only to reachable destinations

Steps:

1. Press Ctrl+K (or ⌘K) → the palette overlay opens with the input focused.
2. As `twcommerce.read` + `staff.read`: results include 商品, 人員,
   角色權限, 五狀態參考 — and exclude 公告文章 (needs `content.publish`).
3. As `twcommerce.read` only: results exclude every `staff.*` and
   `content.*` destination.
4. Type 訂單 → a single result remains; press Enter → the app navigates to
   `/res/minimal-cart-orders` and the palette closes.
5. Press Escape or click the backdrop → the palette closes without
   navigating.

Observed: `SearchPalette.test.ts` (5 tests) — all green. The index is a
`computed` over `navLeaves()` filtered by `auth.can`, so it can only ever
list what the nav surfaces can show; route guards remain authoritative.

## AC-009 — Dashboard stays truthful

Steps:

1. Live data: mount with API responses (2 orders, 1 low-stock product,
   revenue 4280, 1 pending comment) → six KPI cards render the real values
   `[1, 1, 0, 1, 4280, 1]`, each with an icon and a `res` click-through;
   the 待處理 panel lists real rows (TW-1 pending → 開始處理).
2. Error: all four API calls fail → the panel shows 載入失敗 with the error
   text; zero KPI cards render — nothing fabricated.
3. Empty workload: APIs resolve with empty collections → KPIs show real
   zeros, the task panel shows the explicit 沒有待處理項目 empty state.
4. In flight: while requests are pending, skeleton placeholders render and
   no value row appears; the literal string 載入中 is not in the DOM text.

Observed: `DashboardPage.test.ts` (4 tests) — all green. KPI values and
task rows derive only from `/admin/orders`, `/admin/products`,
`/admin/stats`, `/admin/comments`; no fixture, browser-local, or invented
content appears in any state.
