# Deployment

## Railway API

The repository includes `Dockerfile` and `railway.toml`.

Production variables normally include:

```text
APP_ENV=production
DB_DRIVER=postgres
DATABASE_URL=...
AUTH_MODE=supabase
SUPABASE_URL=...
SUPABASE_PUBLISHABLE_KEY=...
SITE_ORIGIN=https://www.example.com
R2_*=...
RESEND_API_KEY=...
RESEND_FROM=...
CONTACT_NOTIFY_TO=...
```

Railway runs `migrate` before starting `api`.

## Cloudflare Pages

For Git integration:

```text
Build command:      go run ./server/tools/render
Build output:       dist
GO_VERSION:         1.26.5 (or another supported 1.26 release)
DB_DRIVER:          postgres
DATABASE_URL:       production read-capable URL
PUBLIC_SITE_URL:    production site URL
PUBLIC_API_BASE:    Railway API URL
```

A CMS publish action may trigger a Pages Deploy Hook. For direct upload instead, an agent/CI can run `go run ./server/tools/publish` after setting `CF_PAGES_PROJECT` and Cloudflare credentials used by Wrangler.

Keep database credentials server/build-side. Never render them into `dist/`.
