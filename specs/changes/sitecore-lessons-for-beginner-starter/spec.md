# Beginner Starter Architecture Hardening Specification

Change ID: sitecore-lessons-for-beginner-starter
Revision: 1
Status: Verifying
Decision authority: Repository owner/user
Approval basis: Repository owner explicitly approved implementation on 2026-08-26 after clarifying that this repository is a beginner-facing single-site starter whose AI machinery must stay hidden behind the propose/apply workflow.
Repository baseline: ade97a8d58dfffe1d2b61633d552dfffcb7ba3f6
Supersedes: none

## Outcome

The starter keeps its simple single-site product shape while borrowing only the SiteCore lessons that reduce AI implementation drift: one enforceable architecture source, explicit typed collaboration boundaries, a bounded abstraction budget, cohesive module growth, and UI-first integration when a user-visible surface already exists.

The non-technical user still describes the desired site, reviews a proposal, and says `apply`. No platform terminology or additional lifecycle command is exposed to them.

## Scope

In scope:

- Make the import-boundary portion of `architecture.yaml` authoritative for `archcheck`.
- Make `archcheck` fail closed when that policy is missing, malformed, unsupported, or unsafe.
- Run the archcheck policy regression tests in repository CI before the runtime import scan.
- Add agent-facing single-site, typed-port, cohesion, and UI-first architecture guardrails.
- Preserve the existing static-first, one-Go-backend, Vue-admin/scoped-island shape.

Out of scope:

- SiteCore Composer, `ResolvedPlan`, selected-site runtime composition, multi-site registries, or dynamic module selection.
- A runtime DI container or service locator.
- Splitting `commerce` into multiple top-level modules in this change.
- Replacing the current OpenAPI/HTTP implementation in this change.
- Any new user-facing workflow command.

## Requirements

### REQ-001: Architecture import rules have one enforceable source

`server/tools/archcheck` MUST read the import-boundary roots and deny rules from the checked-in `architecture.yaml` rather than maintain a second hard-coded architecture map, and its manifest-policy regression tests MUST run in repository CI.

#### AC-001: A configured module root changes what archcheck enforces

- GIVEN an architecture policy whose module root differs from `server/internal/modules`
- WHEN the checker evaluates a cross-module import under that configured root
- THEN it MUST detect the violation from the configured policy without a code change to the checker.

#### AC-002: Missing or malformed enforcement fails closed

- GIVEN `architecture.yaml` is missing a required import-rule field, contains an unsupported rule key, or names an unsafe repository root
- WHEN `archcheck` starts
- THEN it MUST fail with a diagnostic instead of silently falling back to hard-coded defaults or skipping the check.

### REQ-002: The starter remains a beginner-facing single-site product

Agent guidance MUST preserve one site, one Go backend, static-first public delivery, the separate Vue admin, and scoped islands as the default product shape; it MUST NOT introduce SiteCore-style multi-site composition or platform machinery without an explicit product decision that changes the starter's purpose.

#### AC-003: Broad work cannot silently platformize the starter

- GIVEN an agent is planning a broad feature, refactor, or provider change
- WHEN it reads repository architecture guidance
- THEN it MUST find explicit prohibitions against speculative multi-site runtime selection, Composer/`ResolvedPlan`, service locators, dynamic module registries, and abstractions justified only by hypothetical future sites/providers.

### REQ-003: Collaboration and growth boundaries stay explicit

Cross-module synchronous collaboration MUST prefer a consumer-owned typed interface wired in bootstrap; a large module MUST be split by cohesive internal seams before creating new top-level modules; and an existing UI/reference surface MUST remain the user-facing acceptance contract unless a reviewed product/design change supersedes it.

#### AC-004: Agent guidance names the safe collaboration and cohesion path

- GIVEN work crosses module boundaries or expands an already-large module
- WHEN the agent reads the relevant architecture guidance
- THEN it MUST be directed to a consumer-owned typed port plus bootstrap wiring, internal cohesion before module proliferation, and preservation of existing routes/fields/actions when a UI/reference already defines them.

### REQ-004: Governance complexity stays behind the agent

The repository MUST preserve `propose <outcome>` -> proposal summary -> plain `apply` as the normal beginner workflow and MUST NOT require the user to understand architecture manifests, module ports, controlled-change IDs, revisions, evidence IDs, or validation commands.

#### AC-005: No new beginner command is introduced

- GIVEN a non-technical user requests a normal site change
- WHEN the site skill describes the interaction
- THEN the visible lifecycle MUST remain propose/review/apply and all architecture-checking machinery MUST remain agent-owned.

## Amendments

None.
