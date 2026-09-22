package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"

	"github.com/example/ai-site-starter/server/internal/httpx"
	"github.com/example/ai-site-starter/server/internal/platform/abuse"
)

// edgeClientKeyHeader carries the opaque per-client key the Pages Function
// derives from Cloudflare's edge-observed address. The origin trusts it only
// because the edge-secret middleware already authenticated the hop; a
// browser can supply the header but cannot traverse the edge unstamped.
const edgeClientKeyHeader = "X-Edge-Client-Key"

// abuseGuard applies the deployment-wide fixed-window budget to one public
// mutation or expensive endpoint before the handler runs — and therefore
// before JSON decoding, persistence, mail, inventory, or idempotency work.
//
// The client key comes from X-Edge-Client-Key when an edge credential is
// configured. A missing or malformed key fails closed rather than falling
// back to X-Forwarded-For or CF-Connecting-IP, both of which are
// caller-supplied and must never mint an identity. When no edge credential
// exists (local development and tests), the transport peer address is
// hashed into an equivalent opaque key so the same code path still runs.
func abuseGuard(limiter *abuse.Limiter, edgeBound bool, bucket string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := abuseClientKey(r, edgeBound)
		if !ok {
			// Same refusal shape as edge authentication: one response for
			// every identity failure mode tells a prober nothing about which
			// layer rejected the request.
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		err := limiter.Allow(r.Context(), bucket, key)
		var over *abuse.ErrOverQuota
		switch {
		case err == nil:
			// allowed
		case errors.As(err, &over):
			// Whole seconds; at least one so clients wait out the window.
			seconds := int(over.RetryAfter.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			httpx.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		case errors.Is(err, abuse.ErrInvalidClientKey):
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		default:
			// The shared store is the authority; when it is unavailable the
			// endpoint fails closed rather than degrading to a process-local
			// counter that replicas would not share.
			httpx.Error(w, http.StatusServiceUnavailable, "service unavailable")
			return
		}
		next(w, r)
	}
}

// abuseClientKey resolves the opaque limiter identity for this request.
// edgeBound reports whether an edge credential is configured; in that mode
// only the edge-stamped header is trusted. In no-edge mode the transport
// peer is hashed so the limiter table still stores no raw address.
func abuseClientKey(r *http.Request, edgeBound bool) (string, bool) {
	if edgeBound {
		key := r.Header.Get(edgeClientKeyHeader)
		if len(key) != sha256.Size*2 {
			return "", false
		}
		if _, err := hex.DecodeString(key); err != nil {
			return "", false
		}
		return key, true
	}
	sum := sha256.Sum256([]byte("abuse-local-peer\x00" + peerAddress(r)))
	return hex.EncodeToString(sum[:]), true
}
