# Frontend Contract Verification Gate Specification

Change ID: verify-contract-checks
Revision: 3
Status: Verifying
Decision authority: Repository owner/user
Approval basis: Repository owner decided on 2026-08-18 that go run ./server/tools/verify stays Go-only and the frontend contract checks become a separate required gate with its own documented command.
Repository baseline: fcace980a247858b17eb1a25cae8369395b85afc
Supersedes: none

## Outcome

Make the existing admin resource-contract and public-theme OpenAPI checks mandatory and falsifiable in CI and through one documented local command, without silently skipping them or accidentally changing the starter's documented toolchain prerequisites.

## Scope

In scope:

- A single documented local command that runs both existing contract scripts.
- The CI step that makes them a required gate, with an explicitly pinned Node version.
- The README note recording the new command and its prerequisite.

Out of scope:

- `server/tools/verify`. The owner decision is that it stays Go-only, so a Go-only contributor is never blocked by a missing Node toolchain.
- Rewriting the JavaScript contract scripts in Go.
- Generating clients from OpenAPI.
- Running every frontend unit test as part of this contract-only change.
- `npm ci`, package manifests, and lockfiles. Both scripts import only Node standard-library modules (`fs`, `path`, `url`), so installing dependencies would add a failure mode without adding coverage.

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
The repository MUST expose one documented local command that runs both contract scripts directly with `node`. `go run ./server/tools/verify` MUST remain free of any Node dependency. A missing or too-old Node MUST fail with an actionable message naming the minimum version; it MUST NOT be silently skipped.

#### AC-003: Missing Prerequisites Fail Honestly
- GIVEN Node is unavailable or older than the minimum this repository requires
- WHEN the documented local command runs
- THEN it MUST exit nonzero naming the required version and the install action, and MUST NOT claim repository verification succeeded.

#### AC-004: The Go Verifier Stays Node-Free
- GIVEN a checkout on a machine with no Node installed
- WHEN `go run ./server/tools/verify` runs
- THEN it MUST behave exactly as before this change, neither invoking Node nor reporting a frontend contract result.

### REQ-003: Pinned Runtime, Not an Assumed One
CI MUST select an explicit Node version rather than relying on whatever the runner image preinstalls.

#### AC-005: The Required Version Is Declared and Sufficient
- GIVEN `admin/scripts/check-resource-contracts.mjs` uses `import.meta.dirname`, which exists only from Node 20.11
- WHEN CI configures its runtime
- THEN it MUST pin a version at or above that floor rather than accept the runner default, and the same floor MUST appear in the documented prerequisite.

## Amendments

Revision 2 rebases the proposal onto baseline 6a3c113. Seventeen commits landed since revision 1, including the live PostgreSQL gate, the commerce boolean adapter fix, request observability, connection pool bounds, and a single-command site build. Evidence binds to the baseline, so a stale one cannot carry acceptance once an apply begins.

Scope is unchanged. An earlier refresh added `.github/workflows/ci.yml` to `control.json` on the reasoning that AC-001 and AC-002 name CI determinately. That was withdrawn: it contradicted this document's own Scope section and the plan's Scope Lock, both of which state that this revision authorizes no CI change, and it altered what the proposal permits without amending the text that describes it. The proposal remains blocked on the REQ-002 ownership decision, so authorizing CI alone would license a half-implementation.

Revision 3 records the owner decision that revision 2 was blocked on: `go run ./server/tools/verify` stays Go-only, and the frontend contract checks become a separate required gate with its own documented command.

The scope therefore widens from this proposal directory to `.github/workflows/ci.yml`, `Makefile`, and `README.md`. Revision 2 deliberately authorised nothing, because authorising CI while command ownership was undecided would have licensed a half-implementation; that reasoning is now discharged rather than overridden.

Two implementation constraints follow from repository evidence rather than preference. Both scripts import only Node standard-library modules, so the gate runs them directly and never runs `npm ci` — installing dependencies would add a failure mode without adding coverage. And `admin/scripts/check-resource-contracts.mjs` uses `import.meta.dirname`, so a pinned Node version at or above 20.11 is a correctness requirement, not a hygiene preference; REQ-003 exists to stop CI inheriting whatever the runner image happens to ship.
