# Admin IA Port Delivery Plan

Change ID: admin-ia-port
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| `RouteDef.children` and the sidebar's expand/flyout machinery already exist | `lib/types.ts:30-34`, `Sidebar.vue:81-143`; comment notes no PROFILE entry uses children | Grouping is data + activation, not new machinery |
| Nav is a flat 3-section list | `config/profile.ts` sections `primary`/`secondary`/`settings` | Introduce groups inside sections plus a labeled divider |
| `Select.vue` is a native `<select>` | `ui/Select.vue:32` | Dropdown menu is a new primitive; native select stays for forms |
| `ResourceTable.vue` has zero sorting | grep `sort` → 0 matches | Add per-column opt-in sort over loaded rows |
| Loading is text-only | `DashboardPage.vue:150` "載入中…" emptybox | Add `Skeleton` ui component, wire list + dashboard |
| Roles model exists without a page | `config/roles.ts`, `auth.can()` | Matrix page is pure read of config — no backend work |
| Mockup file is an untracked gate violator | `admin/influencer-admin-ia.html` flagged by speccheck/scopecheck | Move it into `specs/changes/admin-ia-port/reference/` |
| Mobile reuses `Sidebar` in a drawer | `AdminShell.vue:36` | Grouped nav must not break the drawer |

## Scope lock

- `admin/src/components/layout/**`, `admin/src/components/ui/**`,
  `admin/src/components/resource/**`, `admin/src/pages/**`
- `admin/src/config/profile.ts`, `admin/src/config/resources/**`,
  `admin/src/config/machines.ts` (deleted: display-only state-machine block)
- `admin/src/lib/types.ts`, `admin/src/router.ts`, `admin/src/router.test.ts`,
  `admin/src/stores/layout.ts`, `admin/src/styles/globals.css`
- `admin/influencer-admin-ia.html` (moved into `specs/changes/admin-ia-port/reference/`)
- `specs/changes/admin-ia-port/**`

## Dependency-ordered slices

### Slice 0: Reference relocation

Outcome: `admin/influencer-admin-ia.html` moves to
`specs/changes/admin-ia-port/reference/influencer-admin-ia.html`; the
standing protected-path violation clears.

Rollback: move the file back.

### Slice 1: Grouped sidebar + compact proportions

Outcome: `profile.ts` gains groups (children + divider between module and
universal base); `Sidebar.vue` activates existing children machinery and
adopts the reference proportions (h-14 brand, 16px icons, bottom user
block); `MobileNav`/drawer verified; `types.ts` extended only if a
divider/group marker needs a field.

Edits: `profile.ts`, `Sidebar.vue`, `Sidebar.test.ts`, `MobileNav.vue`,
`AdminShell.vue` (if brand/footer markup lives there), `globals.css`,
`types.ts`, `router.test.ts` if routes are added.

Acceptance evidence: group expand/collapse, active-child highlight,
collapsed flyout, and hidden-child/group removal proven per role;
walkthrough receipt for AC-001, security-review receipt for AC-002.
Covers REQ-001, REQ-002, AC-001, AC-002.

Rollback: revert listed files; flat nav returns.

### Slice 2: Component primitives — Skeleton, DropdownMenu, Tabs

Outcome: three new `ui/` components with unit tests; no consumers yet.

Acceptance evidence: component tests for render/dismiss/keyboard behavior.
Covers the primitive half of REQ-003, REQ-005, REQ-006 (AC-003/AC-005/AC-006
land with their consumers in Slices 3–4).

Rollback: delete the new components.

### Slice 3: Table sorting + filter/menu integration + skeleton consumers

Outcome: `ResourceTable` gains opt-in column sorting; row actions collapse
into `DropdownMenu`; `ResourceFilters` uses dropdown controls; list pages
and the dashboard render `Skeleton` while loading.

Edits: `ResourceTable.vue`, `ResourceTable.test.ts`, `ResourceFilters.vue`,
`ResourceListPage.vue`, `DashboardPage.vue`, resource configs (per-column
`sortable` flags), `globals.css`.

Acceptance evidence: sort direction/indicators, menu open/act/dismiss,
skeleton-with-no-fixture; walkthrough receipts for AC-004, AC-005. Covers
REQ-003, REQ-004, REQ-005, REQ-006 (tabs consumed where parallel panels
exist), AC-003, AC-004, AC-005, AC-006.

Rollback: revert listed files.

### Slice 4: Roles matrix + search palette + dashboard polish

Outcome: read-only roles/capabilities matrix page (new route guarded by the
staff capability); ⌘K/Ctrl+K palette over visible destinations; dashboard
KPI cards gain icon/delta/click-through.

Edits: new `RolesPage.vue` + palette component, `router.ts`,
`DashboardPage.vue`, `DashboardPage.test.ts`, `Topbar.vue` (palette
trigger), `stores/layout.ts` (palette open state), `globals.css`.

Acceptance evidence: matrix truthfulness + deny path (AC-007), palette
gating + navigation (AC-008 walkthrough), truthful dashboard states
(AC-009 walkthrough). Covers REQ-007, REQ-008, AC-007, AC-008, AC-009.

Rollback: revert listed files.

## Traceability matrix

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 1 | Sidebar tests + walkthrough receipt |
| REQ-002, AC-002 | 1 | Per-role render tests + security-review receipt |
| REQ-003, AC-003 | 2,3 | Skeleton test; loading-state test asserts no fixture rows |
| REQ-004, AC-004 | 3 | Sort table test + walkthrough receipt |
| REQ-005, AC-005 | 2,3 | DropdownMenu test + walkthrough receipt |
| REQ-006, AC-006 | 2,3 | Tabs keyboard/ARIA test |
| REQ-007, AC-007 | 4 | Matrix page test incl. denied principal |
| REQ-008, AC-008 | 4 | Palette gating test + walkthrough receipt |
| REQ-008, AC-009 | 4 | Dashboard real/error/empty tests + walkthrough receipt |

## Risks and controls

- Risk: grouped nav regresses mobile drawer. Control: Sidebar is shared;
  mobile tests run and the drawer walkthrough is part of AC-001's receipt.
- Risk: sorting masks server-authoritative order (e.g. sort_order-driven
  lists). Control: sorting is opt-in per column; default unsorted columns
  keep server order.
- Risk: skeletons become a fixture backdoor. Control: AC-003 forbids
  fixture/local/invented rows in loading states.
- Risk: search palette leaks gated destinations. Control: palette builds
  from the same capability-filtered nav set; AC-008 proves it.
- Risk: dashboard polish reintroduces fabricated metrics. Control:
  AC-009 requires real/error/empty states and forbids fixture substitution;
  charts are skipped where no real series exists.
