# Security Review Receipt — prelaunch-security-hardening rev 3

Date: 2026-09-22
Reviewer: implementer-side self-review; independent review pending holder acceptance (REQ-009 remains pending until then).

## What was reviewed

- `functions/api/[[path]].js`: caller-supplied `X-Edge-Client-Key` is deleted before the HMAC-SHA256 stamp is applied; the key is derived from `CF-Connecting-IP` + `EDGE_SECRET` and never logs or persists a raw address.
- `platform/abuse` + `bootstrap/abuse_guard.go`: fixed-window counting lives in `abuse_buckets` (SQLite/Postgres migration 019); over-quota → 429 + `Retry-After`; missing stamped identity under edge auth → 403; store failure → 503. No `X-Forwarded-For`/`CF-Connecting-IP` trust in Go.
- `platform/turnstile`: Siteverify with strict action + hostname match; production without a secret fails closed; provider outage → 503 without leaking provider detail; token field travels only in request bodies.
- `commerce` manual-test window: `MANUAL_TEST_CHECKOUT_UNTIL` parsed once at startup; empty/malformed/expired/over-cap disables; one predicate covers listing, quote, guest and member orders.
- `admin/scripts/security-headers.mjs`: `_headers` generated at production build; origin inputs strictly validated (scheme/userinfo/query/fragment/path/control characters); no wildcards or hardcoded installation hosts.
- `content/sanitize.go` + render boundary: bluemonday allowlist; sanitizer output is the only value cast to `template.HTML`; bounded output via truncate-then-resanitize.
- Curatory `store.ts` + islands: `RecentOrder` is metadata-only; legacy token fields stripped on load and re-saved de-identified; fresh-order token stays in `sessionStorage`.
- Dependency chain: vitest 5.0.1 / vite 7.3.6 / plugin-vue 6.0.9 / happy-dom override ≥20.8.9; `npm audit` = 0 findings.

## Findings

- No new critical/high issues found in the reviewed diff. The abuse guard intentionally fails closed; a misconfigured edge credential yields 403 rather than silent bypass — documented behavior.
- Known pre-existing gap (out of scope, recorded for follow-up): `admin/scripts/check-resource-contracts.mjs` rejects the tuple-format `opts` already committed on `origin/staging`; the script path is outside this change's `applies_to`.
- Turnstile hostname check depends on `PUBLIC_SITE_URL`/`TURNSTILE_EXPECTED_HOST` configuration matching the deployed origin; live check listed in the holder checklist.
