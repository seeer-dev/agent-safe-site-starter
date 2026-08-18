# Single Command Site Build Specification

Change ID: single-command-site-build
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved the pre-launch operability batch on 2026-08-18. Review found that a fresh clone cannot render, because the theme bundle is git-ignored and no documented command builds it before the renderer runs.
Repository baseline: 75ebf8b896cd484f562e9ce00afe89bc125654f2
Supersedes: none

## Outcome

One documented command takes a fresh clone to a rendered site, and the
deployment instructions name a command that works rather than one that fails on
a clean checkout.

No renderer behavior, route, template, or output format changes.

## Scope

In scope:

- A make target that builds the theme bundle before invoking the renderer.
- The deployment documentation that currently names the renderer alone.

Out of scope:

- Changing `validateThemeAssets`. Failing closed on a missing bundle is
  correct; the problem is that nothing documented produces the bundle.
- Making the renderer shell out to npm. That would put a Node dependency
  inside a Go tool and hide the ordering rather than state it.
- Committing built theme assets.

## Decisions and invariants

- The ordering stays explicit. A caller can still run each step alone.
- The renderer keeps failing closed when the bundle is absent. This change
  removes the surprise, not the guard.
- The existing `render` target keeps its current meaning so nothing that
  already builds the theme separately breaks.

## Requirements

### REQ-001: A fresh clone can produce a rendered site

The repository MUST provide a documented command that builds the theme bundle
and then renders, in that order, without the caller knowing the dependency.

#### AC-001: One command covers the ordering

- GIVEN a checkout whose theme dist directory is absent
- WHEN the documented single command runs
- THEN the theme bundle MUST be built and the renderer MUST then succeed;
  and WHEN the renderer alone runs against that same absent bundle
  THEN it MUST still fail closed with its existing diagnostic.

### REQ-002: Deployment instructions name a command that works

Documentation that tells an operator what to configure as a build command MUST
name one that succeeds on a clean checkout.

#### AC-002: Documented build command succeeds from clean

- GIVEN the deployment section describing the Cloudflare Pages build command
- WHEN an operator follows it on a clean checkout
- THEN the named command MUST include the theme build step, and the ordering
  dependency MUST be stated rather than left implicit.
