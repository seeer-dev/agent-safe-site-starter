# Single Command Site Build Delivery Plan

Change ID: single-command-site-build
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Repository reality and baseline

| Observation | Evidence | Implication |
|---|---|---|
| Theme bundle is not tracked | `.gitignore` rule `dist/` matches `site/themes/minimal-cart/dist`; `git ls-files` returns nothing for it | A fresh clone has no bundle |
| `make render` renders only | `Makefile` `render` target runs `go run ./server/tools/render` | Fails on a fresh clone |
| Renderer fails closed | `render/render.go:77` `validateThemeAssets` errors when the dist directory or `islands.js` is missing | Correct behavior; the gap is upstream |
| Docs name the renderer alone | `README.md:70` gives `go run ./server/tools/render` as the Pages build command | An operator following it hits the same failure |

## Scope lock

- `Makefile`
- `README.md`
- `specs/changes/single-command-site-build/**`

## Dependency-ordered slices

### Slice 1: Ordered build target

Outcome: a `site` target builds the theme bundle then renders; `render` keeps
its current meaning; a `theme` target exposes the bundle step alone.

Acceptance evidence: with the theme dist directory moved aside, the renderer
alone fails with its existing diagnostic and the new target succeeds; the dist
directory is restored afterwards. Covers REQ-001, AC-001.

Rollback: revert the Makefile; the previous targets are unchanged in meaning.

### Slice 2: Deployment documentation

Outcome: the Pages build command names the ordered command, and the dependency
is stated.

Hard dependencies: Slice 1 defines the command being documented.

Acceptance evidence: the documented command includes the theme build step.
Covers REQ-002, AC-002.

Rollback: revert the documentation only.

## Traceability matrix

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 1 | Absent-bundle run of both commands, then restore |
| REQ-002, AC-002 | 2 | Documented command contains the theme step |

## Risks and controls

- Risk: hiding the ordering inside the Go tool. Control: rejected in scope; the
  ordering is expressed in the build target where it is visible.
- Risk: weakening the renderer guard to make the failure go away. Control:
  `validateThemeAssets` is explicitly out of scope, and AC-001 asserts the
  renderer alone still fails closed.
- Risk: a Pages build image without Node. Control: the documentation states the
  requirement rather than assuming it.
