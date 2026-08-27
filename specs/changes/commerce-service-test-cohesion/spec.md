# Commerce Service Test Cohesion Specification

Change ID: commerce-service-test-cohesion
Revision: 1
Status: Accepted
Decision authority: Repository owner
Approval basis: Owner approved continued commerce cohesion refactoring on 2026-08-27; this slice is test-only and may be prepared before full CI quota is available, but must not merge until repository verification succeeds.
Repository baseline: 8c93e03f6f0721f7be061fb539d50e3a7cf2c27e
Supersedes: none

### REQ-001: Test behavior remains unchanged while ownership moves

The implementation MUST preserve existing test bodies, shared fixtures, helper semantics, and production behavior while relocating order-domain tests into cohesion-oriented files inside `package commerce`.

#### AC-001: Test declarations are relocated without semantic edits

- GIVEN the existing commerce service tests execute against shared package-local fixtures and helpers
- WHEN order-domain `Test*` declarations are moved into dedicated files
- THEN moved test bodies and associated assertions remain unchanged, shared helpers remain available package-wide, and the commerce package still compiles

### REQ-002: Test ownership follows commerce service responsibilities

The implementation MUST reduce the central `service_test.go` ownership surface by colocating checkout, order, return, and restock tests with the corresponding service responsibilities.

#### AC-002: The test-only split introduces no runtime contract change

- GIVEN the production service cohesion split is already merged
- WHEN its tests are reorganized by responsibility
- THEN no production source, route, API, Store contract, SQL, migration, provider, authorization, state-machine, idempotency, or UI behavior changes
