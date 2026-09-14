# Pre-Launch SEO + Reliability Hardening Plan

Change ID: prelaunch-seo-reliability
Revision: 1
Status: Verifying

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `specs/changes/prelaunch-seo-reliability/**`
- `site/themes/curatory/**`
- `server/internal/render/**`
- `server/internal/modules/commerce/**`
- `server/internal/bootstrap/**`
- `server/tools/render/**`
- `admin/**`

## Slices (dependency order)

### S01 — Crawler files in renderer

`renderSEOFiles(staging, in)` inside the staging pass emits
`robots.txt` (disallows /cart/ /checkout/ /track/ /order/, references
sitemap) and `sitemap.xml` (absolute `<loc>` from PublicSiteURL for
every indexable route).

Covers: REQ-001 (AC-001, AC-002)

### S02 — Social meta

`chromeFields` gains OGType/OGImage; `doc-head` emits og:*/twitter:*
on all pages; product pages pass `og:type=product` + absolutized
product image; default falls back to the hero asset.

Covers: REQ-002 (AC-003)

### S03 — Product JSON-LD

Renderer builds a schema.org Product JSON string per product page;
`product.html` emits it in an `application/ld+json` script (data block
— readable to crawlers; CSP governs execution only).

Covers: REQ-003 (AC-004)

### S04 — Notification async + admin retry

`notifyOrderEvent` call sites dispatch on a detached context
(`context.WithoutCancel` + goroutine) so order commit/status
transitions never wait on mail. New `RetryNotificationLog` service
method + `POST /api/admin/notification-logs/{id}/retry` route
(bootstrap) re-send the stored recipient/subject/body and insert a new
attempt row. Admin notification-logs resource gains a `retry` row
action shown on failed|skipped rows.

Covers: REQ-004 (AC-005, AC-006)

### S05 — Verification + evidence

Render, inspect dist/ outputs, API retry walkthrough, full verify
chain.

Covers: AC-007 + REQ-001–REQ-004 closure

## Traceability

| REQ / AC | Slice | Evidence |
|---|---|---|
| REQ-001 / AC-001, AC-002 | S01 | dist/sitemap.xml + robots.txt inspection |
| REQ-002 / AC-003 | S02 | rendered `<head>` inspection (home + product) |
| REQ-003 / AC-004 | S03 | dist product page JSON-LD parse |
| REQ-004 / AC-005, AC-006 | S04 | async send test + retry API walkthrough |
| AC-007 | S05 | verify chain output |
