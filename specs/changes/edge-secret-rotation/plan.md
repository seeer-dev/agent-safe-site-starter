# Edge Secret Rotation Delivery Plan

Change ID: edge-secret-rotation
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| Origin accepts exactly one credential | `bootstrap/edge_auth.go` compares the header against a single `secret` via `ConstantTimeCompare` | Every documented rotation order has a 403 window — Pages-first rejects until Railway updates, Railway-first rejects until Pages redeploys |
| The check is opt-in | `withEdgeAuth` returns the inner handler untouched when `secret == ""` | The accepted set must be built from non-empty values so the empty case still disables the guard |
| Health probe is exempt | `edge_auth.go` skips `/healthz` because `railway.toml` probes it directly | Exemption must survive the multi-value change |
| `CFPagesProject` is dead | `config.go:53,123` loads `CF_PAGES_PROJECT`; no consumer exists anywhere — `tools/publish` is trigger-only and never reads it | Remove field and env read; inventories already dropped the name |
| Existing test discipline | `edge_auth_test.go` covers absent/empty/wrong/near-miss values with identical-body assertions | Extend the same table rather than inventing a parallel test |

## Scope lock

- `server/internal/config/config.go`
- `server/internal/config/config_test.go`
- `server/internal/bootstrap/app.go`
- `server/internal/bootstrap/edge_auth.go`
- `server/internal/bootstrap/edge_auth_test.go`
- `.env.development.example`
- `.env.production.example`
- `docs/environment-configuration.md`
- `docs/deployment-guide.html`
- `README.md`
- `specs/changes/edge-secret-rotation/**`

## Dependency-ordered slices

### Slice 1: Dual-value acceptance

Outcome: `Config` carries `EdgeSecretPrevious` from `EDGE_SECRET_PREVIOUS`;
`withEdgeAuth` admits any non-empty configured value via constant-time
comparison, rejects everything else with the identical 403, and keeps the
`/healthz` exemption and opt-in empty-set behavior. `app.go` passes both
fields.

Edits: `config.go` (field + `os.Getenv`), `edge_auth.go` (candidate set
built from non-empty values; admission iterates candidates), `app.go` (wire
both fields), `edge_auth_test.go` (extend the existing table: both-values
admit, wrong/absent refuse identically, previous-only configuration,
no-secrets pass-through, near-miss disclosure gate).

Acceptance evidence: `edge_auth_test.go` run plus mutation checks (drop the
previous candidate → previous-value test red; drop the empty-set early return
→ opt-in test red). Covers REQ-001, REQ-002, REQ-003, AC-001, AC-002,
AC-003, AC-004. AC-004 additionally needs the `security-review` receipt
recording the no-disclosure assertions.

Rollback: revert the four files; single-value behavior returns.

### Slice 2: Dead config removal

Outcome: `CFPagesProject` field and the `CF_PAGES_PROJECT` read are deleted
from `config.go`; a repo-wide search shows no live consumer.

Hard dependencies: none beyond Slice 1 sharing the file; land after Slice 1
to keep each diff readable.

Acceptance evidence: `go build ./...` clean; search finds no
`CFPagesProject`/`CF_PAGES_PROJECT` outside frozen historical records.
Covers REQ-004, AC-005.

Rollback: revert `config.go`.

### Slice 3: Operator documentation

Outcome: `.env.production.example` lists `EDGE_SECRET_PREVIOUS`;
`docs/environment-configuration.md` adds the ownership row; the deployment
guide's rotation note describes the zero-outage sequence and drops the
brief-outage wording; `.env.development.example` and `README.md` gain the
name only where they already discuss `EDGE_SECRET`.

Hard dependencies: Slices 1–2 fix the exact names and behavior being
documented.

Acceptance evidence: each document names `EDGE_SECRET_PREVIOUS`; the guide's
rotation note describes origin-accepts-both → edge-switches → origin-drops-
old with no outage warning. Covers REQ-005, AC-006.

Rollback: revert the documentation files.

## Traceability matrix

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 1 | Dual-admission table test |
| REQ-002, AC-002 | 1 | Empty-set pass-through test (existing, kept) |
| REQ-002, AC-003 | 1 | Previous-only configuration test |
| REQ-003, AC-004 | 1 | Constant-time + no-disclosure assertions; security-review receipt |
| REQ-004, AC-005 | 2 | `go build ./...`; repo search shows no live reference |
| REQ-005, AC-006 | 3 | Name and procedure present in all three operator surfaces |

## Risks and controls

- Risk: accepting two values doubles the credential surface during the
  window. Control: the window is operator-bounded and ends by unsetting
  `EDGE_SECRET_PREVIOUS`; the guide instructs removing it.
- Risk: a supplied value leaks via a log line added while iterating
  candidates. Control: AC-004 keeps the disclosure assertions and requires
  the security-review receipt, matching `edge-origin-authentication`.
- Risk: removing `CFPagesProject` breaks an unnoticed consumer. Control:
  AC-005 requires a clean build and a repo-wide search; the field has no
  consumer today.
- Risk: `EDGE_SECRET_PREVIOUS` left set forever normalizes two live
  credentials. Control: documentation states it exists only for rotation
  windows and must be removed afterward.
