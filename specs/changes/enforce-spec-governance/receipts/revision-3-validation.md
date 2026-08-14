# Revision 3 governance validation receipt

Source revision: working tree based on `7c45b616fbe3a632ffe2a39d872c98485466c991`  
Controlled change: `enforce-spec-governance` revision 3  
Validation date: 2026-08-12  
Secrets or raw PII: none

## Consumer reachability contract

- Surface: `skills/site/references/integration-planning.md`
- Expected: definitions are not treated as consumers; evidence traces a real entry point, actual identity/session/configuration producer, call site, authoritative owner, and visible success/empty/permission/failure states.
- Observed: the reference contains the complete reachability chain, definition-versus-invocation checks, unreachable-auth handling, and pending-evidence rule.
- Check: `TestSiteSkillRequiresReachabilityRecoveryReceiptsAndIndependentReplay` passed.

## Security recovery contract

- Surface: `skills/site/references/auth.md`
- Expected: identifying/contact data alone cannot authorize recovery; the review covers possession proof, expiry, single use, replay, rate limiting, enumeration, atomic rotation, audit, delivery, and safe failure.
- Observed: the recovery matrix and stop condition require those decisions plus a `security-review` receipt before mapped evidence passes.
- Check: targeted `server/tools/speccheck` tests and full `go test ./... -count=1` passed.

## Production content audit contract

- Surface: `skills/site/SKILL.md` and `skills/site/references/integration-planning.md`
- Expected: production claims trace to a published source or reviewed allowlist; acceptance inspects source, a fresh client build, and freshly rendered `dist/`.
- Observed: both routed instructions require the approval inventory and reject a self-selected keyword subset as sufficient proof.
- Check: routed-reference inspection and manual skill validation passed.

## Walkthrough receipt enforcement

- Surface: `skills/site/references/user-walkthrough.md` and `server/tools/speccheck`
- Expected: runtime acceptance names a non-secret receipt bound to the current revision; missing, unsafe, empty, or unmentioned receipts fail.
- Observed: strict evidence supports required receipt kinds and safe in-change paths.
- Checks: current-receipt, missing-receipt, unsafe-path, and required-kind tests passed.

## Independent replay

- Implementer report used as sole evidence: no.
- Repository-facing documentation: inspected `README.md` and `workflows/safe-change.md` for the two-step operator contract, strict revision/receipt rules, reachable-consumer boundary, protected fail-closed behavior, recovery possession proof, fresh production-output audit, structured walkthrough receipt, and provisional implementer-report rule. The stale `docs/architecture.md` pointer was removed because that file is absent from the current repository shape.
- Replay: inspected the actual diff; ran repository-documentation contract assertions, `go test ./server/tools/speccheck -count=1`, `go test ./... -count=1`, `go vet ./...`, `gofmt -l server`, `go run ./server/tools/scopecheck`, manual skill frontmatter/reference validation, and `git diff --check`.
- Result: all listed checks passed. The Python `quick_validate.py` helper could not run because this Windows environment exposes only the Microsoft Store Python alias and no Python runtime; the equivalent frontmatter/name/description/reference checks were replayed in PowerShell.
- External-state note: the repository-wide `speccheck` now correctly rejects contradictory evidence in the separate active `minimal-cart-integration` change. That receiving change must repair its own evidence before the full verifier can pass.
