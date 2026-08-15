# Evidence

## Delivery Status

Revision 2 was independently reviewed and accepted on 2026-08-15. Shared-worktree gates remain non-attributable because other active changes overlap there; the controlled change was validated in an isolated worktree with the same root-change diff.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Independent review confirmed root index.html is absent, while admin/index.html remains the scoped Vite entrypoint. The post-apply isolated full verifier passed with the root changes present. |
| REQ-002 | passed | Independent review confirmed AGENTS.md and README.md distinguish static core assets, selected-theme Vue islands, and the separate admin Vue SPA while retaining static Pages, one Go backend, and no browser database access. |
| AC-001 | passed | The post-apply isolated go run ./server/tools/verify passed with 2 protected changed files covered by this change; targeted renderer tests and go run ./server/tools/render also passed, producing dist/index.html and dist/_headers. |
| AC-002 | passed | Independent diff review found only the authorized root entrypoint deletion and the AGENTS.md/README.md boundary clarification; admin typecheck/build and the selected theme OpenAPI contract check passed. |

## Independent Validation

```text
go run ./server/tools/verify                                      passed in isolated worktree
go test ./server/internal/render/... ./server/tools/render/...    passed
go run ./server/tools/render                                      passed; inspected dist/index.html and dist/_headers
cd admin && npm run typecheck && npm run build                    passed
cd site/themes/minimal-cart && npm run check:openapi-contracts   passed
```
