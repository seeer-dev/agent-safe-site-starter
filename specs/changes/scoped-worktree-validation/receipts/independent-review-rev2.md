# Independent Review — Revision 2

Reviewer: Codex

## Selected mode

- `go test ./server/tools/scopecheck -count=20`: passed (20 repetitions).
- `go test -race ./server/tools/scopecheck -count=1`: passed.
- The temporary-repository suite proves an allowed selected linked worktree excludes foreign primary dirt, and discovers committed-baseline, staged, unstaged, deleted, and untracked selected paths.

## Fail-closed guards

- Named tests passed for primary-worktree rejection, CI rejection, malformed/missing control, status, empty scope, baseline format/non-ancestor, Git discovery/list/diff/untracked failures, and current/other worktree path-resolution failures.
- `go test ./server/...`: passed.

## Legacy compatibility

- Named tests passed for absent scope skip, configured paths, outside/untracked rejection, `SCOPE_BASE_REF`, and the original error-tolerant Git discovery behavior.
- Legacy Git failures remain ignored only in legacy mode; selected mode fails closed.

## Workflow contract

- `TestScopeWorkflowDocumentsIsolationContract` passed for all nine governance documents.
- The handoff recipe creates a detached OS-temporary worktree from an explicit repository root and baseline, then removes it and prunes metadata in `finally`; it contains no provider-specific requirement.

## Mutation replay

Each mutation was run locally, produced the named test failure, then was immediately restored. The final binary diff hash for the two scopecheck files matched the pre-mutation hash `9207455b677e100bf18f39f20d0a2a129661fcdb`.

| Temporary mutation | Expected red proof |
|---|---|
| bypass selected `!allowed` outside-path check | `TestSelectedChangeRejectsOutsidePath` failed because unauthorized path passed |
| bypass cross-worktree `allowed` overlap check | `TestSelectedChangeRejectsDirtyPathInOtherWorktree` failed because overlap passed |
| make selected `gitOutput` ignore errors | all four `TestSelectedChangeGitFailureFailClosed` subtests failed |
| remove README's PR-base unique clause | `TestScopeWorkflowDocumentsIsolationContract/README.md` failed naming the missing clause |

## Gate result and limitation

- `go run ./server/tools/speccheck`: passed (`17 controlled spec(s), 9 protected changed file(s)`).
- `git diff --check`: passed.
- `go run ./server/tools/verify` reached and passed `archcheck`, `migration-parity`, and `speccheck`, then failed at legacy `scopecheck`. The checked-in `.ai/scope.json` belongs to a different shared-worktree task and rejects this change plus unrelated untracked proposals. No attempt was made to alter or hide those paths. This limitation blocks full legacy verification in the shared primary worktree, not the isolated selected-mode behavior covered above.
