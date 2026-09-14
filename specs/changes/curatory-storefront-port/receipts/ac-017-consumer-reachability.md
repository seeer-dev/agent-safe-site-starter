# AC-017 consumer reachability — storefront bootstrap

- `GET /api/storefront/bootstrap` (200, live) returns `settings`,
  `categories`, `site_content`, `payment_methods`, `shipping_methods`,
  `announcements` — assembled in `bootstrap/storefront.go` at the
  wiring layer (no module-to-module import).
- Consumers: `SiteHeader` (categories dropdown, search), `SiteFooter`
  (links/contact), `AnnouncementBar` (promo banner), CartDrawer and
  checkout (free-shipping threshold from settings), product accordions
  (shipping/payment methods).
- Governance isolation verified: `store_settings` draft was saved then
  published; the bootstrap payload only ever reads `published_json` —
  the unpublished draft did not appear until publish ran.
- No PII and no admin-only fields: the public shipping DTO excludes
  fee/free_threshold (asserted by `shipping_methods_test.go`); settings
  payload contains only the seeded public fields.
