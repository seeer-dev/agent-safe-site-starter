# PostgreSQL Lock Semantics and Execution Evidence Delivery Plan

Change ID: postgres-lock-semantics-and-evidence
Revision: 1
Status: Applying

Normative specification: [`spec.md`](spec.md)

## Repository Reality

| Area | Current state | Required correction |
|---|---|---|
| Migration test | CI applies migrations before `TestPostgresLiveMigrateApplyAndTimestamp`; its second `Apply` observes existing records | Apply into a unique empty namespace and assert newly inserted versions/timestamps |
| Staff lock test | Sequentially demotes owner 1, then rejects demoting owner 2 | Use deterministic overlap across distinct operations and assert one winner, one `ErrLastOwner`, one remaining owner |
| Media lock test | Sequential lifecycle calls execute lock-bearing SQL but do not observe blocking or skip behavior | Hold locks from a separate transaction and observe blocking/timeout and `SKIP LOCKED` selection |
| CI live step | `go test -v -run '^TestPostgresLive' ./server/...` relies on package exit | Parse `go test -json`, propagate runner failure, and require an exact pass inventory with zero missing/skip |
| Verifier unit test | Finds `go test` and `-count=10` independently | Compare the complete concurrency-step argv |
| Prior receipt | Marks AC-002 passed while stating live PostgreSQL was not observed | Keep this change pending until a named real CI run is recorded |

## Scope Lock

- `.github/workflows/ci.yml`
- `server/internal/migrate/migrate.go`
- `server/internal/migrate/postgres_integration_test.go`
- `server/internal/modules/media/postgres_integration_test.go`
- `server/internal/modules/media/store_sql.go`
- `server/internal/modules/staff/postgres_integration_test.go`
- `server/internal/modules/staff/store.go`
- `server/tools/postgres-live-gate/**`
- `server/tools/verify/main_test.go`
- `specs/changes/postgres-lock-semantics-and-evidence/**`

Production behavior in `migrate.go`, `media/store_sql.go`, and `staff/store.go` may be temporarily mutated only for red-before-green evidence and must be restored. A final production edit is limited to a minimal unexported synchronization seam when required for deterministic tests; it must be inert when unset and may not alter domain outcomes.

## Dependency-Ordered Slices

### Slice 1: Fresh Migration and Exact Event Gate
- Isolate the migration test from the pre-migrated CI namespace and assert the exact migration filename set plus newly inserted positive timestamps.
- Add `server/tools/postgres-live-gate` using only the standard library. It runs or consumes `go test -json`, propagates process failure, and validates the exact package/test pass inventory with negative fixtures for missing, skipped, and failing events.
- Covers: REQ-001, REQ-003, AC-001, AC-004.

### Slice 2: Staff and Media Semantic Lock Tests
- Replace the staff test's sequential lock claim with deterministic overlapping owner updates over live PostgreSQL. Preserve any useful sequential last-owner regression under a truthful name.
- Add media lock tests that hold a row from a separate transaction, observe the competing `FOR UPDATE` effect within a bound, and prove `SKIP LOCKED` omits the held row while avoiding duplicate claims.
- Add only the smallest inert test seam if direct database coordination cannot prove deterministic overlap.
- Covers: REQ-002, AC-002, AC-003.

### Slice 3: CI Wiring, Mutation Replay, and Independent Evidence
- Replace the CI live-test command with the exact event gate while preserving the ephemeral service, migrations, and SQLite suite.
- Strengthen `server/tools/verify/main_test.go` to compare the complete concurrency command.
- For each mapped branch, temporarily disable the behavior, record the targeted failure, restore it, and replay green.
- Agy hands off implementation without accepting evidence. Claude reviews the diff and negative cases. Any Claude remediation is replayed by Codex.
- Keep status `Verifying` until the real CI run receipt names the run and shows every required test passed without skip.
- Covers: REQ-003, REQ-004, AC-004, AC-005, AC-006, AC-007.

## Validation Plan

```text
go test ./server/tools/postgres-live-gate -count=20
CI=true go test -run '^TestPostgresLive' ./server/...          # must fail without TEST_DATABASE_URL
go test -json -run '^TestPostgresLive' ./server/...            # live environment; exact gate consumes events
go test ./server/tools/verify -count=20
go run ./server/tools/archcheck
go run ./server/tools/speccheck
go run ./server/tools/scopecheck
go run ./server/tools/verify
```

The live PostgreSQL commands are evidence only when `TEST_DATABASE_URL` targets an isolated test service. Do not print the DSN. No status advances to Accepted from local skips or source inspection.

## Implementation expansion context

