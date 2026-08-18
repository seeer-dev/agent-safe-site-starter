# Public Endpoint Abuse-Control Decision Plan

Change ID: public-endpoint-rate-limit
Revision: 3
Status: Draft

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `specs/changes/public-endpoint-rate-limit/**`

No product path is authorized by this revision.

## Implementation expansion context

Proposal revision: 2
Proposal status: Draft
Repository baseline: `6a3c113e9d4957617a532ab2d10701a6fbc66db9`
Observed HEAD: `7e5aa90b92d23c8c316d44ca935be7af8d150a1c`
Pre-existing dirty paths:
- Active PostgreSQL/CI/staff verification changes and their controlled artifacts — preserve; unrelated.
- `Makefile` and the architecture-review proposal directories — attributable to this review task, not abuse-control implementation.
Packet ID mapping:
- Slice 1 -> Packet S01

### Packet S01: Select the authoritative enforcement topology

Status: DECISION_REQUIRED
Covers: REQ-001, REQ-002, AC-001, AC-002, AC-003, AC-004
Hard dependencies: none
Outcome: Repository owner selects an abuse-control owner and failure contract that matches real production replicas and trusted proxy behavior.
Safe failure / rollback: No product code is edited until the decision is recorded and controlled scope is revised.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Public entrypoints | `server/internal/bootstrap/app.go` | `POST /api/contact`, `/api/orders`, `/api/orders/mine`, `/api/quote` | Routes currently have no common abuse-control middleware. |
| Existing request middleware | `server/internal/bootstrap/app.go` | `withCORS`, `withAccessLog` | No trusted-proxy or client-identity contract exists. |
| Order replay safety | `server/internal/modules/commerce/service.go` | order idempotency paths | Prevents duplicate order effects for one key but does not bound request volume. |
| Production topology | `architecture.yaml`, `railway.toml` | API production owner | Names Railway but does not state replica count, proxy header guarantees, or edge rate-limit service. |
| Contract | `contracts/openapi.yaml` | listed public operations | No authoritative 429/failure contract is selected. |

Read set:
- `server/internal/bootstrap/app.go`, `server/internal/config/config.go`, `railway.toml`, `architecture.yaml`, `contracts/openapi.yaml`.

Modify set:
- None in revision 2. The selected option must create revision 3 with exact paths.

Must not modify:
- Any product, provider, contract, or config path before the decision.

#### Behavior contract

Before:
- Public requests are unbounded by a common abuse-control policy.
- Replica topology and trusted client-IP source are unspecified.

After the future executable revision:
- One approved owner enforces a deployment-correct policy before side effects.
- 429 and dependency-failure semantics are documented and observable.

Preserved invariants:
- Idempotency remains authoritative for replay correctness.
- Client-controlled headers are not trusted without a verified proxy contract.
- Browser code does not become the enforcement owner.

Inputs and authority:
- Repository owner chooses cost/topology/failure semantics.
- Railway or another approved proxy contract defines trustworthy forwarding metadata.

Errors and edge states:
- Multi-replica bypass, spoofed headers, memory-state exhaustion, and limiter outage must each be decided explicitly.

#### Candidate decisions

1. Platform/edge enforcement: preferred when Railway or an approved gateway provides deployment-wide policy and trustworthy identity metadata.
2. Shared application limiter: requires an approved shared store, cost, latency budget, and fail behavior.
3. Process-local limiter: acceptable only as documented best-effort defense or with an explicit single-replica production constraint; never claimed as global enforcement.

#### Expansion stop

Marker: DECISION_REQUIRED
Claim that cannot be executed: Select implementation paths and prove AC-001/AC-002 across the production topology.
Repository evidence: `architecture.yaml` and `railway.toml` identify Railway but contain no replica count, trusted header contract, rate-limit service, or shared state provider.
Impact: All requirements and acceptance scenarios; no product slice is authorized.
Smallest unblock action: Repository owner chooses one candidate and provides/approves the production proxy and replica contract. Then revise `applies_to`, add a security-review receipt requirement to executable ACs, and expand exact tests.

## Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass |
| Every AC maps to proof in at least one packet | blocked by decision |
| Every packet maps back to REQ/AC | pass |
| Baseline, observed HEAD, dirty paths, and packet mapping are explicit | pass |
| All current repository anchors were inspected | pass |
| Hard dependency graph has no unexplained edge or cycle | pass |
| Scope covers every modify path and no unrelated path | pass; no product edits authorized |
| Product/authority decisions are approved or blocked | DECISION_REQUIRED |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |
