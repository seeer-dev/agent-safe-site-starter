# Admin IA Port Specification

Change ID: admin-ia-port
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved revision 1 on 2026-09-15 by directing the reference mockup into specs/changes/admin-ia-port/reference/ and instructing the change be landed. The owner supplied the mockup as the reference IA and asked to port its grouped sidebar, compact proportions, internal components (skeleton, sorting, dropdowns, tabs, permissions matrix), dashboard polish, and global search into the live admin SPA.
Repository baseline: 3a196c39ce746d1bf9c078a137fb124f879289ae
Supersedes: none

## Outcome

The admin SPA adopts the information architecture and component vocabulary of
the owner-supplied reference mockup: a two-level grouped sidebar with compact
proportions, skeleton loading, sortable tables, real dropdown menus and
filters, tabs, a roles/permissions matrix, a polished dashboard, and global
search — all driven by real API data and the existing capability model.

The reference file moves under this change directory as `reference/` so it is
versioned alongside the work and stops tripping the protected-path gates as an
untracked file under `admin/`.

## Scope

In scope:

- `admin/src` layout, navigation configuration, shared UI components, resource
  table/filters, pages, router, layout store, and global styles.
- Relocating `admin/influencer-admin-ia.html` to
  `specs/changes/admin-ia-port/reference/` (delete the `admin/` copy).
- Removing the display-only orders state-machine block (`config/machines.ts`,
  its `StateMachineFlow` type, and the `.machine*` styles): absent from the
  mockup, its `expected_version` warning already lives in `ConfirmDialog`,
  and valid transitions are expressed by per-row action buttons.
- Everything renders existing API data and existing capability checks; no new
  endpoints, no backend changes.

Out of scope:

- Influencer/campaign/commission/payout business pages from the mockup — the
  commerce backend has no such resources, and this change adds none.
- The mockup's Tailwind-CDN styling approach; the SPA keeps its
  `globals.css` custom-property design system.
- Charts fed by series the API does not provide. If `/admin/stats` has no
  trend series, no chart is drawn — fabricated data is forbidden.
- A multi-site or platform-shell abstraction. The grouping is a data shape in
  `profile.ts`, not a registry.

## Decisions and invariants

- Grouped navigation is expressed as `children` on `RouteDef` — the existing
  sidebar machinery (inline expand, collapsed-mode flyout) is activated, not
  rewritten. A labeled divider separates business module groups from the
  universal base groups (the mockup's "通用後台 Base" idea).
- Capability gating is recursive: a child the principal cannot use is hidden,
  and a group with zero visible children is hidden. Hidden navigation MUST
  never be the only gate — route-level guards stay authoritative.
- Loading skeletons replace text-only loading. A skeleton MUST never render
  fixture or invented data, and empty/error/authorization states keep their
  current fail-closed behavior.
- Table sorting is client-side over already-loaded rows and opt-in per
  column; it MUST NOT fabricate rows or reorder server-owned semantics
  (e.g. a column explicitly ordered by the API keeps that order unless the
  column is declared sortable).
- Search indexes visible navigation only — it can never surface a destination
  the principal cannot reach.
- Compact proportions follow the reference: 14px-height brand header, 16px
  nav icons, a user block pinned to the sidebar bottom.

## Requirements

### REQ-001: Two-level grouped sidebar

The navigation profile MUST group primary entries into named categories with
children, with a labeled divider before the universal base groups. Expanded
mode MUST offer collapsible groups; collapsed mode MUST keep working via the
existing flyout.

#### AC-001: Groups expand, collapse, and track active children

- GIVEN a grouped profile and the expanded sidebar
- WHEN a category is clicked, or a child route is active
- THEN the group MUST toggle or show the active child, and the active child
  MUST be visually distinct.

### REQ-002: Capability gating applies to groups

Capability filtering MUST apply recursively to grouped navigation: a child
the principal cannot use MUST be hidden, and a group with zero visible
children MUST be hidden.

#### AC-002: Hidden means hidden

- GIVEN a principal lacking a child's capability
- WHEN the sidebar renders in expanded, collapsed, or mobile mode
- THEN the child MUST be absent, and a group whose children are all hidden
  MUST be absent entirely.

### REQ-003: Skeleton loading replaces text-only waiting

List pages and the dashboard MUST render skeleton blocks/rows while data is
in flight instead of a bare "載入中" label.

#### AC-003: Skeleton shows structure, never data

- GIVEN a resource list or dashboard request in flight
- WHEN the loading state renders
- THEN skeleton placeholders MUST appear, and no fixture, browser-local, or
  invented rows may be displayed.

### REQ-004: Sortable table columns

`ResourceTable` MUST support per-column opt-in sorting with a visible
asc/desc indicator, applied client-side to loaded rows.

#### AC-004: Sort toggles and indicates direction

- GIVEN a column declared sortable on a resource
- WHEN its header is clicked once, twice, and a non-sortable column is clicked
- THEN rows MUST sort ascending, then descending, and the non-sortable click
  MUST do nothing; the indicator MUST show direction.

### REQ-005: Real dropdown menus and dropdown filters

Row/overflow actions MUST collapse into a dropdown menu, and filter bars MUST
use dropdown-style controls consistent with the reference instead of raw
native selects where the reference pattern applies.

#### AC-005: Menu opens, acts, and dismisses correctly

- GIVEN a row with multiple or destructive actions
- WHEN the menu trigger is clicked, an item chosen, and Esc/outside-click
  pressed
- THEN the menu MUST open, invoke the action, and dismiss on Esc and outside
  click without invoking anything.

### REQ-006: Tabs component

A reusable tabs primitive MUST exist and be used where a page presents
parallel panels (e.g. notification templates vs. logs, or settings
groupings).

#### AC-006: Tabs switch panels accessibly

- GIVEN a page using the tabs component
- WHEN a tab is activated by click or arrow-key navigation
- THEN its panel MUST show, others hide, and the active tab MUST expose the
  selected state to assistive technology.

### REQ-007: Roles and permissions matrix page

A read-only page MUST render every configured role and its capabilities as a
matrix, sourced from the existing roles config.

#### AC-007: Matrix reflects the configured role model

- GIVEN an authenticated principal permitted to view staff/role data
- WHEN the roles page loads
- THEN every role and capability from config MUST be listed with allow/deny
  shown truthfully, and the page MUST be denied to principals lacking the
  capability.

### REQ-008: Dashboard polish and global search

KPI cards MUST carry icon, delta/context, and click-through to their
resource; the pending-work panel stays real. A ⌘K/Ctrl+K palette MUST search
visible navigation destinations and resource labels.

#### AC-008: Search jumps only to reachable destinations

- GIVEN the palette open
- WHEN a query filters destinations and a result is confirmed
- THEN navigation MUST occur, and no destination outside the principal's
  capabilities may appear.

#### AC-009: Dashboard stays truthful

- GIVEN the dashboard with live data, an error response, and an empty
  workload
- WHEN each state renders
- THEN KPIs and tasks MUST reflect real API values, and error/empty states
  MUST remain explicit — no fixture substitution.

## Amendments

None.
