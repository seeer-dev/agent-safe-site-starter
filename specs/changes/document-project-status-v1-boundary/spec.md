# Project Status Documentation Specification

Change ID: document-project-status-v1-boundary
Revision: 1
Status: Accepted
Decision authority: Repository owner
Approval basis: Owner explicitly requested that completed work, remaining work, README status, and supporting documentation be landed on 2026-08-31.
Repository baseline: c8249301020c6a0eceeda33b875e5d9101b91ebe
Supersedes: none

### REQ-001: README exposes truthful current status

The README MUST summarize the current v1 completion boundary and MUST distinguish source-level completion from deployment acceptance and optional post-v1 commerce operations.

#### AC-001: README does not overclaim deployment or full commerce operations

- GIVEN the architecture/governance and ECPay commerce source flow are merged and CI verified
- WHEN the repository status is described
- THEN the README identifies those capabilities as complete for the current source-level/v1 boundary while keeping public deployment acceptance, refunds, invoices, logistics, and reconciliation outside the completed claim

### REQ-002: Detailed status and acceptance boundaries are durable

The repository MUST provide canonical documentation for the overall v1 roadmap and for commerce source-vs-deployment acceptance so later work can update a stable status source rather than reconstruct progress from conversation history.

#### AC-002: Documents show implemented flow, release path, and non-blocking extensions

- GIVEN maintainers or agents need to understand what to do next
- WHEN they read the detailed status documents
- THEN they can identify the implemented ECPay authority flow, the v1 release sequence, the deployment acceptance requirements, and the commerce capabilities intentionally deferred beyond v1
