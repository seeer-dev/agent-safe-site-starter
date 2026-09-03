# ECPay Official Conformance Hardening Evidence

Change ID: ecpay-official-conformance-hardening
Revision: 2
Status: Accepted

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | Revision-2 official-source audit pins the AIO ReturnURL shape and the branch parser now requires TradeAmt, validates optional SimulatePaid, and correlates the returned amount with the durable TWD attempt before financial authority is considered. |
| AC-001 | passed | Revision-2 independent protocol replay observed signed simulation and signed non-success callbacks acknowledged with zero success claims, followed by a later real callback for the same MerchantTradeNo acquiring exactly one captured claim. |
| REQ-002 | passed | The Go CheckMacValue implementation includes the official apostrophe and tilde corrections plus the seven compatibility replacements, case-insensitive sorting, SHA256, and uppercase output; all five pinned official AIO vectors matched exactly in revision-2 replay. |
| AC-002 | passed | Independent revision-2 replay matched the pinned official baseline, apostrophe, tilde, spaces, and callback SHA256 digests exactly and also observed post-signature callback tampering rejected. |
| REQ-003 | passed | Revision-2 authority replay rejected wrong amount, tampered signature, wrong MerchantID, and invalid SimulatePaid; simulation and pending states consumed no success claim, while the later valid real success captured once. The durable conflicting-success CAS boundary is unchanged from the previously Accepted ECPay payment-flow implementation. |
| AC-003 | passed | Independent revision-2 falsification covered wrong TradeAmt, signature tamper, wrong MerchantID, simulated success, signed pending result, invalid SimulatePaid, and later real success; only the valid real-success path reached the one-effect durable claim boundary. |
| REQ-004 | passed | Revision-2 official-source audit keeps credentials server-only, finite official AIO endpoints, exact provider acknowledgement semantics, and HTTPS callback transport on standard port 443 while retaining deployment-only stage acceptance as a separate gate. |
| AC-004 | passed | Independent revision-2 replay accepted implicit HTTPS 443, explicit HTTPS port 443, and punycode DNS while rejecting non-standard HTTPS port 8443, direct IP, raw Unicode hostname, and non-HTTPS origins. |
| AC-005 | passed | README, project status, commerce acceptance, and the dedicated ECPay audit now mark pinned source-level official conformance complete while explicitly leaving one public HTTPS ECPay stage transaction as deployment and go-live acceptance; the missing PR Actions run is separately recorded rather than reported as green. |
