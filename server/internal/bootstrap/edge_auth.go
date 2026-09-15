package bootstrap

import (
	"crypto/subtle"
	"log/slog"
	"net/http"

	"github.com/example/ai-site-starter/server/internal/httpx"
)

// edgeSecretHeader carries the shared secret the edge injects. The name is
// deliberately generic: it says nothing about which provider is in front.
const edgeSecretHeader = "X-Edge-Secret"

// healthPath stays reachable without the secret. The platform probes it
// directly rather than through the edge (railway.toml sets healthcheckPath),
// so guarding it would fail every probe and take the deployment down.
const healthPath = "/healthz"

// withEdgeAuth refuses requests that did not arrive through the edge.
//
// This authenticates the *hop*, not the caller. It proves a request traversed
// the proxy that injects the secret; it says nothing about who sent it and must
// never influence an authorization decision.
//
// It exists because edge protection is otherwise only as strong as the origin
// being undiscoverable. With no origin check, anyone who finds the origin
// hostname bypasses every edge rule. Certificate Transparency logs and
// historical DNS make that discovery a matter of effort rather than luck.
//
// The check is opt-in: with no configured value it returns the handler
// untouched, so local development, tests, and existing deployments are
// unaffected until an operator configures one.
//
// The accepted set is the non-empty values of current and previous. previous
// exists for rotation windows: while the edge still sends the outgoing value,
// the origin already accepts the incoming one, so rotating never produces a
// 403 gap. A previous-only configuration simply makes that value the sole
// accepted credential — the set has no coupling between the two names.
func withEdgeAuth(current, previous string, next http.Handler) http.Handler {
	var accepted [][]byte
	for _, secret := range []string{current, previous} {
		if secret != "" {
			accepted = append(accepted, []byte(secret))
		}
	}
	if len(accepted) == 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthPath {
			next.ServeHTTP(w, r)
			return
		}

		got := []byte(r.Header.Get(edgeSecretHeader))
		// subtle.ConstantTimeCompare returns 0 for unequal lengths without
		// comparing contents, so a length check would not add anything and a
		// byte-by-byte comparison would leak the secret to a patient caller.
		matched := false
		for _, want := range accepted {
			if subtle.ConstantTimeCompare(got, want) == 1 {
				matched = true
				break
			}
		}
		if !matched {
			// The record names the request, never the value. A rejected
			// attempt is worth seeing; the supplied secret is not, and neither
			// is whether it was absent, empty, or merely wrong.
			slog.Warn("request rejected: missing or invalid edge credential",
				slog.String("request_id", httpx.RequestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("peer", peerAddress(r)),
			)
			// One response for every failure mode. Distinguishing absent from
			// wrong would tell a prober whether the header name is right.
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}

		next.ServeHTTP(w, r)
	})
}
