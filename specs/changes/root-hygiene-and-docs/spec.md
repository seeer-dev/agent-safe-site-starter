# Root Hygiene and Frontend Boundary Specification

Change ID: root-hygiene-and-docs
Revision: 2
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner explicitly approved implementation through Herdr delegation on 2026-08-15.
Repository baseline: 7e5aa90b92d23c8c316d44ca935be7af8d150a1c
Supersedes: none

## Outcome

Remove the dead root Vite entrypoint and make the documented frontend boundaries match the repository that actually builds: static Go-rendered site output, Vue islands inside the selected commerce theme, and a separate Vue admin application.

## Scope

In scope:
- Delete the tracked root `index.html`, which references a nonexistent root `/src/main.ts` and is not used by the renderer, admin Vite root, or theme Vite root.
- Clarify frontend ownership in `AGENTS.md` and `README.md`.

Out of scope:
- Moving or rewriting `INTEGRATION_PLAN.md`.
- Changing `admin/`, `site/`, renderer behavior, build commands, or deployed output.
- Sharing UI components between public and admin surfaces.

## Requirements

### REQ-001: Remove the Dead Root Entrypoint
The repository MUST delete the root `index.html` mock and MUST preserve the real entrypoints under `admin/`, `site/templates/`, and `site/themes/`.

#### AC-001: Real Builds Do Not Depend on Root Index
- GIVEN the root `index.html` references a nonexistent `/src/main.ts`
- WHEN the file is deleted and repository validation runs
- THEN no live build, render, preview, or publish path SHALL fail or reference that deleted entrypoint.

### REQ-002: Document Actual Frontend Ownership
`AGENTS.md` and `README.md` MUST state that plain JavaScript remains the default for the core static site, Vue islands are allowed only inside the selected interactive theme where interaction requires them, and `admin/` is a separate Vue SPA. The documents SHALL preserve the static Cloudflare Pages and single Go backend boundaries.

#### AC-002: Agent Guidance Matches Repository Reality
- GIVEN an agent reads `AGENTS.md` and `README.md` before frontend work
- WHEN it chooses an implementation surface
- THEN the documents MUST distinguish core static assets, theme islands, and the admin SPA without authorizing a site-wide frontend runtime or second backend.

## Amendments

Revision 2 removes the proposed `INTEGRATION_PLAN.md` archive move because its historical references are still live and moving it provides no runtime or maintenance benefit proportional to the churn.
