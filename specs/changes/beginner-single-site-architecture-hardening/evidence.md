# Beginner Single-Site Architecture Hardening Evidence

Change ID: beginner-single-site-architecture-hardening
Revision: 1
Status: Accepted

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | PR #2 CI run 32968611776 passed the archcheck regression tests and manifest-driven repository scan, proving architecture.yaml drives the versioned import roots and hard deny semantics. |
| REQ-002 | passed | PR #2 CI run 32968611776 passed fail-closed tests for unsupported policy values, missing protected roots, and zero module or platform Go-file coverage. |
| REQ-003 | passed | AGENTS.md, the site skill, and architecture-boundaries.md preserve one site, one Go backend, static-first delivery, separate Vue admin, scoped islands, and starter-owned generic terminology. |
| REQ-004 | passed | Agent guidance requires consumer-owned typed ports wired in bootstrap, cohesion-first module growth, preservation of checked-in UI contracts, and the existing propose-review-apply user workflow. |
| AC-001 | passed | TestCheckArchitectureUsesConfiguredRoots passed in PR #2 CI run 32968611776 and detected cross-module plus platform-to-module imports under alternate configured roots. |
| AC-002 | passed | TestLoadImportPolicyFailsClosed, TestCheckArchitectureFailsOnMissingRoots, and TestCoverageFailsClosed passed in PR #2 CI run 32968611776. |
| AC-003 | passed | The checked-in architecture guidance explicitly prohibits speculative runtime multi-site selection, dynamic module selection, global service lookup, provider registries, and runtime DI containers. |
| AC-004 | passed | The checked-in site guidance explicitly names typed ports, bootstrap wiring, cohesion-first splitting, UI-contract preservation, and propose-review-apply as the normal collaboration and user workflow. |
