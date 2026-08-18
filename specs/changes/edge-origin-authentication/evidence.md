# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-08-18. Three controls were mutation-verified: each was observed failing for a named trigger and green after restoration, with no mutation left in the diff.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | withEdgeAuth rejects any request that does not present the configured secret in X-Edge-Secret before the handler runs, wired in bootstrap as withRequestObservability(withEdgeAuth(cfg.EdgeSecret, withCORS(...))) so a rejection is still recorded and a bypass attempt is refused rather than answered with CORS headers. |
| REQ-002 | passed | An empty EDGE_SECRET returns the inner handler untouched, so behavior is unchanged until an operator opts in. The health path is exempt because the platform probes it directly per railway.toml. |
| REQ-003 | passed | Comparison uses crypto/subtle.ConstantTimeCompare. The rejection record names method, path, request id, and peer, and never the configured or supplied value. |
| AC-001 | passed | TestEdgeAuthRefusesNonEdgeTraffic covers absent, empty, wrong, prefix-only, and trailing-space values: each returns 403, no handler runs, and the response never echoes the secret. The bodies of all five failure modes are compared against each other and must be identical, so a prober cannot distinguish them. TestEdgeAuthAdmitsValidCredential confirms the correct value is admitted. |
| AC-002 | passed | TestEdgeAuthIsOptIn serves both a bare request and one carrying an irrelevant header with no secret configured; both reach the handler with 200. Mutation-verified: removing the empty-secret early return turned it red. |
| AC-003 | passed | TestEdgeAuthExemptsHealthProbe requests the health path without the header while a secret is configured and asserts the handler is reached. Mutation-verified: replacing the path check with a constant false produced status 403 and turned it red. |
| AC-004 | passed | Constant-time comparison plus a disclosure gate driving a near-miss supplied value and a bearer token into the rejection path; neither reaches the response nor the log record, while the record retains rejected, the path, and a request id so an over-broad redactor cannot pass. Mutation-verified: adding the supplied value to the record leaked it and turned the assertion red. Recorded in receipts/security-review.md. |
