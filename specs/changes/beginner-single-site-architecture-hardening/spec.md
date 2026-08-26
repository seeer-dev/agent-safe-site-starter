# Beginner Single-Site Architecture Hardening Specification

Change ID: beginner-single-site-architecture-hardening
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner explicitly requested a clean replacement branch and authorized merging the corrected architecture hardening into main on 2026-08-26. Targeted PR CI then passed before acceptance.
Repository baseline: ade97a8d58dfffe1d2b61633d552dfffcb7ba3f6
Supersedes: none

## Outcome

The starter keeps its beginner-facing single-site product shape while making its architecture boundaries harder for coding agents to drift across. Import enforcement is driven by one checked-in, versioned policy; cross-module collaboration uses typed consumer-owned ports; module growth stays cohesion-first; and checked-in UI/reference surfaces remain the user-facing contract unless a reviewed design change supersedes them.

The user-facing workflow remains `propose <outcome>` -> proposal review -> plain `apply`.

## Scope

In scope:

- Make the import-boundary portion of `architecture.yaml` authoritative for `archcheck`.
- Make the enforcement schema versioned and non-disableable within version 1.
- Fail closed when configured roots are invalid, absent, overlapping, or match no protected Go files.
- Run archcheck regression tests in CI before the real repository scan.
- Add starter-owned single-site, typed-port, cohesion-first, and UI-first architecture guidance.

Out of scope:

- Multi-site runtime composition.
- Dynamic module selection or plugin loading.
- Global service lookup or runtime DI containers.
- Splitting `commerce` business code in this change.
- Replacing the current HTTP/OpenAPI implementation.
- Any new user-facing lifecycle command.

## Requirements

### REQ-001: Import rules have one enforceable source

`server/tools/archcheck` MUST read versioned import roots and deny semantics from checked-in `architecture.yaml` rather than keep a second hard-coded architecture map.

#### AC-001
A configured module root changes cross-module detection without checker code changes.

#### AC-002
Version 1 MUST accept only deny semantics; an unsupported version or attempt to disable a hard rule MUST fail.

### REQ-002: Enforcement fails closed on coverage drift

The checker MUST reject missing/non-directory roots, non-strict or overlapping protected roots, and scans that discover no module or platform non-test Go files.

#### AC-003
A nonexistent module/platform root fails before reporting `archcheck: ok`.

#### AC-004
Zero protected-file coverage fails instead of silently passing.

### REQ-003: The starter remains a beginner-facing single-site product

Agent guidance MUST preserve one site, one Go backend, static-first delivery, a separate Vue admin SPA, and scoped Vue islands. It MUST NOT add speculative multi-site/runtime platform machinery without an explicit product decision.

### REQ-004: Collaboration and growth boundaries stay explicit

Cross-module synchronous behavior uses a consumer-owned typed port wired in bootstrap; large modules split by cohesive internal seams before top-level proliferation; existing UI/reference surfaces remain the user-facing acceptance contract unless a reviewed product/design change supersedes them.

### REQ-005: Governance complexity stays behind the agent

Normal controlled work remains propose/review/apply and does not require the user to operate architecture manifests, change IDs, revisions, evidence IDs, or validation commands.
