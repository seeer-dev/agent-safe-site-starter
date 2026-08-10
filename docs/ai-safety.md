# AI change safety

The safety model has three independent layers.

## 1. Static architecture boundary

`architecture.yaml` explains ownership. `go run ./server/tools/archcheck` enforces the most important import rules.

## 2. Impact discovery

CodeGraph is optional. When installed, use it before cross-cutting edits to inspect related symbols, callers/callees, impact radius, and affected tests. The repository must remain understandable without CodeGraph.

Optional setup:

```text
npx @colbymchenry/codegraph
codegraph init
```

Typical agent use:

```text
codegraph explore <task or area>
codegraph impact <symbol> --depth 3
git diff --name-only HEAD | codegraph affected --stdin
```

Do not make CodeGraph a build dependency. If it is unavailable, fall back to repository search, Go imports, tests, and the same scope/architecture gates.

## 3. Per-task write scope

Copy `.ai/scope.example.json` to `.ai/scope.json` and list only the paths the task should need. `scopecheck` compares local changes against that allowlist.

The scope file is intentionally ignored by Git in v0: it protects the local agent session without adding task state to the permanent architecture. Hosted agent runners can persist it as a CI artifact if stricter remote enforcement is needed.
