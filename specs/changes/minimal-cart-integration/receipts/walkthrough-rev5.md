# Independent Walkthrough Receipt (Revision 5)

Date: 2026-08-12
Change: minimal-cart-integration
Revision: 5
Auditor: Independent review (Devin)

## Scope

Independent local walkthrough of two surfaces after the revision 8
no-JS home product baseline + dev same-origin proxy fix and the B8
nullish cell display fix:

1. Admin/B5: content approval + publish gate, product list/edit
   dialog, ResourceTable nullish cell rendering.
2. Public/static/no-JS: home page raw HTML, ProductGrid hydration,
   product/category/content routes, draft exclusion, expired content
   404.

This receipt records only what was observed in the running dev stack.
It does not authorize a release or blanket-pass any requirement.

## Environment

- OS: Windows (MSYS_NT-10.0-19045)
- Dev stack: `go run ./server/tools/dev` (API :8080 + site :4173 with
  same-origin /api/* proxy)
- Admin: `npm run dev` (Vite, connected via dev-admin)
- Database: SQLite (dev)
- Browser: Chrome

## Surface: Admin — product list and edit dialog

### Observation

- Started documented dev stack and admin Vite; connected with
  dev-admin.
- Product list loaded real 6 admin products.
- Edit product dialog opened with focus on SKU field; media chooser
  present; save button disabled until changes.
- Actual file chooser/upload could NOT be exercised through this
  Browser surface (no setInputFiles capability).

### Conclusion

Product list and edit dialog render real data with correct save-gate
behavior. B4 live/real upload walkthrough remains pending — do not
mark it complete.

## Surface: Admin — ResourceTable nullish cell rendering (B8)

### Observation

- Content table initially exposed literal `undefined` for omitted
  legacy approval fields (rows with no approved_version /
  published_version due to json omitempty on legacy published rows).
- ResourceTable fix hot-reloaded and recheck showed em dash for
  absent fields.
- Numeric `sort_order` 0 stayed `0` (not conflated with absent).

### Conclusion

B8 nullish cell display fix is confirmed in the running admin UI.
Full B8 media/accessibility walkthrough evidence for admin surfaces
remains pending.

## Surface: Admin — B5 content approval and publish gate

### Observation

- Using the admin UI, approved then published `home.hero` and
  `footer.about` with expiry 2026-08-13 23:59 Asia/Taipei.
- After settle, dialogs closed.
- Rows showed draft_version=1, approved_version=1,
  published_version=1, local-admin approver/publisher, expiry
  snapshots.

### Conclusion

B5 approve/publish gate is confirmed end-to-end in the running admin
UI: approve with operator-decided expiry, publish with version
snapshot, governance columns populated. B5 acceptance for this local
walkthrough is complete.

## Surface: Public — same-origin API proxy

### Observation

- Clean restart of `go run ./server/tools/dev`.
- same-origin GET http://127.0.0.1:4173/api/products -> 200, 5
  published products.
- Draft `wide-work-pants` absent from API response.

### Conclusion

Dev same-origin proxy works: browser fetches /api/products from the
same origin as the page (no cross-origin fetch, no CORS issue).

## Surface: Public — home page raw HTML (no-JS baseline)

### Observation

- Raw home HTML includes `#shop-static` section.
- Server-rendered product names present: 素面亞麻襯衫, 陶製小碟,
  線裝筆記本.
- Product links and category links present.
- ProductGrid island mount point present.
- Draft 寬版工作褲 absent from raw HTML.
- `Static Publish Starter` absent.

### Conclusion

No-JS home product baseline is confirmed: raw HTML contains
server-rendered product cards and category navigation. No-JS users
see products without JavaScript.

## Surface: Public — ProductGrid hydration

### Observation

- Browser hydrated ProductGrid; loaded 5/5 products.
- Correct low-stock and out-of-stock states displayed.
- No load error (`無法載入商品` absent).

### Conclusion

ProductGrid island hydrates successfully with same-origin API.
Static baseline is hidden on successful load; interactive grid
replaces it.

## Surface: Public — product detail route

### Observation

- GET /products/linen-shirt/ -> 200.
- Product content present.

### Conclusion

Product detail route serves real static content.

## Surface: Public — category route

### Observation

- GET /categories/apparel/ -> 200.
- Active product present.
- Draft product absent.

### Conclusion

Category route serves real static content with correct draft
exclusion.

## Surface: Public — content route after approve/publish

### Observation

- GET /content/footer.about/ -> 200 with title/body after UI
  approve/publish + clean dev restart.
- Prior expired/unapproved footer snapshot correctly returned 404
  before re-approval.

### Conclusion

Content route serves published content after approve/publish.
Expired/unapproved snapshots are correctly absent (404).

## What this receipt does NOT cover

- B4 live/real media upload (no setInputFiles capability) — remains
  pending.
- Full B8 media/accessibility walkthrough evidence — remains pending.
- Live R2 CopyObject / custom domain nosniff / CDN cache behavior —
  remains pending (no live R2 in env).
- Live PostgreSQL migration 014 apply and integration run — remains
  pending (no live PG in env).
- Live Cloudflare Deploy Hook receipt (GATE-006) — remains pending.
- Secure token recovery mechanism — remains pending.
- Member auth consumer reachability (Supabase customer auth
  integration) — remains pending.
- Formal retention/deletion policy approval (GATE-009) — remains
  pending.
- Independent acceptance walkthrough by a non-implementer reviewer
  — remains pending.
- Shipping fee schedule approval — remains pending.

## Conclusion

This independent local walkthrough confirms:

1. The no-JS home product baseline + dev same-origin proxy fix is
   working: raw HTML contains server-rendered products and category
   nav, ProductGrid hydrates without error, draft products are
   excluded, and the dev proxy serves /api/* same-origin.
2. The B5 content approve/publish gate is working end-to-end in the
   admin UI: approve with expiry, publish with version snapshot,
   governance columns populated, published content served, expired
   content 404.
3. The B8 nullish cell display fix is working: absent fields render
   em dash, numeric 0 preserved.

The change remains Applying. Pending items listed above are not
resolved by this receipt.
