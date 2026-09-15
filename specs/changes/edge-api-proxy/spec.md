# Edge API Proxy Specification

Change ID: edge-api-proxy
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Owner direction 2026-09-15 — "放在 cf pages 層其實好切後端，我傾向改成放那裡". Explicit architecture change approving a Pages Functions proxy seam, per the AGENTS.md escape clause.
Repository baseline: 10f805e18e394e7065854dfbceb327114e0d4564
Supersedes: none

## Outcome

Browser traffic never names the Railway origin. Both Cloudflare Pages
projects (storefront + admin) proxy `/api/*` through a Pages Function
that injects the edge secret itself, so the Transform Rule step
disappears and every browser call becomes same-origin. The Go backend,
its routes, and the EDGE_SECRET check on the origin stay exactly as
they are — the proxy is a dumb pass-through, not a second backend.

## Scope

In scope:

- `functions/api/[[path]].js` — the proxy Function (repo root so both
  Pages projects pick it up).
- `AGENTS.md` — record the approved Pages Functions exception, scoped
  to this seam.
- `.env.production.example`, `docs/environment-configuration.md` — new
  Pages-side vars (`API_ORIGIN`, `EDGE_SECRET`), revised
  `PUBLIC_API_BASE` / `ADMIN_API_BASE` guidance.
- `README.md`, `docs/deployment-guide.html` — topology, setup steps,
  checklist.

Out of scope:

- Any Go code change. CORS, edge-secret verification, and route
  registration are unchanged.
- Pages Functions beyond the single `/api/*` catch-all route.

## Requirements

### REQ-001: `/api/*` proxies to the configured origin

A Pages Function at `functions/api/[[path]].js` MUST forward every
method, path, query, body, and header under `/api/*` to
`env.API_ORIGIN` and stream the upstream response back. When
`API_ORIGIN` is unset it MUST fail closed (5xx), never fall through to
static assets.

#### AC-001: Pass-through fidelity
- GIVEN the Function running under `wrangler pages dev`
- WHEN GET /api/products is requested
- THEN the response is the upstream body/status, and POST bodies and
  Authorization headers arrive intact.

### REQ-002: The Function injects the edge credential

Every proxied request MUST carry `X-Edge-Secret: <env.EDGE_SECRET>` so
the origin's existing hop check passes; requests that never traverse
the Function still get 403. The Function MUST NOT forward the site's
`Cookie` header upstream — the API never reads cookies and first-party
cookies must not leak to the origin hop.

#### AC-002: Injection verified at the origin
- GIVEN a local echo upstream
- WHEN a request passes through the Function
- THEN the upstream observes `X-Edge-Secret` with the configured value
  and observes no `Cookie` header.

### REQ-003: Docs and config match the new topology

README, deployment guide, env examples, and the env ownership doc MUST
describe: `PUBLIC_API_BASE=https://<site origin>` (Pages build + Railway,
the latter also feeds ECPay ReturnURL via `site.com/api/...`), unset
`ADMIN_API_BASE` defaulting to same-origin `/api`, `API_ORIGIN` +
`EDGE_SECRET` as Pages project vars on BOTH projects, and the removal
of the Transform Rule step. The checklist MUST gain a
"same-origin `/api` proxied" item in place of the Transform Rule item.

#### AC-003: No stale instructions
- GIVEN the updated docs
- WHEN searching for `api.` subdomain and Transform Rule steps
- THEN the guide no longer instructs operators to configure them, and
  the topology diagram shows Pages → Function → Railway.

### REQ-004: Boundary amendment is explicit and scoped

`AGENTS.md` MUST record that Pages Functions are permitted only for the
`/api/*` edge proxy defined here — not a general opening for edge
compute.

#### AC-004: Narrow exception text
- GIVEN AGENTS.md
- WHEN the hard-boundary list is read
- THEN it names this spec as the sole approved Pages Functions use.
