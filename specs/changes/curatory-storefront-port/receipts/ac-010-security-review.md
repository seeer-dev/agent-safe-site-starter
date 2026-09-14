# AC-010 security review — guest order access

## Decision

The reference's order-number + email lookup was deliberately not
ported: it is an enumerable endpoint (anyone knowing/guessing an order
number plus a plausible email could read the order). The port keeps the
starter's capability-bound model: `GET /api/orders/{id}` requires the
opaque `X-Order-Access-Token` returned only in the order-creation
response.

## Verified live

- `POST /api/orders` returned order `TW-…` plus `access_token`.
- `GET /api/orders/{id}` without the header → 400.
- With the token → 200, and the response PII is masked
  (masked email/phone fields).
- No order-number+email endpoint exists in `bootstrap/app.go` route
  inventory.
- Order/track pages read the token from sessionStorage recent-orders or
  accept manual entry — the token never lives in the URL.

## Idempotency

Checkout sends `crypto.randomUUID()` as `idempotency_key`; duplicate
submissions return the existing order instead of double-charging.
