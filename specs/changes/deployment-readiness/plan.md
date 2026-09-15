# Deployment Readiness Plan

Change ID: deployment-readiness
Revision: 1
Status: Verifying

Normative specification: [`spec.md`](spec.md)

## Scope Lock

- `specs/changes/deployment-readiness/**`
- `Makefile`
- `Dockerfile`
- `admin/public/_redirects`
- `README.md`
- `docs/**`

## Slices

### S01 — Build/deploy artifact fixes

Makefile THEME resolution, Dockerfile writable var dir,
admin/public/_redirects.

Covers: REQ-001, REQ-002, REQ-003 (AC-001, AC-002, AC-003)

### S02 — Documentation

README production section rewrite + docs/deployment-guide.html
single-page guide with checklist.

Covers: REQ-004 (AC-004)

## Traceability

| REQ / AC | Slice | Evidence |
|---|---|---|
| REQ-001 / AC-001 | S01 | make -n theme in-container, both branches |
| REQ-002 / AC-002 | S01 | build:only emits dist/_redirects |
| REQ-003 / AC-003 | S01 | container migrate+api smoke test |
| REQ-004 / AC-004 | S02 | cross-check vs config artifacts |