Proposal revision: 1
Proposal status: Applying
Repository baseline: `7c45b616fbe3a632ffe2a39d872c98485466c991`
Observed HEAD: `7e5aa90b92d23c8c316d44ca935be7af8d150a1c`
Pre-existing dirty paths:
- `.github/workflows/ci.yml` - preserve; postgres-lock CI wiring, not attributable to this expansion
- `server/internal/migrate/postgres_integration_test.go` - preserve; Slice 2 migration test
- `server/internal/modules/media/postgres_integration_test.go` - preserve; Slice 2 media lock test
- `server/internal/modules/staff/postgres_integration_test.go` - preserve; Slice 2 staff lock test
- `server/internal/modules/staff/store.go` - preserve; Slice 2 inert test seam
- `server/tools/verify/main_test.go` - preserve; Slice 3 verifier test
- `specs/changes/postgres-lock-semantics-and-evidence/**` - preserve; this change's own artifacts
Packet ID mapping:
- Existing Slice 1 (gate portion) -> Packet S01G

### Packet S01G: Regression coverage for duplicate-run-without-pass rejection and non-zero-exit override

Status: executable
Covers: REQ-003, AC-004
Hard dependencies: none
Outcome: Two new regression tests prove (1) a required test that emits only duplicate `run` events with no `pass` is rejected as missing, and (2) a non-zero child process exit fails the gate even when the JSON event stream shows every required test passing.
Safe failure / rollback: All edits are additive — one small pure-function extraction (`EvaluateGateResult`) and two new test functions. No production argv, inventory, or control-flow logic changes. If the extraction is reverted, `RunLiveGate` returns to inline logic and the new `EvaluateGateResult` test is removed; the duplicate-run test depends only on existing `ValidateTestEvents` and survives independently.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Behavior owner | `server/tools/postgres-live-gate/main.go` | `ValidateTestEvents` | Sets `passed[pkg][test]=true` only on `Action=="pass"`; `"run"` is not in the switch, so a test with only `run` events falls through to `!passed` → "required test missing from execution stream" |
| Behavior owner | `server/tools/postgres-live-gate/main.go` | `RunLiveGate` | After `ValidateTestEvents`, checks `cmdErr != nil` and returns error even when `valErr == nil`; but this logic is inline and not independently testable |
| Existing proof | `server/tools/postgres-live-gate/main_test.go` | `TestGatePassesExactInventory` | Asserts nil error on a stream where every required test has `run` then `pass` |
| Existing proof | `server/tools/postgres-live-gate/main_test.go` | `TestGateRejectsMissingTest` | Asserts error naming the missing test when one required test is entirely absent from the stream |
| Existing proof | `server/tools/postgres-live-gate/main_test.go` | `TestGateRejectsSkippedTest` | Asserts "was skipped" error when a required test has `skip` instead of `pass` |
| Existing proof | `server/tools/postgres-live-gate/main_test.go` | `TestGateRejectsFailedTest` | Asserts "test failed" error when a required test has `fail` instead of `pass` |
| Production invariant | `server/tools/postgres-live-gate/main.go` | `RunLiveGate` argv | `exec.Command("go", "test", "-json", "-run", "^TestPostgresLive", "./server/...")` — must not change |
| Production invariant | `server/tools/postgres-live-gate/main.go` | `RequiredLiveTests` | 3-package, 3-test inventory — must not change |

Read set:
- `server/tools/postgres-live-gate/main.go` - understand `ValidateTestEvents` switch logic and `RunLiveGate` post-wait decision
- `server/tools/postgres-live-gate/main_test.go` - understand existing test patterns (inline JSON streams, `strings.Contains` assertions)

Modify set:
- `server/tools/postgres-live-gate/main.go` - extract post-wait decision from `RunLiveGate` into `EvaluateGateResult(valErr, cmdErr error) error`; `RunLiveGate` calls it as `return EvaluateGateResult(valErr, cmdErr)`
- `server/tools/postgres-live-gate/main_test.go` - add `TestGateRejectsDuplicateRunWithoutPass` and `TestGateFailsOnNonZeroExitEvenWithAllPasses`

Must not modify:
- `RunLiveGate` argv (`exec.Command` arguments) - production contract
- `RequiredLiveTests` map - production inventory
- `ValidateTestEvents` logic - already correct for both requirements; only tests are added
- `main()` function - entry point unchanged
- Any file outside `server/tools/postgres-live-gate/**` - scope boundary

#### Behavior contract

