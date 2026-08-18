# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-08-18. AC-001 was verified adversarially: the theme bundle was removed, the renderer alone was observed failing closed, and the new target then succeeded from that same state.

Note: `make` is not on PATH in this environment; `mingw32-make` was used to invoke the target itself, not merely its constituent steps.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Makefile gains a theme target that builds the Vue islands bundle and a site target that runs theme then render. The existing render target keeps its meaning, so anything already building the theme separately is unaffected. |
| REQ-002 | passed | README.md now names make site as the Cloudflare Pages Git build command, states that the theme bundle is git-ignored so a clean checkout has none, and records that the build image needs Node as well as Go. |
| AC-001 | passed | Verified with the theme dist directory removed. go run ./server/tools/render alone failed closed with its existing diagnostic (render failed (dist preserved): theme dist directory missing), exit status 1. mingw32-make site then succeeded from the same absent state, building site/themes/minimal-cart/dist/islands.js and rendering 5 products and 4 categories into dist/. The renderer guard was not weakened. |
| AC-002 | passed | README.md:70 names make site rather than the renderer alone, and states the ordering dependency, the git-ignored bundle, and the Node requirement explicitly rather than leaving them implicit. |
