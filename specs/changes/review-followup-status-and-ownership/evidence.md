# Architecture Review Follow-up Evidence

Change ID: review-followup-status-and-ownership
Revision: 1
Status: Accepted

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | README.md and docs/project-status.md now distinguish implemented commerce/payment behavior from HTTP contract truth, make contract restoration a v1 pre-deploy blocker, and keep external stage/deploy acceptance separate. |
| AC-001 | passed | The root and canonical status explicitly say source-level commerce can be implemented while runtime/OpenAPI contract truth still needs restoration before v1 deploy-ready acceptance. |
| REQ-002 | passed | architecture.yaml now records ecpay_payment_attempts under commerce ownership and points the known commerce-to-media data write at store_catalog.go:markMediaAssociationsTx rather than stale store.go line numbers. |
| AC-002 | passed | The ownership map names the current ECPay attempt table and uses a function-level cross-module write location that survives file line movement. |
| REQ-003 | passed | docs/review-status.md records the current interpretation of historical reviews, evidence-complete lifecycle closures versus still-pending changes, and the revised priority order; README.md and docs/README.md route readers to the current status material. |
| AC-003 | passed | The review overlay labels resolved historical findings, preserves external/pending evidence honestly, rejects direct execution of stale Draft plans, and does not introduce a new runtime architecture decision. |
