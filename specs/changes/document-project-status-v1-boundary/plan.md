# Project Status Documentation Delivery Plan

Change ID: document-project-status-v1-boundary
Revision: 1
Status: Accepted

Repository baseline: `c8249301020c6a0eceeda33b875e5d9101b91ebe`

## Scope lock

- `README.md`
- `docs/README.md`
- `docs/project-status.md`
- `docs/commerce-acceptance.md`
- `specs/changes/document-project-status-v1-boundary/**`

## Delivery

Covers REQ-001, AC-001, REQ-002, AC-002.

- add a concise README current-status section with links to canonical detail;
- add a documentation index that points maintainers to the canonical status and commerce acceptance documents;
- record completed starter/governance/commerce/ECPay source-level capabilities;
- record remaining official ECPay conformance, sample acceptance, deploy-readiness, and stage-transaction work;
- distinguish starter-v1 blockers from optional refund/invoice/logistics/reconciliation capabilities;
- add Mermaid diagrams where flow/release sequencing is materially clearer visually;
- keep claims grounded to the current merged main baseline and previously green PR/post-merge CI.
