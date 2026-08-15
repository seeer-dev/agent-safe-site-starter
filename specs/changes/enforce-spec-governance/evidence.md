# Enforce spec governance evidence

| REQ/AC | Status | Proof |
|---|---|---|
| REQ-001 | passed | protected-path coverage tests passed; missing-control CLI test rejected uncovered paths |
| REQ-002 | passed | go test ./server/tools/speccheck -count=1: passed, including CRLF spec-heading regression |
| REQ-003 | passed | go test ./... -count=1 and go vet ./... passed; verifier wiring inspection passed |
| REQ-004 | passed | governance protected-path and wiring tests: passed |
| REQ-005 | passed | README and workflows/safe-change.md two-step workflow contract inspection passed; manual skill frontmatter/reference validation passed |
| REQ-006 | passed | contradictory-proof, revision-binding, required-receipt, missing-file, unsafe-path, and valid-receipt tests passed; receipt receipts/revision-3-validation.md |
| REQ-007 | passed | consumer reachability and protected fail-closed contracts inspected; TestSiteSkillRequiresReachabilityRecoveryReceiptsAndIndependentReplay passed; receipt receipts/revision-3-validation.md |
| REQ-008 | passed | auth recovery possession-proof and threat-model matrix inspected; receipt receipts/revision-3-validation.md |
| REQ-009 | passed | repository documentation and routed references consistently require production-claim audits, structured walkthroughs, and independent replay; receipt receipts/revision-3-validation.md |
| AC-001 | passed | missing-control CLI test: rejected with uncovered protected paths |
| AC-002 | passed | TestCoveringChangesRequiresOneAuthorizingSpec: passed |
| AC-003 | passed | TestValidateControlRejectsArtifactDrift and TestValidateControlAcceptsCRLFSpecHeadings: passed |
| AC-004 | passed | TestValidateControlRejectsAcceptedWithoutPassedEvidence: passed |
| AC-005 | passed | verifier wiring inspection passed; go test ./... -count=1 and go vet ./... passed |
| AC-006 | passed | TestGovernanceWiringKeepsSpeccheckInVerifyAndCI: passed; CI base wiring inspected |
| AC-007 | passed | TestRequiresControlledSpecProtectsImplementationAndGovernance: passed |
| AC-008 | passed | README and safe-change workflow inspection: propose summary and hidden-control contract passed |
| AC-009 | passed | README and safe-change workflow inspection: plain apply, all-slices, walkthrough, and internal-validation contract passed |
| AC-010 | passed | safe-change workflow inspection: internal discovery and material-decision stop boundary passed |
| AC-011 | passed | TestValidateControlRejectsContradictoryPassedProof and negative-language regression test passed |
| AC-012 | passed | strict revision and receipt validation tests passed; receipt receipts/revision-3-validation.md |
| AC-013 | passed | reachable-consumer contract and call-site/session-producer checks inspected; receipt receipts/revision-3-validation.md |
| AC-014 | passed | authoritative empty and permission/network failure contracts prohibit browser-local fallback; routed-reference test passed |
| AC-015 | passed | recovery possession-proof, lifetime, abuse-control, rotation, audit, and failure matrix inspected; receipt receipts/revision-3-validation.md |
| AC-016 | passed | approved-source plus source/fresh-build/fresh-render audit contract inspected; receipt receipts/revision-3-validation.md |
| AC-017 | passed | walkthrough receipt path, revision, completeness, and proof-link validation tests passed; receipt receipts/revision-3-validation.md |
| AC-018 | passed | independent CRLF parser diff inspection and command replay recorded in receipt receipts/crlf-parser-review.md |
