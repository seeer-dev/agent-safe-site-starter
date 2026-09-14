# AC-005 walkthrough — interaction and motion inventory

Exercised against the built theme bundle and running dev site
(localhost:4173 + :8080) after a fresh seed.

## Inventory (all observed in the shipped bundle/source)

- Brand intro loader: `PageLoader` island — plays once per session,
  skipped on repeat visits and under reduced-motion.
- Page transitions: `theme-init.ts` curtain reveal + link-intercept
  transition in `islands/bootstrap.ts`.
- Scroll reveal: `v-reveal` directive (staggered `--reveal-delay`).
- Image fade-in: `FadeImage` shared component on all product imagery.
- Skeleton placeholders: ShopGrid/ProductDetail/comments loading states.
- Toasts: `Toaster` island, fired by cart/comment/checkout actions.
- Cart persistence: localStorage cart rehydrated at bootstrap;
  variant-aware line items.
- Cart drawer: right-side sheet with free-shipping progress bound to
  `settings.freeShippingThreshold`.
- Search dialog: debounced (300ms) live results, count line, Enter to
  first result.
- Checkout: per-step validation, error toasts, processing overlay,
  disabled states while quoting/submitting.
- Hover/focus/disabled/empty/error states ported on controls (buttons,
  steppers, radio cards, dialogs, sheet).

## Reduced motion

`globals.css` gates keyframes/transitions under
`prefers-reduced-motion: reduce`; `bootstrap.ts` checks the media query
before loader/transition work.
