# ECPay Official Conformance Hardening Evidence

Change ID: ecpay-official-conformance-hardening
Revision: 1
Status: Verifying

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | Official-source review pinned in receipts/source-audit-rev1.md found the ReturnURL TradeAmt and optional SimulatePaid semantics; the branch parser now requires TradeAmt and the payment service verifies amount before treating SimulatePaid=1 as a non-mutating acknowledgement path. |
| AC-001 | passed | Isolated replay in receipts/isolated-protocol-replay-rev1.md compiled the changed callback/payment function bodies and observed SimulatePaid=1 return 1\|OK with zero claims, followed by a real callback for the same MerchantTradeNo acquiring one captured claim. |
| REQ-002 | passed | server/internal/modules/commerce/ecpay.go now includes the official Go apostrophe and tilde corrections plus the existing .NET replacement set; all five pinned official SHA256 vectors matched exactly in receipts/isolated-protocol-replay-rev1.md. |
| AC-002 | passed | Independent protocol replay in receipts/isolated-protocol-replay-rev1.md matched the pinned official baseline, apostrophe, tilde, spaces, and TradeAmt callback SHA256 digests exactly; repository tests copy those same expected digests rather than the provider implementation. |
| REQ-003 | pending | Simulation-to-real authority was independently replayed, but the full repository callback/replay/conflict suite has not yet run on this branch because no GitHub Actions run was created for the API-originated PR events. |
| AC-003 | pending | The branch defines wrong-amount, tamper, simulated, simulation-then-real, and existing durable conflict behavior, but full repository CI remains unobserved for this branch and must pass before acceptance. |
| REQ-004 | passed | Official AIO integration cautions and go-live guidance are pinned in receipts/source-audit-rev1.md; ECPay configuration remains HTTPS-only and now rejects explicit ports, direct IPs, and unencoded Unicode hostnames while keeping stage/production endpoints and secrets server-owned. |
| AC-004 | passed | Independent isolated replay in receipts/isolated-protocol-replay-rev1.md observed explicit :8443, explicit :443, and direct-IP callback origins rejected; repository conformance tests also cover Unicode rejection and punycode acceptance. |
| AC-005 | pending | Canonical README/project/commerce completion status will be updated only after full repository verification succeeds; the detailed audit document already preserves the distinction between source conformance and pending public stage acceptance. |
