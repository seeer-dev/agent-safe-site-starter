# Change safety

Before editing across modules or shared infrastructure:

1. Inspect `architecture.yaml`.
2. Use CodeGraph `explore`/`impact` and `affected` when available.
3. Create or select a controlled change under `specs/changes/` before editing protected paths.
4. Create `.ai/scope.json` listing expected paths.
5. Run `go run ./server/tools/speccheck`, then make the change.
6. Run `go run ./server/tools/scopecheck` before broadening scope.
7. If a new path is genuinely required, update both controlled `applies_to` and task scope with a reason rather than silently editing outside them.
8. Run the full verifier.

CodeGraph is context discovery, not write authorization. `speccheck`, the scope gate, and CI provide enforcement.
