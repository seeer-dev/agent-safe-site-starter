# Security review — edge credential rotation window

Change ID: edge-secret-rotation
Revision: 1
Covers: AC-004 (review of the multi-value diff; REQ-001..REQ-003 properties
were exercised by the named tests cited below)
Reviewed: 2026-09-15

## What changed under review

`withEdgeAuth` now admits a **set** — the non-empty values of `EDGE_SECRET`
and `EDGE_SECRET_PREVIOUS` — instead of exactly one value. The purpose is a
rotation window: the origin accepts the outgoing credential while the edge is
switched, then the outgoing name is unset. `CFPagesProject` was removed as
dead configuration; it had no consumer and its removal shrinks the config
surface.

The credential still authenticates the **hop**, never the caller, and nothing
consumes it for an authorization decision. That boundary is unchanged.

## Controls verified in the diff

1. **Constant-time comparison for every candidate.** Each accepted value is
   compared with `crypto/subtle.ConstantTimeCompare`; no byte-wise or
   length-shortcut comparison was introduced. On rejection the loop runs every
   candidate — refusal timing does not depend on which position would have
   matched. A match breaks early, but the only party who can observe that
   already holds a valid credential and learns nothing beyond the admission it
   was entitled to.
2. **Uniform rejection preserved.** Wrong value, near-miss of the previous
   value, absent header, and no credential configured-singly all produce the
   same 403 body and the same log record shape; the new dual-value table test
   asserts body equality across failure modes, not just status codes.
3. **No disclosure of configured or supplied values.** The rejection record
   still names only request_id, method, path, and peer. `TestEdgeAuthDisclosesNothing`
   now drives a wrong value AND a near-miss of the *previous* value plus a
   bearer token through the rejection path and asserts none reach the response
   body or the log; it also requires `rejected`, the path, and `request_id=`
   to survive so a log-nothing "fix" cannot pass.
4. **Opt-in boundary unchanged.** The accepted set is built only from
   non-empty values, so the both-unset case returns the inner handler
   untouched — the edge-origin-authentication opt-in contract holds, and a
   previous-only configuration simply makes that value the sole credential.
5. **`/healthz` still exempt.** The platform probe cannot carry the header;
   the exemption survives the multi-value change unchanged.

## Mutation evidence

Each new or modified check was observed failing for a named trigger, then
green after restoration; no mutation is left in the diff.

| Control | Mutation | Observed failure |
|---|---|---|
| Rotation window | dropped `previous` from the candidate set | `TestEdgeAuthAdmitsRotationWindow/previous_value` and `TestEdgeAuthPreviousOnly/{wrong_value,no_header_at_all}` failed |
| Opt-in default | replaced the empty-set early return with `if false` | `TestEdgeAuthIsOptIn/{no_header,irrelevant_header}` failed |
| Non-disclosure | added `slog.String("supplied", r.Header.Get(edgeSecretHeader))` to the record | `TestEdgeAuthDisclosesNothing` failed with both supplied values visible in the emitted lines |
| Env loading | replaced the `EDGE_SECRET_PREVIOUS` read with a constant | `TestLoadEdgeSecrets` failed: `EdgeSecretPrevious: got ""` |

## Residual

- **Two live credentials during the window.** Accepting the outgoing value
  doubles the valid-credential surface for the length of the rotation. The
  window is operator-bounded, and the deployment guide, env examples, and
  environment inventory all instruct removing `EDGE_SECRET_PREVIOUS` once the
  edge sends only the new value.
- **A forgotten `EDGE_SECRET_PREVIOUS` is silent.** Nothing expires it — an
  operator who skips step 3 leaves the old credential valid indefinitely.
  This is documented rather than enforced; there is no mechanism in scope to
  bound the window automatically.
- **The set semantics are intentionally uncoupled.** `EDGE_SECRET_PREVIOUS`
  alone is a valid sole credential. That is the specified behavior (a pure
  set), not a misconfiguration guard — an operator who mistypes the variable
  name still gets a working, guarded origin.
- **No deployment-level proof.** Tests prove the origin's behavior, not that
  a given Railway/Pages deployment actually configured the values. The
  operator verifies the live rotation by observing old and new both admitted.
