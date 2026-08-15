# B01F static footer local validation (revision 12)

Date: 2026-08-15

## Implemented behavior

- The Go renderer passes the already composed, published footer/policy content blocks to all five public template families.
- Each template emits a semantic `#footer-static` navigation block. It contains only static policy links, or an honest empty state when no eligible blocks exist.
- `Footer.vue` hides that fallback only in `onMounted`; a failed island load leaves the static navigation usable.

## Mechanical evidence

- `go test ./server/internal/render -run 'TestRenderStaticFooter(ContentLinks|EmptyContent)$' -v -count=1` — PASS.
- `go test ./server/tools/internal/rendercompose -run TestComposeAndRenderProducesMinimalCartOutput -v -count=1` — PASS.
- `npm run build:check` in `site/themes/minimal-cart` — PASS (Vite emitted its existing chunk-size warning only).
- `go test ./server/...` — PASS.
- `go run ./server/tools/render` — PASS. With no current eligible content, fresh `dist/index.html` contains `#footer-static`, the honest empty state, and the Footer island mount point.

## Local HTTP replay and cleanup

The local development server was started in development-auth mode. One exact temporary content row with key `policy.b01f-acceptance` was created, approved with a future expiry, published, and rendered. Static HTTP inspection observed a 200 home page containing its `/content/policy.b01f-acceptance/` link and a 200 direct content route containing the temporary body. The same exact row was then deleted, the renderer was run again, the public API returned no matching row, and `dist/content/policy.b01f-acceptance/` no longer existed. The locally started API and site listeners were stopped; no repository scratch file was created.

## Browser limitation

The controlled Chrome browser denied access to `http://localhost:4173/` because its browser security check was unavailable. No bypass was used. Consequently this receipt does **not** prove a JavaScript-disabled click interaction or a successful mounted-DOM `#footer-static[hidden]` observation. AC-012 remains pending until those two browser observations are captured in an environment that permits local-host browser control.
