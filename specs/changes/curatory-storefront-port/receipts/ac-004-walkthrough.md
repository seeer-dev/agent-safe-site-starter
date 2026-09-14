# AC-004 walkthrough — visual fidelity, page by page

Method: section-by-section comparison of the reference source
(`D:\MyTools\AI設計版面\購物車系列\質選所`) against the rendered
`dist/` output and the curatory theme source. No automated visual-diff
tooling exists in this repo; comparison was structural + token-level.

## Design tokens

- Cream/ink/gold palette, serif+sans pairing (Noto Serif TC / Noto Sans
  TC / Cormorant Garamond, self-hosted), spacing scale, radii and
  shadows ported into `shared/styles/globals.css` — including dark-mode
  variants and `prefers-reduced-motion` media handling.
- Reference-specific CSS restored: `.animate-float` hero card,
  `.prose-curatory` article typography, reveal keyframes, skeletons.

## Per-page comparison

- `/` home — hero + stats + float card, value props, featured grid,
  categories strip, story, latest products, news preview, newsletter
  sections all present in `templates/home.html`.
- `/shop/` — heading, category chips, search/sort controls, product
  grid with ProductCard (featured badge, sold-out stamp, quick-add).
- `/products/{slug}/` — gallery, price/original price, variant picker
  with deltas and sold-out disabling, quantity stepper, CTA pair,
  shipping/payment/returns accordions, comments summary + list + form.
- `/news/` + `/news/{slug}/` — pinned-first list, article page has
  公告/置頂 badges, calendar icon, dual footer CTAs.
- `/about/` — story/values/stats/CTA sections match reference.
- `/cart/`, `/checkout/`, `/order/`, `/track/` — reference view
  structure ported to islands (cart rows + progress + coupon, 4-step
  checkout wizard, order detail stepper/timeline, track form).

## Intentional deviations (approved in spec)

- SPA hash-router → real MPA URLs (per REQ-002).
- No-JS static baselines added under every island (reference had none).
- Header search upgraded to debounced live results (8 max) — reference
  navigated to /shop with query.
- Reference order-number+email lookup intentionally replaced by
  access-token model (spec security decision).
