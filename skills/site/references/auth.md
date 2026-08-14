# Auth

Supabase Auth authenticates production users; Go authorizes actions.

Browser sends the Supabase access token to Go. Go turns it into explicit `auth.Principal`. Services receive Principal as a normal argument when authorization matters.

Do not query application tables directly from the browser via Supabase. Do not move business authorization into frontend route guards.

## Recovery preserves authorization strength

Recovery is a separate authentication path, not an exception to object-level authorization. An order ID, username, email address, phone number, or other identifying/contact data does not prove control of the protected object.

Before approving or implementing account, order, token, or secret recovery, record:

| Question | Required answer |
|---|---|
| Protected object and attacker knowledge | What identifiers, contact data, logs, URLs, or support records may already be known? |
| Possession proof | Which approved factor proves control, and why is it no weaker than the original access path? |
| Lifetime and reuse | Expiry, single-use behavior, replay handling, and invalidation |
| Abuse controls | Per-subject and per-origin rate limits, enumeration-safe responses, and audit events |
| Rotation | Atomic replacement, concurrent requests, old-secret invalidation, and rollback behavior |
| Delivery | No plaintext secret at rest or in logs; no sensitive query-string exposure |
| Recovery failure | A lost response, unavailable provider, or invalid proof must not expose data or silently weaken authorization |

Acceptable proof may include an authenticated principal, a short-lived single-use code delivered through a verified channel, or a high-entropy recovery credential the client already possessed before the protected response was lost and whose server-side representation is hashed. Contact-field equality alone is never sufficient.

Recovery behavior changes permissions or data handling. If the approved spec does not already decide the factor, delivery channel, expiry, abuse controls, and rotation semantics, record a blocker and obtain decision authority before product edits. Use the `security-review` skill and attach its receipt to the mapped evidence.
