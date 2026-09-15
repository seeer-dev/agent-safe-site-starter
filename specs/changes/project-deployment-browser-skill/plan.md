# Project Deployment Browser Skill Plan

Change ID: project-deployment-browser-skill
Revision: 1
Status: Accepted

Normative specification: [`spec.md`](spec.md)

## Scope lock

- `skills/site-deployment-browser/**`
- `docs/deployment-guide.html`
- `specs/changes/project-deployment-browser-skill/**`
- `.ai/scope.json`

## Slices

### Slice 1: Project-local skill

Create a concise `SKILL.md` under `skills/site-deployment-browser/` covering:
isolated controllable browser setup, user handoff for login/MFA/CAPTCHA,
safe screenshot rules, short-lived secret transfer, provider value ownership,
confirmation gates, and completion reporting.

Covers REQ-001, REQ-002, AC-001, AC-002.

### Slice 2: HTML guide runbook

Add an agent-assisted runbook to `docs/deployment-guide.html` that lists each
provider, dashboard entry point, values to retrieve, destination, confirmation
points, and verification checkpoint.

Covers REQ-003, AC-003.

### Slice 3: Evidence and security review

Run controlled gates and record a defensive security review confirming no
secret persistence or unsafe browser automation instruction was introduced.

Covers all acceptance IDs.

## Traceability

| REQ / AC | Slice | Verification |
|---|---|---|
| REQ-001, AC-001 | 1 | Inspect project-local skill path and content |
| REQ-002, AC-002 | 1, 3 | Security review and content inspection |
| REQ-003, AC-003 | 2 | Inspect deployment guide runbook |

## Risks and controls

- Risk: a browser walkthrough captures secrets or personal tabs. Control:
  require isolated/explicitly claimed tabs and secret-free screenshots.
- Risk: a dashboard mutation occurs without user confirmation. Control:
  list confirmation gates directly in the skill and guide.
- Risk: production dotenv files are created. Control: state that production
  values belong in provider dashboards, not repository files.
