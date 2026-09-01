# ECPay implementation precheck — revision 1

This receipt intentionally starts before formal acceptance.

Draft PR: `#15`

The branch now contains the protocol corrections identified by the official-source audit:

- ReturnURL amount uses `TradeAmt`.
- `SimulatePaid=1` cannot capture the durable order.
- Go CheckMacValue includes the official apostrophe encoding rule.
- Explicit HTTPS ports other than 443 are rejected for ECPay public origins.
- ItemName is capped at the official recommended 200-character operating boundary.
- OpenAPI and the runtime/OpenAPI contract checker require the corrected callback shape.
- Unit tests include the official SHA256 vectors copied from the pinned ECPay test-vector file.

No acceptance claim is made by this receipt. CI and independent review evidence are added after the branch is exercised.
