# Edge-to-Origin Authentication Delivery Plan

Change ID: edge-origin-authentication
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

Repository baseline: `d3896491a9ccf43cfdaa8bce99a88d806665d91c`

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| No edge-to-origin check exists | `CF-Connecting-IP`, `Origin-Secret`, and mTLS appear nowhere under `server/internal` | The origin serves anyone who reaches it |
| The middleware chain is two deep | `bootstrap/app.go:206` wraps only observability and CORS | A third wrapper is the whole change |
| The health path is probed directly | `railway.toml` sets `healthcheckPath = "/healthz"` | The probe cannot carry an edge header |
| Diagnostic redaction already exists | `httpx.redactDiagnostic`, `auth.WriteError` | Reuse the posture rather than inventing one |

## Scope lock

- `server/internal/config/config.go`
- `server/internal/bootstrap/app.go`
- `server/internal/bootstrap/edge_auth.go`
- `server/internal/bootstrap/edge_auth_test.go`
- `.env.development.example`
- `.env.production.example`
- `docs/environment-configuration.md`
- `specs/changes/edge-origin-authentication/**`

## Dependency-ordered slices

### Slice 1: Opt-in origin guard

Outcome: a middleware rejects requests lacking the configured secret, exempts
the health path, and is inert when no secret is set.

Acceptance evidence: table tests over absent, empty, wrong, and correct header
values, both with and without a configured secret, plus the health path in each
mode. Covers REQ-001, REQ-002, AC-001, AC-002, AC-003.

Rollback: remove the wrapper; the chain returns to its previous two layers.

### Slice 2: Safe comparison and disclosure gate

Outcome: comparison uses `crypto/subtle`, and a test asserts neither the
configured secret nor a supplied wrong value reaches the response or a log
record.

Hard dependencies: Slice 1 provides the rejection path under test.

Acceptance evidence: the disclosure assertion observed failing when the value
is deliberately routed into the record, then passing after restoration. Covers
REQ-003, AC-004.

Rollback: the gate is additive.

### Slice 3: Operator documentation

Outcome: the setting appears in both examples and the ownership table, stating
that it authenticates the hop rather than the caller.

Hard dependencies: Slices 1 and 2 fix the name and semantics.

Acceptance evidence: the name appears in all three documents. Covers REQ-002.

Rollback: revert the documentation only.

## Traceability matrix

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 1 | Absent, empty, and wrong header all rejected identically |
| REQ-002, AC-002 | 1 | Unconfigured mode serves every case unchanged |
| REQ-002, AC-003 | 1 | Health path reachable while the guard is active |
| REQ-003, AC-004 | 2 | Constant-time primitive; red-before-green disclosure gate |

## Risks and controls

- Risk: the secret is mistaken for a client identity and used in authorization.
  Control: the specification states it proves the hop only, and no requirement
  here consumes it for a decision.
- Risk: enabling it takes the deployment down through failed health probes.
  Control: AC-003 exempts the health path and asserts it.
- Risk: a timing side channel. Control: `crypto/subtle` and an explicit AC.
- Risk: it is read as abuse control. Control: the outcome states it is a
  precondition, and rate limiting stays with its own proposal.
