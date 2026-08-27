# Commerce Service Test Cohesion Evidence

Change ID: commerce-service-test-cohesion
Revision: 1
Status: Accepted

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | Only Test* declarations are relocated; shared fixtures and helper declarations remain available in package commerce, production files are untouched, and the relocation job completed a commerce package compile probe successfully. |
| AC-001 | passed | The AST-based relocation preserves each moved test function body and associated doc comment while deriving file-local imports from actual identifier usage. |
| REQ-002 | passed | Checkout, order, return, and restock tests are colocated in dedicated files while the central service_test.go shrinks to shared infrastructure plus remaining non-order commerce tests. |
| AC-002 | passed | The final diff changes test files only and introduces no production code, route, API, Store, SQL, migration, provider, authorization, state-machine, idempotency, or UI change. |
