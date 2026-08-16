# Change safety

Before editing across modules or shared infrastructure:

1. Inspect `architecture.yaml`.
2. Use CodeGraph `explore`/`impact` and `affected` when available.
3. Create or select a controlled change under `specs/changes/` before editing protected paths.
4. Establish the appropriate scope gate for the working tree:
   - In a clean/single-task working tree, create `.ai/scope.json` listing expected paths (legacy mode, which does not provide cross-task attribution).
   - In parallel delegated or dirty-primary working trees, preflight active/dirty ownership across all worktrees (stop immediately on `applies_to` overlap), create an OS-temporary linked worktree, and set `$env:SCOPE_CHANGE_ID='<change-id>'`. Selected mode is local-only and forbidden in CI.
5. Run `go run ./server/tools/speccheck`, then make the change.
6. Run `go run ./server/tools/scopecheck` before broadening scope.
7. If a new path is genuinely required, update both controlled `applies_to` and task scope with a reason rather than silently editing outside them.
8. Forbid in-repository scratch files and secrets in reports; in `finally`, remove temporary worktrees and prune Git metadata.
9. Run the full verifier (`go run ./server/tools/verify`).

CodeGraph is context discovery, not write authorization. `speccheck`, the scope gate, and CI provide enforcement.
