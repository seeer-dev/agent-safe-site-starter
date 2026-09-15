# Deployment Readiness — Evidence

Change ID: deployment-readiness
Revision: 1
Status: Verifying

| Gate | Status | Proof |
|---|---|---|
| REQ-001 | passed | Makefile: THEME ?= $(or $(SITE_THEME),minimal-cart). Verified in-container (golang:1.26-alpine + make): SITE_THEME=curatory make -n theme -> npm --prefix site/themes/curatory run build; unset -> minimal-cart. |
| REQ-002 | passed | admin/public/_redirects contains '/* /index.html 200'; npm --prefix admin run build:only produced admin/dist/_redirects verbatim. |
| REQ-003 | passed | Dockerfile adds 'RUN mkdir -p /app/var && chown app:app /app/var'. Container smoke: migrate -> 'migrations applied (sqlite)'; api -> /healthz 200, /api/products 200; APP_ENV=production without config -> 'AUTH_MODE=dev is forbidden in production'. Image 42.8MB. |
| REQ-004 | passed | docs/deployment-guide.html covers Supabase/R2/Resend/Railway/Pages/edge/ECPay with signup URLs, value tables, and a localStorage checklist. README Production shape section links it and mirrors environment-configuration.md ownership. Commands verified: make site, npm --prefix admin ci && run build:only, output dirs dist/ and admin/dist. |
| AC-001 | passed | SITE_THEME=curatory make -n theme -> site/themes/curatory build; unset -> minimal-cart (verified in-container). |
| AC-002 | passed | admin/dist/_redirects produced by build:only with '/* /index.html 200'. |
| AC-003 | passed | docker build + run: migrate applies 18 migrations as app user; api serves /healthz 200 and /api/products 200; production mode without config refuses to start. |
| AC-004 | passed | Every command/path/var in the guide cross-checked against Makefile, railway.toml, Dockerfile, .env.production.example, and docs/environment-configuration.md. |
