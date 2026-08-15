# Expand implementation skill evidence

Change ID: expand-implementation-skill
Revision: 1
Status: Accepted

Revision 1 was implemented and independently replayed against two materially different controlled proposals.

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | skills/expand-implementation/SKILL.md defines expansion only from controlled artifacts and repository evidence, explicitly excludes implementer selection and identity branching, and a specific-name scan of the new skill returned no matches. |
| REQ-002 | passed | The skill and blueprint-format reference require expansion context, stable packet mapping, inspected anchors, read/modify boundaries, behavior contracts, ordered symbol-level work, integration traces, drift stops, and completion gates; both independent replays produced repository-grounded packets. |
| REQ-003 | passed | The verification matrix requires working directory, exact argv, selected assertion or artifact, expected result, negative case, bounded failure trigger, and restoration/residue check. The targeted governance test was observed red after mutating the required SPEC_DRIFT marker and green after exact restoration. |
| REQ-004 | passed | skills/site and README route non-trivial proposal preparation through the new skill while retaining the existing plan.md and propose/plain apply lifecycle; the complete repository verifier passed. |
| AC-001 | passed | The skill states that blueprint semantics cannot vary by implementer identity, provider, model, or role; scans found no specific agent or provider names, and both forward-test contexts used the same mandatory format. |
| AC-002 | passed | Independent admin and PostgreSQL replays produced locally executable representative packets containing the required context, anchors, contracts, ordered work, boundaries, traces, proof, and completion fields; recorded in receipts/independent-review-rev1.md. |
| AC-003 | passed | The skill defines SPEC_DRIFT, DECISION_REQUIRED, and ENVIRONMENT_BLOCKED with claim, evidence, impact, and smallest unblock action. The PostgreSQL replay classified absent live and CI proof explicitly without substituting another database or mock. |
| AC-004 | passed | The mandatory verification matrix carries exact command context, selected proof, negative behavior, bounded mutation, and restoration. Mutating SPEC_DRIFT to SPEC_DRIFT_MUTATED made TestSiteSkillRoutesAgentNeutralImplementationExpansion fail for the intended assertion; restoration returned it to pass with no marker residue. |
| AC-005 | passed | workflow-usage.md keeps expansion inside proposal preparation, writes to existing plan.md, labels it not a third lifecycle action, and retains plain apply as the implementation approval. TestSiteSkillRoutesAgentNeutralImplementationExpansion passed. |
| AC-006 | passed | Two isolated read-only reviewers replayed revision 1 against admin-configurable-api-base and postgres-lock-semantics-and-evidence. Their packets exercised browser configuration/consumer reachability and backend CI/concurrency evidence, exposed format omissions that were repaired, and confirmed the revised skill transfers without model-specific guidance; receipt receipts/independent-review-rev1.md. |
