# Change safety

Before editing across modules or shared infrastructure:

1. Inspect `architecture.yaml`.
2. Use CodeGraph `explore`/`impact` and `affected` when available.
3. Create `.ai/scope.json` listing expected paths.
4. Make the change.
5. Run `go run ./server/tools/scopecheck` before broadening scope.
6. If a new path is genuinely required, update the scope with a reason in the task plan rather than silently editing outside it.
7. Run the full verifier.

CodeGraph is context discovery, not write authorization. The scope gate and CI/tool checks provide enforcement.
