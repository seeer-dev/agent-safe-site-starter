# Frontend Contract Verification Gate Specification

Change ID: verify-contract-checks
Revision: 2
Status: Draft
Decision authority: Repository owner/user
Approval basis: Pending repository owner decision whether Node becomes a required dependency of the full verifier or frontend contract checks remain a separate mandatory gate.
Repository baseline: 6a3c113e9d4957617a532ab2d10701a6fbc66db9
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

Revision 2 rebases the proposal onto baseline 6a3c113. Seventeen commits landed since revision 1, including the live PostgreSQL gate, the commerce boolean adapter fix, request observability, connection pool bounds, and a single-command site build. Evidence binds to the baseline, so a stale one cannot carry acceptance once an apply begins.

Scope is unchanged. An earlier refresh added `.github/workflows/ci.yml` to `control.json` on the reasoning that AC-001 and AC-002 name CI determinately. That was withdrawn: it contradicted this document's own Scope section and the plan's Scope Lock, both of which state that this revision authorizes no CI change, and it altered what the proposal permits without amending the text that describes it. The proposal remains blocked on the REQ-002 ownership decision, so authorizing CI alone would license a half-implementation.
