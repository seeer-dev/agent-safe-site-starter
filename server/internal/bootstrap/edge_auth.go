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
// The check is opt-in: an empty secret returns the handler untouched, so local
// development, tests, and existing deployments are unaffected until an operator
// configures one.
func withEdgeAuth(secret string, next http.Handler) http.Handler {
	if secret == "" {
		return next
	}
	want := []byte(secret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthPath {
			next.ServeHTTP(w, r)
			return
		}

		got := []byte(r.Header.Get(edgeSecretHeader))
		// subtle.ConstantTimeCompare returns 0 for unequal lengths without
		// comparing contents, so a length check would not add anything and a
		// byte-by-byte comparison would leak the secret to a patient caller.
		if subtle.ConstantTimeCompare(got, want) != 1 {
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
