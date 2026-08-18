# Frontend Contract Verification Gate Specification
| 2 | none | Proposal targeted baseline 7e5aa90 | Proposal targets baseline 3ecc1b6 | Seventeen commits landed since revision 1. | Owner instruction to refresh draft baselines on 2026-08-18 | No evidence existed to invalidate; all records remain pending. |
| 2 | REQ-001 | applies_to named only this proposal directory, so an apply could authorize no CI change | applies_to adds .github/workflows/ci.yml | AC-001 and AC-002 name CI determinately, so the surface is known. Without it an apply would be rejected by speccheck for touching an unauthorized path. | same | — |
| 2 | REQ-002 | — | still no implementation path | Deliberate. Whether the local command is server/tools/verify or a separate frontend verifier is an unresolved owner decision that changes the development prerequisite contract. Scoping it now would pre-decide it. | same | — |

Change ID: verify-contract-checks
Revision: 2
Status: Draft
Decision authority: Repository owner/user
Approval basis: Pending repository owner decision whether Node becomes a required dependency of the full verifier or frontend contract checks remain a separate mandatory gate.
Repository baseline: 3ecc1b63b85f13be773af79453a2c29f868903e4
Supersedes: none

## Outcome

Make the existing admin resource-contract and public-theme OpenAPI checks mandatory and falsifiable in CI and through one documented local command, without silently skipping them or accidentally changing the starter's documented toolchain prerequisites.

## Scope

This revision records the required gate behavior and the unresolved command ownership decision only. It does not authorize changes to CI, `server/tools/verify`, package manifests, or lockfiles.

Out of scope:
- Rewriting the JavaScript contract scripts in Go.
- Generating clients from OpenAPI.
- Running every frontend unit test as part of this contract-only change.
- Silently downloading packages or mutating lockfiles during verification.

## Requirements

### REQ-001: Mandatory Consumer Contract Checks
CI MUST execute `admin/scripts/check-resource-contracts.mjs` and `site/themes/minimal-cart/scripts/check-openapi-contracts.mjs` against the repository contract. A failure in either check SHALL fail the required gate.

#### AC-001: Contract Drift Fails CI
- GIVEN a bounded temporary mismatch between one real frontend consumer and `contracts/openapi.yaml`
- WHEN the required gate runs
- THEN the corresponding existing script MUST fail for that mismatch and the pipeline MUST not report success.

#### AC-002: Both Consumers Execute
- GIVEN a clean checkout with declared prerequisites installed
- WHEN the required gate runs
- THEN output MUST identify both the admin resource-contract check and the public-theme OpenAPI check as executed and passed.

### REQ-002: Explicit Local Toolchain Contract
The repository MUST expose one documented local command for the same checks. Missing Node or package prerequisites SHALL fail with an actionable message; they MUST NOT be silently skipped. Whether that command is `go run ./server/tools/verify` or a separate mandatory frontend verifier requires owner approval because it changes the development prerequisite contract.

#### AC-003: Missing Prerequisites Fail Honestly
- GIVEN Node or required frontend dependencies are unavailable
- WHEN the selected local command runs
- THEN it MUST exit nonzero with the exact install/setup action and MUST NOT claim repository verification succeeded.

## Amendments

None.
