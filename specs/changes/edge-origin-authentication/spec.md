# Edge-to-Origin Authentication Specification

Change ID: edge-origin-authentication
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved on 2026-08-18 after review established that the Go API performs no edge-to-origin authentication, so any edge protection can be bypassed by reaching the origin directly.
Repository baseline: d3896491a9ccf43cfdaa8bce99a88d806665d91c
Supersedes: none

## Outcome

The origin can refuse requests that did not pass through the edge, so edge
protection stops depending on the origin being undiscoverable.

This is a precondition for edge-based abuse control, not abuse control itself.
No rate limiting is added or implied.

## Scope

In scope:

- A shared-secret header check in front of the request handlers.
- The configuration that carries the secret and the operator documentation.

Out of scope:

- Rate limiting, quotas, and any decision about enforcement owner. Those remain
  with `public-endpoint-rate-limit`.
- Deriving a trusted client identity. The header proves the hop, not the caller.
- mTLS, Authenticated Origin Pull, and Cloudflare Tunnel. Each is a deployment
  choice this change neither requires nor forecloses.

## Decisions and invariants

- The check is **opt-in**. With no secret configured the API behaves exactly as
  before, so local development, tests, and existing deployments are unaffected
  until an operator opts in.
- The secret proves a request traversed the edge. It is not an identity, not a
  capability, and must never influence authorization.
- Comparison is constant-time. A byte-by-byte early exit leaks the secret to a
  patient caller.
- The health check stays reachable, because the platform probes it directly
  rather than through the edge.
- A rejection tells the caller nothing about why.

## Requirements

### REQ-001: The origin can reject non-edge traffic

When a shared secret is configured, a request that does not present it MUST be
rejected before reaching any handler.

#### AC-001: Missing or wrong secret is refused

- GIVEN a configured edge secret
- WHEN a request arrives with no secret header, an empty one, or a wrong value
- THEN it MUST receive 403 with a generic body, no handler MUST run, and the
  response MUST NOT reveal whether the header was absent, empty, or merely
  incorrect.

### REQ-002: The check is opt-in and cannot be enabled by accident

Absent configuration MUST leave behavior unchanged, and the platform health
probe MUST remain reachable when the check is active.

#### AC-002: No secret means no change

- GIVEN no edge secret is configured
- WHEN any request arrives, with or without the header
- THEN it MUST be served exactly as before this change.

#### AC-003: Health probes bypass the check

- GIVEN a configured edge secret
- WHEN the platform requests the health path without the header
- THEN it MUST still succeed, because the probe reaches the origin directly and
  a failing probe would take the deployment down.

### REQ-003: The secret is compared and logged safely

Comparison MUST be constant-time, and the secret MUST never appear in a
response, a log record, or a diagnostic.

#### AC-004: No timing or disclosure leak

- GIVEN a configured edge secret and a request carrying a wrong value
- WHEN the comparison runs and the rejection is recorded
- THEN the comparison MUST use a constant-time primitive, and neither the
  configured secret nor the supplied value may appear in the response body or
  in any emitted log record.

## Amendments

None.
