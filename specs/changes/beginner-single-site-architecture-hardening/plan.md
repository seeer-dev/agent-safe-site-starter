# Beginner Single-Site Architecture Hardening Delivery Plan

Change ID: beginner-single-site-architecture-hardening
Revision: 1
Status: Accepted

Repository baseline: `ade97a8d58dfffe1d2b61633d552dfffcb7ba3f6`

## Scope lock

- `architecture.yaml`
- `AGENTS.md`
- `.github/workflows/ci.yml`
- `server/tools/archcheck/**`
- `skills/site/SKILL.md`
- `skills/site/references/architecture-boundaries.md`
- `skills/site/references/change-safety.md`
- `specs/changes/beginner-single-site-architecture-hardening/**`

## Slices

### Slice 1: Versioned manifest-driven import enforcement

Covers REQ-001, AC-001, REQ-002, AC-002.

- add versioned import roots and hard deny semantics to `architecture.yaml`;
- make `server/tools/archcheck/**` consume that policy;
- reject invalid roots, unsupported hard-rule values, and zero protected-file coverage;
- run archcheck regression tests in `.github/workflows/ci.yml` before the real repository scan.

### Slice 2: Starter-owned architecture guidance

Covers REQ-003, AC-003, REQ-004, AC-004.

- update `AGENTS.md` and `skills/site/SKILL.md`;
- add `skills/site/references/architecture-boundaries.md`;
- align `skills/site/references/change-safety.md`;
- preserve single-site product shape, typed ports, cohesion-first growth, UI-contract preservation, and propose-review-apply.

### Slice 3: Controlled evidence and merge

- keep `specs/changes/beginner-single-site-architecture-hardening/**` synchronized with revision 1;
- targeted architecture checks passed in PR #2 CI run 32968611776;
- ensure a fresh CI checkout installs/builds the theme bundle before renderer-dependent Go tests;
- provide the CI PostgreSQL test URL to `go test ./...` because live database tests intentionally fail rather than skip under CI;
- preserve the existing scope-workflow documentation contract while extending site guidance;
- rerun the full repository CI after Accepted status;
- merge only after every CI gate succeeds.