Before:
- `ValidateTestEvents` already rejects a test with only `run` events as missing (the `"run"` action is not in the switch, so `passed` stays false). But no test exercises this with duplicate `run` events — the existing `TestGateRejectsMissingTest` uses a stream where the missing test is entirely absent, not one where `run` events appear without a matching `pass`.
- `RunLiveGate` already returns an error when `cmdErr != nil` even if `valErr == nil`. But this logic is inline inside `RunLiveGate` and cannot be exercised by a unit test without launching a real `go test` subprocess.

After:
- `TestGateRejectsDuplicateRunWithoutPass` feeds `ValidateTestEvents` a stream where one required test (migrate) has two `run` events but no `pass`, while the other two required tests pass normally. The test asserts the returned error contains "missing" and names `TestPostgresLiveMigrateApplyAndTimestamp`.
- `EvaluateGateResult(nil, errors.New("exit status 1"))` returns a non-nil error whose message contains "process exited". `RunLiveGate` delegates its post-wait decision to this function with no change to its observable behavior.
- `TestGateFailsOnNonZeroExitEvenWithAllPasses` calls `EvaluateGateResult(nil, errors.New("exit status 1"))` and asserts the error is non-nil and contains "process exited". It also calls `EvaluateGateResult(nil, nil)` and asserts nil, proving the happy path is preserved.

Preserved invariants:
- `RunLiveGate` argv: `exec.Command("go", "test", "-json", "-run", "^TestPostgresLive", "./server/...")` — unchanged
- `RequiredLiveTests` map: 3 packages, 3 tests — unchanged
- `ValidateTestEvents` signature and logic — unchanged
- `main()` function — unchanged
- Existing 4 tests — unchanged and still passing

Inputs and authority:
- `ValidateTestEvents` takes an `io.Reader` (JSON stream) and a `map[string][]string` (required inventory); both are test-supplied
- `EvaluateGateResult` takes two `error` values (validation error, process error); both are caller-supplied
- `RunLiveGate` takes two `io.Writer` values; production behavior unchanged

Errors and edge states:
- Duplicate `run` without `pass` → "required test missing from execution stream: <pkg> <test>"
- `valErr == nil`, `cmdErr != nil` → "test process exited with error: <cmdErr>"
- `valErr != nil`, `cmdErr != nil` → "tests failed (<valErr>) and process exited: <cmdErr>"
- `valErr != nil`, `cmdErr == nil` → `<valErr>`
- `valErr == nil`, `cmdErr == nil` → nil

#### Ordered edits

1. `server/tools/postgres-live-gate/main.go` - `RunLiveGate` post-wait block (lines 64-76)
   - Change: Extract the `if cmdErr != nil { ... } if valErr != nil { ... } return nil` block into a new function `func EvaluateGateResult(valErr, cmdErr error) error` placed immediately after `RunLiveGate`. Replace the extracted block in `RunLiveGate` with `return EvaluateGateResult(valErr, cmdErr)`.
   - Constraint: `RunLiveGate`'s `exec.Command` argv, `cmd.Env`, pipe setup, `cmd.Start`, `io.TeeReader`, and `ValidateTestEvents` call must remain byte-for-byte identical. The `errors`, `fmt` imports remain used.
   - Result: `RunLiveGate` ends with `return EvaluateGateResult(valErr, cmdErr)`; `EvaluateGateResult` contains the exact same decision logic that was inline before.

2. `server/tools/postgres-live-gate/main_test.go` - new test `TestGateRejectsDuplicateRunWithoutPass`
   - Change: Add a new test function. Feed `ValidateTestEvents` a JSON stream where `TestPostgresLiveMigrateApplyAndTimestamp` has two `run` events but no `pass`, while `TestPostgresLiveStaffLockActiveOwners` and `TestPostgresLiveMediaLifecycleAndLocks` have `run` then `pass`. Assert `err != nil`, `strings.Contains(err.Error(), "missing")`, and `strings.Contains(err.Error(), "TestPostgresLiveMigrateApplyAndTimestamp")`.
   - Constraint: `RequiredLiveTests` is used as-is (not a custom map). Existing 4 tests must remain and pass.
   - Result: Before implementation, this test does not exist. After implementation, it passes against the existing `ValidateTestEvents` logic (proving the behavior is already correct and regression-protected).

3. `server/tools/postgres-live-gate/main_test.go` - new test `TestGateFailsOnNonZeroExitEvenWithAllPasses`
   - Change: Add a new test function. Call `EvaluateGateResult(nil, errors.New("exit status 1"))` and assert `err != nil` and `strings.Contains(err.Error(), "process exited")`. Call `EvaluateGateResult(nil, nil)` and assert `err == nil`. Add `"errors"` to the import block.
   - Constraint: Does not call `RunLiveGate` (no subprocess). Existing 4 tests must remain and pass.
   - Result: Before the `EvaluateGateResult` extraction (edit 1), this test does not compile. After both edits, it passes.

