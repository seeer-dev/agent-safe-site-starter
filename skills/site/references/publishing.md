# Publishing

Translate user language into these policies:

- "按發布才更新" -> Publish.
- "內容發布就更新網站" -> Auto publish.
- "某時間上線 / 每天更新" -> Scheduled publish.
- "網站很大，只重建受影響頁" -> Incremental publish, but only after scale proves full render is a problem.

The implementation remains data -> Go renderer -> `dist/` -> Cloudflare Pages. Do not introduce Nuxt/Next/SSR infrastructure for these policies.
