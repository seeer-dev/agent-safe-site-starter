# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-09-15. Owner approval recorded via
plain apply; status ran Draft -> Applying -> Accepted. Four controls were
mutation-verified: each was observed failing for a named trigger and green
after restoration, with no mutation left in the diff. The AC-004
security-review receipt is at `receipts/security-review.md`.

Standing gate note: `speccheck`/`scopecheck` in this working tree also report
`admin/influencer-admin-ia.html` and `specs/changes/admin-ia-port/**` —
owner-retained files belonging to a separate pending proposal. They sit
outside this change's `applies_to`; no file inside the change's scope is
uncovered.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | withEdgeAuth builds the accepted set from the non-empty values of EDGE_SECRET and EDGE_SECRET_PREVIOUS (loaded as Config.EdgeSecretPrevious and wired in bootstrap alongside EdgeSecret) and admits any of them before the handler runs. TestEdgeAuthAdmitsRotationWindow proves the current and previous values both reach the handler with 200 while a wrong value and an absent header get the identical generic 403 with no handler run. Mutation-verified: dropping previous from the candidate set turned TestEdgeAuthAdmitsRotationWindow/previous_value and TestEdgeAuthPreviousOnly red. |
| REQ-002 | passed | The candidate set is built only from non-empty values, so with neither variable configured withEdgeAuth returns the inner handler untouched — TestEdgeAuthIsOptIn serves a bare request and one carrying an irrelevant header with 200, and was mutation-verified by replacing the empty-set early return with if false. A previous-only configuration makes that value the sole accepted credential (TestEdgeAuthPreviousOnly), keeping the set semantics uncoupled between the two names. |
| REQ-003 | passed | Every accepted value is compared with subtle.ConstantTimeCompare inside the candidate loop; a rejection runs every candidate and emits one record naming request_id, method, path, and peer — never a configured or supplied value. TestEdgeAuthDisclosesNothing drives a wrong value and a near-miss of the previous value plus a bearer token through the rejection path with both secrets configured and asserts none appear in the response body or the emitted log. |
| REQ-004 | passed | The CFPagesProject field and the CF_PAGES_PROJECT read are deleted from config.go; go build ./... is clean and a repository search finds no live reference — the name survives only inside this change's own spec artifacts, which are the frozen records the acceptance scenario allows. |
| REQ-005 | passed | docs/deployment-guide.html describes the zero-outage sequence (Railway sets EDGE_SECRET_PREVIOUS=old plus EDGE_SECRET=new so the origin accepts both, both Pages projects switch to the new value, then Railway drops the previous value) instead of the brief-outage warning; docs/environment-configuration.md adds the EDGE_SECRET_PREVIOUS ownership row and the same three-step procedure; .env.production.example lists it as a rotation-window-only Railway secret. |
| AC-001 | passed | TestEdgeAuthAdmitsRotationWindow: with EDGE_SECRET and EDGE_SECRET_PREVIOUS configured to different values, the current and previous values are admitted (handler reached, 200); a wrong value and an absent header receive 403 with byte-identical bodies and no handler run. TestEdgeAuthRefusesNonEdgeTraffic additionally proves five failure modes produce indistinguishable bodies under a single-value configuration. |
| AC-002 | passed | TestEdgeAuthIsOptIn: with neither variable configured, a bare request and one carrying an irrelevant header are both served with 200 exactly as before this change. Mutation-verified: removing the empty-set early return turned both cases red. |
| AC-003 | passed | TestEdgeAuthPreviousOnly: with EDGE_SECRET_PREVIOUS configured and EDGE_SECRET empty, the previous value is admitted while a wrong value and an absent header are refused with 403. |
| AC-004 | passed | Constant-time comparison per candidate plus the disclosure gate: with both secrets configured, requests carrying a wrong value and a near-miss of the previous value plus a bearer token are rejected without any configured or supplied value reaching the response body or log, while the record retains rejected, the path, and a request id so a redact-everything implementation cannot pass. Mutation-verified: adding the supplied value to the record leaked it and turned the assertion red. Recorded in receipts/security-review.md. |
| AC-005 | passed | go build ./... is clean after deleting the field and env read; a repository search for CFPagesProject/CF_PAGES_PROJECT finds references only inside specs/changes/edge-secret-rotation/, this change's own normative records. |
| AC-006 | passed | EDGE_SECRET_PREVIOUS is named in .env.production.example (rotation-window-only Railway secret), .env.development.example, docs/environment-configuration.md (ownership row plus procedure), and README.md (EDGE_SECRET bullet). docs/deployment-guide.html names it in the rotation note, which describes origin-accepts-both, edge-switches, origin-drops-old with no outage warning. |
