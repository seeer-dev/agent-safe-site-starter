# Evidence

## Delivery Status

Revision 1 is Ready against baseline `f8053f8ea3ac640586384e24667933e5f556b689`. All implementation and runtime evidence remains pending.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | pending | Pending implementation and verification for revision 1. |
| REQ-002 | pending | Pending implementation and verification for revision 1. |
| REQ-003 | pending | Pending implementation and verification for revision 1. |
| REQ-004 | pending | Pending implementation and verification for revision 1. |
| AC-001 | pending | Pending fresh-schema consumer reachability evidence for revision 1. |
| AC-002 | pending | Pending live PostgreSQL staff-lock reachability and independent review for revision 1. |
| AC-003 | pending | Pending live PostgreSQL media-lock reachability and independent review for revision 1. |
| AC-004 | pending | Pending exact named-test gate implementation and negative-path verification for revision 1. |
| AC-005 | pending | Pending observed live CI run, consumer reachability, and independent review for revision 1. |
| AC-006 | pending | Pending verifier wiring regression implementation for revision 1. |
| AC-007 | pending | Pending mutation-sensitive independent review for revision 1. |

## Known Historical Gap

The superseded receipt under `postgres-execution-gate` explicitly stated that live PostgreSQL had not been observed while its AC-002 record was marked passed. Its migration test also ran after the shared CI migration step, and its staff/media lock tests did not observe concurrency semantics. This proposal does not rewrite that immutable history; it replaces the affected evidence claim with falsifiable tests and an observed live-run requirement.
