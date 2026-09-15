# Edge Secret Rotation Specification

Change ID: edge-secret-rotation
Revision: 1
Status: Accepted
Decision authority: Repository owner/user
Approval basis: Repository owner approved revision 1 on 2026-09-15 via plain apply on the sole review-ready proposal. The deployment guide documented EDGE_SECRET rotation, but the origin accepted exactly one value, so any documented order produced a 403 window; and CFPagesProject was loaded by config but consumed by nothing.
Repository baseline: 3a196c39ce746d1bf9c078a137fb124f879289ae
Supersedes: none

## Outcome

Operators can rotate `EDGE_SECRET` without an API outage: the origin accepts a
previous value alongside the current one for the length of a rotation window,
then the previous value is removed.

As ride-along hygiene in the same config surface, the `CFPagesProject` field —
loaded from `CF_PAGES_PROJECT` but consumed by nothing since the publish tool
became trigger-only — is removed so configuration only carries live settings.

## Scope

In scope:

- A second optional credential, `EDGE_SECRET_PREVIOUS`, accepted alongside
  `EDGE_SECRET` during a rotation window.
- Removal of the dead `CFPagesProject` configuration field and its env read.
- Operator documentation of the zero-outage rotation procedure.

Out of scope:

- More than two simultaneously accepted values. One current plus one previous
  covers a rotation; arbitrary credential lists are not needed.
- Automatic rotation, secret generation, or provider API integration.
- Changing what the credential proves. It still authenticates the hop, never
  the caller, and never influences authorization.

## Decisions and invariants

- The accepted credential set is exactly the non-empty values of
  `EDGE_SECRET` and `EDGE_SECRET_PREVIOUS`. With neither set the check stays
  disabled — the opt-in boundary from `edge-origin-authentication` is
  unchanged. `EDGE_SECRET_PREVIOUS` alone therefore acts as the sole accepted
  value, which keeps the semantics a pure set with no coupling between the
  two names.
- Every accepted value is compared with `crypto/subtle.ConstantTimeCompare`.
- A rejection still tells the caller nothing about why: one generic 403 body
  and one log record shape regardless of which value — or none — was sent.
- Neither a configured value nor a supplied value ever appears in a response
  or a log record.
- `/healthz` stays exempt so the platform probe keeps working mid-rotation.
- The documented rotation order becomes: set `EDGE_SECRET_PREVIOUS=old` and
  `EDGE_SECRET=new` on the origin and redeploy (both values accepted, no
  outage) → point both Pages projects at the new value and redeploy → remove
  `EDGE_SECRET_PREVIOUS` on the origin and redeploy. No step refuses the
  credential the other side is still sending.

## Requirements

### REQ-001: The origin accepts a rotation window

When `EDGE_SECRET_PREVIOUS` is configured, a request presenting either the
current or the previous secret MUST be admitted; any other value MUST be
rejected.

#### AC-001: Current and previous both admit

- GIVEN `EDGE_SECRET` and `EDGE_SECRET_PREVIOUS` configured to different values
- WHEN requests present the current value, the previous value, a wrong value,
  or no value
- THEN the two configured values MUST be admitted and the others MUST receive
  the identical generic 403, with no handler running.

### REQ-002: The check stays opt-in and uniform

With neither variable configured, behavior MUST be exactly as before this
change; with only `EDGE_SECRET_PREVIOUS` configured, it MUST act as the sole
accepted value.

#### AC-002: No secrets means no change

- GIVEN neither `EDGE_SECRET` nor `EDGE_SECRET_PREVIOUS` is configured
- WHEN any request arrives, with or without the header
- THEN it MUST be served exactly as before this change.

#### AC-003: Previous-only configuration still guards the origin

- GIVEN `EDGE_SECRET_PREVIOUS` configured and `EDGE_SECRET` empty
- WHEN a request presents the previous value versus a wrong or absent value
- THEN the previous value MUST be admitted and the rest refused with 403.

### REQ-003: Comparison and logging stay safe for multiple values

Each configured value MUST be compared in constant time, and neither a
configured nor a supplied value may appear in a response or log record.

#### AC-004: No timing or disclosure leak across the credential set

- GIVEN both secrets configured and requests carrying a wrong value, a
  near-miss of the previous value, and a bearer token
- WHEN comparisons run and rejections are recorded
- THEN every comparison MUST use a constant-time primitive, and no configured
  or supplied value may appear in the response body or any emitted log
  record.

### REQ-004: Configuration carries no dead fields

`Config` MUST NOT load or expose a setting that no code consumes.
`CFPagesProject` and the `CF_PAGES_PROJECT` read MUST be removed.

#### AC-005: Dead field removed without dangling references

- GIVEN the field and its env read are deleted
- WHEN the module compiles and the repository is searched for
  `CFPagesProject` / `CF_PAGES_PROJECT`
- THEN compilation MUST succeed and no live reference may remain (historical
  specs and INTEGRATION_PLAN.md excepted as frozen records).

### REQ-005: Operators can find the zero-outage procedure

The deployment guide, the environment inventory, and the production env
example MUST describe `EDGE_SECRET_PREVIOUS` and the three-step rotation
order, replacing the brief-outage wording added when only one value was
accepted.

#### AC-006: Rotation procedure documented where operators look

- GIVEN the implemented dual-value behavior
- WHEN an operator reads `docs/deployment-guide.html`,
  `docs/environment-configuration.md`, or `.env.production.example`
- THEN each MUST name `EDGE_SECRET_PREVIOUS`, and the guide MUST describe the
  zero-outage sequence (origin accepts old+new → edge switches to new →
  origin drops old) instead of warning about a brief outage.

## Amendments

None.
