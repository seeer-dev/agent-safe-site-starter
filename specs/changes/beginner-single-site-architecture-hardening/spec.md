# Beginner Single-Site Architecture Hardening Specification

Change ID: beginner-single-site-architecture-hardening
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner requested a clean replacement branch and authorized merging the corrected starter-owned architecture hardening into main on 2026-08-26 after review.
Repository baseline: ade97a8d58dfffe1d2b61633d552dfffcb7ba3f6
Supersedes: none

## Outcome

The starter keeps its beginner-facing single-site product shape while making architecture drift mechanically harder. Import enforcement is driven by one checked-in, versioned policy; cross-module collaboration uses typed consumer-owned ports; module growth stays cohesion-first; and checked-in UI/reference surfaces remain the user-facing contract unless a reviewed design change supersedes them.

## Requirements

### REQ-001: Import rules have one enforceable source

`server/tools/archcheck` MUST read versioned import roots and hard deny semantics from checked-in `architecture.yaml` rather than keep a second hard-coded architecture map.

#### AC-001: Configured roots drive detection
- GIVEN an import policy whose module and platform roots differ from the repository defaults
- WHEN archcheck evaluates imports beneath those configured roots
- THEN it detects cross-module and platform-to-module violations from the configured policy without a checker code change

### REQ-002: Enforcement fails closed on invalid policy or coverage

The checker MUST reject unsupported version-1 rule values, missing or invalid protected roots, and scans that do not cover module or platform non-test Go files.

#### AC-002: Invalid roots or coverage cannot silently pass
- GIVEN an unsupported rule value, nonexistent protected root, or zero protected-file coverage
- WHEN archcheck validates the policy and repository
- THEN it exits with an error instead of reporting `archcheck: ok`

### REQ-003: The starter remains a beginner-facing single-site product

Agent guidance MUST preserve one site, one Go backend, static-first public delivery, a separate Vue admin SPA, and scoped Vue islands, and MUST NOT introduce speculative runtime platform machinery without an explicit product decision.

#### AC-003: Broad work cannot silently platformize the starter
- GIVEN an agent plans a broad feature, refactor, provider change, or new module
- WHEN it reads the repository architecture guidance
- THEN it finds explicit prohibitions against speculative multi-site selection, dynamic module selection, global service lookup, provider registries, and runtime DI containers

### REQ-004: Collaboration, module growth, UI contracts, and user workflow stay bounded

Cross-module synchronous behavior MUST use a consumer-owned typed port wired in bootstrap; large modules MUST split by cohesive internal seams before top-level proliferation; existing UI/reference surfaces MUST remain the user-facing acceptance contract unless a reviewed design change supersedes them; and the normal user lifecycle MUST remain propose-review-apply.

#### AC-004: Agent guidance names the bounded collaboration path
- GIVEN work crosses modules, expands a large module, or implements behavior behind an existing UI
- WHEN the agent reads the site guidance
- THEN it is directed to typed ports plus bootstrap wiring, cohesion-first splitting, UI-contract preservation, and the existing propose-review-apply workflow
