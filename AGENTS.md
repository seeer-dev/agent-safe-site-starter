# Agent Rules

Read `architecture.yaml` and `skills/site/SKILL.md` before broad changes.

## Hard boundaries

- Keep Cloudflare Pages static. Do not add Nuxt, Next.js, Pages Functions, or a second backend unless the user explicitly changes the architecture.
- Go is the only application backend.
- Browser code must not query PostgreSQL/Supabase Database directly.
- `server/internal/modules/<name>` must not import another module directly.
- `server/internal/platform` must not import business modules.
- Keep `auth.Principal` explicit at handler/service boundaries. Do not hide it in `context.Context`.
- Database changes require a SQLite migration and a PostgreSQL migration unless the feature explicitly drops one driver.
- Prefer portable SQL. Do not add PostgreSQL-only features silently.
- Public content defaults to static publish: CMS/data -> Go renderer -> `dist/` -> Cloudflare Pages.
- Add client JavaScript only for interaction that actually needs it.

## Change procedure

1. Identify the user's intent, not a technology label.
2. Read the closest file under `skills/site/references/`.
3. Inspect existing modules before creating new ones.
4. If CodeGraph is installed, use `codegraph explore` / impact queries before cross-cutting edits.
5. For a non-trivial task, create `.ai/scope.json` from `.ai/scope.example.json` and keep the allowed paths narrow.
6. Make the smallest coherent change.
7. Run `go run ./server/tools/verify`.
8. If publishing output changed, also run `go run ./server/tools/render` and inspect `dist/`.

Do not make architecture more generic for hypothetical future providers. Add a seam only when two real implementations exist or tests need one.
