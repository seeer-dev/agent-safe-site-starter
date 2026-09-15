# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-09-15. Owner approval recorded
via plain apply; status ran Draft -> Applying -> Accepted. All four slices
landed in dependency order. A post-acceptance regression (collapsed flyout
mis-anchored; mobile layout/drawer broken) was reported during owner review
and fixed in scope: `.nav-item-wrapper` positioning, `.mobile-drawer
.sidebar` re-show override, and a higher-specificity mobile grid column
rule; see the AC-001 regression note in `receipts/walkthrough.md`.
`npx vitest run` is green at 246 tests across 22 files; `npm run typecheck`
and `npm run build` are clean; `speccheck`/`scopecheck`/`verify` all pass.

Walkthrough receipt for AC-001/AC-004/AC-005/AC-008/AC-009 is at
`receipts/walkthrough.md`; the AC-002 security-review receipt is at
`receipts/security-review.md`.

Standing gate note: `npm run check:resource-contracts` reports one
pre-existing baseline failure unrelated to this change — the content
resource's placement select uses tuple `opts` while the check regex expects
a flat string list. `admin/src/config/resources/content.ts` is byte-identical
to baseline (`git show HEAD:` confirmed); no file this change touched is
implicated. The script is a protected governance surface; correcting the
check or the config is out of this change's scope.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Grouped profile (children + labeled 通用後台 divider); Sidebar activates existing expand/flyout machinery and auto-expands the active child's group |
| REQ-002 | passed | visibleNav recursively filters children and drops empty groups across sidebar, flyout, drawer, MobileNav, and palette; route guards stay authoritative (router.test.ts /roles deny) |
| REQ-003 | passed | Skeleton.vue consumed by ResourceListPage (table variant), DashboardPage, StoreSettingsPage; no fixture/invented rows in loading states (DashboardPage.test.ts) |
| REQ-004 | passed | Col.sortable opt-in; asc/desc toggle with aria-sort + icon; numeric vs zh-Hant compare; empties last; original indexes emitted; flagged across seven resource configs |
| REQ-005 | passed | DropdownMenu backs row-action overflow in ResourceTable and select filters in ResourceFilters; Esc/outside-click dismiss; disabled items show reason; danger styling |
| REQ-006 | passed | Tabs.vue with tablist/tab/tabpanel, aria-selected, roving tabindex, arrow-key nav; consumed by StoreSettingsPage parallel panels |
| REQ-007 | passed | RolesPage read-only matrix lists every role and capability from config/roles.ts; /roles guarded by staff.read |
| REQ-008 | passed | KPI cards carry icon/context/click-through with real API values; SearchPalette on Ctrl+K/Cmd+K indexes capability-filtered navLeaves only |
| AC-001 | passed | Sidebar.test.ts 11 tests + AdminShell.test.ts 3 tests: toggle, auto-expand active child, flyout anchoring, drawer outside .app with navigate-close; walkthrough at receipts/walkthrough.md (incl. post-acceptance flyout/mobile regression fix verified in real Chromium) |
| AC-002 | passed | Recursive gating proven in expanded/collapsed/mobile surfaces; fully gated group absent; security review at receipts/security-review.md |
| AC-003 | passed | Skeleton.test.ts + DashboardPage loading test: skeletons render, no fixture/invented rows, no 載入中 text while in flight |
| AC-004 | passed | ResourceTable.test.ts sortable block (7 tests): asc/desc toggle + indicator, inert non-sortable, numeric order, empties last, original-index actions; walkthrough at receipts/walkthrough.md |
| AC-005 | passed | DropdownMenu.test.ts 7 tests + ResourceTable dropdown block 4 tests: open/act/dismiss, disabled-with-reason, danger; walkthrough at receipts/walkthrough.md |
| AC-006 | passed | Tabs.test.ts 4 tests: roles, aria-selected, roving tabindex, arrow-key activation; live consumer StoreSettingsPage |
| AC-007 | passed | RolesPage.test.ts 4 tests: all roles/capabilities listed, allow/deny truthful, read-only; router.test.ts denies staff.read-less principals |
| AC-008 | passed | SearchPalette.test.ts 5 tests: only cap-visible destinations indexed, query filter, Enter navigates, Esc/backdrop close; walkthrough at receipts/walkthrough.md |
| AC-009 | passed | DashboardPage.test.ts 4 tests: real KPI values + click-through, explicit error with no fabricated KPIs, explicit empty state, skeleton without values; walkthrough at receipts/walkthrough.md |
