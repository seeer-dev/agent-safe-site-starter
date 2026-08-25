# Beginner Starter Architecture Hardening Evidence

Change ID: sitecore-lessons-for-beginner-starter
Revision: 1
Status: Verifying

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | PR #1 CI run 32873109532 ran `go test ./server/tools/archcheck -count=1` successfully and then ran the manifest-driven real-repository scan with `archcheck: ok`; frontend contract checks and migration parity also passed before the expected Verifying-vs-Accepted spec gate stop. |
| REQ-002 | passed | AGENTS.md, skills/site/SKILL.md, and skills/site/references/architecture-boundaries.md explicitly preserve one site, one Go backend, static-first delivery, separate Vue admin, scoped islands, and forbid speculative multi-site/runtime platform machinery. |
| REQ-003 | passed | AGENTS.md and architecture-boundaries.md require consumer-owned typed ports wired in bootstrap, cohesive internal splits before top-level module proliferation, and existing UI/reference surfaces as the user-facing acceptance contract unless a reviewed design change supersedes them. |
| REQ-004 | passed | skills/site/SKILL.md retains propose <outcome> -> concise proposal review -> plain apply and states architecture machinery remains agent-owned; no new user lifecycle command was added. |
| AC-001 | passed | PR #1 CI run 32873109532 passed TestCheckArchitectureUsesConfiguredModuleRoot as part of `go test ./server/tools/archcheck -count=1`, proving an alternate configured module root drives cross-module detection without checker code changes. |
| AC-002 | passed | PR #1 CI run 32873109532 passed TestLoadImportPolicyFailsClosedOnInvalidPolicy, covering a missing required field, an unsupported rule key, and an unsafe repository root. |
| AC-003 | passed | The new architecture-boundaries reference and AGENTS.md explicitly prohibit speculative runtime site selection, provider/module registries, Composer/ResolvedPlan-style planning, service locators, runtime DI containers, and abstractions justified only by hypothetical future sites/providers. |
| AC-004 | passed | The site skill and architecture-boundaries reference directly specify consumer-owned typed ports plus bootstrap wiring, cohesion-first commerce growth, and preservation of checked-in UI routes, visible fields, actions, states, and navigation. |
| AC-005 | passed | The site skill still exposes only propose/review/apply for normal controlled work and explicitly keeps spec IDs, architecture choices, verification commands, and provider details behind the agent. |
