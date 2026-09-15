# Security review — grouped navigation capability gating

Change ID: admin-ia-port
Revision: 1
Covers: AC-002 (capability-gated navigation surfaces: expanded sidebar,
collapsed flyout, mobile drawer/nav, and the ⌘K search palette)
Reviewed: 2026-09-15

## What changed under review

`Sidebar.vue` now renders a two-level grouped profile. Capability filtering
is recursive: `visibleNav` drops any parent whose `caps` the principal does
not hold, filters each group's `children` individually, and drops a group
whose visible-children set is empty. The same filtered set feeds the
expanded-mode inline children, the collapsed-mode hover flyout, the mobile
drawer (which renders the same `Sidebar` component), `MobileNav`'s flattened
leaf list, and `SearchPalette`'s result index.

## Controls verified in the diff

1. **Hiding is presentation, never the gate.** Route protection stays in
   `router.ts` (`authCapabilityGuard` on `meta.caps`), and every API call is
   authorized server-side. Removing a nav item only removes a link; a direct
   visit to `/roles` or `/res/<key>` still hits the route guard, and a
   verified principal without `staff.read` is redirected to `/` —
   `router.test.ts` asserts `/roles` denies without `staff.read` after
   verification. UI hiding cannot grant anything.
2. **Recursive filtering is fail-closed in all three nav surfaces.**
   `Sidebar.test.ts` drives principals with partial caps and asserts:
   a gated child (人員, `staff.read`) is absent from the expanded list AND
   the collapsed flyout; a group whose children are all gated (內容,
   `content.*`) disappears entirely; `MobileNav.test.ts` asserts the mobile
   set contains only leaves whose caps the principal holds. No filtered
   path renders an `<a>`/`RouterLink` for the hidden key.
3. **The search palette cannot leak gated destinations.** `SearchPalette.vue`
   builds its index from `navLeaves()` — the same capability-filtered leaf
   set the nav surfaces use — plus the always-open `/states` reference.
   `SearchPalette.test.ts` proves a `twcommerce.read`-only principal sees
   商品/訂單 but never 人員, 角色權限, 公告文章, or 商店設定 in results, and
   that Enter on a result navigates to a route the guard would admit anyway.
4. **Disabled ≠ hidden only where visibility is informative.** Row actions
   the principal lacks stay visible-but-disabled with a visible reason
   (`需要 <cap>`) — they are already rendered on a page the principal could
   open, so no gated destination leaks; the action itself still hits the
   server-authorized endpoint.
5. **No new trust surface.** No new endpoints, no client-side capability
   grants, no persistence of caps. The roles matrix page is a read-only
   rendering of `config/roles.ts` (the declared model), labeled as such;
   it neither mutates roles nor asserts live server grants.

## Residual

- **Nav hiding is cosmetic for a crafted URL.** Intended and documented —
  the route guard plus server authorization are authoritative. A principal
  probing `/roles` without `staff.read` gets the SPA's redirect, and the
  underlying APIs remain server-gated.
- **Roles matrix reflects config, not live grants.** The page reads the
  declared `ROLES` map; if server capabilities drift from config, the matrix
  could show a stale model. It is labeled "實際授權以伺服器為準" and is
  informational only — no authorization decision consumes it.
- **Palette index is rebuilt reactively.** If a principal's caps change
  mid-session (e.g. logout/login as a different role), the next palette
  open reflects the new set; no stale destination survives because the
  index is `computed` over the live store.
