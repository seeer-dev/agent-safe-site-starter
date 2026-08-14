# Backend Verification Discipline Specification

Change ID: backend-verification-discipline
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved implementation on 2026-08-14 after reviewing the proposed global, agent-specific, and server-local rule split.
Repository baseline: e30f583c00c9ceecbbdf856016c2a01fab9c3cd2
Supersedes: none

## Outcome

Add backend-local verification rules that prevent evidence-shaped but ineffective tests without placing Go or PostgreSQL mechanics into global agent policy. The rules must constrain test deletion, silent skips, branch claims, concurrency/locking claims, wiring assertions, and self-review while preserving the repository's existing controlled-change workflow.

## Scope

In scope:
- A nearest-scope `server/AGENTS.md` inherited by work under `server/`.
- Backend-specific requirements for test inventory preservation, red-before-green evidence, exact required-test execution, deterministic concurrency overlap, database lock semantics, and receipt honesty.

Out of scope:
- Product code, tests, CI workflows, verification tools, migrations, and existing accepted specifications.
- Global Grok/Agy user configuration, which is outside this repository and is maintained separately under the user's explicit instruction.

## Requirements

### REQ-001: Falsifiable Backend Evidence
Backend implementation and verification work MUST preserve existing tests and MUST demonstrate that new safety or governance checks can fail for the defect they claim to detect.

#### AC-001: Test Inventory Preservation
- GIVEN a backend change that edits tests
- WHEN the final diff is reviewed
- THEN removed or weakened test cases MUST be explicitly identified and justified by an approved behavior change rather than silently disappearing.

#### AC-002: Red-Before-Green Evidence
- GIVEN a new regression, security, governance, branch, or integration check
- WHEN acceptance evidence is recorded
- THEN a bounded temporary failure trigger MUST have made the targeted check fail before the restored implementation passes.

### REQ-002: Honest Execution and Concurrency Claims
Backend evidence MUST distinguish reachable code from observed semantics and MUST reject missing, skipped, sequential-only, or substring-only substitutes for required execution.

#### AC-003: Required Test Execution
- GIVEN a required integration or acceptance test set
- WHEN it runs in CI or another required environment
- THEN the exact expected tests MUST be observed as executed and passed, with missing or skipped tests treated as failure.

#### AC-004: Semantic Concurrency and Wiring Proof
- GIVEN a claim about concurrency, database locking, skip-locked behavior, or command wiring
- WHEN evidence is accepted
- THEN it MUST prove the claimed overlap/outcome or exact command structure; sequential execution or unrelated substring presence MUST NOT satisfy the claim.

## Amendments

None.
