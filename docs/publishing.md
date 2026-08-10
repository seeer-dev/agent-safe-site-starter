# Publishing

Users should think in three concepts, not SSR/SSG/ISR terminology.

## Publish

Content is edited, then a full static render produces `dist/` and Pages deploys it.

## Auto publish

A CMS publish event triggers the same render/deploy workflow automatically. A Cloudflare Pages Deploy Hook is one simple trigger when the Pages build can read the production database.

## Scheduled publish

A scheduler triggers the same render/deploy workflow at a requested time. This works for scheduled posts, catalog refreshes, or periodic data output.

## Incremental publish — later

V0 deliberately renders the full site. If rendering becomes measurably expensive, add a content dependency graph such as:

```text
Article 123 -> article page
            -> article index
            -> homepage latest posts
            -> sitemap
            -> feed
```

Then regenerate affected outputs. Do not add request-time frontend SSR merely to obtain incremental publishing.
