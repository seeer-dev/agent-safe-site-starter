# Site skill

Use this skill as the single entry point for website changes. The user should describe outcomes in ordinary language; do not ask them to choose database adapters, rendering jargon, or vendor APIs unless a real product decision requires it.

## Goal

Translate user intent into the smallest safe change in this starter.

## Procedure

1. Read `architecture.yaml` and `AGENTS.md`.
2. Classify the request by user outcome:
   - public content/page -> `references/content-and-pages.md`
   - data/model/form -> `references/data-and-forms.md`
   - login/permissions -> `references/auth.md`
   - images/files -> `references/media.md`
   - notifications -> `references/email.md`
   - publish/deploy/freshness -> `references/publishing.md`
   - broad/refactor/debug -> `references/change-safety.md`
3. Inspect the existing module that owns the behavior before adding a module.
4. Prefer the repository's golden path. Do not introduce a new framework/provider to solve a local problem.
5. If the change spans multiple areas, inspect impact first. Use CodeGraph when available; otherwise search imports/routes/tests manually.
6. For non-trivial edits, create `.ai/scope.json` with a narrow allowlist.
7. Implement one coherent vertical slice: migration -> store/service -> handler/contract -> render/client behavior only where required.
8. Run `go run ./server/tools/verify`.
9. If public output changed, run `go run ./server/tools/render` and inspect `dist/`.

## Defaults the agent should choose without bothering the user

- Public informational content: static publish to Cloudflare Pages.
- Local database: SQLite.
- Production database: PostgreSQL.
- Production auth: Supabase Auth.
- File bytes: direct browser upload to R2 through a Go-generated presigned URL.
- Transactional email: Resend through Go.
- Small interaction: plain browser JS; no site-wide frontend framework.
- Full render before incremental render until actual scale justifies complexity.

## Stop conditions

Ask the user only when the answer changes product behavior or cost materially, for example: public vs private content, who may edit, payment semantics, retention rules, or whether a feature must work offline. Do not ask technical-choice questions the architecture already answers.
