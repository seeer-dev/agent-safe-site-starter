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
- Core site assets default to plain static JavaScript/CSS. Vue islands are permitted only inside the selected interactive theme (`site/themes/<theme>/`) where component interaction requires them. The admin interface (`admin/`) is a separate Vue SPA.
- Add client JavaScript only for interaction that actually needs it.
- Protected implementation and governance changes require one valid controlled change under `specs/changes/<change-id>/`; a Draft or Superseded spec never authorizes product edits.
- Treat `server/tools/speccheck`, `server/tools/verify`, CI workflows, `AGENTS.md`, `architecture.yaml`, and `skills/` as protected governance surfaces. Changes to the gate must pass the gate.
- Keep the normal controlled-change interface to `propose <outcome>` -> proposal summary -> plain `apply`. The agent owns change IDs, revisions, statuses, REQ/AC traceability, slice ordering, required walkthroughs, and verification; do not ask the user to operate that machinery.

## Change procedure

1. Identify the user's intent, not a technology label.
2. Read the closest file under `skills/site/references/`.
3. Inspect existing modules before creating new ones.
4. If CodeGraph is installed, use `codegraph explore` / impact queries before cross-cutting edits.
5. On `propose`, keep product code read-only, create or update the controlled artifacts, and return a concise review summary rather than internal IDs or commands.
6. On plain `apply`, approve the sole current review-ready proposal, execute all in-scope slices in dependency order, and maintain its status, scope, REQ/AC traceability, and evidence without asking the user to name a revision or slice.
7. Before editing a protected path, run `go run ./server/tools/speccheck` for the selected controlled change.
8. For a non-trivial task, create `.ai/scope.json` from `.ai/scope.example.json` and keep the allowed paths narrow.
9. Make the smallest coherent change and run the targeted checks, required user walkthrough, and `go run ./server/tools/verify` before completion.
10. If publishing output changed, also run `go run ./server/tools/render` and inspect `dist/`.

Interrupt the two-step flow only when multiple current proposals make plain `apply` ambiguous, a newly discovered decision materially changes product behavior, cost, permissions, data handling, or another trust boundary, or a genuine blocker prevents implementation or verification.

Do not make architecture more generic for hypothetical future providers. Add a seam only when two real implementations exist or tests need one.
