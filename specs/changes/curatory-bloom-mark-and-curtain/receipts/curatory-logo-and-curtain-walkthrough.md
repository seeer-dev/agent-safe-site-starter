# Curatory Logo and curtain walkthrough

Source: selected linked worktree based on bac7c78fd6c5b4793e6705cefbdc7cabb56095bb
Revision: 1

## Shared static mark

- Rendered Curatory output at / loaded /assets/images/curatory-bloom-mark.svg in the shared header, footer, loader, and static chrome.
- Desktop and 390 px mobile checks showed the approved 02 花間 flower-and-butterfly mark. The dark Hero header uses the light treatment; the footer and loader use the same mark.
- The mounted mark has no SVG animation elements, and the checkout processing overlay no longer applies a pulse class.

## Standard-motion transition

- From /, the primary-header 全部商品 link entered the internal-navigation flow to /shop/.
- During coverage, page-curtain had curtain-cover with a non-zero vertical transform. The central brand computed animation-name: none and opacity: 1.
- The curtain displayed the flower-and-butterfly mark with 質選所 and CURATORY, then the destination route loaded and removed the reveal state.

## Motion preference

- navigate() checks prefers-reduced-motion before adding the curtain class and performs direct navigation in that branch.
- The Logo component and curtain brand have no independent animation rule.

## Build and render evidence

- npm run typecheck and npm run build passed for the Curatory theme.
- go run ./server/tools/render rendered 3 articles, 13 products, 6 categories, and 7 static pages using the Curatory theme.
- SCOPE_CHANGE_ID=curatory-bloom-mark-and-curtain go run ./server/tools/verify passed.
