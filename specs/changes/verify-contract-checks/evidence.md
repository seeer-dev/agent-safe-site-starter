# Evidence

## Delivery Status

Revision 3 implemented on 2026-08-18 and now Verifying. AC-001 and AC-003 require independent-review receipts, which the implementer cannot write for their own work, so both remain pending until another reviewer replays them.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | CI runs make verify-contracts as a required step, which executes both admin/scripts/check-resource-contracts.mjs and site/themes/minimal-cart/scripts/check-openapi-contracts.mjs. A failure in either fails the step and therefore the gate. |
| REQ-002 | passed | make verify-contracts is the single documented local command. It checks the Node version first and reports the required floor plainly, then runs both scripts directly with node. No npm install step exists, because both scripts import only Node standard-library modules (fs, path, url). go run ./server/tools/verify is unchanged and contains no Node reference. |
| REQ-003 | passed | CI selects Node explicitly via actions/setup-node pinned to 22.11.0 rather than relying on the runner default, with a comment recording why the floor exists. |
| AC-001 | pending | Implementer observation only, awaiting independent review. Renaming the guarded slug field in admin/src/config/resources/products.ts made the gate exit 2 with: Resource contract check FAILED: - product form must preserve slug and product_images. Restored with no residue. Recorded honestly: a first mutation renaming sku did NOT turn the gate red, because the checker asserts the presence of specific named fields rather than validating every field, so the gate proves drift only for the fields it names. A reviewer who did not implement this change must replay both mutations before this can pass. |
| AC-002 | passed | make verify-contracts prints both check names and both results: Resource contract check PASSED and OpenAPI contract check PASSED. |
| AC-003 | pending | Implementer observation only, awaiting independent review. The target runs a Node version guard before either script; below 20.11 it exits nonzero naming the observed version, the required floor, the reason (import.meta.dirname), and the install action. The below-floor path was not exercised on this machine, which runs Node 22.22.2, so a reviewer must exercise it with an older Node or an equivalent forced condition. |
| AC-004 | passed | server/tools/verify/main.go contains zero occurrences of node, npm, or Node, and this change did not touch it. A machine without Node runs the Go verifier exactly as before. |
| AC-005 | passed | The workflow pins node-version 22.11.0, above the 20.11 floor that import.meta.dirname requires, and README records the same floor as the documented prerequisite. |
