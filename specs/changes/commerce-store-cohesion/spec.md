# Commerce Store Cohesion Specification

Change ID: commerce-store-cohesion
Revision: 1
Status: Accepted
Decision authority: Repository owner
Approval basis: Owner approved continued Phase 2 commerce cohesion refactoring on 2026-08-27 with behavior-preserving slices merged to main after verification.
Repository baseline: 7d44ebb42928ab13a8fa28864c1fa054e803b473
Supersedes: none

### REQ-001: Store contract and core composition remain stable

The implementation MUST keep the existing `Store` contract, `SQLStore` type, constructor, package ownership, and caller-visible persistence method signatures stable while relocating implementation bodies by cohesion.

#### AC-001: Persistence contracts remain behavior-compatible

- GIVEN existing commerce services and tests depend on the current `Store` and `SQLStore` contracts
- WHEN persistence implementations are split into cohesion-oriented files
- THEN method signatures, SQL statements, error behavior, idempotency behavior, and transaction semantics remain unchanged and no subinterface split is introduced

### REQ-002: Persistence implementations are colocated by commerce responsibility

The implementation MUST keep persistence inside `package commerce` while colocating catalog, member, order, promotion, payment-method, and shipping-method SQLStore behavior into dedicated files.

#### AC-002: Structural split introduces no product or storage contract change

- GIVEN the existing commerce persistence implementation is already grouped by responsibility inside `store.go`
- WHEN those groups are moved into dedicated files
- THEN no route, API payload, database schema, migration, provider, UI, authorization, idempotency, or transaction-semantic change is introduced
