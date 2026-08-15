---
name: site
description: Route website changes and controlled-spec propose/apply/walkthrough workflows through this starter's architecture, safety, data, auth, media, publishing, and verification rules. Use when work affects the public site, admin, Go API, persistence, rendering, requirements, acceptance evidence, or multiple architectural regions.
---

# Site skill

Use this skill as the single entry point for website changes, implementation plans, and integration reviews. The user should describe outcomes in ordinary language; do not ask them to choose database adapters, rendering jargon, spec IDs, revisions, slices, or verification commands unless a real product decision requires it. For controlled work, keep the normal user interaction to `propose <outcome>`, a concise proposal summary, then plain `apply`; manage the SDD machinery internally.

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
   - workflow guidance or lifecycle verbs (`propose`, `apply`, `walkthrough`) -> `references/workflow-usage.md`
   - controlled specification, SDD, requirements, acceptance, or contract changes -> `references/spec-driven-delivery.md` and `references/change-safety.md`
   - non-trivial proposal preparation or a request for implementation-ready detail -> read `../expand-implementation/SKILL.md` and its required blueprint reference in full
   - proposal/application lifecycle or gaps discovered during implementation -> `references/delivery-lifecycle.md`, `references/spec-driven-delivery.md`, and `references/change-safety.md`
   - UI surfaces, journeys, states, or experience contracts -> `references/ui-surfaces.md`
   - runtime UX acceptance or gap discovery -> `references/user-walkthrough.md`, `references/ui-surfaces.md`, and `references/spec-driven-delivery.md`
   - plan/review spanning multiple architectural regions -> `references/integration-planning.md` and `references/change-safety.md`
   - auth, money, PII, permissions, secrets, or another trust boundary -> use the available `security-review` skill in addition to the relevant site references before accepting the plan or implementation
   - broad/refactor/debug -> `references/change-safety.md`
3. Inspect the existing module that owns the behavior before adding a module.
4. Prefer the repository's golden path. Do not introduce a new framework/provider to solve a local problem.
5. If the change spans multiple areas, inspect impact first. Use CodeGraph when available; otherwise search imports/routes/tests manually.
6. For non-trivial edits, create `.ai/scope.json` with a narrow allowlist.
7. Respect the selected workflow boundary and controlled-spec status. `Propose` changes only planning/control artifacts. For a non-trivial proposal, use `../expand-implementation/SKILL.md` to expand repository-grounded implementation packets into the existing `plan.md`, then return the outcome, boundaries, decisions, slices, risks, and proof strategy. Plain `apply` approves the sole current review-ready proposal and delivers all of its dependency-ordered slices; do not make the user select each slice.
8. During apply, maintain spec state, traceability, scope, targeted checks, required UX walkthroughs, and evidence internally. A definition, route, client helper, or authenticated branch is not integrated until a real entry point and identity/session producer can reach it. Keep affected evidence pending until authoritative success, empty, permission, and failure states are observed without fixture or browser-local fallback.
9. Treat implementer completion reports as provisional. Independently inspect the diff, replay the relevant commands, consumer paths, fresh generated output, and required walkthrough receipts before restoring passed evidence. Use strict evidence mode for new non-trivial controlled changes so evidence is bound to the current revision and required receipts.
10. Run `go run ./server/tools/speccheck` before claiming a spec is ready or a protected slice is complete, then run `go run ./server/tools/verify`.
11. If public output changed, build affected client assets, run `go run ./server/tools/render`, and inspect both source and fresh `dist/`. Production policy, privacy, payment, shipping, warranty, support, promotion, and brand claims require an approved source; a self-selected keyword search is not sufficient proof.

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

Ask the user only when multiple current proposals make plain `apply` ambiguous, a discovery changes product behavior or cost materially (for example public vs private content, who may edit, payment semantics, retention rules, or offline support), a trust-boundary decision lacks authority, or a genuine implementation/verification blocker remains. Handle ordinary implementation gaps, artifact updates, slice ordering, and validation without interrupting the user.
