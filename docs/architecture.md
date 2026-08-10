# Architecture

The starter optimizes for one thing: a beginner describes a site, while the repository stays understandable to both a person and an agent.

## Golden path

```text
content changes
    -> database
    -> Go renderer
    -> dist/
    -> Cloudflare Pages

interactive request
    -> Go API
    -> module
    -> database / auth / R2 / Resend
```

Static pages do not need a request-time renderer. This keeps the public site cheap, cacheable, and easy to inspect.

## Dependency rules

`bootstrap` wires concrete dependencies. Modules own business behavior. Platform packages wrap vendor/infrastructure details.

```text
bootstrap -> modules -> platform abstractions/helpers
    |           |
    +---------> auth
    +---------> platform implementations

platform -X-> modules
module A -X-> module B
```

`server/tools/archcheck` enforces the two `-X->` rules with Go AST imports.

## Why no generic provider registry

Only the database has two deliberate drivers because local SQLite and production PostgreSQL solve a concrete usability problem. R2, Resend, and Supabase Auth each have one production adapter. Their small interfaces exist for isolation/testing, not runtime plugin selection.

## Why plain HTML/CSS/JS

The generated site is an artifact, not another server application. Add client JavaScript only around forms, carts, galleries, account controls, or other genuinely interactive areas. If interaction becomes large later, introduce a small island runtime without turning the entire site into a hydrated app.
