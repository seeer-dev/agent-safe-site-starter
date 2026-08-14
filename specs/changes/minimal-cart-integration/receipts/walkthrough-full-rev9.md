# Independent Full Acceptance Walkthrough (Revision 9)

Date: 2026-08-13

## Source and environment

- Controlled change: `minimal-cart-integration`, revision 9, Applying.
- Local dev stack: Go API on `:8080`, rendered public site on `:4173`,
  admin Vite app on `:5173`.
- Database driver: SQLite. Supabase provider configuration and live PostgreSQL
  were absent. R2 credentials, bucket and public base URL were present.
- Personas: public visitor and local development owner/admin. The dev token was
  submitted only to the local admin/API and was not printed or stored by the
  browser.

## Accepted observations

- Public home returned 200 and hydrated five authoritative products with
  low-stock/out-of-stock states, no horizontal overflow, 13 island mounts and
  no images missing `alt`. The server-rendered `#shop-static` baseline remained
  in raw HTML and was hidden only after the runtime product fetch succeeded.
- Cart add/open worked. Checkout truthfully disabled all shipping methods and
  progression because no shipping fee schedule has been approved; no order was
  submitted.
- Product, category and published-content routes returned 200. Unknown product
  returned 404. The same-origin public products API returned 200 JSON.
- Admin local auth succeeded without browser persistence. Product list loaded
  six authoritative rows. Product edit opened an accessible modal with
  `role=dialog`, `aria-modal=true`, label association and initial focus on SKU.
  No mutation was submitted.
- Admin orders showed an honest empty state and the expected operation contract.
- Public and admin console warning/error logs were empty for the covered paths.
- Live R2 verified JPEG returned 200 on HEAD and GET with `image/jpeg`, 196145
  bytes, and `Cache-Control: public, max-age=31536000, immutable`.

## Routed gaps

### GAP-FULL-001 — broken, blocking AC-018

The shared public `Dialog.vue` renders visual modals without `role=dialog`,
`aria-modal`, or an accessible-name relationship. Opening checkout leaves focus
on the trigger. Tabbing from the last enabled modal control escapes to `body`,
and Escape closes without restoring trigger focus. All public consumers of the
shared primitive are affected. ImageZoomModal is a separate component.

### GAP-FULL-002 — broken, blocking AC-005/AC-016

`category.html` emits `data-category`, but the islands bootstrap only consumes
JSON from `data-props`, and `ProductGrid.vue` declares no initial-category prop.
After hydration, `/categories/apparel/` shows a second interactive grid titled
"全部商品" with all five products instead of preserving the apparel scope.

### GAP-FULL-003 — broken, blocking AC-015/AC-018

Admin sidebar/mobile navigation renders `<a>` elements with click handlers but
no `href`. The accessibility tree exposes their labels as text rather than
links, and standard link behavior (copy/open in new tab) is unavailable.

### GAP-FULL-004 — production-content residue, blocking AC-019

Rendered home, product, category, content and article titles still contain
`AI Site Starter`; the visible product uses the `質物選物` brand everywhere
else. This scaffold title is present in `dist/` and is not approved public copy.

### GAP-FULL-005 — external configuration, blocking AC-023/AC-024

The live verified R2 object omitted `X-Content-Type-Options: nosniff` on both
HEAD and GET. Cache and MIME headers were correct. This cannot be repaired by
the static Pages `_headers` file; the R2/custom-domain response path needs a
Cloudflare response-header rule and live replay.

## Remediation replay

The local gaps were implemented by Grok and independently replayed by Codex on
the rebuilt output:

- GAP-FULL-001 resolved: the shared modal now exposes `role=dialog`,
  `aria-modal=true` and a truthful accessible name. Account-dialog initial
  focus enters the visible close control; Tab and Shift+Tab remain trapped;
  Escape closes and restores the Header account trigger. A runtime replay found
  and fixed a second defect where a control under a `display:none` ancestor was
  incorrectly selected for focus.
- GAP-FULL-002 resolved: `/categories/apparel/` hydrates with the `服飾`
  heading and exactly one apparel product. The server `#shop-static` baseline
  remains available before a successful fetch and is replaced after success.
- GAP-FULL-003 resolved: desktop and mobile admin navigation use real
  `RouterLink` anchors. The accessibility tree exposes correct `href` values;
  activating 商品 navigates to `/res/minimal-cart-products` and marks the link
  active.
- GAP-FULL-004 resolved: rendered home is titled `質物選物`, apparel is titled
  `服飾 — 質物選物`, the known local `hello-starter` dev fixture was removed by
  an exact ID/content/byte match, render produced zero articles, and its route
  returns 404. Minimal-cart dev no longer auto-seeds the sample article.
- Public/admin browser logs contained no warning or error entries during the
  remediation replay.

Independent commands passed: admin 145 tests, typecheck, production build and
resource contracts; theme typecheck, dialog/category/source contracts and
production build; full `go test ./server/... -count=1`; `go vet ./server/...`;
targeted render/dev tool tests; `gofmt`; `speccheck`; and `go run
./server/tools/render`. `verify` reached and passed archcheck/speccheck, then
stopped at scopecheck because the repository already contains hundreds of
pre-existing dirty paths outside this slice; no full verify pass is claimed.

## Disposition

The covered local visitor/admin remediation is accepted. GAP-FULL-001 through
004 are closed. GAP-FULL-005 remains an operator-owned external configuration
blocker: the live R2/custom-domain response still needs
`X-Content-Type-Options: nosniff` and a live HEAD/GET replay. Live Supabase
success and live PostgreSQL validation remain separately pending, so the
overall controlled change stays `Applying`.
