# Content and pages

Public content flows from data to `server/internal/render` to `site/templates` to `dist/`.

When adding a new public content type:

1. Find or create the owning module under `server/internal/modules/`.
2. Add portable schema migrations for SQLite and PostgreSQL.
3. Add store/service behavior.
4. Add templates or renderer output only for pages that need publishing.
5. Keep browser JavaScript optional and local to interaction.

Do not add request-time SSR for content freshness. Read `publishing.md` instead.
