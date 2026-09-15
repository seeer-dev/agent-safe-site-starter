# Edge API Proxy Plan

Change ID: edge-api-proxy
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `specs/changes/edge-api-proxy/**`
- `functions/**`
- `.gitignore`
- `AGENTS.md`
- `README.md`
- `.env.production.example`
- `.env.development.example`

`docs/**` changes ride under the accepted `curatory-storefront-port`
ownership in the branch-wide comparison; this spec deliberately does not
claim `docs/**` so it cannot double-own it.

## Slices

### S01 — Proxy Function + boundary note

`functions/api/[[path]].js` dumb pass-through injecting `X-Edge-Secret`,
stripping `Cookie`, failing closed without `API_ORIGIN`. AGENTS.md gains
the scoped exception.

Covers: REQ-001, REQ-002, REQ-004 (AC-001, AC-002, AC-004)

### S02 — Config + docs alignment

Env examples, environment-configuration.md, README, deployment guide,
checklist.

Covers: REQ-003 (AC-003)

## Traceability

| REQ / AC | Slice | Evidence |
|---|---|---|
| REQ-001 / AC-001 | S01 | wrangler pages dev + curl through proxy |
| REQ-002 / AC-002 | S01 | echo upstream observes secret header, no cookie |
| REQ-003 / AC-003 | S02 | doc grep: no api./Transform-Rule instructions remain |
| REQ-004 / AC-004 | S01 | AGENTS.md diff names this spec only |
