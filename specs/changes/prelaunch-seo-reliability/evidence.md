# Evidence — prelaunch-seo-reliability

| ID | Status | Proof |
|----|--------|-------|
| REQ-001 | passed | go run ./server/tools/render emits dist/robots.txt + dist/sitemap.xml in the staging pass (2026-09-14 run: 3 articles, 12 products, 5 categories, 1 content page, 7 static pages). |
| REQ-002 | passed | dist/index.html head carries og:site_name/type/title/description/url/image/image:alt plus twitter:card/title/description/image; product page emits og:type=product with product photo og:image. |
| REQ-003 | passed | dist/products/glazed-mug-grey/index.html embeds application/ld+json with @type=Product, name, description, brand, offers (TWD 480, InStock), image list, absolute url. |
| REQ-004 | passed | service_checkout.go:610 and service_orders.go:208 dispatch via go s.notifyOrderEvent(context.WithoutCancel(ctx), ...); POST /api/admin/notification-logs/{id}/retry verified live (401 unauthenticated, 204 authorized, new attempt row, original preserved). |
| AC-001 | passed | dist/sitemap.xml holds 25 url entries: /, /about/, /shop/, 5 categories, 12 products, /news/ + 3 articles, /content/footer.about/; zero cart/checkout/track/order matches. |
| AC-002 | passed | dist/robots.txt disallows /cart/ /checkout/ /track/ /order/ and references the absolute sitemap URL. |
| AC-003 | passed | Home: og:type=website, og:image=/assets/images/hero.jpg absolutized. Product glazed-mug-grey: og:type=product, og:image=/assets/images/p01.png, og:image:alt=手作釉彩馬克杯・霧灰. |
| AC-004 | passed | Rendered JSON-LD parses as valid JSON: @type=Product, brand=質選所, offers price=480 priceCurrency=TWD availability=InStock, image and url absolute. |
| AC-005 | passed | TestOrderPlacedNotificationDoesNotBlockCheckout: gated fakeSender blocks Send; CreateOrder returned inside 2s deadline; log row landed after gate release (sent). Trigger observed: synchronous-dispatch mutation fails at 'CreateOrder blocked on sender' (2.04s). |
| AC-006 | passed | TestRetryNotificationLogCreatesNewAttempt and TestRetryNotificationLogRequiresAdmin pass; no-op-retry mutation fails at 'logs = 1, want 2'. Live e2e: retry unauthenticated 401, authorized 204, new attempt row d6ef3f53 (sent), original 007e5379 preserved. |
| AC-007 | passed | go run ./server/tools/verify 2026-09-14: archcheck ok, migration-parity ok (18), speccheck ok (33 specs, 15 protected), scopecheck ok (19 files), go test ./... all ok, go test commerce/staff/media -count=10 ok, vet ok — 'verify: ok'. |
