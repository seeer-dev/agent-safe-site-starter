# Evidence

## Delivery Status

Revision 2 remains Draft. No product or dependency edit is authorized or implemented. Local implementation evidence is pending, and live Supabase compatibility evidence is environment-blocked.

## Observed Evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | pending |  |
| REQ-002 | pending |  |
| REQ-003 | pending |  |
| REQ-004 | pending |  |
| AC-001 | pending |  |
| AC-002 | pending |  |
| AC-003 | pending |  |
| AC-004 | pending |  |
| AC-005 | pending |  |
| AC-006 | blocked | Requires non-secret signing, issuer, audience, access-token-lifetime, protected-endpoint, outage, and explicit rollback observations from the actual Supabase deployment environment. |

## Planning Evidence

- Expansion inspected HEAD `bc1d17f10d258c337efab975466949c92a5ec956` and the current auth/config/bootstrap seams.
- `go test ./server/internal/auth ./server/internal/config -count=1` passed before revision 2 planning; this is baseline evidence only and does not prove the proposed behavior.
- The proposed `github.com/lestrrat-go/jwx/v3` v3.2.0 module declares Go 1.25, matching `go.mod`; it is not yet a repository dependency.
