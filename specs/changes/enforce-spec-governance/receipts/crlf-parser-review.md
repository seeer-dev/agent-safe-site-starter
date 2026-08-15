# CRLF parser independent-review receipt

Controlled change: `enforce-spec-governance` revision 3
Reviewed state: working tree based on `adae154`
Review date: 2026-08-15
Secrets or raw PII: none

## Scope

- Reviewed files: `server/tools/speccheck/main.go`, `server/tools/speccheck/main_test.go`, and this controlled change's plan/evidence artifacts.
- Intent: make `speccheck` parse REQ and AC headings in a Windows CRLF checkout without weakening its heading grammar.

## Independent replay

- Diff inspection: `\r?$` is permitted only at a heading line end; heading text remains constrained by `[^\r\n]+` and cannot span lines.
- Test inventory: all prior 19 `Test*` functions remain; one direct CRLF fixture regression test was added.
- `go test ./server/tools/speccheck -count=1`: passed.
- `go test ./server/tools/speccheck -run '^TestValidateControlAcceptsCRLFSpecHeadings$' -count=1 -v`: passed.
- Clean temporary worktree check: controlled Markdown had `w/crlf`; after applying the parser fix, `go run ./server/tools/speccheck` passed with `speccheck: ok (9 controlled spec(s), 1 protected changed file(s))`.

## Negative and external-state evidence

- Mutation: removing the optional line-ending CR made `TestValidateControlAcceptsCRLFSpecHeadings` fail with missing `REQ-001` and `AC-001`; the exact parser fix was restored before final validation.
- The mixed root worktree's `speccheck` rejects only `.github/workflows/ci.yml`, because two separately active changes currently declare that file. This CRLF fix does not modify the workflow; the overlap is external dirty work, not a parser defect.

## Result

No blocking defect found. The CRLF parser correction and its regression test are accepted for this controlled change.
