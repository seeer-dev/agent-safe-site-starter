# Public Endpoint Abuse-Control Specification
| 3 | none | Proposal targeted baseline 7e5aa90 | Proposal targets baseline 3ecc1b6 | Seventeen commits landed since revision 2, including request observability, which now records the transport peer and an X-Forwarded-For claim. Those are diagnostic only and nothing consumes them for a decision, so they do not supply the trusted client identity AC-002 requires. | Owner instruction to refresh draft baselines on 2026-08-18 | No evidence existed to invalidate; all records remain pending. |
| 3 | scope | — | unchanged, deliberately | Revision 2 withdrew the implementation assumption, and the surface still depends on four unresolved owner decisions: replica topology, trusted client-IP source, enforcement owner, and acceptable operating cost. Edge enforcement would touch no Go file; shared-state in-app enforcement would add a dependency and middleware. Naming a surface now would pre-decide the topology. | same | — |

Change ID: public-endpoint-rate-limit
Revision: 3
Status: Draft
Decision authority: Repository owner/user
Approval basis: Pending repository owner decision on production replica topology, trusted client-IP source, enforcement owner, and acceptable operating cost.
Repository baseline: 3ecc1b63b85f13be773af79453a2c29f868903e4
Supersedes: none

## Outcome

Protect public mutation and expensive quote endpoints from unbounded automated abuse without trusting spoofable forwarding headers or pretending a process-local limiter is a deployment-wide security boundary.

## Scope

This revision records normative behavior and the unresolved trust-boundary decision only. It does not authorize product edits.

Candidate protected surfaces after the decision may include Go middleware/configuration, OpenAPI, operator documentation, or an approved external edge/gateway configuration. The next revision must narrow `applies_to` to the selected owner.

Out of scope:
- Assuming Railway runs exactly one replica.
- Trusting arbitrary `X-Forwarded-For` values.
- Adding Redis or another paid provider without owner approval.
- Treating idempotency as a substitute for abuse controls.

## Requirements

### REQ-001: Deployment-Wide Abuse Enforcement
The selected control MUST enforce one authoritative policy across the production API topology for `POST /api/contact`, `POST /api/orders`, `POST /api/orders/mine`, and `POST /api/quote`. A process-local limiter SHALL be described only as best-effort unless production is contractually single-replica.

#### AC-001: Replica Bypass Is Not Accepted
- GIVEN requests are distributed across every production API replica
- WHEN one caller exceeds the approved policy
- THEN the selected enforcement owner MUST reject excess traffic consistently rather than granting a fresh bucket per process.

#### AC-002: Client Identity Comes From a Trusted Boundary
- GIVEN a request may contain spoofed forwarding headers
- WHEN the abuse-control key is derived
- THEN only a repository-documented trusted proxy chain or provider-issued identity SHALL influence the key.

### REQ-002: Safe Endpoint and Failure Semantics
Excess requests MUST receive a documented response, normally HTTP 429 with Retry-After, before contact mail or order side effects occur. Limiter unavailability and state exhaustion MUST have explicit fail-open or fail-closed behavior approved per endpoint.

#### AC-003: Side Effects Do Not Run Over Quota
- GIVEN the authoritative policy reports a request over quota
- WHEN the request targets contact or order creation
- THEN no mail send, stock mutation, order insert, or idempotency record MUST occur.

#### AC-004: Failure Mode Is Explicit
- GIVEN the selected enforcement dependency is unavailable or its state bound is exhausted
- WHEN a protected request arrives
- THEN behavior MUST follow the approved endpoint-specific failure policy and MUST be observable without leaking caller data.

## Amendments

Revision 2 withdraws the previous in-memory per-IP implementation assumption. Repository evidence does not establish a single-replica Railway topology or a trustworthy client-IP header contract.
