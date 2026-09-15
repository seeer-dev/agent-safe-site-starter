# Project Deployment Browser Skill Specification

Change ID: project-deployment-browser-skill
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: User requested a project-scoped deployment browser skill and updated docs/deployment-guide.html on 2026-09-15, including authenticated dashboard operation, value retrieval, safe screenshots, and environment-variable placement.
Repository baseline: 862ea46b0fa22ff507ed887660bda187c86bf6f7
Supersedes: none

## Outcome

This repository has its own deployment-browser skill for the concrete
agent-safe-site-starter deployment path, and the HTML deployment guide tells an
agent/operator which provider dashboards to open, which values to retrieve, and
where those values belong.

The workflow supports authenticated dashboards after a user performs sign-in,
MFA, and CAPTCHA steps. It must not persist or expose secrets in the
repository, screenshots, chat, or reports.

## Scope

In scope:

- A project-local `skills/site-deployment-browser/SKILL.md`.
- Deployment guide updates in `docs/deployment-guide.html`.
- This controlled-change directory and the task scope file.

Out of scope:

- Provider account creation, dashboard mutations, DNS changes, key creation,
  deploys, payment actions, or production environment-variable saves without a
  separate action-time confirmation.
- Changes to application runtime behavior, provider choices, database schema,
  admin UI, public site UI, or deployment infrastructure.
- Persisting screenshots or secrets into the repository.

## Requirements

### REQ-001: Project-scoped browser deployment skill

The repository MUST provide a project-local skill that activates for this
starter's deployment setup/audit workflow and is not installed as a global
Codex skill.

#### AC-001: Skill location and scope

- GIVEN an agent is working in this repository
- WHEN the user asks for deployment browser help
- THEN the instructions are available at `skills/site-deployment-browser/SKILL.md`
  and describe the starter-specific provider map.

### REQ-002: Authenticated dashboard handoff and secret safety

The skill MUST distinguish user-only login/MFA/CAPTCHA steps from agent-run
dashboard navigation and value retrieval. It MUST require action-time
confirmation before resource creation, key creation, environment-variable
saves, deployment, DNS, or payment actions.

#### AC-002: Secrets and screenshots are bounded

- GIVEN a provider dashboard reveals a secret or one-time credential
- WHEN the agent records evidence or screenshots
- THEN the secret is neither printed nor persisted, and screenshots are taken
  only on safe overview/status pages or with secrets hidden.

### REQ-003: Deployment guide records dashboard sources and destinations

The HTML guide MUST include an agent-assisted runbook that maps Supabase,
Cloudflare R2/Pages/DNS, Resend, Railway, and optional ECPay to the values to
retrieve, destinations to configure, and checkpoints to verify.

#### AC-003: Guide can drive provider-by-provider execution

- GIVEN an operator opens `docs/deployment-guide.html`
- WHEN they use the agent-assisted runbook
- THEN they can see each provider's dashboard entry point, required values,
  destination variables, confirmation points, and non-secret verification
  checkpoint.
