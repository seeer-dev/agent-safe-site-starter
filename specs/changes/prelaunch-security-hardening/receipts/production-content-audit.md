# Production Content Audit Receipt — AC-009

Date: 2026-09-22

## Scope scanned

`README.md`, `docs/**`, `skills/**`, `.env*.example`, `contracts/`, `functions/`, `server/`, `site/` sources, `admin/src`, `admin/scripts` — excluding `node_modules` and build output.

## Method and result

- 29 substitutions applied to `docs/deployment-guide.html` + `docs/environment-configuration.md`: Railway public origin, Railway workspace/project/service/deployment UUIDs (9 distinct), and the installation bucket name → `<railway-*>`, `<*-pages-deployment-id>`, `<r2-bucket-name>` placeholders.
- Residual scan for `[0-9a-f]{8}-[0-9a-f]{4}-…{12}` UUIDs, `up.railway.app` origins, `re_*`/`pub-*` provider IDs, and `r2.dev` hostnames: only generic placeholders (`xxx.up.railway.app`, `<railway-service>.up.railway.app`), test fixtures (`re_familymart`), and node_modules noise remain.
- Retained `agent-safe-site-starter.pages.dev` / `agent-safe-site-starter-admin.pages.dev` are already labelled in the documents as intentional public test/proof URLs for the open-source starter — permitted by REQ-008/D-002.
- No production domain, credential, database URI, private recipient, or deployment receipt value is committed.
- Operator instructions still state where each value is obtained and entered, using placeholders with the bilingual UI labels preserved.
