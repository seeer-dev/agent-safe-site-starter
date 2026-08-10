# Publish workflow

V0 uses full-site rendering.

```text
published database content
  -> go run ./server/tools/render
  -> dist/
  -> Cloudflare Pages
```

Trigger options:

- Git push / Pages build.
- CMS publish -> Pages Deploy Hook.
- Scheduler -> same Deploy Hook/build.
- AI/CI machine -> `go run ./server/tools/publish` for Direct Upload.

Do not add incremental rendering until full render duration or page count becomes an observed problem.
