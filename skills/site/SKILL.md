---
name: site
description: Route website changes and controlled-spec propose/apply/walkthrough workflows through this starter's architecture, safety, data, auth, media, publishing, and verification rules. Use when work affects the public site, admin, Go API, persistence, rendering, requirements, acceptance evidence, or multiple architectural regions.
---

# Site skill

Use this skill as the single entry point for website changes, implementation plans, and integration reviews. The user should describe outcomes in ordinary language; do not ask them to choose database adapters, rendering jargon, spec IDs, revisions, slices, or verification commands unless a real product decision requires it. For controlled work, keep the normal user interaction to `propose <outcome>`, a concise proposal summary, then plain `apply`; manage the SDD machinery internally.

This is a beginner-facing **single-site** starter, not a general site platform. Architecture should make the user's one site easier for an AI agent to evolve safely, not prepare for hypothetical future sites.

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
   - plan/review spanning multiple architectural regions -> `references/integration-planning.md`, `references/architecture-boundaries.md`, and `references/change-safety.md`
   - auth, money, PII, permissions, secrets, or another trust boundary -> use the available `security-review` skill in addition to the relevant site references before accepting the plan or implementation
   - broad/refactor/new module/provider/framework -> `references/architecture-boundaries.md` and `references/change-safety.md`
3. Inspect the existing module that owns the behavior before adding a module. If a module is large, split cohesive internals before proposing new top-level modules unless ownership/lifecycle/contracts clearly differ.
4. Prefer the repository's golden path. Do not introduce a new framework/provider to solve a local problem. Do not add multi-site runtime selection, Composer/`ResolvedPlan`-style planning, a dynamic module registry, a service locator, or a runtime DI container for hypothetical future sites.
5. When synchronous behavior crosses modules, keep module imports isolated: let the consumer define the smallest typed port and wire the provider adapter in bootstrap. Do not turn the port into a plugin system.
6. When a checked-in UI, mockup, route, or reference flow already defines the product surface, preserve its information architecture, visible fields, actions, states, and navigation unless a reviewed product/design change supersedes it. Backend ownership alone does not authorize a new UI surface.
7. If the change spans multiple areas, inspect impact first. Use CodeGraph when available; otherwise search imports/routes/tests manually.
8. For non-trivial edits in a clean/single-task worktree, create `.ai/scope.json` with a narrow allowlist (legacy mode, which does not provide cross-task attribution). For parallel delegated packets or dirty-primary worktrees, route work through selected linked-worktree mode: preflight active/dirty ownership across all worktrees and stop immediately on `applies_to` overlap; create one OS-temporary linked worktree per packet using the controlled change and baseline as authority; run `$env:SCOPE_CHANGE_ID='<change-id>'; go run ./server/tools/scopecheck` and `go run ./server/tools/verify` inside the linked root; forbid in-repository scratch files and secrets in reports; report the complete selected diff, gate results, and scope incidents; and in `finally`, remove the linked worktree and prune Git metadata. Selected mode is local-only and forbidden in CI.
9. Respect the selected workflow boundary and controlled-spec status. `Propose` changes only planning/control artifacts. For a non-trivial proposal, use `../expand-implementation/SKILL.md` to expand repository-grounded implementation packets into the existing `plan.md`, then return the outcome, boundaries, decisions, slices, risks, and proof strategy. Plain `apply` approves the sole current review-ready proposal and delivers all of its dependency-ordered slices; do not make the user select each slice.
10. During apply, maintain spec state, traceability, scope, targeted checks, required UX walkthroughs, and evidence internally. A definition, route, client helper, or authenticated branch is not integrated until a real entry point and identity/session producer can reach it. Keep affected evidence pending until authoritative success, empty, permission, and failure states are observed without fixture or browser-local fallback.
11. Treat implementer completion reports as provisional. Independently inspect the diff, replay the relevant commands, consumer paths, fresh generated output, and required walkthrough receipts before restoring passed evidence. Use strict evidence mode for new non-trivial controlled changes so evidence is bound to the current revision and required receipts.
12. Run `go run ./server/tools/speccheck` before claiming a spec is ready or a protected slice is complete, then run `go run ./server/tools/verify`.
13. If public output changed, build affected client assets, run `go run ./server/tools/render`, and inspect both source and fresh `dist/`. Production policy, privacy, payment, shipping, warranty, support, promotion, and brand claims require an approved source; a self-selected keyword search is not sufficient proof.

## Defaults the agent should choose without bothering the user

- Product shape: one site; do not invent multi-site runtime composition.
- Public informational content: static publish to Cloudflare Pages.
- Local database: SQLite.
- Production database: PostgreSQL.
- Production auth: Supabase Auth.
- File bytes: direct browser upload to R2 through a Go-generated presigned URL.
- Transactional email: Resend through Go.
- Small interaction: plain browser JS; no site-wide frontend framework.
- Full render before incremental render until actual scale justifies complexity.
- Cross-module synchronous collaboration: consumer-owned typed port wired in bootstrap.
- Large-module cleanup: cohesive internal split before top-level module proliferation.
- Existing UI/reference: preserve the approved surface before adding backend-driven UI.

## Stop conditions

Ask the user only when multiple current proposals make plain `apply` ambiguous, a discovery changes product behavior or cost materially (for example public vs private content, who may edit, payment semantics, retention rules, or offline support), a trust-boundary decision lacks authority, or a genuine implementation/verification blocker remains. Handle ordinary implementation gaps, artifact updates, slice ordering, architecture defaults, and validation without interrupting the user.
