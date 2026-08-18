# Security review — edge-to-origin shared secret

Change ID: edge-origin-authentication
Revision: 1
Covers: AC-004
Reviewed: 2026-08-18

## What this control does and does not do

It authenticates the **hop**. A request carrying the secret traversed the proxy
that injects it. That is all. It is not an identity, not a capability, and
nothing in this repository consumes it for an authorization decision.

It is a **precondition** for edge-based abuse control, not abuse control. Rate
limiting remains with `public-endpoint-rate-limit` and its unresolved owner
decisions.

## Why it exists

Edge rules only apply to traffic that reaches the edge. Before this change the
Go API had no edge-to-origin check at all — `CF-Connecting-IP`, a shared
secret, and mTLS appear nowhere under `server/internal` — so any WAF or rate
limit could be bypassed by connecting to the origin directly.

The counter-argument is that the origin hostname is not published. That is
obscurity, not access control: Certificate Transparency logs record every
certificate issued, and historical DNS survives a later proxy change. This
control converts "hard to find" into "refused".

## Controls

1. **Constant-time comparison.** `crypto/subtle.ConstantTimeCompare`. It
   returns 0 for unequal lengths without comparing contents, so an explicit
   length check adds nothing, and a byte-by-byte comparison would leak the
   secret to a patient caller.
2. **Uniform rejection.** Absent, empty, wrong, and prefix-correct all produce
   the same 403 and the same body. A prober cannot learn from the response
   whether the header name is even right. Asserted by comparing bodies across
   all five failure cases rather than checking each in isolation.
3. **The value is never recorded.** The log record names the request — method,
   path, request id, peer — and never the supplied or configured value.
4. **Opt-in.** An empty secret returns the handler untouched, so this cannot be
   enabled by accident and does not change local development.
5. **Health probe exempt.** `railway.toml` sets `healthcheckPath = /healthz`
   and the platform probes it directly, so guarding it would fail every probe
   and take the deployment down.

## Mutation evidence

Each control was observed failing for a named trigger, then green after
restoration, with no mutation left in the diff.

| Control | Mutation | Observed failure |
|---|---|---|
| Non-disclosure | added `slog.String("supplied", string(got))` to the record | `log record leaked "attacker-guess-edge-sec"` with the value visible in the emitted line |
| Health exemption | replaced the path check with `if false` | `the health path must stay reachable without the edge credential, got status 403` |
| Opt-in default | removed the empty-secret early return | `TestEdgeAuthIsOptIn/irrelevant_header` failed |

The disclosure test supplies a **near-miss** value (`attacker-guess-` plus the
first eight characters of the real secret) precisely so a naive "log what we
received" implementation cannot pass it, and asserts the record still contains
`rejected`, the path, and a request id — a redactor that logged nothing would
otherwise satisfy the negative checks while leaving a bypass attempt invisible.

## Residual

- **Rotation has a window.** Changing the value at the edge and the origin is
  not atomic. Running two edge rules briefly, or accepting a short window of
  refusals, is an operator choice; the code accepts exactly one value.
- **The secret is only as private as its configuration.** It sits in the
  Railway environment and in an edge rule. Anyone with access to either has it.
- **This does not bound request volume.** An attacker who obtains the secret,
  or traffic that legitimately passes the edge, is unaffected. Abuse control is
  a separate change.
- **No test proves the deployment is actually closed.** That requires the edge
  rule to exist and the origin to be configured, neither of which is visible
  from this repository. The operator must verify it.
