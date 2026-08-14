# Backend Agent Rules

These rules apply under `server/` and supplement the repository-root
`AGENTS.md`. The root architecture, controlled-change, scope, and verification
requirements remain authoritative.

## Preserve tests and gates

- Inventory existing `Test*` functions and other named checks before editing a
  test or verification file; compare the inventory with the final diff.
- Do not remove, skip, quarantine, rename, or weaken an existing check unless
  the approved specification explicitly changes that behavior and the handoff
  identifies every affected check.
- A broader table test does not replace narrower regression, governance,
  permission, migration, or security tests unless independent review approves
  the removal.

## Require falsifiable evidence

- Every new regression, security, integration, branch, or governance check must
  name a bounded failure trigger and be observed failing for that trigger before
  the trigger is restored and the final passing run is recorded.
- The failure must come from the assertion that protects the claimed behavior,
  not an unrelated compile, setup, environment, or timeout failure.
- Do not commit the temporary mutation. Confirm the working tree contains only
  the authorized implementation after restoration.
- Source reachability, a green exit code, or the presence of a command is not
  evidence when removal of the claimed behavior would leave the check green.

## Prove required test execution

- An environment-gated integration test may skip locally, but a required CI or
  acceptance environment must fail when its dependency or configuration is
  missing.
- For a required test set, inspect `go test -json` events or equivalent exact
  runner output and confirm every expected test appears with `pass`; missing or
  skipped tests fail the evidence claim.
- Record the exact command, environment class, expected test names, and observed
  result. A package-level `ok` line alone does not prove named tests executed.

## Match concurrency evidence to the claim

- Distinguish sequential stale-version/interleaving tests from tests that claim
  simultaneous execution. The former need not spawn workers; the latter must
  prove overlap through a deterministic barrier, hook, or transaction state.
- A simultaneous-concurrency test needs at least two independently controlled
  operations, a bounded deadline, collected outcomes, and a final invariant.
  Do not infer concurrency from a test name, `t.Parallel`, timing luck, or a
  repeated sequential run.
- A PostgreSQL locking claim requires separate connections or transactions and
  an observable lock effect. Prove `FOR UPDATE` through bounded blocking and
  release behavior; prove `SKIP LOCKED` while another transaction holds the
  target row and assert that the competing transaction skips it without
  duplicating work.

## Verify wiring exactly

- Tests for verifier or CI wiring must compare the relevant complete argument
  vector, structured configuration, or exact step. Independent substring checks
  such as finding `go test` somewhere and `-count=10` elsewhere are insufficient.
- When a check is required to run named tests, verify both selection and
  execution; command construction alone is not consumer evidence.

## Keep acceptance honest

- Do not mark evidence passed when its receipt says execution was unavailable,
  not observed, skipped, still outstanding, or dependent on a future CI run.
- An implementer's report is provisional. A reviewer who changes the
  implementation or test cannot provide independent-review evidence for that
  remediation; another reviewer must replay it.
- Preserve unresolved product semantics as explicit blockers or quarantines
  authorized by the controlled spec. Do not silently redefine the expected
  loser error, permission result, or data invariant to make a test deterministic.
