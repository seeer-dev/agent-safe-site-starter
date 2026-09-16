# Architecture Review Follow-up: Status and Ownership

Change ID: review-followup-status-and-ownership
Revision: 1
Status: Accepted
Decision authority: Repository owner
Approval basis: Repository owner authorized applying the reviewed architecture-feedback corrections on 2026-08-31, including lifecycle cleanup, current-status clarification, and architecture ownership-map currency without broad runtime refactoring.
Repository baseline: 10f805e18e394e7065854dfbceb327114e0d4564
Supersedes: none

## Outcome

Keep the starter's agent-facing truth current after the 2026-08-31 architecture review without turning the review into a broad refactor. The repository should clearly distinguish completed runtime behavior from contract/deployment acceptance, keep architecture ownership metadata current, and route historical review documents through a current status overlay.

### REQ-001: Canonical status separates runtime completion from contract truth

`docs/project-status.md` MUST state that the commerce/ECPay runtime path is implemented while HTTP contract truth restoration remains required before the starter is called v1 deploy-ready.

#### AC-001: Contract drift is not hidden by source-level completion
- GIVEN the commerce checkout and ECPay runtime path are implemented and CI verified
- WHEN an agent reads the canonical project status
- THEN it sees contract truth restoration as a separate current blocker rather than interpreting source-level completion as full API-contract acceptance

### REQ-002: Architecture data ownership metadata is current

`architecture.yaml` MUST name the current commerce-owned ECPay persistence table and MUST locate the known commerce-to-media SQL write using the current cohesive file/function rather than stale line numbers.

#### AC-002: Ownership map follows current code structure
- GIVEN ECPay persistence exists and commerce cohesion moved product-media association SQL into `store_catalog.go`
- WHEN an agent reads `architecture.yaml`
- THEN `ecpay_payment_attempts` is owned by commerce and the known cross-module write points to `store_catalog.go:markMediaAssociationsTx`

### REQ-003: Historical review findings are contextualized

The documentation index MUST route architecture-review readers through a current overlay that distinguishes resolved findings, still-open evidence, stale Draft proposals, and future work without rewriting history or falsely accepting blocked evidence.

#### AC-003: Review material cannot masquerade as current backlog
- GIVEN historical backend/admin reviews contain findings that later controlled changes resolved
- WHEN an agent follows the documentation index and review overlay
- THEN resolved items identify their accepted change, pending/blocked items remain explicit, stale Drafts require re-proposal from the current baseline, and no new runtime architecture is implied
