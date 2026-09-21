# Delivery plan

Change ID: curatory-bloom-mark-and-curtain
Revision: 3
Status: Accepted

## Scope Lock

- specs/changes/curatory-bloom-mark-and-curtain/**
- site/assets/images/curatory-bloom-mark.svg
- site/assets/images/curatory-bloom-curtain-motion.svg
- site/themes/curatory/shared/components/LogoMark.vue
- site/themes/curatory/shared/components/LogoLockup.vue
- site/themes/curatory/shared/components/LogoLoading.vue
- site/themes/curatory/islands/CheckoutPage/CheckoutPage.vue
- site/themes/curatory/islands/SiteHeader/SiteHeader.vue
- site/themes/curatory/shared/components/LogoLoading.vue
- site/themes/curatory/shared/lib/transition.ts
- site/themes/curatory/shared/styles/globals.css
- site/themes/curatory/templates/partials.html

| Slice | Requirements | Surface | Work | Proof |
|---|---|---|---|---|
| S01 | REQ-001, AC-001 | Shared chrome and loader | Add the vector mark, replace the legacy circular mark, and update no-JS chrome. | Client build, render, desktop/mobile walkthrough. |
| S02 | REQ-002, AC-002, AC-003 | Internal page transition | Match the supplied motion reference in the curtain-only mark, timing, palette, lockup, and reduced-motion static fallback while keeping shared marks static. | Transition walkthrough and reduced-motion inspection. |

## Surface contract

| Surface | REQ/AC | Primary task | States | Evidence |
|---|---|---|---|---|
| Shared header/footer/loader | REQ-001 / AC-001 | Identify the storefront and return home | standard, dark footer, no-JS fallback | rendered output and browser walkthrough |
| Page-transition curtain | REQ-002 / AC-002 / AC-003 | Give navigation a short visual handoff | standard motion, reduced motion, interrupted navigation | browser walkthrough |

The mark is a packaged static asset. It has no data dependency, write operation, or error state. Existing navigation still owns link interception and falls back to normal browser navigation if the curtain is unavailable.
