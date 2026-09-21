# Curatory 花間品牌標記與頁面遮罩

Change ID: curatory-bloom-mark-and-curtain
Revision: 3
Status: Accepted
Decision authority: Repository owner/user
Approval basis: On 2026-09-20, the user selected the approved flower-and-butterfly 02 direction, approved the restrained colour accent, asked to keep shared Logo locations still, then identified http://127.0.0.1:4187/curatory-logo-motion.html as the required animation reference for the Logo inside the page-transition curtain.
Repository baseline: bac7c78fd6c5b4793e6705cefbdc7cabb56095bb
Supersedes: curatory-storefront-port

## Outcome

The storefront presents the approved 02 花間 butterfly-and-flower mark at the shared Logo locations. Its page-transition mask follows the supplied animation reference: a warm-ink curtain, a cream-and-terracotta lockup, butterfly wing folds, flower sway, and the accompanying brand line. Shared Logo locations remain static. This change supersedes only the brand-mark and transition portion of the earlier storefront port.

## Scope

In scope:

- Shared Vue Logo mark, loader, static HTML fallbacks, and the rendered page-transition curtain.
- A self-hosted static vector mark plus a dedicated animated curtain variant copied into the published static asset path.
- Existing transition interception and reduced-motion behavior.

Out of scope:

- New routes, data sources, payments, or administration behavior.
- Butterfly, flower, or wordmark motion outside the page-transition curtain.
- Changing existing page content or navigation rules.

## Requirements

### REQ-001: Shared static 花間 mark

The storefront SHALL use the approved butterfly-and-flower Logo mark in the interactive shared chrome and the static fallback chrome. The mark SHALL include the restrained terracotta flower accent and remain stationary at every location.

#### AC-001: Header, footer, loader, and rendered fallback agree

- GIVEN a rendered Curatory page with and without Vue islands mounted
- WHEN a visitor sees the shared Logo locations
- THEN they see the same flower-and-butterfly mark with no independently animated Logo path or element.

### REQ-002: Animated-logo transition curtain

The existing internal-navigation curtain SHALL follow the supplied motion reference: 0.52-second cover, 0.68-second reveal, a cream-and-terracotta animated mark with independently folding wings and swaying flower, and the reference brand line. Motion preference SHALL continue to bypass extended transition motion and use the static mark.

#### AC-002: Standard-motion navigation covers then reveals

- GIVEN a visitor follows an eligible internal link with standard motion enabled
- WHEN navigation starts
- THEN the warm-ink curtain covers the current page before navigation and lifts from the next page with the reference butterfly-and-flower Logo animation and brand line appearing only inside that curtain.

#### AC-003: Reduced-motion navigation remains concise

- GIVEN a visitor requests reduced motion
- WHEN they follow an internal link
- THEN navigation occurs without the extended curtain sequence and the curtain mark remains visually static.
