# Publish workflow

V0 uses full-site rendering.

```text
published database content
  -> go run ./server/tools/render
  -> dist/
  -> Cloudflare Pages
```

Trigger options:

- Git push -> Pages build (`make site` renders from the production database).
- `go run ./server/tools/publish` (CI or manual) -> renders `dist/` as a
  pre-check, then POSTs the Pages Deploy Hook to trigger that same Git
  rebuild. It is trigger-only; no dist upload.
- Scheduler -> same Deploy Hook/build.

Admin/CMS publish actions only mark content published in the database.
They do not invoke render or the hook — a trigger above must run before
new content appears on the static site.

Do not add incremental rendering until full render duration or page count becomes an observed problem.
