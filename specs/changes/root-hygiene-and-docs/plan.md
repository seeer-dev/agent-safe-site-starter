# Root Hygiene and Frontend Boundary Plan

Change ID: root-hygiene-and-docs
Revision: 2
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `index.html`
- `AGENTS.md`
- `README.md`
- `specs/changes/root-hygiene-and-docs/**`

## Implementation expansion context

Proposal revision: 2
Proposal status: Accepted
Repository baseline: `7e5aa90b92d23c8c316d44ca935be7af8d150a1c`
Observed HEAD: `7e5aa90b92d23c8c316d44ca935be7af8d150a1c`
Pre-existing dirty paths:
- `.github/workflows/ci.yml`, `server/internal/migrate/postgres_integration_test.go`, `server/internal/modules/media/postgres_integration_test.go`, `server/internal/modules/staff/postgres_integration_test.go`, `server/internal/modules/staff/store.go`, `server/tools/verify/main_test.go`, `server/tools/postgres-live-gate/**`, and their controlled-spec artifacts — preserve; unrelated work owned by other active changes.
- `Makefile` — direct unprotected cleanup from this architecture-review task; outside this proposal.
Packet ID mapping:
- Slice 1 -> Packet S01

### Packet S01: Remove the false entrypoint and align frontend guidance

Status: executable
Covers: REQ-001, REQ-002, AC-001, AC-002
Hard dependencies: none
Outcome: Repository entrypoint documentation names only the static renderer, theme islands, and separate admin SPA; the unused root Vite mock is absent.
Safe failure / rollback: The deletion is independently reversible from Git; documentation edits do not change runtime output.

#### Repository anchors

| Purpose | Path | Symbol or structural anchor | Current evidence |
|---|---|---|---|
| Dead entrypoint | `index.html` | `<script type="module" src="/src/main.ts">` | No root `src/main.ts` exists; admin and theme each own separate Vite roots. |
| Architecture rule | `AGENTS.md` | Hard boundaries / client JavaScript rule | Requires JS only for interaction but does not name the already-existing theme/admin exceptions. |
| Operator map | `README.md` | Default path, Repository map, Intentional v0 limits | Describes static-first output and admin/theme behavior but does not state their frontend ownership boundary together. |
| Real admin entrypoint | `admin/index.html` | `/src/main.ts` | Vite root scoped to `admin/`; must remain unchanged. |
| Real public sources | `site/templates/**`, `site/themes/minimal-cart/**` | Go templates and island bootstrap | Renderer consumes templates and built theme assets; neither consumes root `index.html`. |

Read set:
- `server/tools/dev/main.go`, `server/tools/render/main.go`, `server/internal/render/render.go` — confirm live render/preview entrypoints.
- `admin/vite.config.ts`, `site/themes/minimal-cart/vite.config.ts` — confirm separate Vite roots.

Modify set:
- `index.html` — delete only.
- `AGENTS.md` — clarify the existing frontend boundary without changing backend or deployment rules.
- `README.md` — align the repository map and frontend guidance.

Must not modify:
- `admin/**`, `site/**`, `server/**` — no runtime or build behavior change.
- `INTEGRATION_PLAN.md` and `skills/**` — revision 2 explicitly drops archival work.

#### Behavior contract

Before:
- A tracked root Vite mock points to a nonexistent source file and can be mistaken for a deployed entrypoint.
- Frontend rules do not explicitly distinguish core site assets, theme islands, and admin SPA.

After:
- Only real entrypoints remain discoverable.
- Documentation states the three frontend ownership boundaries without authorizing SSR, a second backend, or browser database access.

Preserved invariants:
- Cloudflare Pages stays static.
- Go remains the only application backend.
- Admin and selected theme keep their current independent builds.

Inputs and authority:
- `architecture.yaml`, live Vite configs, renderer code, and repository owner approval define the boundary.

Errors and edge states:
- If any live command references root `index.html`, deletion must stop and be reported as `SPEC_DRIFT` rather than changing that command opportunistically.

#### Ordered edits

1. `index.html` — root file
   - Change: delete the unused mock.
   - Constraint: do not delete or edit `admin/index.html` or theme templates.
   - Result: no false root Vite entrypoint remains.
2. `AGENTS.md` — Hard boundaries
   - Change: add the three-surface frontend ownership rule adjacent to the existing client-JS rule.
   - Constraint: retain static Pages, one-Go-backend, and no-direct-database rules verbatim.
   - Result: future agents choose the correct surface.
3. `README.md` — architecture/repository guidance
   - Change: describe static core assets, theme islands, and admin SPA consistently.
   - Constraint: do not claim a new deployment topology or framework.
   - Result: operator documentation matches repository reality.

#### Integration trace

```text
repository operator or agent
  -> AGENTS.md / README.md guidance
  -> server/tools/render and admin/theme Vite roots
  -> dist/ static output and separate admin build
  -> Cloudflare Pages / browser
```

The root `index.html` is not present in this trace and therefore is safe to remove if the negative searches remain empty.

#### Verification matrix

| Claim | Working directory | Exact argv or walkthrough | Selected assertion/artifact | Expected result | Negative case | Failure trigger | Restoration/residue check |
|---|---|---|---|---|---|---|---|
| Root mock has no consumer | repository root | `rg -n "index\\.html|/src/main\\.ts" Makefile Dockerfile railway.toml server site admin .github README.md` | Search result inspection | No live path points to the root mock; `admin/index.html` references remain scoped to admin. | A live command names root `index.html`. | Temporarily reintroduce a reference in a scratch copy of the search corpus and confirm the negative check detects it. | Remove scratch mutation and confirm `git diff --check`. |
| Architecture remains valid | repository root | `go run ./server/tools/archcheck` | `archcheck: ok` | Exit 0. | Import boundary regression. | Existing archcheck is established proof; no new architecture assertion is introduced. | No temporary mutation. |
| Controlled scope is valid | repository root | `go run ./server/tools/speccheck` and `go run ./server/tools/scopecheck` | Both gates report ok. | Exit 0. | Uncovered protected path. | Add then remove an out-of-scope scratch path only if needed to prove scopecheck during apply. | `git status --short` contains no scratch path. |
| Repository remains buildable | repository root | `go run ./server/tools/verify` | Full verifier terminal result. | Exit 0, subject to preserving unrelated dirty work. | Any live build depended on the deleted file. | Deletion itself is the bounded trigger: validation must fail if the file was actually required. | Restore only if dependency is observed; otherwise retain deletion. |

#### Completion gate

- [ ] Every ordered edit is inside the modify set and controlled scope.
- [ ] Before/after behavior and preserved invariants match the approved spec.
- [ ] Every mapped AC has a selected assertion or observable artifact.
- [ ] Existing real entrypoints remain unchanged.
- [ ] No temporary mutation or generated residue remains.
- [ ] Expected evidence has been replaced with attributable observed evidence after apply.

## Blueprint-wide gates

| Check | Result |
|---|---|
| Every REQ maps to at least one packet | pass |
| Every AC maps to proof in at least one packet | pass |
| Every packet maps back to REQ/AC | pass |
| Baseline, observed HEAD, dirty paths, and packet mapping are explicit | pass |
| All named modify paths and repository anchors were inspected | pass |
| Hard dependency graph has no unexplained edge or cycle | pass |
| Scope covers every modify path and no unrelated path | pass |
| Product/authority decisions are approved or blocked | pending repository owner approval |
| No agent/provider/model identity changes packet semantics | pass |
| No expected result is presented as observed evidence | pass |