#### Integration trace

```text
TestGateRejectsDuplicateRunWithoutPass
  -> strings.NewReader (synthetic JSON stream)
  -> ValidateTestEvents (existing, unchanged)
  -> switch on Action: "run" not handled, "pass" sets passed=true
  -> final check: !passed[migrate][TestPostgresLiveMigrateApplyAndTimestamp]
  -> error: "required test missing from execution stream"

TestGateFailsOnNonZeroExitEvenWithAllPasses
  -> EvaluateGateResult(nil, errors.New("exit status 1"))
  -> cmdErr != nil, valErr == nil
  -> return fmt.Errorf("test process exited with error: %w", cmdErr)
  -> error: "test process exited with error: exit status 1"

RunLiveGate (production, unchanged behavior)
  -> exec.Command("go", "test", "-json", "-run", "^TestPostgresLive", "./server/...")
  -> ValidateTestEvents(tee, RequiredLiveTests)
  -> cmd.Wait()
  -> EvaluateGateResult(valErr, cmdErr)
  -> same decision as before extraction
```

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Duplicate `run` without `pass` rejected as missing | `D:\Projects\AI-go-starter` | `go test ./server/tools/postgres-live-gate -run '^TestGateRejectsDuplicateRunWithoutPass$' -v -count=1` | `TestGateRejectsDuplicateRunWithoutPass` | PASS, error contains "missing" and "TestPostgresLiveMigrateApplyAndTimestamp" | If `ValidateTestEvents` were changed to set `passed=true` on `"run"`, this test would fail (error would be nil) | Temporarily add `case "run": passed[event.Package][event.Test] = true` to `ValidateTestEvents`, run test, observe failure, remove the line | `git diff server/tools/postgres-live-gate/main.go` shows no `case "run"` addition |
| Non-zero exit fails even with all passes | `D:\Projects\AI-go-starter` | `go test ./server/tools/postgres-live-gate -run '^TestGateFailsOnNonZeroExitEvenWithAllPasses$' -v -count=1` | `TestGateFailsOnNonZeroExitEvenWithAllPasses` | PASS, error contains "process exited" | If `EvaluateGateResult` returned nil when `cmdErr != nil`, this test would fail | Temporarily change `EvaluateGateResult` to `if valErr != nil { return valErr }; return nil` (dropping cmdErr check), run test, observe failure, restore | `git diff server/tools/postgres-live-gate/main.go` shows the cmdErr check restored |
| Existing tests still pass | `D:\Projects\AI-go-starter` | `go test ./server/tools/postgres-live-gate -v -count=1` | All 6 tests (4 existing + 2 new) | All PASS | If extraction changed RunLiveGate behavior, existing tests that call `ValidateTestEvents` directly would still pass, but the extraction would be wrong | N/A (existing tests don't exercise RunLiveGate) | N/A |
| Production argv unchanged | `D:\Projects\AI-go-starter` | `git diff server/tools/postgres-live-gate/main.go` | `exec.Command` line in `RunLiveGate` | No diff on the `exec.Command` line | N/A | N/A | `git diff` shows only the extraction and `return EvaluateGateResult(valErr, cmdErr)` |
| Required inventory unchanged | `D:\Projects\AI-go-starter` | `git diff server/tools/postgres-live-gate/main.go` | `RequiredLiveTests` map | No diff on the map | N/A | N/A | `git diff` shows no change to `RequiredLiveTests` |
| gofmt clean | `D:\Projects\AI-go-starter` | `gofmt -l server/tools/postgres-live-gate/` | empty output | exit 0, no files listed | N/A | N/A | N/A |
| go vet clean | `D:\Projects\AI-go-starter` | `go vet ./server/tools/postgres-live-gate/` | exit 0 | no output | N/A | N/A | N/A |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] The bounded failure trigger fails for the claimed reason and is restored.
- [ ] Existing relevant tests remain present and unweakened.
- [ ] No temporary mutation, fixture, credential, or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.

#### Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass (REQ-003 -> S01G) |
| Every AC maps to proof in at least one packet | pass (AC-004 -> S01G) |
| Every packet maps back to REQ/AC | pass (S01G -> REQ-003, AC-004) |
| Baseline, observed HEAD, dirty paths, and packet ID mapping are explicit | pass |
| All paths and symbols were inspected | pass (main.go and main_test.go read in full) |
| Hard dependency graph has no unexplained edge or cycle | pass (no hard dependencies) |
| Scope covers every modify path and no unrelated path | pass (only `server/tools/postgres-live-gate/**`) |
| Product/authority decisions are approved or blocked | pass (no new decisions; extraction is implementation detail) |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |
