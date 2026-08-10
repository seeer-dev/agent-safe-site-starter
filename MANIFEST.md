# Example package contents

This v0 package demonstrates the architecture, not a finished CMS product.

## Runnable path

- `server/tools/dev`: migrate -> seed -> render -> API + local static server
- SQLite is the no-account local default
- production adapters are PostgreSQL, Supabase Auth, R2, and Resend
- `server/tools/render`: full static publish into `dist/`
- `server/tools/publish`: render + Cloudflare Pages Direct Upload

## Vertical examples

- `content`: admin-only content write + published reads + static page rendering
- `contact`: public form persistence + optional Resend notification
- `media`: authenticated R2 presigned image upload

## AI guardrails

- one `skills/site/SKILL.md` intent router
- `architecture.yaml` ownership map
- `AGENTS.md` hard rules
- optional CodeGraph impact discovery
- `.ai/scope.json` per-task allowlist
- `archcheck`, `scopecheck`, tests and vet

## Deliberately absent

- Nuxt / Next / frontend SSR runtime
- Pages Functions or a second backend
- generic provider registry / DI container / plugin framework
- admin frontend
- request-time ISR
- incremental content graph (add only when full render is measurably expensive)
